package proxy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// CacheEntry represents a cached response
type CacheEntry struct {
	Key         string        `json:"key"`
	Response    []byte        `json:"response"`
	StatusCode  int           `json:"status_code"`
	Headers     map[string]string `json:"headers"`
	CreatedAt   time.Time     `json:"created_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	Size        int64         `json:"size"`
	AccessCount int64         `json:"access_count"`
	LastAccess  time.Time     `json:"last_access"`
}

// RequestCache implements request/response caching
type RequestCache struct {
	entries map[string]*CacheEntry
	mu      sync.RWMutex
	config  CacheConfig
	stats   CacheStats
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	MaxEntries    int
	MaxSizeBytes  int64
	DefaultTTL    time.Duration
	MaxTTL        time.Duration
	CleanupInterval time.Duration
}

// CacheStats holds cache statistics
type CacheStats struct {
	Hits        int64 `json:"hits"`
	Misses      int64 `json:"misses"`
	Evictions   int64 `json:"evictions"`
	TotalEntries int64 `json:"total_entries"`
	TotalSize   int64 `json:"total_size_bytes"`
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		MaxEntries:      1000,
		MaxSizeBytes:    100 * 1024 * 1024, // 100MB
		DefaultTTL:      5 * time.Minute,
		MaxTTL:          30 * time.Minute,
		CleanupInterval: 10 * time.Minute,
	}
}

// NewRequestCache creates a new request cache
func NewRequestCache(config CacheConfig) *RequestCache {
	if config.MaxEntries == 0 {
		config = DefaultCacheConfig()
	}

	cache := &RequestCache{
		entries: make(map[string]*CacheEntry),
		config:  config,
		stats: CacheStats{
			Hits:        0,
			Misses:      0,
			Evictions:   0,
			TotalEntries: 0,
			TotalSize:   0,
		},
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached response
func (c *RequestCache) Get(key string) (*CacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, exists := c.entries[key]
	if !exists {
		c.stats.Misses++
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		delete(c.entries, key)
		c.stats.TotalEntries--
		c.stats.TotalSize -= entry.Size
		c.stats.Misses++
		return nil, false
	}

	// Update access statistics
	entry.AccessCount++
	entry.LastAccess = time.Now()
	c.stats.Hits++

	return entry, true
}

// Put stores a response in the cache
func (c *RequestCache) Put(key string, response []byte, statusCode int, headers map[string]string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Apply TTL constraints
	if ttl == 0 {
		ttl = c.config.DefaultTTL
	}
	if ttl > c.config.MaxTTL {
		ttl = c.config.MaxTTL
	}

	entry := &CacheEntry{
		Key:        key,
		Response:   response,
		StatusCode: statusCode,
		Headers:    headers,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(ttl),
		Size:       int64(len(response)),
		AccessCount: 0,
		LastAccess: time.Now(),
	}

	// Check if we need to make space
	c.makeSpace(entry.Size)

	// Store the entry
	c.entries[key] = entry
	c.stats.TotalEntries++
	c.stats.TotalSize += entry.Size

	// Update stats
	if c.stats.TotalEntries > 0 {
		c.stats.Hits = c.stats.Hits
		c.stats.Misses = c.stats.Misses
	}
}

// makeSpace ensures there's enough room for a new entry
func (c *RequestCache) makeSpace(newEntrySize int64) {
	// Check if we're at capacity
	if len(c.entries) < c.config.MaxEntries && c.stats.TotalSize+newEntrySize <= c.config.MaxSizeBytes {
		return
	}

	// Remove expired entries first
	c.removeExpired()

	// If still need space, remove oldest entries (LRU)
	if c.stats.TotalSize+newEntrySize > c.config.MaxSizeBytes || len(c.entries) >= c.config.MaxEntries {
		c.evictOldest(newEntrySize)
	}
}

// removeExpired removes all expired entries
func (c *RequestCache) removeExpired() {
	now := time.Now()
	var toRemove []string

	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			toRemove = append(toRemove, key)
		}
	}

	for _, key := range toRemove {
		entry := c.entries[key]
		delete(c.entries, key)
		c.stats.TotalEntries--
		c.stats.TotalSize -= entry.Size
	}
}

// evictOldest removes the oldest accessed entries
func (c *RequestCache) evictOldest(spaceNeeded int64) {
	// Sort by last access time
	type entryInfo struct {
		key        string
		lastAccess time.Time
		entry      *CacheEntry
	}

	var entries []entryInfo
	for key, entry := range c.entries {
		entries = append(entries, entryInfo{
			key:        key,
			lastAccess: entry.LastAccess,
			entry:      entry,
		})
	}

	// Sort by last access (oldest first)
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[i].lastAccess.After(entries[j].lastAccess) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// Evict oldest entries until we have enough space
	for _, info := range entries {
		if c.stats.TotalSize+spaceNeeded <= c.config.MaxSizeBytes && len(c.entries) < c.config.MaxEntries {
			break
		}

		delete(c.entries, info.key)
		c.stats.TotalEntries--
		c.stats.TotalSize -= info.entry.Size
		c.stats.Evictions++
	}
}

// cleanup removes expired entries periodically
func (c *RequestCache) cleanup() {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		c.removeExpired()
		c.mu.Unlock()
	}
}

// GetStats returns cache statistics
func (c *RequestCache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// Invalidate removes entries matching a prefix
func (c *RequestCache) Invalidate(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var toRemove []string
	for key := range c.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			toRemove = append(toRemove, key)
		}
	}

	for _, key := range toRemove {
		entry := c.entries[key]
		delete(c.entries, key)
		c.stats.TotalEntries--
		c.stats.TotalSize -= entry.Size
	}
}

// Clear removes all entries from the cache
func (c *RequestCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.stats = CacheStats{
		Hits:        c.stats.Hits,
		Misses:      c.stats.Misses,
		Evictions:   c.stats.Evictions,
		TotalEntries: 0,
		TotalSize:   0,
	}
}

// GenerateCacheKey generates a cache key from request parameters
func GenerateCacheKey(method string, url string, body []byte, headers map[string]string) string {
	// Create a hash of method + url + body + relevant headers
	hashData := method + ":" + url + ":"

	if body != nil && len(body) > 0 {
		bodyHash := sha256.Sum256(body)
		hashData += hex.EncodeToString(bodyHash[:])
	}

	// Add relevant headers
	if headers != nil {
		headersJSON, _ := json.Marshal(headers)
		headersHash := sha256.Sum256(headersJSON)
		hashData += hex.EncodeToString(headersHash[:])
	}

	// Hash the combined data
	fullHash := sha256.Sum256([]byte(hashData))
	return hex.EncodeToString(fullHash[:])
}

// ResponseCacheMiddleware provides HTTP middleware for caching
func ResponseCacheMiddleware(cache *RequestCache) func(*http.Request) *http.Request {
	return func(req *http.Request) *http.Request {
		// This is a placeholder for middleware integration
		// In a real implementation, this would cache responses
		return req
	}
}
