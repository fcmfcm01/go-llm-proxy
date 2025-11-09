package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/auth"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
)

// TestSessionCreationAndValidation tests session creation and validation
// T029 [P] [US5] Unit test for session creation and validation
func TestSessionCreationAndValidation(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepository()
	userRepo := newMockUserRepository()

	// Create test user
	user := &models.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: "hashed-password",
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Create auth service
	authService := auth.NewAuthService(userRepo, sessionRepo, auth.DefaultArgon2Config(), nil)

	// Create session
	session, err := authService.CreateSession(ctx, user.ID, "192.168.1.1", "Test-Agent/1.0")
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.NotEmpty(t, session.ID)
	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, "192.168.1.1", session.IPAddress)
	assert.Equal(t, "Test-Agent/1.0", session.UserAgent)
	assert.NotEmpty(t, session.CSRFToken)
	assert.True(t, session.ExpiresAt.After(time.Now()))

	// Validate session
	validSession, err := authService.ValidateSession(ctx, session.ID)
	require.NoError(t, err)
	assert.NotNil(t, validSession)
	assert.Equal(t, session.ID, validSession.ID)
	assert.Equal(t, user.ID, validSession.UserID)
}

// TestSessionTimeout tests 24-hour session timeout
// T031 [P] [US5] Unit test for 24-hour session timeout
func TestSessionTimeout(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepository()
	userRepo := newMockUserRepository()

	// Create test user
	user := &models.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: "hashed-password",
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	authService := auth.NewAuthService(userRepo, sessionRepo, auth.DefaultArgon2Config(), nil)

	tests := []struct {
		name          string
		sessionAge    time.Duration
		shouldBeValid bool
	}{
		{
			name:          "fresh session (5 minutes old)",
			sessionAge:    5 * time.Minute,
			shouldBeValid: true,
		},
		{
			name:          "session within 24 hours (12 hours old)",
			sessionAge:    12 * time.Hour,
			shouldBeValid: true,
		},
		{
			name:          "session at 23 hours 59 minutes",
			sessionAge:    23*time.Hour + 59*time.Minute,
			shouldBeValid: true,
		},
		{
			name:          "expired session (24 hours + 1 minute)",
			sessionAge:    24*time.Hour + 1*time.Minute,
			shouldBeValid: false,
		},
		{
			name:          "old expired session (48 hours)",
			sessionAge:    48 * time.Hour,
			shouldBeValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create session with specific age
			session := &models.Session{
				ID:         generateSessionID(),
				UserID:     user.ID,
				Username:   user.Username,
				Role:       user.Role,
				CreatedAt:  time.Now().Add(-tt.sessionAge),
				ExpiresAt:  time.Now().Add(-tt.sessionAge).Add(24 * time.Hour),
				LastActive: time.Now().Add(-tt.sessionAge),
				IPAddress:  "192.168.1.1",
				UserAgent:  "Test-Agent/1.0",
				CSRFToken:  "test-csrf-token",
			}

			err := sessionRepo.Create(ctx, session)
			require.NoError(t, err)

			// Validate session
			validSession, err := authService.ValidateSession(ctx, session.ID)

			if tt.shouldBeValid {
				require.NoError(t, err)
				assert.NotNil(t, validSession)
				assert.Equal(t, session.ID, validSession.ID)
			} else {
				assert.Error(t, err)
				assert.Nil(t, validSession)
			}
		})
	}
}

