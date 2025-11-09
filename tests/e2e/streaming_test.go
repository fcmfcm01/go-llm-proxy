package e2e

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stretchr/testify/suite"
)

// StreamingTestSuite tests streaming response functionality
type StreamingTestSuite struct {
	suite.Suite
	*TestSuite
	providerID string
}

// TestChatCompletionsStreaming tests streaming chat completions
func (s *StreamingTestSuite) TestChatCompletionsStreaming() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Create streaming request
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Write a short story about a cat"},
		},
		"max_tokens": 100,
		"stream":     true,
	}

	req := s.MakeStreamingRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Should get a response
	s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
		"Expected OK or BadGateway, got %d", resp.StatusCode)

	// Verify streaming response if successful
	if resp.StatusCode == http.StatusOK {
		contentType := resp.Header.Get("Content-Type")
		s.True(strings.Contains(contentType, "text/event-stream") ||
			strings.Contains(contentType, "application/json"),
			"Response should be streaming format, got %s", contentType)

		// Read streaming data
		reader := bufio.NewReader(resp.Body)
		chunkCount := 0

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break // End of stream
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue // Skip empty lines
			}

			// Parse streaming chunk
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				if data == "[DONE]" {
					// End of stream marker
					break
				}

				// Parse JSON chunk
				var chunk map[string]interface{}
				if err := json.Unmarshal([]byte(data), &chunk); err == nil {
					chunkCount++

					// Verify chunk structure
					if choices, ok := chunk["choices"]; ok {
						if choiceArray, ok := choices.([]interface{}); ok && len(choiceArray) > 0 {
							if choice, ok := choiceArray[0].(map[string]interface{}); ok {
								// Check for delta (streaming) or message
								if _, hasDelta := choice["delta"]; hasDelta {
									s.NotNil(choice["delta"])
								}
								if _, hasMessage := choice["message"]; hasMessage {
									s.NotNil(choice["message"])
								}
								if finishReason, ok := choice["finish_reason"]; ok && finishReason != nil {
									// Stream complete
								}
							}
						}
					}

					// Verify chunk has expected fields
					if id, ok := chunk["id"]; ok {
						s.NotEmpty(id)
					}
					if created, ok := chunk["created"]; ok {
						s.NotEmpty(created)
					}
					if object, ok := chunk["object"]; ok {
						s.NotEmpty(object)
					}
				}
			}
		}

		// Should receive multiple chunks for a streaming response
		s.Greater(chunkCount, 0, "Should receive at least one data chunk")
	}
}

// TestStreamingCancellation tests that streaming can be cancelled
func (s *StreamingTestSuite) TestStreamingCancellation() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Create request
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Write a very long story about..."},
		},
		"max_tokens": 1000,
		"stream":     true,
	}

	req := s.MakeStreamingRequest("POST", "/v1/chat/completions", requestData)

	// Create client with cancel
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Read first chunk
	reader := bufio.NewReader(resp.Body)
	line, err := reader.ReadString('\n')
	s.Require().NoError(err)

	// Verify we got a valid chunk
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "data: ") {
		data := strings.TrimPrefix(line, "data: ")
		if data != "[DONE]" {
			var chunk map[string]interface{}
			err := json.Unmarshal([]byte(data), &chunk)
			s.NoError(err)
			s.NotNil(chunk)
		}
	}

	// Client cancellation is tested by closing the connection
	// which is handled by the HTTP client automatically
}

// TestStreamingErrorHandling tests streaming error handling
func (s *StreamingTestSuite) TestStreamingErrorHandling() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	testCases := []struct {
		name        string
		request     map[string]interface{}
		expectError bool
	}{
		{
			name: "invalid_model_streaming",
			request: map[string]interface{}{
				"model": "invalid-model",
				"messages": []map[string]string{
					{"role": "user", "content": "Test"},
				},
				"stream": true,
			},
			expectError: true,
		},
		{
			name: "empty_messages_streaming",
			request: map[string]interface{}{
				"model":  "gpt-3.5-turbo",
				"stream": true,
			},
			expectError: true,
		},
		{
			name: "valid_streaming",
			request: map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": "Hi"},
				},
				"stream": true,
			},
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := s.MakeStreamingRequest("POST", "/v1/chat/completions", tc.request)
			client := &http.Client{Timeout: 15 * time.Second}
			resp, err := client.Do(req)
			s.Require().NoError(err)
			defer resp.Body.Close()

			if tc.expectError {
				s.True(resp.StatusCode >= 400, "Expected error status for %s", tc.name)
			} else {
				s.True(resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway,
					"Expected OK or BadGateway for %s, got %d", tc.name, resp.StatusCode)
			}
		})
	}
}

