package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/server"
)

// TestReadyContract tests the GET /healthz/ready endpoint
func TestReadyContract(t *testing.T) {
	// Set Gin to Test mode
	gin.SetMode(gin.TestMode)

	// Create logger
	logger := logging.NewSimpleLogger("debug")

	// Create a test server
	srv := server.NewServer(&server.ServerConfig{
		Port:         8080,
		ReadTimeout:  30,
		WriteTimeout: 30,
		IdleTimeout:  60,
	}, logger.Logger)

	// Setup routes
	srv.SetupRoutes()

	t.Run("ReadyEndpointExists", func(t *testing.T) {
		// Test that the ready endpoint exists
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200 (ready) or 503 (not ready)
		assert.True(t, w.Code == 200 || w.Code == 503,
			"Ready endpoint should return 200 or 503, got: %d", w.Code)
	})

	t.Run("ReturnsReadinessStatus", func(t *testing.T) {
		// Test that the endpoint returns readiness status
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should have valid JSON response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Should have status field
		assert.Contains(t, response, "status", "Response should have status field")

		// Status should be either "ready" or "not ready"
		if w.Code == 200 {
			assert.Equal(t, "ready", response["status"], "Status should be 'ready' when returning 200")
		} else if w.Code == 503 {
			assert.Equal(t, "not ready", response["status"], "Status should be 'not ready' when returning 503")
		}
	})

	t.Run("IncludesReadinessChecks", func(t *testing.T) {
		// Test that the response includes readiness check details
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// May include checks field with details
		if checks, ok := response["checks"]; ok {
			// If checks are present, they should be an object or array
			assert.NotNil(t, checks, "Checks should not be nil if present")
		}
	})

	t.Run("ResponseFormat", func(t *testing.T) {
		// Test response format
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify basic structure
		if status, ok := response["status"]; ok {
			assert.NotEmpty(t, status, "Status should not be empty")
		}
	})

	t.Run("ValidJSONContentType", func(t *testing.T) {
		// Test that the endpoint returns JSON content type
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return application/json content type
		contentType := w.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json",
			"Content-Type should be application/json, got: %s", contentType)
	})

	t.Run("FastResponse", func(t *testing.T) {
		// Test that ready endpoint responds quickly
		start := time.Now()

		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		duration := time.Since(start)

		// Should respond within 50ms for readiness check
		assert.Less(t, duration, 50*time.Millisecond,
			"Ready endpoint should respond quickly, took %v", duration)
	})

	t.Run("StatusCodeReflectsReadiness", func(t *testing.T) {
		// Test that status code reflects readiness
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response to get status
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if status, ok := response["status"]; ok {
			// Status code should match status
			if status == "ready" {
				assert.Equal(t, 200, w.Code, "Should return 200 when ready")
			} else {
				assert.Equal(t, 503, w.Code, "Should return 503 when not ready")
			}
		}
	})
}
