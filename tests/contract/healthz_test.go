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

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/server"
)

// TestHealthzContract tests the GET /healthz endpoint per health-metrics.yaml
func TestHealthzContract(t *testing.T) {
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

	t.Run("HealthEndpointExists", func(t *testing.T) {
		// Test that the health endpoint exists
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Health endpoint should return 200")
	})

	t.Run("ReturnsHealthyStatus", func(t *testing.T) {
		// Test that the endpoint returns a healthy status
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Expected 200 OK")

		// Should have valid JSON response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Should have status field
		assert.Contains(t, response, "status", "Response should have status field")
		assert.Equal(t, "healthy", response["status"], "Status should be 'healthy'")
	})

	t.Run("ReturnsUptime", func(t *testing.T) {
		// Test that the endpoint returns uptime information
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Should have uptime field
		assert.Contains(t, response, "uptime", "Response should have uptime field")

		// Uptime should be a duration string
		if uptime, ok := response["uptime"].(string); ok {
			assert.NotEmpty(t, uptime, "Uptime should not be empty")
			// Can be in formats like "1h30m45s", "5m", "30s", etc.
		}
	})

	t.Run("ResponseFormatMatchesContract", func(t *testing.T) {
		// Test response format matches health-metrics.yaml contract
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify structure
		status, ok := response["status"].(string)
		assert.True(t, ok, "status should be a string")
		assert.NotEmpty(t, status, "status should not be empty")

		uptime, ok := response["uptime"].(string)
		assert.True(t, ok, "uptime should be a string")
		assert.NotEmpty(t, uptime, "uptime should not be empty")
	})

	t.Run("ValidJSONContentType", func(t *testing.T) {
		// Test that the endpoint returns JSON content type
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return application/json content type
		contentType := w.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json",
			"Content-Type should be application/json, got: %s", contentType)
	})

	t.Run("FastResponse", func(t *testing.T) {
		// Test that health endpoint responds quickly
		start := time.Now()

		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		duration := time.Since(start)

		// Should respond within 10ms for basic health check
		assert.Less(t, duration, 10*time.Millisecond,
			"Health endpoint should respond quickly, took %v", duration)

		// Should still return 200
		assert.Equal(t, 200, w.Code)
	})

	t.Run("AlwaysReturnsHealthyWhenServerRunning", func(t *testing.T) {
		// Test that health endpoint returns healthy when server is running
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			// Should always return 200
			assert.Equal(t, 200, w.Code, "Should return 200 on call %d", i+1)

			// Should always return healthy status
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)

			assert.Contains(t, response, "status", "Call %d should have status", i+1)
		}
	})

	t.Run("NoAuthenticationRequired", func(t *testing.T) {
		// Test that health endpoint doesn't require authentication
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		// No auth headers
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200 without authentication
		assert.Equal(t, 200, w.Code, "Health endpoint should not require authentication")
	})
}
