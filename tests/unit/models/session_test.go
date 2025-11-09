package models

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSession(t *testing.T) {
	t.Parallel()

	userID := "user-123"
	username := "testuser"
	role := RoleAdmin
	ipAddress := "192.168.1.1"
	userAgent := "Test Browser 1.0"
	duration := 24 * time.Hour

	session, err := NewSession(userID, username, role, ipAddress, userAgent, duration)

	require.NoError(t, err)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, userID, session.UserID)
	assert.Equal(t, username, session.Username)
	assert.Equal(t, role, session.Role)
	assert.Equal(t, ipAddress, session.IPAddress)
	assert.Equal(t, userAgent, session.UserAgent)
	assert.False(t, session.CreatedAt.IsZero())
	assert.False(t, session.ExpiresAt.IsZero())
	assert.True(t, session.CreatedAt.Before(session.ExpiresAt))
	assert.Equal(t, session.CreatedAt, session.LastActive)
	assert.NotEmpty(t, session.CSRFToken)
}

func TestSession_IsExpired(t *testing.T) {
	t.Parallel()

	// Test non-expired session
	now := time.Now()
	futureSession := &Session{
		CreatedAt:  now.Add(-time.Hour),
		ExpiresAt:  now.Add(time.Hour),
		LastActive: now,
	}
	assert.False(t, futureSession.IsExpired())

	// Test expired session
	expiredSession := &Session{
		CreatedAt:  now.Add(-3 * time.Hour),
		ExpiresAt:  now.Add(-time.Hour),
		LastActive: now.Add(-2 * time.Hour),
	}
	assert.True(t, expiredSession.IsExpired())

	// Test session expiring now
	nowTime := now
	expireNowSession := &Session{
		CreatedAt:  nowTime.Add(-time.Hour),
		ExpiresAt:  nowTime,
		LastActive: nowTime,
	}
	assert.True(t, expireNowSession.IsExpired())
}

func TestSession_IsValid(t *testing.T) {
	t.Parallel()

	now := time.Now()
	ipAddress := "192.168.1.1"
	userAgent := "Test Browser"

	// Test valid session
	validSession := &Session{
		ID:         "session-123",
		UserID:     "user-123",
		Username:   "testuser",
		Role:       RoleAdmin,
		CreatedAt:  now.Add(-time.Hour),
		ExpiresAt:  now.Add(time.Hour),
		LastActive: now,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CSRFToken:  "csrf-token",
	}
	assert.True(t, validSession.IsValid())

	// Test expired session
	expiredSession := &Session{
		ID:         "session-123",
		UserID:     "user-123",
		Username:   "testuser",
		Role:       RoleAdmin,
		CreatedAt:  now.Add(-3 * time.Hour),
		ExpiresAt:  now.Add(-time.Hour),
		LastActive: now,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CSRFToken:  "csrf-token",
	}
	assert.False(t, expiredSession.IsValid())

	// Test session with too much idle time
	idleSession := &Session{
		ID:         "session-123",
		UserID:     "user-123",
		Username:   "testuser",
		Role:       RoleAdmin,
		CreatedAt:  now.Add(-time.Hour),
		ExpiresAt:  now.Add(time.Hour),
		LastActive: now.Add(-2 * time.Hour), // 2 hours idle
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		CSRFToken:  "csrf-token",
	}
	assert.False(t, idleSession.IsValid())
}

func TestSession_Extend(t *testing.T) {
	t.Parallel()

	now := time.Now()
	initialDuration := time.Hour
	extension := 30 * time.Minute

	session := &Session{
		ID:         "session-123",
		UserID:     "user-123",
		CreatedAt:  now,
		ExpiresAt:  now.Add(initialDuration),
		LastActive: now,
	}

	err := session.Extend(extension)

	require.NoError(t, err)
	assert.Equal(t, now.Add(initialDuration+extension), session.ExpiresAt)
	assert.Equal(t, now.Add(extension), session.LastActive)
}

func TestSession_Refresh(t *testing.T) {
	t.Parallel()

	now := time.Now()
	initialTime := now.Add(-30 * time.Minute)

	session := &Session{
		ID:         "session-123",
		UserID:     "user-123",
		CreatedAt:  initialTime,
		ExpiresAt:  now.Add(time.Hour),
		LastActive: initialTime,
		IPAddress:  "192.168.1.1",
		UserAgent:  "Test Browser",
	}

	session.Refresh()

	assert.Equal(t, now, session.LastActive)
	assert.Greater(t, session.LastActive, initialTime)
}

func TestSession_Invalidate(t *testing.T) {
	t.Parallel()

	now := time.Now()
	expireTime := now.Add(time.Hour)

	session := &Session{
		ID:        "session-123",
		UserID:    "user-123",
		ExpiresAt: expireTime,
		CSRFToken: "csrf-token",
	}

	session.Invalidate()

	assert.True(t, session.ExpiresAt.Before(now))
	assert.Empty(t, session.CSRFToken)
}