// TestSessionCleanup tests cleanup of expired sessions
func TestSessionCleanup(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepository()

	// Create mix of valid and expired sessions
	sessions := []*models.Session{
		{
			ID:        "session-1",
			UserID:    "user-1",
			CreatedAt: time.Now().Add(-1 * time.Hour),
			ExpiresAt: time.Now().Add(23 * time.Hour),
		},
		{
			ID:        "session-2",
			UserID:    "user-2",
			CreatedAt: time.Now().Add(-25 * time.Hour),
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		},
		{
			ID:        "session-3",
			UserID:    "user-3",
			CreatedAt: time.Now().Add(-30 * time.Hour),
			ExpiresAt: time.Now().Add(-6 * time.Hour),
		},
	}

	for _, s := range sessions {
		err := sessionRepo.Create(ctx, s)
		require.NoError(t, err)
	}

	// Cleanup expired sessions
	err := sessionRepo.DeleteExpired(ctx)
	require.NoError(t, err)

	// Verify only valid session remains
	session1, err := sessionRepo.GetByID(ctx, "session-1")
	require.NoError(t, err)
	assert.NotNil(t, session1)

	_, err = sessionRepo.GetByID(ctx, "session-2")
	assert.Error(t, err)

	_, err = sessionRepo.GetByID(ctx, "session-3")
	assert.Error(t, err)
}

// TestCSRFTokenGeneration tests CSRF token generation and validation
func TestCSRFTokenGeneration(t *testing.T) {
	ctx := context.Background()
	sessionRepo := newMockSessionRepository()
	userRepo := newMockUserRepository()

	user := &models.User{
		ID:           "user-1",
		Username:     "testuser",
		PasswordHash: "hashed-password",
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	err := userRepo.Create(ctx, user)
	require.NoError(t, err)

	authService := auth.NewAuthService(userRepo, sessionRepo, auth.DefaultArgon2Config(), nil)

	// Create multiple sessions
	session1, err := authService.CreateSession(ctx, user.ID, "192.168.1.1", "Agent1")
	require.NoError(t, err)

	session2, err := authService.CreateSession(ctx, user.ID, "192.168.1.2", "Agent2")
	require.NoError(t, err)

	// CSRF tokens should be unique
	assert.NotEmpty(t, session1.CSRFToken)
	assert.NotEmpty(t, session2.CSRFToken)
	assert.NotEqual(t, session1.CSRFToken, session2.CSRFToken)

	// CSRF token should have minimum length (32 characters)
	assert.GreaterOrEqual(t, len(session1.CSRFToken), 32)
	assert.GreaterOrEqual(t, len(session2.CSRFToken), 32)
}

// Mock implementations

type mockSessionRepository struct {
	sessions map[string]*models.Session
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]*models.Session),
	}
}

func (r *mockSessionRepository) Create(ctx context.Context, session *models.Session) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *mockSessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		return nil, auth.ErrSessionExpired
	}

	return session, nil
}

func (r *mockSessionRepository) GetByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	var sessions []*models.Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (r *mockSessionRepository) Update(ctx context.Context, session *models.Session) error {
	if _, ok := r.sessions[session.ID]; !ok {
		return auth.ErrSessionNotFound
	}
	r.sessions[session.ID] = session
	return nil
}

func (r *mockSessionRepository) Delete(ctx context.Context, id string) error {
	delete(r.sessions, id)
	return nil
}

func (r *mockSessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	for id, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *mockSessionRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	for id, s := range r.sessions {
		if now.After(s.ExpiresAt) {
			delete(r.sessions, id)
		}
	}
	return nil
}

type mockUserRepository struct {
	users map[string]*models.User
}

func newMockUserRepository() *mockUserRepository {
	return &mockUserRepository{
		users: make(map[string]*models.User),
	}
}

func (r *mockUserRepository) Create(ctx context.Context, user *models.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *mockUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return user, nil
}

func (r *mockUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (r *mockUserRepository) Update(ctx context.Context, user *models.User) error {
	if _, ok := r.users[user.ID]; !ok {
		return auth.ErrUserNotFound
	}
	r.users[user.ID] = user
	return nil
}

func (r *mockUserRepository) Delete(ctx context.Context, id string) error {
	delete(r.users, id)
	return nil
}

func (r *mockUserRepository) Exists(ctx context.Context, username string) (bool, error) {
	for _, u := range r.users {
		if u.Username == username {
			return true, nil
		}
	}
	return false, nil
}

// Helper functions

func generateSessionID() string {
	return "session-" + time.Now().Format("20060102150405")
}
