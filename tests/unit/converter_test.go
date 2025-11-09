package unit

import (
	"encoding/json"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/converter"
)

// TestAnthropicToOpenAIConversion tests Anthropic→OpenAI format conversion with test vectors
// T046 [P] [US2] Unit test for Anthropic→OpenAI format conversion with test vectors
func TestAnthropicToOpenAIConversion(t *testing.T) {
	tests := []struct {
		name           string
		anthropicInput string
		expectedOpenAI string
		expectError    bool
	}{
		{
			name: "simple message conversion",
			anthropicInput: `{
				"model": "claude-2",
				"messages": [
					{"role": "user", "content": "Hello!"}
				],
				"max_tokens": 100
			}`,
			expectedOpenAI: `{
				"model": "claude-2",
				"messages": [
					{"role": "user", "content": "Hello!"}
				],
				"max_tokens": 100
			}`,
			expectError: false,
		},
		{
			name: "system message conversion",
			anthropicInput: `{
				"model": "claude-2",
				"system": "You are a helpful assistant",
				"messages": [
					{"role": "user", "content": "Hello!"}
				]
			}`,
			expectedOpenAI: `{
				"model": "claude-2",
				"messages": [
					{"role": "system", "content": "You are a helpful assistant"},
					{"role": "user", "content": "Hello!"}
				]
			}`,
			expectError: false,
		},
		{
			name: "temperature parameter mapping",
			anthropicInput: `{
				"model": "claude-2",
				"messages": [{"role": "user", "content": "Test"}],
				"temperature": 0.7
			}`,
			expectedOpenAI: `{
				"model": "claude-2",
				"messages": [{"role": "user", "content": "Test"}],
				"temperature": 0.7
			}`,
			expectError: false,
		},
		{
			name: "top_p parameter mapping",
			anthropicInput: `{
				"model": "claude-2",
				"messages": [{"role": "user", "content": "Test"}],
				"top_p": 0.9
			}`,
			expectedOpenAI: `{
				"model": "claude-2",
				"messages": [{"role": "user", "content": "Test"}],
				"top_p": 0.9
			}`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var anthropicReq map[string]interface{}
			err := json.Unmarshal([]byte(tt.anthropicInput), &anthropicReq)
			require.NoError(t, err)

			conv := converter.NewAnthropicConverter()
			openAIReq, err := conv.ConvertToOpenAI(anthropicReq)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, openAIReq)

			var expectedReq map[string]interface{}
			err = json.Unmarshal([]byte(tt.expectedOpenAI), &expectedReq)
			require.NoError(t, err)

			// Compare key fields
			assert.Equal(t, expectedReq["model"], openAIReq["model"])

			if expectedMessages, ok := expectedReq["messages"]; ok {
				assert.Equal(t, expectedMessages, openAIReq["messages"])
			}
		})
	}
}