func TestSession_UpdateIP(t *testing.T) {
	t.Parallel()

	session := &Session{
		ID:        "session-123",
		IPAddress: "192.168.1.1",
	}

	newIP := "10.0.0.1"
	session.UpdateIP(newIP)

	assert.Equal(t, newIP, session.IPAddress)
}

func TestSession_GetRemoteAddr(t *testing.T) {
	t.Parallel()

	// Test with IP address
	session1 := &Session{
		IPAddress: "192.168.1.1",
	}
	assert.Equal(t, "192.168.1.1", session1.GetRemoteAddr())

	// Test with IP:port
	session2 := &Session{
		IPAddress: "192.168.1.1:8080",
	}
	assert.Equal(t, "192.168.1.1:8080", session2.GetRemoteAddr())

	// Test with empty IP
	session3 := &Session{
		IPAddress: "",
	}
	assert.Equal(t, "unknown", session3.GetRemoteAddr())
}

func TestSession_ToJSON(t *testing.T) {
	t.Parallel()

	now := time.Now()
	session := &Session{
		ID:         "session-123",
		UserID:     "user-123",
		Username:   "testuser",
		Role:       RoleAdmin,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Hour),
		LastActive: now,
		IPAddress:  "192.168.1.1",
		UserAgent:  "Test Browser",
		CSRFToken:  "csrf-token",
	}

	jsonSession := session.ToJSON()

	assert.Equal(t, session.ID, jsonSession.ID)
	assert.Equal(t, session.Username, jsonSession.Username)
	assert.Equal(t, session.Role, jsonSession.Role)
	assert.Equal(t, session.CreatedAt, jsonSession.CreatedAt)
	assert.Equal(t, session.ExpiresAt, jsonSession.ExpiresAt)
	assert.Equal(t, session.LastActive, jsonSession.LastActive)
	assert.Equal(t, session.IPAddress, jsonSession.IPAddress)
	assert.Equal(t, session.UserAgent, jsonSession.UserAgent)
	assert.Empty(t, jsonSession.CSRFToken) // Should not be exposed
	assert.Empty(t, jsonSession.UserID)    // Should not be exposed
}

func TestSession_HasRole(t *testing.T) {
	t.Parallel()

	session := &Session{
		Role: RoleAdmin,
	}

	assert.True(t, session.HasRole(RoleAdmin))
	assert.True(t, session.HasRole(RoleUser))   // Admin can access user resources
	assert.True(t, session.HasRole(RoleViewer)) // Admin can access viewer resources
	assert.False(t, session.HasRole("invalid"))

	// Test user role
	userSession := &Session{
		Role: RoleUser,
	}
	assert.True(t, userSession.HasRole(RoleUser))
	assert.False(t, userSession.HasRole(RoleAdmin))
}

func TestSession_SetID(t *testing.T) {
	t.Parallel()

	session := &Session{
		UserID:   "user-123",
		Username: "testuser",
	}

	newID := "new-session-id"
	session.SetID(newID)

	assert.Equal(t, newID, session.ID)
}

func TestSession_SetCreatedAt(t *testing.T) {
	t.Parallel()

	session := &Session{
		Username: "testuser",
	}

	newTime := time.Now()
	session.SetCreatedAt(newTime)

	assert.Equal(t, newTime, session.CreatedAt)
}

func TestSession_SetUpdatedAt(t *testing.T) {
	t.Parallel()

	session := &Session{
		Username: "testuser",
	}

	newTime := time.Now()
	session.SetUpdatedAt(newTime)

	assert.Equal(t, newTime, session.UpdatedAt)
}

func TestSession_Duration(t *testing.T) {
	t.Parallel()

	createdAt := time.Now()
	expiresAt := createdAt.Add(2 * time.Hour)

	session := &Session{
		CreatedAt: createdAt,
		ExpiresAt: expiresAt,
	}

	duration := session.Duration()
	assert.Equal(t, 2*time.Hour, duration)
}

func TestSession_Age(t *testing.T) {
	t.Parallel()

	createdAt := time.Now().Add(-time.Hour)
	session := &Session{
		CreatedAt: createdAt,
	}

	age := session.Age()
	assert.GreaterOrEqual(t, age, time.Hour-5*time.Second)
	assert.LessOrEqual(t, age, time.Hour+5*time.Second)
}

func TestSession_IdleTime(t *testing.T) {
	t.Parallel()

	now := time.Now()
	lastActive := now.Add(-30 * time.Minute)
	session := &Session{
		LastActive: lastActive,
	}

	idleTime := session.IdleTime()
	assert.GreaterOrEqual(t, idleTime, 30*time.Minute-5*time.Second)
	assert.LessOrEqual(t, idleTime, 30*time.Minute+5*time.Second)
}
