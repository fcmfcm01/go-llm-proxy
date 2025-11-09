package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/example/go-llm-proxy/internal/auth"
	"github.com/example/go-llm-proxy/internal/models"
)

// TestAuthFlowIntegration tests complete authentication flow
// T030 [P] [US5] Integration test for login/logout flow
func TestAuthFlowIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup repositories
	userRepo := newInMemoryUserRepository()
	sessionRepo := newInMemorySessionRepository()

	// Create auth service
	passwordConfig := auth.DefaultArgon2Config()
	authService := auth.NewAuthService(userRepo, sessionRepo, passwordConfig, nil)

	// 1. Create a test user
	password := "SecurePassword123!"
	passwordHash, err := auth.HashPassword(password, passwordConfig)
	require.NoError(t, err)

	user := &models.User{
		ID:           "user-test-1",
		Username:     "testadmin",
		PasswordHash: passwordHash,
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	// 2. Attempt login with correct credentials
	loginReq := &auth.LoginRequest{
		Username: "testadmin",
		Password: password,
	}

	loginResp, err := authService.Login(ctx, loginReq, "192.168.1.100", "Test-Browser/1.0")
	require.NoError(t, err)
	assert.NotNil(t, loginResp)
	assert.NotEmpty(t, loginResp.SessionID)
	assert.NotEmpty(t, loginResp.Token)
	assert.NotNil(t, loginResp.User)
	assert.Equal(t, "testadmin", loginResp.User.Username)
	assert.Equal(t, "admin", loginResp.User.Role)

	// 3. Validate session is active
	session, err := authService.ValidateSession(ctx, loginResp.SessionID)
	require.NoError(t, err)
	assert.NotNil(t, session)
	assert.Equal(t, user.ID, session.UserID)
	assert.Equal(t, "192.168.1.100", session.IPAddress)
	assert.Equal(t, "Test-Browser/1.0", session.UserAgent)
	assert.True(t, session.ExpiresAt.After(time.Now()))

	// 4. Attempt login with wrong password
	wrongLoginReq := &auth.LoginRequest{
		Username: "testadmin",
		Password: "WrongPassword123!",
	}

	wrongLoginResp, err := authService.Login(ctx, wrongLoginReq, "192.168.1.100", "Test-Browser/1.0")
	assert.Error(t, err)
	assert.Nil(t, wrongLoginResp)
	assert.Equal(t, auth.ErrInvalidCredentials, err)

	// 5. Attempt login with non-existent user
	nonExistentLoginReq := &auth.LoginRequest{
		Username: "nonexistent",
		Password: password,
	}

	nonExistentResp, err := authService.Login(ctx, nonExistentLoginReq, "192.168.1.100", "Test-Browser/1.0")
	assert.Error(t, err)
	assert.Nil(t, nonExistentResp)

	// 6. Update session activity (simulate active user)
	time.Sleep(100 * time.Millisecond)
	err = authService.RefreshSession(ctx, loginResp.SessionID)
	require.NoError(t, err)

	// Verify last active time updated
	refreshedSession, err := sessionRepo.GetByID(ctx, loginResp.SessionID)
	require.NoError(t, err)
	assert.True(t, refreshedSession.LastActive.After(session.LastActive))

	// 7. Logout
	err = authService.Logout(ctx, loginResp.SessionID)
	require.NoError(t, err)

	// 8. Verify session is invalidated
	invalidSession, err := authService.ValidateSession(ctx, loginResp.SessionID)
	assert.Error(t, err)
	assert.Nil(t, invalidSession)

	// 9. Attempt to logout again with same session (should fail)
	err = authService.Logout(ctx, loginResp.SessionID)
	assert.Error(t, err)
}

