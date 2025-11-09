package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig holds security headers configuration
type SecurityHeadersConfig struct {
	ContentSecurityPolicy       string // CSP header value
	XSSProtection               string // X-XSS-Protection header
	ContentTypeOptions          string // X-Content-Type-Options header
	FrameOptions                string // X-Frame-Options header
	ReferrerPolicy              string // Referrer-Policy header
	PermittedCrossDomainPolicies string // X-Permitted-Cross-Domain-Policies header
	StrictTransportSecurity     string // Strict-Transport-Security header
}

// DefaultSecurityHeadersConfig returns default security headers configuration
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' https://cdn.jsdelivr.net; connect-src 'self' https:; frame-ancestors 'none';",
		XSSProtection:         "1; mode=block",
		ContentTypeOptions:    "nosniff",
		FrameOptions:          "DENY",
		ReferrerPolicy:        "strict-origin-when-cross-origin",
		PermittedCrossDomainPolicies: "none",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains; preload",
	}
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(config *SecurityHeadersConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityHeadersConfig()
	}

	return func(c *gin.Context) {
		// Content Security Policy
		c.Header("Content-Security-Policy", config.ContentSecurityPolicy)

		// XSS Protection
		c.Header("X-XSS-Protection", config.XSSProtection)

		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", config.ContentTypeOptions)

		// Clickjacking protection
		c.Header("X-Frame-Options", config.FrameOptions)

		// Referrer policy
		c.Header("Referrer-Policy", config.ReferrerPolicy)

		// Cross-domain policies
		c.Header("X-Permitted-Cross-Domain-Policies", config.PermittedCrossDomainPolicies)

		// Strict Transport Security (only for HTTPS)
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", config.StrictTransportSecurity)
		}

		// Permissions Policy (formerly Feature Policy)
		c.Header("Permissions-Policy",
			"geolocation=(), microphone=(), camera=(), payment=(), usb=(), vr=(), xr-spatial-tracking=()")

		// Additional security headers
		c.Header("X-Download-Options", "noopen")
		c.Header("X-Permitted-Cross-Domain-Policies", "master-only")

		c.Next()
	}
}

// NoSniffMiddleware prevents MIME type sniffing
func NoSniffMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}

// HSTSMiddleware adds HTTP Strict Transport Security header
func HSTSMiddleware(maxAge int, includeSubDomains bool, preload bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.TLS != nil {
			value := "max-age="

			if maxAge > 0 {
				value += string(rune(maxAge))
			} else {
				value += "31536000" // Default 1 year
			}

			if includeSubDomains {
				value += "; includeSubDomains"
			}

			if preload {
				value += "; preload"
			}

			c.Header("Strict-Transport-Security", value)
		}
		c.Next()
	}
}

// CORSConfigMiddleware configures CORS headers
func CORSConfigMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range allowedOrigins {
			if allowedOrigin == "*" || allowedOrigin == origin {
				allowed = true
				c.Header("Access-Control-Allow-Origin", allowedOrigin)
				break
			}
		}

		if allowed {
			c.Header("Vary", "Origin")
		}

		// Set common CORS headers
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
		c.Header("Access-Control-Max-Age", "86400") // 24 hours

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// SecureHeadersForStaticFile adds security headers for static file responses
func SecureHeadersForStaticFile() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Don't cache static files with sensitive information
		if c.Request.URL.Path[0:8] == "/admin/" {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
		} else {
			// Cache static files for 1 hour
			c.Header("Cache-Control", "public, max-age=3600")
		}

		// Set content type headers
		if c.Request.URL.Path[len(c.Request.URL.Path)-3:] == ".js" {
			c.Header("Content-Type", "application/javascript")
		} else if c.Request.URL.Path[len(c.Request.URL.Path)-4:] == ".css" {
			c.Header("Content-Type", "text/css")
		}

		// Always include security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}
