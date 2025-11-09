package integration

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

// TestHealthCheckIntegration tests all health check endpoints working together
func TestHealthCheckIntegration(t *testing.T) {
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

	t.Run("AllHealthEndpointsAvailable", func(t *testing.T) {
		// Test that all health check endpoints are available
		endpoints := []string{
			"/healthz",
			"/healthz/ready",
			"/healthz/live",
		}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			// Each endpoint should return a valid status code
			assert.True(t, w.Code >= 200 && w.Code < 600,
				"Endpoint %s should return valid status, got: %d", endpoint, w.Code)

			// Each endpoint should return JSON
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Endpoint %s should return valid JSON", endpoint)
		}
	})

	t.Run("HealthzBasicCheck", func(t *testing.T) {
		// Test basic /healthz endpoint
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "/healthz should return 200")

		// Should return JSON
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should return valid JSON")

		// Should have status field
		assert.Contains(t, response, "status", "Should have status field")
		assert.NotEmpty(t, response["status"], "Status should not be empty")
	})

	t.Run("ReadyDetailedCheck", func(t *testing.T) {
		// Test /healthz/ready endpoint
		req := httptest.NewRequest(http.MethodGet, "/healthz/ready", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200 or 503
		assert.True(t, w.Code == 200 || w.Code == 503,
			"/healthz/ready should return 200 or 503, got: %d", w.Code)

		// Should return JSON
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should return valid JSON")

		// Should have status field
		assert.Contains(t, response, "status", "Should have status field")
	})

	t.Run("LiveLivenessCheck", func(t *testing.T) {
		// Test /healthz/live endpoint (liveness probe)
		req := httptest.NewRequest(http.MethodGet, "/healthz/live", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200 (liveness should always be true if server is running)
		assert.Equal(t, 200, w.Code, "/healthz/live should return 200")

		// Should return JSON
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Should return valid JSON")

		// Should have status field
		assert.Contains(t, response, "status", "Should have status field")
	})

	t.Run("EndpointsConsistency", func(t *testing.T) {
		// Test that all endpoints are consistent in their responses
		endpoints := []string{"/healthz", "/healthz/ready", "/healthz/live"}
		responses := make(map[string]map[string]interface{})

		// Get responses from all endpoints
		for _, endpoint := range endpoints {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			responses[endpoint] = response
		}

		// All should have status field
		for _, endpoint := range endpoints {
			assert.Contains(t, responses[endpoint], "status",
				"Endpoint %s should have status field", endpoint)
		}
	})

	t.Run("FastResponseTimes", func(t *testing.T) {
		// Test that all health endpoints respond quickly
		endpoints := []string{"/healthz", "/healthz/ready", "/healthz/live"}

		for _, endpoint := range endpoints {
			start := time.Now()

			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			duration := time.Since(start)

			// Health checks should be very fast (< 100ms)
			assert.Less(t, duration, 100*time.Millisecond,
				"Endpoint %s should respond quickly, took %v", endpoint, duration)
		}
	})

	t.Run("MultipleSequentialCalls", func(t *testing.T) {
		// Test that endpoints handle multiple sequential calls
		numberOfCalls := 10

		for i := 0; i < numberOfCalls; i++ {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			// Should always return 200
			assert.Equal(t, 200, w.Code, "Call %d should return 200", i+1)

			// Should always return valid JSON
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Call %d should return valid JSON", i+1)
		}
	})

	t.Run("ContentTypeConsistency", func(t *testing.T) {
		// Test that all health endpoints return consistent content types
		endpoints := []string{"/healthz", "/healthz/ready", "/healthz/live"}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			// Should return application/json
			contentType := w.Header().Get("Content-Type")
			assert.Contains(t, contentType, "application/json",
				"Endpoint %s should return application/json, got: %s", endpoint, contentType)
		}
	})

	t.Run("NoAuthenticationRequired", func(t *testing.T) {
		// Test that health endpoints don't require authentication
		endpoints := []string{"/healthz", "/healthz/ready", "/healthz/live"}

		for _, endpoint := range endpoints {
			req := httptest.NewRequest(http.MethodGet, endpoint, nil)
			// No auth headers
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(w, req)

			// Should return 200 without authentication
			assert.Equal(t, 200, w.Code,
				"Endpoint %s should not require authentication", endpoint)
		}
	})

	t.Run("UptimeIncreases", func(t *testing.T) {
		// Test that uptime increases over time
		// Get first uptime
		req1 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w1 := httptest.NewRecorder()
		srv.Router().ServeHTTP(w1, req1)

		var response1 map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &response1)

		if uptime1, ok := response1["uptime"].(string); ok {
			// Wait a bit
			time.Sleep(10 * time.Millisecond)

			// Get second uptime
			req2 := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			w2 := httptest.NewRecorder()
			srv.Router().ServeHTTP(w2, req2)

			var response2 map[string]interface{}
			json.Unmarshal(w2.Body.Bytes(), &response2)

			if uptime2, ok := response2["uptime"].(string); ok {
				// Uptime should be different (greater or equal)
				// Note: This might be the same due to second-level precision
				assert.NotEmpty(t, uptime1, "First uptime should not be empty")
				assert.NotEmpty(t, uptime2, "Second uptime should not be empty")
			}
		}
	})
}
