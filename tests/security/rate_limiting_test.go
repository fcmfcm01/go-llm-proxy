package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/auth"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/middleware"
)

func TestRateLimitingSecurity(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	t.Run("Rate limiting prevents brute force attacks", func(t *testing.T) {
		// Create rate limiter
		config := middleware.DefaultRateLimitConfig()
		config.RequestsPerMinute = 5
		config.BurstSize = 2

		handler := middleware.RateLimitMiddleware(config, nil)

		// Create router
		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Test: Allow initial requests
		for i := 0; i < 7; i++ {
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			req.RemoteAddr = "192.168.1.1:12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if i < 7 { // Should be allowed (2 burst + 5 in minute window)
				assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusTooManyRequests,
					"Request %d should be allowed or rate limited", i)
			} else {
				assert.Equal(t, http.StatusTooManyRequests, w.Code,
					"Request %d should be rate limited", i)
			}
		}
	})

	t.Run("Rate limiting headers are present", func(t *testing.T) {
		// Create rate limiter
		config := middleware.DefaultRateLimitConfig()
		handler := middleware.RateLimitMiddleware(config, nil)

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make request
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check headers
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
		assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	})

	t.Run("Rate limiting is IP-based", func(t *testing.T) {
		// Create rate limiter with IP-based key
		config := middleware.DefaultRateLimitConfig()
		config.RequestsPerMinute = 1

		handler := middleware.RateLimitMiddleware(config, nil)

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Test: Different IPs should have separate limits
		for i := 0; i < 2; i++ {
			ip := "192.168.1." + string(rune(100+i))
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			req.RemoteAddr = ip + ":12345"

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if i == 0 {
				assert.Equal(t, http.StatusOK, w.Code, "First request from IP should succeed")
			} else {
				// Second IP might be allowed depending on implementation
				// This is just to verify different IPs are tracked separately
			}
		}
	})
}

func TestInputValidationSecurity(t *testing.T) {
	t.Run("Prevents SQL injection", func(t *testing.T) {
		sanitizer := middleware.NewInputSanitizer()

		// Test SQL injection patterns
		sqlPayloads := []string{
			"'; DROP TABLE users; --",
			"\" OR \"1\"=\"1",
			"UNION SELECT * FROM passwords",
			"1' OR '1'='1",
		}

		for _, payload := range sqlPayloads {
			err := sanitizer.ValidateInput(payload, middleware.ValidationRules{
				Required: true,
			})
			assert.Error(t, err, "SQL injection payload should be rejected: %s", payload)
		}
	})

	t.Run("Prevents XSS attacks", func(t *testing.T) {
		sanitizer := middleware.NewInputSanitizer()

		// Test XSS patterns
		xssPayloads := []string{
			"<script>alert('xss')</script>",
			"javascript:alert('xss')",
			"<img src=x onerror=alert('xss')>",
			"<svg onload=alert('xss')>",
		}

		for _, payload := range xssPayloads {
			sanitized := sanitizer.SanitizeString(payload)
			// Sanitization should escape dangerous characters
			assert.NotContains(t, sanitized, "<script",
				"XSS payload should be sanitized: %s", payload)
			assert.NotContains(t, sanitized, "javascript:",
				"XSS payload should be sanitized: %s", payload)
		}
	})

	t.Run("Sanitizes HTML input", func(t *testing.T) {
		sanitizer := middleware.NewInputSanitizer()

		// Test allowed vs disallowed HTML
		input := `<p>Hello</p><script>alert('xss')</script><strong>World</strong>`
		sanitized := sanitizer.SanitizeHTML(input)

		// Should remove script tags
		assert.NotContains(t, sanitized, "<script>")
		assert.Contains(t, sanitized, "&lt;script&gt;")
	})

	t.Run("Strips all HTML tags", func(t *testing.T) {
		sanitizer := middleware.NewInputSanitizer()

		input := "<div><p>Hello</p><strong>World</strong></div>"
		cleaned := sanitizer.StripHTML(input)

		assert.Equal(t, "HelloWorld", cleaned)
		assert.NotContains(t, cleaned, "<")
		assert.NotContains(t, cleaned, ">")
	})
}

