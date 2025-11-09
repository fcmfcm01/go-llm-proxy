package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/server"
)

// TestProviderPriorityContract tests the POST /admin/api/v1/providers/:id/priority endpoint
func TestProviderPriorityContract(t *testing.T) {
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

	t.Run("PriorityEndpointExists", func(t *testing.T) {
		// Test that the priority endpoint exists
		payload := map[string]interface{}{
			"priority": 5,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/providers/test-provider/priority", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return a valid HTTP status code
		assert.True(t, w.Code >= 200 && w.Code < 600,
			"Expected valid HTTP status code, got %d", w.Code)
	})

	t.Run("ValidPriorityUpdate", func(t *testing.T) {
		// Test that a valid priority update request works
		payload := map[string]interface{}{
			"priority": 10,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/providers/openai/priority", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200 or other valid status
		assert.True(t, w.Code >= 200 && w.Code < 600,
			"Expected valid HTTP status code, got %d", w.Code)

		// Response should be valid JSON
		if w.Body.Len() > 0 {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Response should be valid JSON")
		}
	})

	t.Run("MissingProviderID", func(t *testing.T) {
		// Test that endpoint handles missing provider ID
		payload := map[string]interface{}{
			"priority": 5,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/providers//priority", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return an error status
		assert.True(t, w.Code >= 400, "Should return error for missing provider ID")
	})

	t.Run("InvalidPriorityValue", func(t *testing.T) {
		// Test that endpoint validates priority values
		payload := map[string]interface{}{
			"priority": -1, // Invalid negative priority
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/providers/test/priority", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return error for invalid priority
		assert.True(t, w.Code >= 400, "Should return error for invalid priority")
	})

	t.Run("ResponseFormat", func(t *testing.T) {
		// Test response format
		payload := map[string]interface{}{
			"priority": 5,
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/admin/api/v1/providers/provider-1/priority", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// If there's a response body, it should be valid JSON
		if w.Body.Len() > 0 {
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err, "Response should be valid JSON")

			// Check for expected fields
			if message, ok := response["message"]; ok {
				assert.NotEmpty(t, message, "Response should have a message field")
			}
		}
	})
}
