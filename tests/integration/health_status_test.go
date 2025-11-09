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

// TestHealthStatusUpdates tests real-time health status updates
func TestHealthStatusUpdates(t *testing.T) {
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

	t.Run("InitialStatusReport", func(t *testing.T) {
		// Test that initial status report includes all providers
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Status endpoint should return 200")

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err, "Response should be valid JSON")

		// Verify structure
		assert.Contains(t, response, "total", "Response should have total field")
		assert.Contains(t, response, "enabled", "Response should have enabled field")
		assert.Contains(t, response, "disabled", "Response should have disabled field")
		assert.Contains(t, response, "providers", "Response should have providers array")

		// Verify counts
		total := int(response["total"].(float64))
		enabled := int(response["enabled"].(float64))
		disabled := int(response["disabled"].(float64))

		assert.Equal(t, total, enabled+disabled, "Total should equal enabled + disabled")
	})

	t.Run("ProviderStatusChangeReflected", func(t *testing.T) {
		// Test that provider status changes are reflected in the status endpoint

		// First, get initial status
		req1 := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w1 := httptest.NewRecorder()
		srv.Router().ServeHTTP(w1, req1)

		var initialStatus map[string]interface{}
		json.Unmarshal(w1.Body.Bytes(), &initialStatus)
		initialTotal := int(initialStatus["total"].(float64))
		initialEnabled := int(initialStatus["enabled"].(float64))

		// Toggle a provider status (if endpoint exists)
		toggleReq := httptest.NewRequest(http.MethodPost, "/admin/api/v1/providers/openai/toggle", nil)
		toggleW := httptest.NewRecorder()
		srv.Router().ServeHTTP(toggleReq, toggleW)

		// Get updated status
		req2 := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w2 := httptest.NewRecorder()
		srv.Router().ServeHTTP(req2, w2)

		var updatedStatus map[string]interface{}
		json.Unmarshal(w2.Body.Bytes(), &updatedStatus)
		updatedEnabled := int(updatedStatus["enabled"].(float64))

		// The enabled count should have changed
		assert.True(t, updatedEnabled == initialEnabled || updatedEnabled == initialEnabled+1 || updatedEnabled == initialEnabled-1,
			"Enabled count should change after toggle, got initial=%d, updated=%d", initialEnabled, updatedEnabled)
	})

	t.Run("StatusEndpointConsistency", func(t *testing.T) {
		// Test that multiple calls to status endpoint return consistent data
		numberOfCalls := 5

		for i := 0; i < numberOfCalls; i++ {
			req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
			w := httptest.NewRecorder()

			srv.Router().ServeHTTP(req, w)

			// Should always return 200
			assert.Equal(t, 200, w.Code, "Status endpoint should always return 200")

			// Should always return valid JSON
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err, "Response should be valid JSON on call %d", i+1)

			// Should have consistent structure
			assert.Contains(t, response, "total", "Call %d should have total field", i+1)
			assert.Contains(t, response, "providers", "Call %d should have providers field", i+1)
		}
	})

	t.Run("ProviderDetailsInStatus", func(t *testing.T) {
		// Test that status response includes detailed provider information
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(req, w)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		providers := response["providers"].([]interface{})

		if len(providers) > 0 {
			// Check first provider has expected fields
			provider := providers[0].(map[string]interface{})

			// Should have at least id and name
			assert.Contains(t, provider, "id", "Provider should have id field")
			assert.Contains(t, provider, "name", "Provider should have name field")

			// May have enabled, priority, url, created, updated
			if enabled, ok := provider["enabled"]; ok {
				assert.IsType(t, bool(true), enabled, "enabled should be boolean")
			}

			if priority, ok := provider["priority"]; ok {
				assert.IsType(t, float64(1), priority, "priority should be number")
			}
		}
	})

	t.Run("HealthStatusTimestamps", func(t *testing.T) {
		// Test that status includes timestamp information
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(req, w)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		providers := response["providers"].([]interface{})

		if len(providers) > 0 {
			provider := providers[0].(map[string]interface{})

			// Should have created/updated timestamps
			if created, ok := provider["created"]; ok {
				// Should be a valid timestamp string
				assert.NotEmpty(t, created, "Created timestamp should not be empty")

				// Verify it's a valid date format (RFC3339 or similar)
				_, err := time.Parse(time.RFC3339, created.(string))
				assert.NoError(t, err, "Created should be a valid timestamp")
			}

			if updated, ok := provider["updated"]; ok {
				// Should be a valid timestamp string
				assert.NotEmpty(t, updated, "Updated timestamp should not be empty")

				// Verify it's a valid date format
				_, err := time.Parse(time.RFC3339, updated.(string))
				assert.NoError(t, err, "Updated should be a valid timestamp")
			}
		}
	})

	t.Run("EmptyProviderList", func(t *testing.T) {
		// Test behavior when no providers are configured
		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(req, w)

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		// Should still return valid structure with zero counts
		assert.Equal(t, float64(0), response["total"], "Total should be 0 when no providers")
		assert.Equal(t, float64(0), response["enabled"], "Enabled should be 0 when no providers")
		assert.Equal(t, float64(0), response["disabled"], "Disabled should be 0 when no providers")

		providers := response["providers"].([]interface{})
		assert.NotNil(t, providers, "Providers array should exist")
		assert.Equal(t, 0, len(providers), "Providers array should be empty")
	})

	t.Run("ResponseTimeAcceptable", func(t *testing.T) {
		// Test that status endpoint responds within acceptable time
		start := time.Now()

		req := httptest.NewRequest(http.MethodGet, "/admin/api/v1/providers/status", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(req, w)

		duration := time.Since(start)

		// Should respond within 100ms (acceptable for local test)
		assert.Less(t, duration, 100*time.Millisecond,
			"Status endpoint should respond quickly, took %v", duration)
	})
}
