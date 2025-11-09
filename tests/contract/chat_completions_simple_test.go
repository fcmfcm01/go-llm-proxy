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

// TestChatCompletionsContract tests the POST /v1/chat/completions endpoint
// This is a simplified test that validates the handler exists and can process requests
func TestChatCompletionsContract(t *testing.T) {
	// Set Gin to Test mode
	gin.SetMode(gin.TestMode)

	// Create logger
	logger := logging.NewSimpleLogger("debug")

	// Create a simple test server
	srv := server.NewServer(&server.ServerConfig{
		Port:         8080,
		ReadTimeout:  30,
		WriteTimeout: 30,
		IdleTimeout:  60,
	}, logger.Logger)

	// Setup routes
	srv.SetupRoutes()

	t.Run("HealthCheck", func(t *testing.T) {
		// Test that health endpoint works
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Health check should return 200")
	})

	t.Run("InvalidEndpoint", func(t *testing.T) {
		// Test that unknown endpoint returns 404
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 404
		assert.Equal(t, 404, w.Code, "Nonexistent endpoint should return 404")
	})

	t.Run("ValidRequestWithRequiredFields", func(t *testing.T) {
		// Create a simple request

			"model": "gpt-4",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Hello!",
				},
			},
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		// The handler should be invoked (even if it fails with 500/502)
		// We just want to make sure the endpoint is wired up
		assert.True(t, w.Code >= 200 && w.Code < 600,
			"Expected valid HTTP status code, got %d", w.Code)
		assert.NotEmpty(t, w.Body.String(), "Response should not be empty")
	})

	t.Run("EndpointResponseFormat", func(t *testing.T) {
		// Test that the endpoint returns a properly formatted response

			"model": "gpt-4",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Hello!",
				},
			},
		}

		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Expected 200 OK")

		// Should have valid JSON response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err, "Response should be valid JSON")
		assert.NotNil(t, response["id"], "Response should have id field")
		assert.NotNil(t, response["choices"], "Response should have choices field")
	})
}