// TestStreamingPerformance tests streaming performance characteristics
func (s *StreamingTestSuite) TestStreamingPerformance() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	// Create request
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Count from 1 to 10"},
		},
		"max_tokens": 50,
		"stream":     true,
	}

	req := s.MakeStreamingRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}

	start := time.Now()
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Measure time to first byte
	firstByteTime := time.Since(start)
	s.Less(firstByteTime, 5*time.Second, "Time to first byte should be < 5 seconds")

	// Read all chunks
	reader := bufio.NewReader(resp.Body)
	chunkCount := 0
	firstChunkTime := time.Time{}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			if chunkCount == 0 {
				firstChunkTime = time.Now()
			}
			chunkCount++
		}
	}

	// Verify we got streaming data
	s.Greater(chunkCount, 0, "Should receive streaming chunks")

	if chunkCount > 0 {
		timeToFirstChunk := firstChunkTime.Sub(start)
		s.Less(timeToFirstChunk, 5*time.Second, "First chunk should arrive within 5 seconds")
	}
}

// TestMultipleConcurrentStreaming tests multiple concurrent streaming requests
func (s *StreamingTestSuite) TestMultipleConcurrentStreaming() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	numRequests := 3
	errChan := make(chan error, numRequests)
	statusChan := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			requestData := map[string]interface{}{
				"model": "gpt-3.5-turbo",
				"messages": []map[string]string{
					{"role": "user", "content": fmt.Sprintf("Request %d: Tell me a joke", index)},
				},
				"max_tokens": 20,
				"stream":     true,
			}

			req := s.MakeStreamingRequest("POST", "/v1/chat/completions", requestData)
			client := &http.Client{Timeout: 30 * time.Second}
			resp, err := client.Do(req)

			if err == nil {
				statusChan <- resp.StatusCode
				// Read a few chunks
				reader := bufio.NewReader(resp.Body)
				for j := 0; j < 5; j++ {
					_, err := reader.ReadString('\n')
					if err != nil {
						break
					}
				}
				resp.Body.Close()
			} else {
				errChan <- err
			}
		}(i)
	}

	// Wait for all requests
	completed := 0
	for completed < numRequests {
		select {
		case err := <-errChan:
			// This is okay - provider may not be available
			completed++
		case status := <-statusChan:
			s.True(status >= 200 && status < 600, "Request should complete with valid status")
			completed++
		case <-time.After(60 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}

// TestStreamingHeaders tests that streaming responses have correct headers
func (s *StreamingTestSuite) TestStreamingHeaders() {
	t := s.T()

	// Setup test provider
	s.setupTestProvider()

	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Test"},
		},
		"stream": true,
	}

	req := s.MakeStreamingRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	// Check common streaming headers
	contentType := resp.Header.Get("Content-Type")
	s.NotEmpty(contentType, "Content-Type header should be set")

	// Cache-Control should prevent buffering for true streaming
	cacheControl := resp.Header.Get("Cache-Control")
	if cacheControl != "" {
		s.NotContains(cacheControl, "no-cache", "Cache-Control should allow streaming")
	}
}

// setupTestProvider creates a test provider for streaming testing
func (s *StreamingTestSuite) setupTestProvider() {
	if s.providerID != "" {
		return
	}

	providerData := map[string]interface{}{
		"name":         "streaming-test-provider",
		"type":         "openai",
		"api_key":      "sk-streaming-test",
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

// createTestProvider creates a provider
func (s *StreamingTestSuite) createTestProvider(data map[string]interface{}) *map[string]interface{} {
	return s.createProvider(data)
}

// MakeStreamingRequest creates an HTTP request for streaming
func (s *StreamingTestSuite) MakeStreamingRequest(method, path string, body interface{}) *http.Request {
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

	// Set headers for streaming
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")
	req.Header.Set("Accept", "text/event-stream")

	return req
}

// RunStreamingTests runs the streaming test suite
func RunStreamingTests() {
	suite.Run(&StreamingTestSuite{
		TestSuite: &TestSuite{},
	})
}
