package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/auth"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
)

// AuthMiddleware creates an authentication middleware
func AuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract session ID from Authorization header or cookie
		sessionID := extractSessionID(c)
		if sessionID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		// Validate session
		session, err := authService.ValidateSession(c.Request.Context(), sessionID)
		if err != nil {
			if err == auth.ErrSessionExpired {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "session expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid session"})
			}
			c.Abort()
			return
		}

		// Store session in context
		c.Set("session", session)
		c.Set("user_id", session.UserID)
		c.Set("username", session.Username)
		c.Set("user_role", session.Role)

		c.Next()
	}
}

// OptionalAuthMiddleware creates an optional authentication middleware
// Sets user context if authenticated, but doesn't block unauthenticated requests
func OptionalAuthMiddleware(authService *auth.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID := extractSessionID(c)
		if sessionID == "" {
			c.Next()
			return
		}

		session, err := authService.ValidateSession(c.Request.Context(), sessionID)
		if err == nil && session != nil {
			c.Set("session", session)
			c.Set("user_id", session.UserID)
			c.Set("username", session.Username)
			c.Set("user_role", session.Role)
		}

		c.Next()
	}
}

// RequireRole creates a middleware that requires a specific role
func RequireRole(roles ...string) gin.HandlerFunc {
	roleMap := make(map[string]bool)
	for _, role := range roles {
		roleMap[role] = true
	}

	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok || !roleMap[role] {
			c.JSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetSession retrieves the session from context
func GetSession(c *gin.Context) (*models.Session, bool) {
	session, exists := c.Get("session")
	if !exists {
		return nil, false
	}

	sess, ok := session.(*models.Session)
	return sess, ok
}

// GetUserID retrieves the user ID from context
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		return "", false
	}

	id, ok := userID.(string)
	return id, ok
}

// extractSessionID extracts session ID from request
// Checks Authorization header (Bearer token) and session cookie
func extractSessionID(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		// Support "Bearer <token>" format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
		// Also support direct token
		return authHeader
	}

	// Try session cookie
	cookie, err := c.Cookie("session_id")
	if err == nil && cookie != "" {
		return cookie
	}

	// Try X-Session-ID header
	sessionHeader := c.GetHeader("X-Session-ID")
	if sessionHeader != "" {
		return sessionHeader
	}

	return ""
}
