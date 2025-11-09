package models

import (
	"net"
	"time"
)

// Session represents a user authentication session
type Session struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Role       UserRole  `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	LastActive time.Time `json:"last_active"`
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	CSRFToken  string    `json:"csrf_token"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewSession creates a new session
func NewSession(userID, username, role, ipAddress, userAgent string, duration time.Duration) (*Session, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	if username == "" {
		return nil, ErrUsernameRequired
	}

	if duration <= 0 {
		return nil, ErrInvalidSessionDuration
	}

	now := time.Now()

	// Get IP without port
	cleanIP := ipAddress
	if host, _, err := net.SplitHostPort(ipAddress); err == nil {
		cleanIP = host
	}

	session := &Session{
		ID:         generateID(),
		UserID:     userID,
		Username:   username,
		Role:       UserRole(role),
		CreatedAt:  now,
		ExpiresAt:  now.Add(duration),
		LastActive: now,
		IPAddress:  cleanIP,
		UserAgent:  userAgent,
		CSRFToken:  generateCSRFToken(),
		UpdatedAt:  now,
	}

	return session, nil
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid checks if the session is valid (not expired and active)
func (s *Session) IsValid() bool {
	now := time.Now()

	// Check if expired
	if s.IsExpired() {
		return false
	}

	// Check if too idle (e.g., 2 hours)
	maxIdleTime := 2 * time.Hour
	if now.Sub(s.LastActive) > maxIdleTime {
		return false
	}

	return true
}

// Extend extends the session expiration time
func (s *Session) Extend(duration time.Duration) error {
	if duration <= 0 {
		return ErrInvalidSessionDuration
	}

	s.ExpiresAt = s.ExpiresAt.Add(duration)
	s.LastActive = time.Now()
	s.UpdatedAt = time.Now()

	return nil
}

// Refresh updates the last active time
func (s *Session) Refresh() {
	s.LastActive = time.Now()
	s.UpdatedAt = time.Now()
}

// Invalidate marks the session as invalid
func (s *Session) Invalidate() {
	now := time.Now()
	s.ExpiresAt = now
	s.CSRFToken = ""
	s.UpdatedAt = now
}

// UpdateIP updates the IP address
func (s *Session) UpdateIP(newIP string) {
	// Get IP without port
	cleanIP := newIP
	if host, _, err := net.SplitHostPort(newIP); err == nil {
		cleanIP = host
	}

	s.IPAddress = cleanIP
	s.UpdatedAt = time.Now()
}

// GetRemoteAddr returns the remote address
func (s *Session) GetRemoteAddr() string {
	if s.IPAddress == "" {
		return "unknown"
	}
	return s.IPAddress
}

// ToJSON returns a JSON-safe version of the session
func (s *Session) ToJSON() *Session {
	return &Session{
		ID:         s.ID,
		UserID:     "", // Don't expose user ID
		Username:   s.Username,
		Role:       s.Role,
		CreatedAt:  s.CreatedAt,
		ExpiresAt:  s.ExpiresAt,
		LastActive: s.LastActive,
		IPAddress:  s.IPAddress,
		UserAgent:  s.UserAgent,
		CSRFToken:  "", // Don't expose CSRF token
		UpdatedAt:  s.UpdatedAt,
	}
}

// HasRole checks if the session has the required role
func (s *Session) HasRole(requiredRole UserRole) bool {
	// Higher roles can access lower roles
	roleHierarchy := map[UserRole]int{
		RoleAdmin:  3,
		RoleUser:   2,
		RoleViewer: 1,
	}

	userLevel := roleHierarchy[s.Role]
	requiredLevel := roleHierarchy[requiredRole]

	return userLevel >= requiredLevel
}

// SetID implements the Entity interface
func (s *Session) SetID(id string) {
	s.ID = id
}

// SetCreatedAt implements the Entity interface
func (s *Session) SetCreatedAt(t time.Time) {
	s.CreatedAt = t
}

// SetUpdatedAt implements the Entity interface
func (s *Session) SetUpdatedAt(t time.Time) {
	s.UpdatedAt = t
}

// Duration returns the session duration
func (s *Session) Duration() time.Duration {
	return s.ExpiresAt.Sub(s.CreatedAt)
}

// Age returns the age of the session
func (s *Session) Age() time.Duration {
	return time.Since(s.CreatedAt)
}

// IdleTime returns the idle time since last activity
func (s *Session) IdleTime() time.Duration {
	return time.Since(s.LastActive)
}
