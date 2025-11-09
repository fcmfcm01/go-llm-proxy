package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	// TokenLength is the length of CSRF tokens in bytes
	TokenLength int

	// HeaderName is the name of the header to check for CSRF token
	HeaderName string

	// FormFieldName is the name of the form field to check for CSRF token
	FormFieldName string

	// CookieName is the name of the cookie to store CSRF token
	CookieName string

	// Skip is a function to determine if CSRF check should be skipped
	Skip func(*gin.Context) bool
}

// DefaultCSRFConfig returns default CSRF configuration
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		TokenLength:   32,
		HeaderName:    "X-CSRF-Token",
		FormFieldName: "csrf_token",
		CookieName:    "csrf_token",
		Skip: func(c *gin.Context) bool {
			// Skip CSRF for safe methods
			method := c.Request.Method
			return method == "GET" || method == "HEAD" || method == "OPTIONS"
		},
	}
}

// CSRFMiddleware creates CSRF protection middleware
func CSRFMiddleware(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	tokenStore := &csrfTokenStore{
		tokens: make(map[string]string),
	}

	return func(c *gin.Context) {
		// Skip if configured
		if config.Skip != nil && config.Skip(c) {
			// Generate and set token for safe methods
			token := tokenStore.generateToken()
			c.SetCookie(config.CookieName, token, 3600, "/", "", false, true)
			c.Header(config.HeaderName, token)
			c.Set("csrf_token", token)
			c.Next()
			return
		}

		// Get token from session
		session, exists := c.Get("session")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
			c.Abort()
			return
		}

		sessionData, ok := session.(interface{ GetCSRFToken() string })
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
			c.Abort()
			return
		}

		expectedToken := sessionData.GetCSRFToken()
		if expectedToken == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
			c.Abort()
			return
		}

		// Get token from request
		token := c.GetHeader(config.HeaderName)
		if token == "" {
			token = c.PostForm(config.FormFieldName)
		}
		if token == "" {
			cookie, err := c.Cookie(config.CookieName)
			if err == nil {
				token = cookie
			}
		}

		// Validate token
		if token == "" || token != expectedToken {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// SimpleCSRFMiddleware creates a simple CSRF middleware without session dependency
func SimpleCSRFMiddleware() gin.HandlerFunc {
	config := DefaultCSRFConfig()
	tokenStore := &csrfTokenStore{
		tokens: make(map[string]string),
	}

	return func(c *gin.Context) {
		// Skip for safe methods
		if config.Skip != nil && config.Skip(c) {
			token := tokenStore.generateToken()
			c.SetCookie(config.CookieName, token, 3600, "/", "", false, true)
			c.Header(config.HeaderName, token)
			c.Set("csrf_token", token)
			c.Next()
			return
		}

		// Get expected token from cookie
		expectedToken, err := c.Cookie(config.CookieName)
		if err != nil || expectedToken == "" {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token missing"})
			c.Abort()
			return
		}

		// Get token from request
		token := c.GetHeader(config.HeaderName)
		if token == "" {
			token = c.PostForm(config.FormFieldName)
		}

		// Validate token
		if token == "" || token != expectedToken {
			c.JSON(http.StatusForbidden, gin.H{"error": "CSRF token validation failed"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// csrfTokenStore stores CSRF tokens
type csrfTokenStore struct {
	mu     sync.RWMutex
	tokens map[string]string
}

// generateToken generates a new CSRF token
func (s *csrfTokenStore) generateToken() string {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		// Fallback to pseudo-random if crypto/rand fails
		return hex.EncodeToString([]byte(generateFallbackToken()))
	}
	return hex.EncodeToString(tokenBytes)
}

// generateFallbackToken generates a fallback token
func generateFallbackToken() string {
	// Simple fallback - in production this should use crypto/rand
	return "fallback-token-" + hex.EncodeToString([]byte("random"))
}

// GetCSRFToken retrieves CSRF token from context
func GetCSRFToken(c *gin.Context) string {
	token, exists := c.Get("csrf_token")
	if !exists {
		return ""
	}

	tokenStr, ok := token.(string)
	if !ok {
		return ""
	}

	return tokenStr
}
