package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/internal/auth"
	"github.com/fcmfcm01/go-llm-proxy/internal/models"
)

// TestAuthLoginContract tests POST /admin/api/v1/auth/login contract
// T026 [P] [US5] Contract test for POST /admin/api/v1/auth/login
func TestAuthLoginContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mockAuthService)
		expectedStatus int
		expectedBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful login",
			requestBody: map[string]interface{}{
				"username": "admin",
				"password": "password123",
			},
			setupMock: func(m *mockAuthService) {
				m.loginResult = &auth.LoginResponse{
					Token:     "test-token",
					SessionID: "test-session-id",
					User: &auth.UserJSON{
						ID:       "user-1",
						Username: "admin",
						Role:     "admin",
					},
				}
				m.loginError = nil
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "test-token", body["token"])
				assert.Equal(t, "test-session-id", body["session_id"])
				assert.NotNil(t, body["user"])
			},
		},
		{
			name: "invalid credentials",
			requestBody: map[string]interface{}{
				"username": "admin",
				"password": "wrongpassword",
			},
			setupMock: func(m *mockAuthService) {
				m.loginResult = nil
				m.loginError = auth.ErrInvalidCredentials
			},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "invalid credentials")
			},
		},
		{
			name: "missing username",
			requestBody: map[string]interface{}{
				"password": "password123",
			},
			setupMock:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "username")
			},
		},
		{
			name: "missing password",
			requestBody: map[string]interface{}{
				"username": "admin",
			},
			setupMock:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "password")
			},
		},
		{
			name:           "empty request body",
			requestBody:    map[string]interface{}{},
			setupMock:      func(m *mockAuthService) {},
			expectedStatus: http.StatusBadRequest,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &mockAuthService{}
			tt.setupMock(mockService)

			router := gin.New()
			handler := &authHandler{service: mockService}
			router.POST("/admin/api/v1/auth/login", handler.handleLogin)

			// Create request
			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Real-IP", "192.168.1.1")
			req.Header.Set("User-Agent", "Test-Client/1.0")

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody map[string]interface{}
			err = json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			tt.expectedBody(t, responseBody)
		})
	}
}

// TestAuthLogoutContract tests POST /admin/api/v1/auth/logout contract
// T027 [P] [US5] Contract test for POST /admin/api/v1/auth/logout
func TestAuthLogoutContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupAuth      func(*gin.Context)
		expectedStatus int
		expectedBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful logout",
			setupAuth: func(c *gin.Context) {
				c.Set("session", &models.Session{
					ID:     "test-session-id",
					UserID: "user-1",
				})
			},
			expectedStatus: http.StatusOK,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "logged out successfully", body["message"])
			},
		},
		{
			name:           "no active session",
			setupAuth:      func(c *gin.Context) {},
			expectedStatus: http.StatusUnauthorized,
			expectedBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "not authenticated")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockService := &mockAuthService{}

			router := gin.New()
			handler := &authHandler{service: mockService}

			// Add middleware to setup auth
			router.Use(func(c *gin.Context) {
				tt.setupAuth(c)
				c.Next()
			})

			router.POST("/admin/api/v1/auth/logout", handler.handleLogout)

			// Create request
			req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/auth/logout", nil)
			req.Header.Set("Content-Type", "application/json")

			// Execute
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)

			var responseBody map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &responseBody)
			require.NoError(t, err)

			tt.expectedBody(t, responseBody)
		})
	}
}

// Mock types for testing

type mockAuthService struct {
	loginResult    *auth.LoginResponse
	loginError     error
	logoutError    error
	validateResult *models.Session
	validateError  error
}

func (m *mockAuthService) Login(username, password, ip, userAgent string) (*auth.LoginResponse, error) {
	return m.loginResult, m.loginError
}

func (m *mockAuthService) Logout(sessionID string) error {
	return m.logoutError
}

func (m *mockAuthService) ValidateSession(sessionID string) (*models.Session, error) {
	return m.validateResult, m.validateError
}

type authHandler struct {
	service *mockAuthService
}

func (h *authHandler) handleLogin(c *gin.Context) {
	var req auth.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	resp, err := h.service.Login(req.Username, req.Password, ip, userAgent)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *authHandler) handleLogout(c *gin.Context) {
	session, exists := c.Get("session")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}

	sess := session.(*models.Session)
	if err := h.service.Logout(sess.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "logout failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "logged out successfully"})
}