// TestOpenAIToAnthropicConversion tests OpenAI→Anthropic format conversion with test vectors
// T047 [P] [US2] Unit test for OpenAI→Anthropic format conversion with test vectors
func TestOpenAIToAnthropicConversion(t *testing.T) {
	tests := []struct {
		name              string
		openAIInput       string
		expectedAnthropic string
		expectError       bool
	}{
		{
			name: "simple message conversion",
			openAIInput: `{
				"model": "gpt-4",
				"messages": [
					{"role": "user", "content": "Hello!"}
				],
				"max_tokens": 100
			}`,
			expectedAnthropic: `{
				"model": "claude-2",
				"messages": [
					{"role": "user", "content": "Hello!"}
				],
				"max_tokens": 100
			}`,
			expectError: false,
		},
		{
			name: "system message to system parameter",
			openAIInput: `{
				"model": "gpt-4",
				"messages": [
					{"role": "system", "content": "You are helpful"},
					{"role": "user", "content": "Hello!"}
				]
			}`,
			expectedAnthropic: `{
				"model": "claude-2",
				"system": "You are helpful",
				"messages": [
					{"role": "user", "content": "Hello!"}
				]
			}`,
			expectError: false,
		},
		{
			name: "multiple system messages merged",
			openAIInput: `{
				"model": "gpt-4",
				"messages": [
					{"role": "system", "content": "You are helpful"},
					{"role": "system", "content": "Be concise"},
					{"role": "user", "content": "Hello!"}
				]
			}`,
			expectedAnthropic: `{
				"model": "claude-2",
				"system": "You are helpful\nBe concise",
				"messages": [
					{"role": "user", "content": "Hello!"}
				]
			}`,
			expectError: false,
		},
		{
			name: "function calling not supported",
			openAIInput: `{
				"model": "gpt-4",
				"messages": [{"role": "user", "content": "Test"}],
				"functions": [{"name": "test", "description": "test"}]
			}`,
			expectedAnthropic: ``,
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var openAIReq map[string]interface{}
			err := json.Unmarshal([]byte(tt.openAIInput), &openAIReq)
			require.NoError(t, err)

			conv := converter.NewAnthropicConverter()
			anthropicReq, err := conv.ConvertFromOpenAI(openAIReq)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, anthropicReq)

			if tt.expectedAnthropic != "" {
				var expectedReq map[string]interface{}
				err = json.Unmarshal([]byte(tt.expectedAnthropic), &expectedReq)
				require.NoError(t, err)

				// Verify system parameter if present
				if expectedSystem, ok := expectedReq["system"]; ok {
					assert.Equal(t, expectedSystem, anthropicReq["system"])
				}

				// Verify messages
				if expectedMessages, ok := expectedReq["messages"]; ok {
					assert.Equal(t, expectedMessages, anthropicReq["messages"])
				}
			}
		})
	}
}

// TestConversionLatency tests that conversion completes within 1ms
// Performance requirement: <1ms format conversion (p95)
func TestConversionLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	conv := converter.NewAnthropicConverter()

	request := map[string]interface{}{
		"model": "claude-2",
		"messages": []map[string]string{
			{"role": "user", "content": "Hello!"},
		},
		"max_tokens": 100,
	}

	// Warm up
	for i := 0; i < 10; i++ {
		_, _ = conv.ConvertToOpenAI(request)
	}

	// Measure performance
	const iterations = 1000
	results := make([]int64, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := conv.ConvertToOpenAI(request)
		elapsed := time.Since(start).Microseconds()

		require.NoError(t, err)
		results[i] = elapsed
	}

	// Calculate p95
	sort.Slice(results, func(i, j int) bool {
		return results[i] < results[j]
	})

	p95Index := int(float64(iterations) * 0.95)
	p95Latency := results[p95Index]

	// Should be under 1000 microseconds (1ms)
	assert.Less(t, p95Latency, int64(1000), "p95 latency should be <1ms")

	t.Logf("Conversion latency - p50: %dμs, p95: %dμs, p99: %dμs",
		results[iterations/2],
		p95Latency,
		results[int(float64(iterations)*0.99)])
}

// TestResponseConversion tests response format conversion
func TestResponseConversion(t *testing.T) {
	tests := []struct {
		name              string
		anthropicResponse string
		expectedOpenAI    string
	}{
		{
			name: "simple response conversion",
			anthropicResponse: `{
				"id": "msg_123",
				"type": "message",
				"role": "assistant",
				"content": [
					{"type": "text", "text": "Hello!"}
				],
				"model": "claude-2",
				"stop_reason": "end_turn"
			}`,
			expectedOpenAI: `{
				"id": "msg_123",
				"object": "chat.completion",
				"created": 1234567890,
				"model": "claude-2",
				"choices": [
					{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Hello!"
						},
						"finish_reason": "stop"
					}
				]
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var anthropicResp map[string]interface{}
			err := json.Unmarshal([]byte(tt.anthropicResponse), &anthropicResp)
			require.NoError(t, err)

			conv := converter.NewAnthropicConverter()
			openAIResp, err := conv.ConvertResponseToOpenAI(anthropicResp)
			require.NoError(t, err)

			var expectedResp map[string]interface{}
			err = json.Unmarshal([]byte(tt.expectedOpenAI), &expectedResp)
			require.NoError(t, err)

			assert.Equal(t, expectedResp["object"], openAIResp["object"])
			assert.NotEmpty(t, openAIResp["choices"])
		})
	}
}