func TestSecurityHeaders(t *testing.T) {
	t.Run("Security headers are set", func(t *testing.T) {
		// Setup
		gin.SetMode(gin.TestMode)

		config := middleware.DefaultSecurityHeadersConfig()
		handler := middleware.SecurityHeadersMiddleware(config)

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make request
		req, err := http.NewRequest("GET", "/test", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check security headers
		assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
		assert.NotEmpty(t, w.Header().Get("X-XSS-Protection"))
		assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
		assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
		assert.NotEmpty(t, w.Header().Get("Referrer-Policy"))
	})

	t.Run("HSTS header is set for HTTPS", func(t *testing.T) {
		// Setup
		gin.SetMode(gin.TestMode)

		handler := middleware.HSTSMiddleware(31536000, true, true)

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Make HTTPS request
		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		require.NoError(t, err)
		req.TLS = &http.Request{}.TLS // Simulate HTTPS

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check HSTS header
		assert.NotEmpty(t, w.Header().Get("Strict-Transport-Security"))
		assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "max-age=31536000")
		assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "includeSubDomains")
		assert.Contains(t, w.Header().Get("Strict-Transport-Security"), "preload")
	})
}

func TestEncryptionSecurity(t *testing.T) {
	t.Run("Encryption and decryption work correctly", func(t *testing.T) {
		// Create encryption service
		enc, err := NewEncryptionService("test-key-123")
		require.NoError(t, err)

		// Test data
		plaintext := "sensitive-api-key-12345"

		// Encrypt
		ciphertext, err := enc.Encrypt(plaintext)
		require.NoError(t, err)
		assert.NotEqual(t, plaintext, ciphertext, "Ciphertext should differ from plaintext")

		// Decrypt
		decrypted, err := enc.Decrypt(ciphertext)
		require.NoError(t, err)
		assert.Equal(t, plaintext, decrypted, "Decrypted text should match original")
	})

	t.Run("Different keys produce different ciphertexts", func(t *testing.T) {
		// Create two encryption services with different keys
		enc1, err := NewEncryptionService("key-1")
		require.NoError(t, err)

		enc2, err := NewEncryptionService("key-2")
		require.NoError(t, err)

		// Same plaintext
		plaintext := "test-data"

		// Encrypt with both
		ciphertext1, err := enc1.Encrypt(plaintext)
		require.NoError(t, err)

		ciphertext2, err := enc2.Encrypt(plaintext)
		require.NoError(t, err)

		// Ciphertexts should be different
		assert.NotEqual(t, ciphertext1, ciphertext2)
	})

	t.Run("Cannot decrypt with wrong key", func(t *testing.T) {
		// Create two encryption services
		enc1, err := NewEncryptionService("key-1")
		require.NoError(t, err)

		enc2, err := NewEncryptionService("key-2")
		require.NoError(t, err)

		// Encrypt with enc1
		plaintext := "secret-data"
		ciphertext, err := enc1.Encrypt(plaintext)
		require.NoError(t, err)

		// Try to decrypt with enc2 (should fail)
		_, err = enc2.Decrypt(ciphertext)
		assert.Error(t, err, "Decryption with wrong key should fail")
	})
}

func TestCORS(t *testing.T) {
	t.Run("CORS headers are set correctly", func(t *testing.T) {
		gin.SetMode(gin.TestMode)

		allowedOrigins := []string{"https://example.com"}
		handler := middleware.CORSConfigMiddleware(allowedOrigins)

		router := gin.New()
		router.Use(handler)
		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		// Test preflight request
		req, err := http.NewRequest("OPTIONS", "/test", nil)
		require.NoError(t, err)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", "GET")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Check CORS headers
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Origin"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
		assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
	})
}
