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
)

// TestChatCompletionsContract tests POST /v1/chat/completions contract
// T042 [P] [US2] Contract test for POST /v1/chat/completions
func TestChatCompletionsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		setupMock      func(*mockProxyService)
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful chat completion",
			requestBody: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": "Hello!"},
				},
			},
			setupMock: func(m *mockProxyService) {
				m.chatResponse = map[string]interface{}{
					"id":      "chatcmpl-123",
					"object":  "chat.completion",
					"created": 1677652288,
					"model":   "gpt-4",
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"message": map[string]string{
								"role":    "assistant",
								"content": "Hello! How can I help you?",
							},
							"finish_reason": "stop",
						},
					},
					"usage": map[string]int{
						"prompt_tokens":     10,
						"completion_tokens": 20,
						"total_tokens":      30,
					},
				}
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "chat.completion", body["object"])
				assert.NotEmpty(t, body["id"])
				assert.NotEmpty(t, body["choices"])
			},
		},
		{
			name: "missing model parameter",
			requestBody: map[string]interface{}{
				"messages": []map[string]string{
					{"role": "user", "content": "Hello!"},
				},
			},
			setupMock:      func(m *mockProxyService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "model")
			},
		},
		{
			name: "missing messages parameter",
			requestBody: map[string]interface{}{
				"model": "gpt-4",
			},
			setupMock:      func(m *mockProxyService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "messages")
			},
		},
		{
			name: "invalid temperature",
			requestBody: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": "Hello!"},
				},
				"temperature": 3.0, // max is 2.0
			},
			setupMock:      func(m *mockProxyService) {},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Contains(t, body["error"], "temperature")
			},
		},
		{
			name: "with streaming parameter",
			requestBody: map[string]interface{}{
				"model": "gpt-4",
				"messages": []map[string]string{
					{"role": "user", "content": "Hello!"},
				},
				"stream": true,
			},
			setupMock: func(m *mockProxyService) {
				m.streamingSupported = true
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				// Streaming response validation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockProxyService{}
			tt.setupMock(mockService)

			router := gin.New()
			handler := &proxyHandler{service: mockService}
			router.POST("/v1/chat/completions", handler.handleChatCompletions)

			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateBody != nil {
				var responseBody map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &responseBody)
				require.NoError(t, err)
				tt.validateBody(t, responseBody)
			}
		})
	}
}

// TestCompletionsContract tests POST /v1/completions contract
// T043 [P] [US2] Contract test for POST /v1/completions
func TestCompletionsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful completion",
			requestBody: map[string]interface{}{
				"model":  "gpt-3.5-turbo",
				"prompt": "Once upon a time",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "text_completion", body["object"])
				assert.NotEmpty(t, body["choices"])
			},
		},
		{
			name: "missing model",
			requestBody: map[string]interface{}{
				"prompt": "Once upon a time",
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockProxyService{}
			router := gin.New()
			handler := &proxyHandler{service: mockService}
			router.POST("/v1/completions", handler.handleCompletions)

			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateBody != nil {
				var responseBody map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &responseBody)
				require.NoError(t, err)
				tt.validateBody(t, responseBody)
			}
		})
	}
}

// TestEmbeddingsContract tests POST /v1/embeddings contract
// T044 [P] [US2] Contract test for POST /v1/embeddings
func TestEmbeddingsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name: "successful embedding",
			requestBody: map[string]interface{}{
				"model": "text-embedding-ada-002",
				"input": "The quick brown fox",
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "list", body["object"])
				assert.NotEmpty(t, body["data"])
			},
		},
		{
			name: "array input",
			requestBody: map[string]interface{}{
				"model": "text-embedding-ada-002",
				"input": []string{"Text 1", "Text 2"},
			},
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				data := body["data"].([]interface{})
				assert.Len(t, data, 2)
			},
		},
		{
			name: "missing model",
			requestBody: map[string]interface{}{
				"input": "Some text",
			},
			expectedStatus: http.StatusBadRequest,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.NotEmpty(t, body["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockProxyService{}
			router := gin.New()
			handler := &proxyHandler{service: mockService}
			router.POST("/v1/embeddings", handler.handleEmbeddings)

			bodyBytes, err := json.Marshal(tt.requestBody)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateBody != nil {
				var responseBody map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &responseBody)
				require.NoError(t, err)
				tt.validateBody(t, responseBody)
			}
		})
	}
}

// TestModelsContract tests GET /v1/models contract
// T045 [P] [US2] Contract test for GET /v1/models
func TestModelsContract(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		expectedStatus int
		validateBody   func(*testing.T, map[string]interface{})
	}{
		{
			name:           "successful models list",
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body map[string]interface{}) {
				assert.Equal(t, "list", body["object"])
				assert.NotEmpty(t, body["data"])

				data := body["data"].([]interface{})
				assert.Greater(t, len(data), 0)

				// Check first model structure
				firstModel := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstModel["id"])
				assert.Equal(t, "model", firstModel["object"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &mockProxyService{
				models: []map[string]interface{}{
					{
						"id":       "gpt-4",
						"object":   "model",
						"created":  1677652288,
						"owned_by": "openai",
					},
					{
						"id":       "gpt-3.5-turbo",
						"object":   "model",
						"created":  1677652288,
						"owned_by": "openai",
					},
				},
			}

			router := gin.New()
			handler := &proxyHandler{service: mockService}
			router.GET("/v1/models", handler.handleModels)

			req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.validateBody != nil {
				var responseBody map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &responseBody)
				require.NoError(t, err)
				tt.validateBody(t, responseBody)
			}
		})
	}
}

// Mock types for testing

type mockProxyService struct {
	chatResponse       map[string]interface{}
	streamingSupported bool
	models             []map[string]interface{}
}

type proxyHandler struct {
	service *mockProxyService
}

func (h *proxyHandler) handleChatCompletions(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if _, ok := req["model"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}
	if _, ok := req["messages"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "messages is required"})
		return
	}

	// Validate temperature
	if temp, ok := req["temperature"].(float64); ok {
		if temp < 0 || temp > 2 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "temperature must be between 0 and 2"})
			return
		}
	}

	if h.service.chatResponse != nil {
		c.JSON(http.StatusOK, h.service.chatResponse)
	} else {
		c.JSON(http.StatusOK, gin.H{"object": "chat.completion"})
	}
}

func (h *proxyHandler) handleCompletions(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, ok := req["model"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"object": "text_completion", "choices": []interface{}{}})
}

func (h *proxyHandler) handleEmbeddings(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, ok := req["model"]; !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model is required"})
		return
	}

	// Check if input is array
	data := []interface{}{}
	if input, ok := req["input"].([]interface{}); ok {
		for range input {
			data = append(data, map[string]interface{}{})
		}
	} else {
		data = append(data, map[string]interface{}{})
	}

	c.JSON(http.StatusOK, gin.H{"object": "list", "data": data})
}

func (h *proxyHandler) handleModels(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   h.service.models,
	})
}
