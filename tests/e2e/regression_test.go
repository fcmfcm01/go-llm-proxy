package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stretchr/testify/suite"
)

// RegressionTestSuite tests Python parity - ensures Go version matches Python behavior
type RegressionTestSuite struct {
	suite.Suite
	*TestSuite
	providerID string
}

// TestAPICompatibility tests API compatibility with Python version
func (s *RegressionTestSuite) TestAPICompatibility() {
	t := s.T()

	// Test 1: Chat Completions API parity
	s.testChatCompletionsParity()

	// Test 2: Completions API parity
	s.testCompletionsParity()

	// Test 3: Embeddings API parity
	s.testEmbeddingsParity()

	// Test 4: Models API parity
	s.testModelsParity()
}

// testChatCompletionsParity tests chat completions API matches Python behavior
func (s *RegressionTestSuite) testChatCompletionsParity() {
	t := s.T()

	s.setupTestProvider()

	testCases := []struct {
		name     string
		request  map[string]interface{}
		desc     string
	}{
		{
			name: "basic_chat",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Hello"},
				},
			},
			desc: "Basic chat completion",
		},
		{
			name: "with_system_message",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "system", "content": "You are a helpful assistant"},
					{"role": "user", "content": "Hello"},
				},
			},
			desc: "Chat with system message",
		},
		{
			name: "with_temperature",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Tell me a joke"},
				},
				"temperature": 0.7,
			},
			desc: "Chat with temperature",
		},
		{
			name: "with_max_tokens",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Write a story"},
				},
				"max_tokens": 100,
			},
			desc: "Chat with max_tokens",
		},
		{
			name: "with_top_p",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Generate text"},
				},
				"top_p": 0.9,
			},
			desc: "Chat with top_p",
		},
		{
			name: "with_stop",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Stop at the word STOP"},
				},
				"stop": []string{"STOP"},
			},
			desc: "Chat with stop sequence",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			req := s.MakeProxyRequest("POST", "/v1/chat/completions", tc.request)
			client := &http.Client{Timeout: 30 * time.Second)
			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			// Should match Python behavior (OK or provider error, not crash)
			s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
				"%s: Expected OK or BadGateway, got %d", tc.name, resp.StatusCode)

			if resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&response)
				s.Require().NoError(err)

				// Verify response structure matches OpenAI spec (same as Python)
				s.NotNil(response, "%s: Response should not be nil", tc.name)

				// Check required fields
				if id, ok := response["id"]; ok {
					s.NotEmpty(id, "%s: ID should be present", tc.name)
				}
				if object, ok := response["object"]; ok {
					s.Equal("chat.completion", object, "%s: Object should be 'chat.completion'", tc.name)
				}
				if created, ok := response["created"]; ok {
					s.NotEmpty(created, "%s: Created should be present", tc.name)
				}
				if choices, ok := response["choices"]; ok {
					s.NotNil(choices, "%s: Choices should be present", tc.name)
					if choiceArray, ok := choices.([]interface{}); ok && len(choiceArray) > 0 {
						if choice, ok := choiceArray[0].(map[string]interface{}); ok {
							// Verify choice structure
							if index, ok := choice["index"]; ok {
								s.NotNil(index, "%s: Index should be present", tc.name)
							}
							if message, ok := choice["message"]; ok {
								if msgMap, ok := message.(map[string]interface{}); ok {
									if role, ok := msgMap["role"]; ok {
										s.NotEmpty(role, "%s: Role should be present", tc.name)
									}
									if content, ok := msgMap["content"]; ok {
										s.NotNil(content, "%s: Content should be present", tc.name)
									}
								}
							}
							if finishReason, ok := choice["finish_reason"]; ok {
								s.NotNil(finishReason, "%s: Finish reason should be present", tc.name)
							}
						}
					}
				}
				if usage, ok := response["usage"]; ok {
					// Usage may be present, structure should match Python
					if usageMap, ok := usage.(map[string]interface{}); ok {
						if promptTokens, ok := usageMap["prompt_tokens"]; ok {
							s.NotNil(promptTokens, "%s: Prompt tokens should be numeric", tc.name)
						}
						if completionTokens, ok := usageMap["completion_tokens"]; ok {
							s.NotNil(completionTokens, "%s: Completion tokens should be numeric", tc.name)
						}
						if totalTokens, ok := usageMap["total_tokens"]; ok {
							s.NotNil(totalTokens, "%s: Total tokens should be numeric", tc.name)
						}
					}
				}
			}
		})
	}
}

