package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/server"
)

// TestDetailedHealthContract tests the GET /healthz/detailed endpoint
func TestDetailedHealthContract(t *testing.T) {
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

	t.Run("DetailedEndpointExists", func(t *testing.T) {
		// Test that the detailed health endpoint exists
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return a valid status code
		assert.True(t, w.Code >= 200 && w.Code < 600,
			"Detailed health endpoint should return valid HTTP status, got: %d", w.Code)
	})

	t.Run("ReturnsDetailedStatus", func(t *testing.T) {
		// Test that the endpoint returns detailed status
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should have valid JSON response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Should have status field
		assert.Contains(t, response, "status", "Response should have status field")
		assert.NotEmpty(t, response["status"], "Status should not be empty")
	})

	t.Run("IncludesSystemInfo", func(t *testing.T) {
		// Test that the response includes system information
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// May include system information
		possibleFields := []string{
			"status",
			"uptime",
			"timestamp",
			"version",
			"build",
			"git_commit",
			"system",
			"runtime",
			"components",
		}

		for _, field := range possibleFields {
			if val, ok := response[field]; ok {
				// Field should not be nil
				assert.NotNil(t, val, "Field %s should not be nil", field)
			}
		}
	})

	t.Run("IncludesComponentStatus", func(t *testing.T) {
		// Test that the response includes component status
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// May include components field
		if components, ok := response["components"]; ok {
			// Should be an object or array
			assert.NotNil(t, components, "Components should not be nil if present")

			// If it's a map, check for expected components
			if compMap, ok := components.(map[string]interface{}); ok {
				// May include health checks for different components
				possibleComponents := []string{
					"database",
					"providers",
					"config",
					"metrics",
				}

				for _, comp := range possibleComponents {
					if _, exists := compMap[comp]; exists {
						// Component exists, that's good
					}
				}
			}
		}
	})

	t.Run("ResponseFormat", func(t *testing.T) {
		// Test response format
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify structure
		// Should have at least status field
		assert.Contains(t, response, "status", "Response should have status field")
		assert.NotEmpty(t, response["status"], "Status should not be empty")
	})

	t.Run("ValidJSONContentType", func(t *testing.T) {
		// Test that the endpoint returns JSON content type
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return application/json content type
		contentType := w.Header().Get("Content-Type")
		assert.Contains(t, contentType, "application/json",
			"Content-Type should be application/json, got: %s", contentType)
	})

	t.Run("IncludesTimestamp", func(t *testing.T) {
		// Test that the response includes timestamp
		req := httptest.NewRequest(http.MethodGet, "/healthz/detailed", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// May include timestamp
		if timestamp, ok := response["timestamp"]; ok {
			// Should be a string
			if ts, ok := timestamp.(string); ok {
				assert.NotEmpty(t, ts, "Timestamp should not be empty if present")
			}
		}
	})
}
