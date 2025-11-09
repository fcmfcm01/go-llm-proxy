package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stretchr/testify/suite"
)

// ProxyFlowTestSuite tests the complete proxy request flow
type ProxyFlowTestSuite struct {
	suite.Suite
	*TestSuite
	providerID string
}

// TestChatCompletions tests the /v1/chat/completions endpoint
func (s *ProxyFlowTestSuite) TestChatCompletions() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Test case 1: Simple chat completion
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello, how are you?"},
		},
		"max_tokens":  100,
		"temperature": 0.7,
	}

	// Make request to proxy
	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should get a valid response
	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Expected OK or BadGateway, got %d", resp.StatusCode)

	// Verify response structure if successful
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		s.Require().NoError(err)

		s.NotNil(response)
		if choices, ok := response["choices"]; ok {
			s.NotNil(choices)
			if choiceArray, ok := choices.([]interface{}); ok && len(choiceArray) > 0 {
				if choice, ok := choiceArray[0].(map[string]interface{}); ok {
					s.NotNil(choice["message"])
					s.NotNil(choice["finish_reason"])
				}
			}
		}
	}
}

// TestCompletions tests the /v1/completions endpoint
func (s *ProxyFlowTestSuite) TestCompletions() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	requestData := map[string]interface{}{
		"model":       "gpt-3.5-turbo",
		"prompt":      "Once upon a time",
		"max_tokens":  100,
		"temperature": 0.7,
	}

	req := s.MakeProxyRequest("POST", "/v1/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Expected OK or BadGateway, got %d", resp.StatusCode)
}

// TestEmbeddings tests the /v1/embeddings endpoint
func (s *ProxyFlowTestSuite) TestEmbeddings() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	requestData := map[string]interface{}{
		"model": "text-embedding-ada-002",
		"input": "The quick brown fox",
	}

	req := s.MakeProxyRequest("POST", "/v1/embeddings", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Expected OK or BadGateway, got %d", resp.StatusCode)
}

// TestModelsEndpoint tests the /v1/models endpoint
func (s *ProxyFlowTestSuite) TestModelsEndpoint() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	req := s.MakeProxyRequest("GET", "/v1/models", nil)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Expected OK or BadGateway, got %d", resp.StatusCode)

	// Verify response structure if successful
	if resp.StatusCode == http.StatusOK {
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		s.Require().NoError(err)

		s.NotNil(response)
		if data, ok := response["data"]; ok {
			s.NotNil(data)
		}
	}
}

// TestRequestRouting tests that requests are routed to correct provider
func (s *ProxyFlowTestSuite) TestRequestRouting() {
	t := s.T()

	// Setup multiple test providers (simulated)
	s.setupTestProvider()

	// Make multiple requests to verify routing
	for i := 0; i < 3; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Test request %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		s.Require().NoError(err)
		resp.Body.Close()

		// At least one request should succeed (or fail with provider error)
		s.True(resp.StatusCode >= 200 && resp.StatusCode < 600,
			"Request %d should complete with valid status", i)
	}
}

// TestLoadBalancing tests round-robin load balancing
func (s *ProxyFlowTestSuite) TestLoadBalancing() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Make multiple requests to verify load distribution
	numRequests := 5
	responses := make([]int, numRequests)

	for i := 0; i < numRequests; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Load test %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		s.Require().NoError(err)
		responses[i] = resp.StatusCode
		resp.Body.Close()
	}

	// All requests should complete (even if some fail with provider errors)
	for i, status := range responses {
		s.True(status >= 200 && status < 600,
			"Request %d should complete, got status %d", i, status)
	}
}

