package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/auth"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
)

// AuthHandler handles authentication requests
type AuthHandler struct {
	authService *auth.AuthService
	auditLogger *logging.AuditLogger
	logger      *logrus.Logger
}

// NewAuthHandler creates a new authentication handler
func NewAuthHandler(authService *auth.AuthService, auditLogger *logging.AuditLogger, logger *logrus.Logger) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		auditLogger: auditLogger,
		logger:      logger,
	}
}

// HandleLogin handles POST /admin/api/v1/auth/login
func (h *AuthHandler) HandleLogin(c *gin.Context) {
	var req auth.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Get client IP and user agent
	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Attempt login
	resp, err := h.authService.Login(c.Request.Context(), &req, ip, userAgent)
	if err != nil {
		// Log failed login attempt
		if h.auditLogger != nil {
			h.auditLogger.LogAuthEvent(c.Request.Context(), logging.AuditEventAuthLogin, req.Username, ip, "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		if err == auth.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			return
		}

		h.logger.WithError(err).Error("Login failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "internal server error",
		})
		return
	}

	// Set session cookie
	c.SetCookie(
		"session_id",
		resp.SessionID,
		86400, // 24 hours
		"/",
		"",
		false, // Set to true in production with HTTPS
		true,  // HttpOnly
	)

	// Log successful login
	if h.auditLogger != nil {
		h.auditLogger.LogAuthEvent(c.Request.Context(), logging.AuditEventAuthLogin, req.Username, ip, "success", map[string]interface{}{
			"session_id": resp.SessionID,
		})
	}

	c.JSON(http.StatusOK, resp)
}

// HandleLogout handles POST /admin/api/v1/auth/logout
func (h *AuthHandler) HandleLogout(c *gin.Context) {
	// Get session from context (set by auth middleware)
	sessionInterface, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "not authenticated",
		})
		return
	}

	session, ok := sessionInterface.(*SessionInfo)
	if !ok {
		// Try models.Session
		if sess, ok := sessionInterface.(interface {
			GetID() string
			GetUsername() string
		}); ok {
			sessionID := sess.GetID()
			username := sess.GetUsername()

			// Logout
			if err := h.authService.Logout(c.Request.Context(), sessionID); err != nil {
				h.logger.WithError(err).Error("Logout failed")
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "logout failed",
				})
				return
			}

			// Clear session cookie
			c.SetCookie("session_id", "", -1, "/", "", false, true)

			// Log logout
			if h.auditLogger != nil {
				h.auditLogger.LogAuthEvent(c.Request.Context(), logging.AuditEventAuthLogout, username, c.ClientIP(), "success", nil)
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "logged out successfully",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid session type",
		})
		return
	}

	// Logout
	if err := h.authService.Logout(c.Request.Context(), session.SessionID); err != nil {
		h.logger.WithError(err).Error("Logout failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "logout failed",
		})
		return
	}

	// Clear session cookie
	c.SetCookie("session_id", "", -1, "/", "", false, true)

	// Log logout
	if h.auditLogger != nil {
		h.auditLogger.LogAuthEvent(c.Request.Context(), logging.AuditEventAuthLogout, session.Username, c.ClientIP(), "success", nil)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "logged out successfully",
	})
}

// HandleRefreshSession handles POST /admin/api/v1/auth/refresh
func (h *AuthHandler) HandleRefreshSession(c *gin.Context) {
	sessionInterface, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "not authenticated",
		})
		return
	}

	// Extract session ID
	var sessionID string
	if sess, ok := sessionInterface.(interface{ GetID() string }); ok {
		sessionID = sess.GetID()
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid session",
		})
		return
	}

	// Refresh session
	resp, err := h.authService.RefreshSessionExtended(c.Request.Context(), sessionID)
	if err != nil {
		if err == auth.ErrSessionExpired {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "session expired",
			})
			return
		}

		h.logger.WithError(err).Error("Session refresh failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "refresh failed",
		})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// HandleGetCurrentUser handles GET /admin/api/v1/auth/me
func (h *AuthHandler) HandleGetCurrentUser(c *gin.Context) {
	sessionInterface, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "not authenticated",
		})
		return
	}

	// Extract session ID
	var sessionID string
	if sess, ok := sessionInterface.(interface{ GetID() string }); ok {
		sessionID = sess.GetID()
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "invalid session",
		})
		return
	}

	// Get user info
	user, err := h.authService.GetUserBySession(c.Request.Context(), sessionID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get user info")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get user info",
		})
		return
	}

	c.JSON(http.StatusOK, user)
}

// SessionInfo is a helper struct for session information
type SessionInfo struct {
	SessionID string
	UserID    string
	Username  string
	Role      string
}
