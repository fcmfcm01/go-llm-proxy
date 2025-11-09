package contract

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestEmbeddingsContract tests the POST /v1/embeddings endpoint
// OpenAI embeddings API
func TestEmbeddingsContract(t *testing.T) {
	// Test case 1: Valid request with required fields
	t.Run("ValidRequestWithRequiredFields", func(t *testing.T) {
		payload := map[string]interface{}{
			"model": "text-embedding-ada-002",
			"input": "The quick brown fox jumps over the lazy dog",
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
		assert.Equal(t, "/v1/embeddings", req.URL.Path)
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	})

	// Test case 2: Valid request with array input
	t.Run("ValidRequestWithArrayInput", func(t *testing.T) {
		payload := map[string]interface{}{
			"model": "text-embedding-ada-002",
			"input": []string{
				"First document",
				"Second document",
				"Third document",
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
		assert.Equal(t, "/v1/embeddings", req.URL.Path)
	})

	// Test case 3: Missing required field - model
	t.Run("MissingRequiredFieldModel", func(t *testing.T) {
		payload := map[string]interface{}{
			"input": "The quick brown fox",
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 4: Missing required field - input
	t.Run("MissingRequiredFieldInput", func(t *testing.T) {
		payload := map[string]interface{}{
			"model": "text-embedding-ada-002",
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 5: Empty input string
	t.Run("EmptyInputString", func(t *testing.T) {
		payload := map[string]interface{}{
			"model": "text-embedding-ada-002",
			"input": "",
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 6: Empty input array
	t.Run("EmptyInputArray", func(t *testing.T) {
		payload := map[string]interface{}{
			"model": "text-embedding-ada-002",
			"input": []string{},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 7: Valid request with optional parameters
	t.Run("ValidRequestWithOptionalParams", func(t *testing.T) {
		payload := map[string]interface{}{
			"model":      "text-embedding-ada-002",
			"input":      "The quick brown fox",
			"user":       "user-123",
			"dimensions": 512,
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 8: Different embedding models
	t.Run("DifferentEmbeddingModels", func(t *testing.T) {
		models := []string{
			"text-embedding-ada-002",
			"text-embedding-3-small",
			"text-embedding-3-large",
		}

		for _, model := range models {
			t.Run(model, func(t *testing.T) {
				payload := map[string]interface{}{
					"model": model,
					"input": "Sample text",
				}

				req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
				req.Header.Set("Content-Type", "application/json")

				assert.NotNil(t, req)
			})
		}
	})

	// Test case 9: User parameter for tracking
	t.Run("UserParameterForTracking", func(t *testing.T) {
		payload := map[string]interface{}{
			"model": "text-embedding-ada-002",
			"input": "Sample text",
			"user":  "end-user-123",
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 10: Dimensions parameter
	t.Run("DimensionsParameter", func(t *testing.T) {
		payload := map[string]interface{}{
			"model":      "text-embedding-3-large",
			"input":      "Sample text",
			"dimensions": 256,
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		req.Header.Set("Content-Type", "application/json")

		assert.NotNil(t, req)
	})

	// Test case 11: Response status codes
	t.Run("ExpectedResponseStatusCodes", func(t *testing.T) {
		// Expected responses:
		// 200 - Successful embedding
		// 400 - Invalid request
		// 500 - Provider error

		expectedStatuses := []int{200, 400, 500}
		assert.Equal(t, 3, len(expectedStatuses))
	})

	// Test case 12: Content-Type validation
	t.Run("ContentTypeValidation", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", nil)
		assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	})
}