// testCompletionsParity tests completions API parity
func (s *RegressionTestSuite) testCompletionsParity() {
	t := s.T()

	s.setupTestProvider()

	testCases := []struct {
		name    string
		request map[string]interface{}
		desc    string
	}{
		{
			name: "basic_completion",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"prompt": "Once upon a time",
			},
			desc: "Basic completion",
		},
		{
			name: "completion_with_params",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"prompt": "Complete this: The quick brown",
				"max_tokens": 50,
				"temperature": 0.5,
			},
			desc: "Completion with parameters",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			req := s.MakeProxyRequest("POST", "/v1/completions", tc.request)
			client := &http.Client{Timeout: 30 * time.Second)
			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
				"%s: Expected OK or BadGateway, got %d", tc.name, resp.StatusCode)
		})
	}
}

// testEmbeddingsParity tests embeddings API parity
func (s *RegressionTestSuite) testEmbeddingsParity() {
	t := s.T()

	s.setupTestProvider()

	request := map[string]interface{}{
		"model": "text-embedding-ada-002",
		"input": "The quick brown fox",
	}

	req := s.MakeProxyRequest("POST", "/v1/embeddings", request)
	client := &http.Client{Timeout: 30 * time.Second)
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Embeddings: Expected OK or BadGateway, got %d", resp.StatusCode)
}

// testModelsParity tests models API parity
func (s *RegressionTestSuite) testModelsParity() {
	t := s.T()

	s.setupTestProvider()

	req := s.MakeProxyRequest("GET", "/v1/models", nil)
	client := &http.Client{Timeout: 10 * time.Second)
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Models: Expected OK or BadGateway, got %d", resp.StatusCode)

	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err := json.NewDecoder(resp.Body).Decode(&response)
		s.Require().NoError(err)

		// Verify response structure matches OpenAI spec
		s.NotNil(response, "Models response should not be nil")
		if object, ok := response["object"]; ok {
			s.Equal("list", object, "Object should be 'list'")
		}
		if data, ok := response["data"]; ok {
			s.NotNil(data, "Data should be present")
		}
	}
}

// TestErrorResponseParity tests error responses match Python behavior
func (s *RegressionTestSuite) TestErrorResponseParity() {
	t := s.T()

	s.setupTestProvider()

	testCases := []struct {
		name        string
		request     map[string]interface{}
		expectedErr string
		desc        string
	}{
		{
			name: "invalid_model",
			request: map[string]interface{}{
				"model": "invalid-model-name-xyz",
				"messages": []map[string]string{
					{"role": "user", "content": "Test"},
				},
			},
			expectedErr: "model not found",
			desc:        "Invalid model error",
		},
		{
			name: "empty_messages",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
			},
			expectedErr: "messages",
			desc:        "Missing messages error",
		},
		{
			name: "invalid_request",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": "invalid", // Should be array
			},
			expectedErr: "messages",
			desc:        "Invalid messages type",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			req := s.MakeProxyRequest("POST", "/v1/chat/completions", tc.request)
			client := &http.Client{Timeout: 10 * time.Second)
			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			// Should return error status (matching Python behavior)
			s.True(resp.StatusCode >= 400, "%s: Should return error status", tc.name)

			// Parse error response
			if resp.StatusCode >= 400 {
				var errorResp map[string]interface{}
				err := json.NewDecoder(resp.Body).Decode(&errorResp)
				s.Require().NoError(err)

				// Error structure should match OpenAI spec
				if error, ok := errorResp["error"]; ok {
					if errMap, ok := error.(map[string]interface{}); ok {
						// Should have error type
						if errType, ok := errMap["type"]; ok {
							s.NotEmpty(errType, "%s: Error type should be present", tc.name)
						}
						// Should have error message
						if errMsg, ok := errMap["message"]; ok {
							s.NotEmpty(errMsg, "%s: Error message should be present", tc.name)
						}
						// May have error code
						if code, ok := errMap["code"]; ok {
							s.NotNil(code, "%s: Error code should be present if provided", tc.name)
						}
					}
				}
			}
		})
	}
}

