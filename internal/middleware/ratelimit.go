package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	buckets map[string]*tokenBucket
	mu      sync.RWMutex
	config  *RateLimitConfig
	logger  *logrus.Logger
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int           // Maximum requests per minute
	BurstSize         int           // Maximum burst size
	KeyFunc           KeyFunc       // Function to generate rate limit key
	CleanupInterval   time.Duration // How often to clean up old buckets
}

// KeyFunc generates a rate limit key from the request context
type KeyFunc func(*gin.Context) string

// tokenBucket implements the token bucket algorithm
type tokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefillTime time.Time
	mu             sync.Mutex
}

// DefaultRateLimitConfig returns default rate limit configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		RequestsPerMinute: 60,
		BurstSize:         10,
		KeyFunc:           IPBasedKeyFunc,
		CleanupInterval:   5 * time.Minute,
	}
}

// IPBasedKeyFunc generates key based on IP address
func IPBasedKeyFunc(c *gin.Context) string {
	return c.ClientIP()
}

// UserBasedKeyFunc generates key based on user ID
func UserBasedKeyFunc(c *gin.Context) string {
	// Try to get user ID from session
	if sessionVal, exists := c.Get("session"); exists {
		if session, ok := sessionVal.(interface{ GetUserID() string }); ok {
			return "user:" + session.GetUserID()
		}
	}

	// Fallback to IP
	return "ip:" + c.ClientIP()
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig, logger *logrus.Logger) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	limiter := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		config:  config,
		logger:  logger,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(config *RateLimitConfig, logger *logrus.Logger) gin.HandlerFunc {
	limiter := NewRateLimiter(config, logger)

	return func(c *gin.Context) {
		// Generate key for this request
		key := config.KeyFunc(c)

		// Check if request is allowed
		allowed, retryAfter := limiter.Allow(key)

		if !allowed {
			if logger != nil {
				logger.WithFields(logrus.Fields{
					"key":         key,
					"path":        c.Request.URL.Path,
					"retry_after": retryAfter,
				}).Warn("Rate limit exceeded")
			}

			// Set rate limit headers
			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(retryAfter).Unix()))
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": map[string]interface{}{
					"message":     "Rate limit exceeded. Please try again later.",
					"type":        "rate_limit_error",
					"retry_after": int(retryAfter.Seconds()),
				},
			})
			c.Abort()
			return
		}

		// Set rate limit headers for successful requests
		bucket := limiter.getBucket(key)
		bucket.mu.Lock()
		remaining := int(bucket.tokens)
		bucket.mu.Unlock()

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))

		c.Next()
	}
}

// Allow checks if a request with the given key is allowed
func (rl *RateLimiter) Allow(key string) (bool, time.Duration) {
	bucket := rl.getBucket(key)
	return bucket.allow()
}

// getBucket gets or creates a token bucket for the given key
func (rl *RateLimiter) getBucket(key string) *tokenBucket {
	rl.mu.RLock()
	bucket, exists := rl.buckets[key]
	rl.mu.RUnlock()

	if exists {
		return bucket
	}

	// Create new bucket
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Double-check after acquiring write lock
	if bucket, exists := rl.buckets[key]; exists {
		return bucket
	}

	bucket = &tokenBucket{
		tokens:         float64(rl.config.BurstSize),
		maxTokens:      float64(rl.config.BurstSize),
		refillRate:     float64(rl.config.RequestsPerMinute) / 60.0, // per second
		lastRefillTime: time.Now(),
	}

	rl.buckets[key] = bucket
	return bucket
}

// allow checks if a token can be consumed from the bucket
func (tb *tokenBucket) allow() (bool, time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(tb.lastRefillTime).Seconds()
	tb.tokens = min(tb.maxTokens, tb.tokens+elapsed*tb.refillRate)
	tb.lastRefillTime = now

	// Check if we have enough tokens
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true, 0
	}

	// Calculate retry after duration
	tokensNeeded := 1.0 - tb.tokens
	retryAfter := time.Duration(tokensNeeded/tb.refillRate) * time.Second

	return false, retryAfter
}

// cleanup removes old token buckets
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		rl.performCleanup()
	}
}

// performCleanup removes buckets that haven't been used recently
func (rl *RateLimiter) performCleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	threshold := rl.config.CleanupInterval

	for key, bucket := range rl.buckets {
		bucket.mu.Lock()
		age := now.Sub(bucket.lastRefillTime)
		bucket.mu.Unlock()

		if age > threshold {
			delete(rl.buckets, key)
		}
	}

	if rl.logger != nil {
		rl.logger.WithField("bucket_count", len(rl.buckets)).Debug("Rate limiter cleanup completed")
	}
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// PerEndpointRateLimitConfig holds per-endpoint rate limit configuration
type PerEndpointRateLimitConfig struct {
	DefaultLimit   *RateLimitConfig
	EndpointLimits map[string]*RateLimitConfig
}

// PerEndpointRateLimitMiddleware creates a rate limiter with different limits per endpoint
func PerEndpointRateLimitMiddleware(config *PerEndpointRateLimitConfig, logger *logrus.Logger) gin.HandlerFunc {
	limiters := make(map[string]*RateLimiter)

	// Create limiter for each endpoint
	for endpoint, endpointConfig := range config.EndpointLimits {
		limiters[endpoint] = NewRateLimiter(endpointConfig, logger)
	}

	// Create default limiter
	defaultLimiter := NewRateLimiter(config.DefaultLimit, logger)

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Get limiter for this endpoint
		limiter, exists := limiters[path]
		if !exists {
			limiter = defaultLimiter
		}

		// Generate key
		key := limiter.config.KeyFunc(c)

		// Check rate limit
		allowed, retryAfter := limiter.Allow(key)

		if !allowed {
			if logger != nil {
				logger.WithFields(logrus.Fields{
					"key":         key,
					"path":        path,
					"retry_after": retryAfter,
				}).Warn("Rate limit exceeded")
			}

			c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("Retry-After", fmt.Sprintf("%d", int(retryAfter.Seconds())))

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": map[string]interface{}{
					"message":     "Rate limit exceeded. Please try again later.",
					"type":        "rate_limit_error",
					"retry_after": int(retryAfter.Seconds()),
				},
			})
			c.Abort()
			return
		}

		// Set rate limit headers
		bucket := limiter.getBucket(key)
		bucket.mu.Lock()
		remaining := int(bucket.tokens)
		bucket.mu.Unlock()

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", limiter.config.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		c.Next()
	}
}
