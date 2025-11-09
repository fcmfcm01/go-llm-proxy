package contract

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/server"
)

// TestProviderStatusContract tests the GET /admin/api/v1/providers/status endpoint
func TestProviderStatusContract(t *testing.T) {
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

	t.Run("StatusEndpointExists", func(t *testing.T) {
		// Test that the status endpoint exists and returns a valid response
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Status endpoint should return 200")
	})

	t.Run("ValidStatusResponse", func(t *testing.T) {
		// Test that the endpoint returns a properly formatted response
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Expected 200 OK")

		// Should have valid JSON response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Response should be valid JSON")
		assert.NotNil(t, response["total"], "Response should have total field")
		assert.NotNil(t, response["enabled"], "Response should have enabled field")
		assert.NotNil(t, response["disabled"], "Response should have disabled field")
		assert.NotNil(t, response["providers"], "Response should have providers field")
	})

	t.Run("ResponseFormatMatchesContract", func(t *testing.T) {
		// Test response format matches admin-api.yaml contract
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		// Verify structure
		total, ok := response["total"].(float64)
		assert.True(t, ok, "total should be a number")
		assert.GreaterOrEqual(t, total, float64(0), "total should be >= 0")

		enabled, ok := response["enabled"].(float64)
		assert.True(t, ok, "enabled should be a number")
		assert.GreaterOrEqual(t, enabled, float64(0), "enabled should be >= 0")

		disabled, ok := response["disabled"].(float64)
		assert.True(t, ok, "disabled should be a number")
		assert.GreaterOrEqual(t, disabled, float64(0), "disabled should be >= 0")

		providers, ok := response["providers"].([]interface{})
		assert.True(t, ok, "providers should be an array")
		assert.NotNil(t, providers)
	})
}
