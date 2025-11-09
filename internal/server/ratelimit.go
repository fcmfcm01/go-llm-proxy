package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	mu        sync.RWMutex
	rate      float64
	burst     int
	perIP     bool
	whiteList map[string]bool
	blackList map[string]bool
	logger    *logrus.Logger
}

// TokenBucket implements a token bucket rate limiter
type TokenBucket struct {
	tokens     float64
	capacity   int
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rate float64, burst int, perIP bool, whiteList, blackList []string, logger *logrus.Logger) *RateLimiter {
	whiteListMap := make(map[string]bool)
	for _, ip := range whiteList {
		whiteListMap[ip] = true
	}

	blackListMap := make(map[string]bool)
	for _, ip := range blackList {
		blackListMap[ip] = true
	}

	return &RateLimiter{
		rate:      rate,
		burst:     burst,
		perIP:     perIP,
		whiteList: whiteListMap,
		blackList: blackListMap,
		logger:    logger,
	}
}

// RateLimitMiddleware returns a Gin middleware for rate limiting
func (r *RateLimiter) RateLimitMiddleware() gin.HandlerFunc {
	buckets := make(map[string]*TokenBucket)
	var mu sync.RWMutex

	return func(c *gin.Context) {
		// Get client IP
		clientIP := c.ClientIP()

		// Check whitelist
		r.mu.RLock()
		if r.whiteList[clientIP] {
			r.mu.RUnlock()
			c.Next()
			return
		}

		// Check blacklist
		if r.blackList[clientIP] {
			r.mu.RUnlock()
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "IP blacklisted",
				"retry_after": 3600,
			})
			c.Abort()
			return
		}
		r.mu.RUnlock()

		// Get or create token bucket
		key := clientIP
		if !r.perIP {
			key = "global"
		}

		mu.Lock()
		bucket, exists := buckets[key]
		if !exists {
			bucket = NewTokenBucket(r.burst, r.rate)
			buckets[key] = bucket
		}
		mu.Unlock()

		// Check if a token is available
		if !bucket.Allow() {
			r.logger.WithField("ip", clientIP).Warn("Rate limit exceeded")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": int(time.Second),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// NewTokenBucket creates a new token bucket
func NewTokenBucket(capacity int, refillRate float64) *TokenBucket {
	return &TokenBucket{
		tokens:     float64(capacity),
		capacity:   capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a token is available
func (t *TokenBucket) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastRefill)

	// Calculate tokens to add
	tokensToAdd := elapsed.Seconds() * t.refillRate

	// Add tokens, capped at capacity
	t.tokens = min(t.tokens+tokensToAdd, float64(t.capacity))
	t.lastRefill = now

	// Check if token is available
	if t.tokens >= 1 {
		t.tokens -= 1
		return true
	}

	return false
}

// min returns the minimum of two float64 values
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GlobalRateLimit provides simple global rate limiting
type GlobalRateLimit struct {
	bucket *TokenBucket
}

// NewGlobalRateLimit creates a new global rate limiter
func NewGlobalRateLimit(rate float64, burst int) *GlobalRateLimit {
	return &GlobalRateLimit{
		bucket: NewTokenBucket(burst, rate),
	}
}

// Middleware returns the rate limiting middleware
func (g *GlobalRateLimit) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !g.bucket.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "rate limit exceeded",
				"retry_after": 1,
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// DynamicRateLimit provides dynamic rate limiting
type DynamicRateLimit struct {
	*RateLimiter
	dynamicRules map[string]*RateRule
	mu           sync.RWMutex
}

// RateRule defines rate limiting rules
type RateRule struct {
	Path    string
	Rate    float64
	Burst   int
	Enabled bool
}

// NewDynamicRateLimit creates a dynamic rate limiter
func NewDynamicRateLimit(baseRate float64, baseBurst int, perIP bool, logger *logrus.Logger) *DynamicRateLimit {
	return &DynamicRateLimit{
		RateLimiter:  NewRateLimiter(baseRate, baseBurst, perIP, nil, nil, logger),
		dynamicRules: make(map[string]*RateRule),
	}
}

// AddRule adds a rate limiting rule
func (d *DynamicRateLimit) AddRule(rule *RateRule) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dynamicRules[rule.Path] = rule
}

// Middleware returns dynamic rate limiting middleware
func (d *DynamicRateLimit) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		d.mu.RLock()
		rule, exists := d.dynamicRules[path]
		d.mu.RUnlock()

		if exists && rule.Enabled {
			// Use specific rule
			bucket := NewTokenBucket(rule.Burst, rule.Rate)
			if !bucket.Allow() {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":       "rate limit exceeded",
					"retry_after": 1,
				})
				c.Abort()
				return
			}
		} else {
			// Use default rate limiter
			d.RateLimiter.RateLimitMiddleware()(c)
		}
	}
}