// TestMultipleSessionsIntegration tests multiple concurrent sessions
func TestMultipleSessionsIntegration(t *testing.T) {
	ctx := context.Background()

	userRepo := newInMemoryUserRepository()
	sessionRepo := newInMemorySessionRepository()
	passwordConfig := auth.DefaultArgon2Config()
	authService := auth.NewAuthService(userRepo, sessionRepo, passwordConfig, nil)

	// Create test user
	password := "TestPassword123!"
	passwordHash, err := auth.HashPassword(password, passwordConfig)
	require.NoError(t, err)

	user := &models.User{
		ID:           "user-multi-1",
		Username:     "multiuser",
		PasswordHash: passwordHash,
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Login from multiple devices
	devices := []struct {
		ip        string
		userAgent string
	}{
		{"192.168.1.10", "Browser/1.0"},
		{"192.168.1.20", "Mobile/1.0"},
		{"10.0.0.5", "Tablet/1.0"},
	}

	var sessions []string

	for _, device := range devices {
		loginReq := &auth.LoginRequest{
			Username: "multiuser",
			Password: password,
		}

		resp, err := authService.Login(ctx, loginReq, device.ip, device.userAgent)
		require.NoError(t, err)
		sessions = append(sessions, resp.SessionID)
	}

	// Verify all sessions are valid
	for i, sessionID := range sessions {
		session, err := authService.ValidateSession(ctx, sessionID)
		require.NoError(t, err)
		assert.Equal(t, devices[i].ip, session.IPAddress)
		assert.Equal(t, devices[i].userAgent, session.UserAgent)
	}

	// Logout from first device
	err = authService.Logout(ctx, sessions[0])
	require.NoError(t, err)

	// Verify first session is invalid
	_, err = authService.ValidateSession(ctx, sessions[0])
	assert.Error(t, err)

	// Verify other sessions are still valid
	for _, sessionID := range sessions[1:] {
		session, err := authService.ValidateSession(ctx, sessionID)
		require.NoError(t, err)
		assert.NotNil(t, session)
	}
}

// TestSessionTimeoutIntegration tests session timeout behavior
func TestSessionTimeoutIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	ctx := context.Background()

	userRepo := newInMemoryUserRepository()
	sessionRepo := newInMemorySessionRepository()
	passwordConfig := auth.DefaultArgon2Config()
	authService := auth.NewAuthService(userRepo, sessionRepo, passwordConfig, nil)

	// Create test user
	password := "TestPassword123!"
	passwordHash, err := auth.HashPassword(password, passwordConfig)
	require.NoError(t, err)

	user := &models.User{
		ID:           "user-timeout-1",
		Username:     "timeoutuser",
		PasswordHash: passwordHash,
		Role:         "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = userRepo.Create(ctx, user)
	require.NoError(t, err)

	// Login
	loginReq := &auth.LoginRequest{
		Username: "timeoutuser",
		Password: password,
	}

	resp, err := authService.Login(ctx, loginReq, "192.168.1.1", "Browser/1.0")
	require.NoError(t, err)

	// Session should be valid immediately
	session, err := authService.ValidateSession(ctx, resp.SessionID)
	require.NoError(t, err)
	assert.NotNil(t, session)

	// Manually expire the session for testing
	session.ExpiresAt = time.Now().Add(-1 * time.Hour)
	err = sessionRepo.Update(ctx, session)
	require.NoError(t, err)

	// Session should now be invalid
	_, err = authService.ValidateSession(ctx, resp.SessionID)
	assert.Error(t, err)
	assert.Equal(t, auth.ErrSessionExpired, err)
}

// Mock in-memory repositories for integration testing

type inMemoryUserRepository struct {
	users map[string]*models.User
}

func newInMemoryUserRepository() *inMemoryUserRepository {
	return &inMemoryUserRepository{
		users: make(map[string]*models.User),
	}
}

func (r *inMemoryUserRepository) Create(ctx context.Context, user *models.User) error {
	r.users[user.ID] = user
	return nil
}

func (r *inMemoryUserRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, auth.ErrUserNotFound
	}
	return user, nil
}

func (r *inMemoryUserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	for _, u := range r.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, auth.ErrUserNotFound
}

func (r *inMemoryUserRepository) Update(ctx context.Context, user *models.User) error {
	if _, ok := r.users[user.ID]; !ok {
		return auth.ErrUserNotFound
	}
	r.users[user.ID] = user
	return nil
}

func (r *inMemoryUserRepository) Delete(ctx context.Context, id string) error {
	delete(r.users, id)
	return nil
}

func (r *inMemoryUserRepository) Exists(ctx context.Context, username string) (bool, error) {
	for _, u := range r.users {
		if u.Username == username {
			return true, nil
		}
	}
	return false, nil
}

type inMemorySessionRepository struct {
	sessions map[string]*models.Session
}

func newInMemorySessionRepository() *inMemorySessionRepository {
	return &inMemorySessionRepository{
		sessions: make(map[string]*models.Session),
	}
}

func (r *inMemorySessionRepository) Create(ctx context.Context, session *models.Session) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *inMemorySessionRepository) GetByID(ctx context.Context, id string) (*models.Session, error) {
	session, ok := r.sessions[id]
	if !ok {
		return nil, auth.ErrSessionNotFound
	}
	return session, nil
}

func (r *inMemorySessionRepository) GetByUserID(ctx context.Context, userID string) ([]*models.Session, error) {
	var sessions []*models.Session
	for _, s := range r.sessions {
		if s.UserID == userID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
}

func (r *inMemorySessionRepository) Update(ctx context.Context, session *models.Session) error {
	if _, ok := r.sessions[session.ID]; !ok {
		return auth.ErrSessionNotFound
	}
	r.sessions[session.ID] = session
	return nil
}

func (r *inMemorySessionRepository) Delete(ctx context.Context, id string) error {
	delete(r.sessions, id)
	return nil
}

func (r *inMemorySessionRepository) DeleteByUserID(ctx context.Context, userID string) error {
	for id, s := range r.sessions {
		if s.UserID == userID {
			delete(r.sessions, id)
		}
	}
	return nil
}

func (r *inMemorySessionRepository) DeleteExpired(ctx context.Context) error {
	now := time.Now()
	for id, s := range r.sessions {
		if now.After(s.ExpiresAt) {
			delete(r.sessions, id)
		}
	}
	return nil
}
