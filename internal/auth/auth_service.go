package auth

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/example/go-llm-proxy/internal/models"
)

// AuthService handles authentication
type AuthService struct {
	userRepo    UserRepository
	sessionRepo SessionRepository
	passwordCfg *Config
	logger      *logrus.Logger
}

// UserRepository defines user repository interface
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id string) error
	Exists(ctx context.Context, username string) (bool, error)
}

// SessionRepository defines session repository interface
type SessionRepository interface {
	Create(ctx context.Context, session *models.Session) error
	GetByID(ctx context.Context, id string) (*models.Session, error)
	GetByUserID(ctx context.Context, userID string) ([]*models.Session, error)
	Update(ctx context.Context, session *models.Session) error
	Delete(ctx context.Context, id string) error
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteExpired(ctx context.Context) error
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required,alphanum,min=3,max=50"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string    `json:"token"`
	SessionID string    `json:"session_id"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *UserJSON `json:"user"`
}

// UserJSON represents user data in JSON responses
type UserJSON struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LogoutRequest represents a logout request
type LogoutRequest struct {
	SessionID string `json:"session_id" binding:"required"`
}

// AuthContext represents authentication context
type AuthContext struct {
	Session   *models.Session
	IP        string
	UserAgent string
	RequestID string
}

// NewAuthService creates a new authentication service
func NewAuthService(
	userRepo UserRepository,
	sessionRepo SessionRepository,
	passwordCfg *Config,
	logger *logrus.Logger,
) *AuthService {
	if passwordCfg == nil {
		passwordCfg = DefaultArgon2Config()
	}

	return &AuthService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		passwordCfg: passwordCfg,
		logger:      logger,
	}
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *LoginRequest, ip, userAgent string) (*LoginResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if s.logger != nil {
			s.logger.WithField("username", req.Username).Warning("Login failed: user not found")
		}
		return nil, ErrInvalidCredentials
	}

	// Check password
	if err := user.CheckPassword(req.Password); err != nil {
		if s.logger != nil {
			s.logger.WithField("username", req.Username).Warning("Login failed: invalid password")
		}
		return nil, ErrInvalidCredentials
	}

	// Create session
	duration := 24 * time.Hour // 24 hours
	session, err := models.NewSession(user.ID, user.Username, string(user.Role), ip, userAgent, duration)
	if err != nil {
		if s.logger != nil {
			s.logger.WithField("username", req.Username).Error("Login failed: session creation error")
		}
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Save session
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		if s.logger != nil {
			s.logger.WithField("username", req.Username).Error("Login failed: session save error")
		}
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	// Mark user as logged in
	user.MarkLogin()
	if err := s.userRepo.Update(ctx, user); err != nil {
		if s.logger != nil {
			s.logger.WithField("username", req.Username).Error("Failed to update user last login")
		}
	}

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"username":  req.Username,
			"sessionID": session.ID,
			"ip":        ip,
		}).Info("User logged in successfully")
	}

	return &LoginResponse{
		Token:     session.ID,
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
		User: &UserJSON{
			ID:        user.ID,
			Username:  user.Username,
			Role:      string(user.Role),
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
	}, nil
}

// Logout invalidates a session
func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	// Get session
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	// Delete session
	if err := s.sessionRepo.Delete(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"username":  session.Username,
			"sessionID": sessionID,
			"ip":        session.IPAddress,
		}).Info("User logged out successfully")
	}

	return nil
}

// GetUserBySession gets user information from session
func (s *AuthService) GetUserBySession(ctx context.Context, sessionID string) (*UserJSON, error) {
	// Get session
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return &UserJSON{
		ID:        user.ID,
		Username:  user.Username,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// LogoutAll logout all sessions for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	// Delete all sessions for user
	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	if s.logger != nil {
		s.logger.WithField("userID", userID).Info("All sessions invalidated")
	}

	return nil
}

// CleanupExpiredSessions cleans up expired sessions
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) (int, error) {
	err := s.sessionRepo.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	// For now, return 0 as we don't track deleted count
	// Can be enhanced later if needed
	if s.logger != nil {
		s.logger.Info("Cleaned up expired sessions")
	}

	return 0, nil
}
func (s *AuthService) ValidateSession(ctx context.Context, sessionID string) (*models.Session, error) {
	// Get session
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// Check if session is valid
	if !session.IsValid() {
		// Invalidate session
		session.Invalidate()
		s.sessionRepo.Update(ctx, session)
		return nil, ErrSessionExpired
	}

	// Refresh session activity
	session.Refresh()
	s.sessionRepo.Update(ctx, session)

	return session, nil
}

// CreateSession creates a new session for a user
func (s *AuthService) CreateSession(ctx context.Context, userID, ip, userAgent string) (*models.Session, error) {
	// Get user for session data
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	// Create session
	duration := 24 * time.Hour // 24 hours timeout (FR-013)
	session, err := models.NewSession(user.ID, user.Username, string(user.Role), ip, userAgent, duration)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Save session
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to save session: %w", err)
	}

	return session, nil
}

// RefreshSession updates the last active time for a session
func (s *AuthService) RefreshSession(ctx context.Context, sessionID string) error {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if !session.IsValid() {
		return ErrSessionExpired
	}

	session.Refresh()
	return s.sessionRepo.Update(ctx, session)
}

// RefreshSessionExtended refreshes and extends a session
func (s *AuthService) RefreshSessionExtended(ctx context.Context, sessionID string) (*LoginResponse, error) {
	// Get session
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, ErrUnauthorized
	}

	// Check if session is valid
	if !session.IsValid() {
		return nil, ErrSessionExpired
	}

	// Extend session
	duration := 24 * time.Hour
	if err := session.Extend(duration); err != nil {
		return nil, fmt.Errorf("failed to extend session: %w", err)
	}

	// Save session
	if err := s.sessionRepo.Update(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"username":  session.Username,
			"sessionID": sessionID,
		}).Info("Session refreshed successfully")
	}

	return &LoginResponse{
		Token:     session.ID,
		SessionID: session.ID,
		ExpiresAt: session.ExpiresAt,
	}, nil
}

// GetUserBySession gets user information from session
func (s *AuthService) GetUserBySession(ctx context.Context, sessionID string) (*UserJSON, error) {
	// Get session
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, models.ErrUnauthorized
	}

	return &UserJSON{
		ID:        user.ID,
		Username:  user.Username,
		Role:      string(user.Role),
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

// LogoutAll logout all sessions for a user
func (s *AuthService) LogoutAll(ctx context.Context, userID string) error {
	// Delete all sessions for user
	if err := s.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		return fmt.Errorf("failed to delete sessions: %w", err)
	}

	s.logger.WithField("userID", userID).Info("All sessions invalidated")

	return nil
}

// CleanupExpiredSessions cleans up expired sessions
func (s *AuthService) CleanupExpiredSessions(ctx context.Context) (int, error) {
	deleted, err := s.sessionRepo.DeleteExpired(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	if deleted > 0 {
		s.logger.Infof("Cleaned up %d expired sessions", deleted)
	}

	return deleted, nil
}

// GetClientIP extracts the client IP from the request
func GetClientIP(r interface{}) string {
	// Try to extract IP from common headers
	switch v := r.(type) {
	case map[string]interface{}:
		// Check headers
		if headers, ok := v["headers"].(map[string]interface{}); ok {
			// Try X-Forwarded-For
			if ip, ok := headers["X-Forwarded-For"].(string); ok && ip != "" {
				return ip
			}
			// Try X-Real-IP
			if ip, ok := headers["X-Real-IP"].(string); ip != "" {
				return ip
			}
		}
		// Check remote address
		if addr, ok := v["remote_addr"].(string); ok {
			host, _, _ := net.SplitHostPort(addr)
			return host
		}
	case *http.Request:
		// For HTTP requests
		ip := v.Header.Get("X-Forwarded-For")
		if ip == "" {
			ip = v.Header.Get("X-Real-IP")
		}
		if ip == "" {
			ip, _, _ = net.SplitHostPort(v.RemoteAddr)
		}
		return ip
	}

	return "unknown"
}

// GetUserAgent extracts the user agent from the request
func GetUserAgent(r interface{}) string {
	switch v := r.(type) {
	case map[string]interface{}:
		if headers, ok := v["headers"].(map[string]interface{}); ok {
			if ua, ok := headers["User-Agent"].(string); ok {
				return ua
			}
		}
	case *http.Request:
		return v.UserAgent()
	}

	return "unknown"
}
