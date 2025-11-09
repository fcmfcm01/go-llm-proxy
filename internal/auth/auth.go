package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SessionManager manages user sessions
type SessionManager struct {
	sessions map[string]session
	logger   *logrus.Logger
}

type session struct {
	Username  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewSessionManager creates a new session manager
func NewSessionManager(logger *logrus.Logger) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]session),
		logger:   logger,
	}
}

// Login handles user login
func (sm *SessionManager) Login(c *gin.Context, username, password string, sessionTimeout time.Duration) (string, bool) {
	// Simple authentication - in production, use proper password hashing
	if username == "admin" && password == "admin123" {
		sessionID := generateSessionID()
		now := time.Now()
		sm.sessions[sessionID] = session{
			Username:  username,
			CreatedAt: now,
			ExpiresAt: now.Add(sessionTimeout),
		}

		c.SetCookie("session_id", sessionID, int(sessionTimeout.Seconds()), "/", "", false, true)
		sm.logger.Infof("User logged in: %s", username)
		return sessionID, true
	}

	sm.logger.Warnf("Failed login attempt for user: %s", username)
	return "", false
}

// Logout handles user logout
func (sm *SessionManager) Logout(c *gin.Context, sessionID string) {
	delete(sm.sessions, sessionID)
	c.SetCookie("session_id", "", -1, "/", "", false, true)
	sm.logger.Info("User logged out")
}

// Middleware checks if user is authenticated
func (sm *SessionManager) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip auth for certain paths
		path := c.Request.URL.Path
		if path == "/admin/login" || path == "/admin/do-login" || path == "/healthz" {
			c.Next()
			return
		}

		// Check for session cookie
		sessionID, err := c.Cookie("session_id")
		if err != nil {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Validate session
		sess, exists := sm.sessions[sessionID]
		if !exists {
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Check expiration
		if time.Now().After(sess.ExpiresAt) {
			delete(sm.sessions, sessionID)
			c.Redirect(http.StatusFound, "/admin/login")
			c.Abort()
			return
		}

		// Add user to context
		c.Set("username", sess.Username)
		c.Set("session_id", sessionID)
		c.Next()
	}
}

func generateSessionID() string {
	// Simple session ID generation - use crypto/rand in production
	return "session_" + time.Now().Format("20060102150405")
}