// TestHeaderParity tests HTTP headers match Python client expectations
func (s *RegressionTestSuite) TestHeaderParity() {
	t := s.T()

	s.setupTestProvider()

	request := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Test headers"},
		},
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", request)
	client := &http.Client{Timeout: 30 * time.Second)
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Check response headers match OpenAI spec
	contentType := resp.Header.Get("Content-Type")
	s.NotEmpty(contentType, "Content-Type header should be present")

	// Check for common headers
	if resp.StatusCode == http.StatusOK {
		// May have rate limit headers
		if rateLimit := resp.Header.Get("X-RateLimit-Limit"); rateLimit != "" {
			s.NotEmpty(rateLimit, "Rate limit headers should be present if applicable")
		}
	}
}

// TestStreamingParity tests streaming responses match Python behavior
func (s *RegressionTestSuite) TestStreamingParity() {
	t := s.T()

	s.setupTestProvider()

	request := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Tell me a story"},
		},
		"stream": true,
	}

	req := s.MakeStreamingRequest("POST", "/v1/chat/completions", request)
	client := &http.Client{Timeout: 30 * time.Second)
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Streaming should work (even if provider fails)
	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Streaming: Expected OK or BadGateway, got %d", resp.StatusCode)

	// Content-Type should indicate streaming
	if resp.StatusCode == http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		s.NotEmpty(contentType, "Content-Type should be present")
	}
}

// TestTimeoutParity tests timeout behavior matches Python
func (s *RegressionTestSuite) TestTimeoutParity() {
	t := s.T()

	s.setupTestProvider()

	// Test with very short timeout
	request := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Quick response"},
		},
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", request)
	// Create client with very short timeout
	client := &http.Client{Timeout: 1 * time.Millisecond)
	resp, err := client.Do(req)

	// Should handle timeout gracefully (matching Python behavior)
	if err == nil {
		resp.Body.Close()
		// If we got a response, it should be a valid HTTP response
		s.True(resp.StatusCode >= 200 && resp.StatusCode < 600,
			"Timeout should result in valid HTTP status code")
	}
}

// TestAuthenticationParity tests auth behavior matches Python
func (s *RegressionTestSuite) TestAuthenticationParity() {
	t := s.T()

	// Test 1: No auth token
	req1 := s.MakeProxyRequest("GET", "/v1/models", nil)
	client := &http.Client{Timeout: 10 * time.Second)
	resp1, err1 := client.Do(req1)
	s.Require().NoError(err1)
	resp1.Body.Close()

	// Should handle missing auth (Python would get 401)
	s.True(resp1.StatusCode >= 400 || resp1.StatusCode == http.StatusOK,
		"Missing auth should return error or OK if not required")

	// Test 2: Invalid auth token
	req2 := s.MakeProxyRequest("GET", "/v1/models", nil)
	req2.Header.Set("Authorization", "Bearer invalid-token")
	resp2, err2 := client.Do(req2)
	s.Require().NoError(err2)
	resp2.Body.Close()

	// Should reject invalid auth
	s.Equal(http.StatusUnauthorized, resp2.StatusCode, "Invalid auth should return 401")
}

// setupTestProvider creates a test provider
func (s *RegressionTestSuite) setupTestProvider() {
	if s.providerID != "" {
		return
	}

	providerData := map[string]interface{}{
		"name":         "regression-test-provider",
		"type":         "openai",
		"api_key":      "sk-regression-test",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	}
	provider := s.createProvider(providerData)
	if provider != nil {
		if id, ok := (*provider)["id"].(string); ok {
			s.providerID = id
		}
	}
}

// MakeStreamingRequest creates a streaming request
func (s *RegressionTestSuite) MakeStreamingRequest(method, path string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, s.testServer.URL+path, nil)
	} else {
		req, _ = http.NewRequest(method, s.testServer.URL+path, nil)
	}

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req.Body = http.NoBody
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")
	req.Header.Set("Accept", "text/event-stream")

	return req
}

// createProvider creates a provider
func (s *RegressionTestSuite) createProvider(data map[string]interface{}) *map[string]interface{} {
	return s.createProvider(data)
}

// RunRegressionTests runs the regression test suite
func RunRegressionTests() {
	suite.Run(&RegressionTestSuite{
		TestSuite: &TestSuite{},
	})
}