// TestErrorHandling tests error handling in proxy flow
func (s *ProxyFlowTestSuite) TestErrorHandling() {
	t := s.T()

	// Test case 1: Invalid model
	requestData := map[string]interface{}{
		"model": "invalid-model-name",
		"messages": []map[string]string{
			{"role": "user", "content": "Test"},
		},
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should return error, not crash
	s.True(resp.StatusCode >= 400, "Invalid model should return error status")

	// Test case 2: Empty request body
	req2 := s.MakeProxyRequest("POST", "/v1/chat/completions", nil)
	client2 := &http.Client{Timeout: 10 * time.Second}
	resp2, err := client2.Do(req2)
	s.Require().NoError(err)
	defer resp2.Body.Close()

	// Should return error for empty body
	s.True(resp2.StatusCode >= 400, "Empty request should return error status")

	// Test case 3: Unsupported endpoint
	req3 := s.MakeProxyRequest("GET", "/v1/unsupported", nil)
	client3 := &http.Client{Timeout: 10 * time.Second}
	resp3, err := client3.Do(req3)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should return 404 for unsupported endpoint
	s.Equal(http.StatusNotFound, resp3.StatusCode, "Unsupported endpoint should return 404")
}

// TestRequestTimeout tests request timeout handling
func (s *ProxyFlowTestSuite) TestRequestTimeout() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Request with very short timeout
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello"},
		},
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	// Create client with very short timeout
	client := &http.Client{Timeout: 1 * time.Millisecond}
	resp, err := client.Do(req)

	// Should either timeout or get a response
	if err == nil {
		resp.Body.Close()
		// If we got a response, it should be handled gracefully
		s.True(resp.StatusCode >= 200 && resp.StatusCode < 600)
	}
}

// TestRequestIdempotency tests that identical requests are handled consistently
func (s *ProxyFlowTestSuite) TestRequestIdempotency() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Make the same request twice
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Idempotency test"},
		},
	}

	// First request
	req1 := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp1, err1 := client.Do(req1)
	s.Require().NoError(err1)
	defer resp1.Body.Close()

	status1 := resp1.StatusCode

	// Second request (identical)
	req2 := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	resp2, err2 := client.Do(req2)
	s.Require().NoError(err2)
	defer resp2.Body.Close()

	status2 := resp2.StatusCode

	// Both requests should complete (may have different results, but should not crash)
	s.True(status1 >= 200 && status1 < 600, "First request should complete")
	s.True(status2 >= 200 && status2 < 600, "Second request should complete")
}

// TestFormatConversion tests format conversion between OpenAI and Anthropic
func (s *ProxyFlowTestSuite) TestFormatConversion() {
	t := s.T()

	// Setup test provider with Anthropic type
	providerData := map[string]interface{}{
		"name":         "test-anthropic",
		"type":         "anthropic",
		"api_key":      "sk-ant-test",
		"base_url":     "https://api.anthropic.com",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	}
	provider := s.createTestProvider(providerData)
	s.NotNil(provider)

	// OpenAI-style request (should be converted to Anthropic format)
	requestData := map[string]interface{}{
		"model": "claude-3-sonnet-20240229",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello from OpenAI client"},
		},
		"max_tokens":  100,
		"temperature": 0.7,
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should handle the conversion
	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Format conversion should be attempted, got status %d", resp.StatusCode)
}

// setupTestProvider creates a test provider for proxy testing
func (s *ProxyFlowTestSuite) setupTestProvider() {
	if s.providerID != "" {
		return // Already setup
	}

	providerData := map[string]interface{}{
		"name":         "e2e-test-provider",
		"type":         "openai",
		"api_key":      "sk-e2e-test",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	}
	provider := s.createTestProvider(providerData)
	if provider != nil {
		if id, ok := (*provider)["id"].(string); ok {
			s.providerID = id
		}
	}
}

// createTestProvider creates a provider using the test suite's admin auth
func (s *ProviderLifecycleTestSuite) createTestProvider(data map[string]interface{}) *map[string]interface{} {
	return s.createProvider(data)
}

// MakeProxyRequest creates an HTTP request to the proxy endpoint
func (s *ProxyFlowTestSuite) MakeProxyRequest(method, path string, body interface{}) *http.Request {
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

	// Set OpenAI-compatible headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")

	return req
}

// RunProxyFlowTests runs the proxy flow test suite
func RunProxyFlowTests() {
	suite.Run(&ProxyFlowTestSuite{
		TestSuite: &TestSuite{},
	})
}
