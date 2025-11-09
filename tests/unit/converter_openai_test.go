package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// OpenAI to Anthropic format conversion test vectors
var openAIToAnthropicTestVectors = []struct {
	name      string
	openai    OpenAIMessage
	anthropic AnthropicMessage
}{
	{
		name: "Simple user message",
		openai: OpenAIMessage{
			Role:    "user",
			Content: "Hello!",
		},
		anthropic: AnthropicMessage{
			Role:    "user",
			Content: "Hello!",
		},
	},
	{
		name: "System message",
		openai: OpenAIMessage{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
		anthropic: AnthropicMessage{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
	},
	{
		name: "Assistant message",
		openai: OpenAIMessage{
			Role:    "assistant",
			Content: "I can help you with that.",
		},
		anthropic: AnthropicMessage{
			Role:    "assistant",
			Content: "I can help you with that.",
		},
	},
	{
		name: "Tool call message",
		openai: OpenAIMessage{
			Role:    "assistant",
			Content: "Let me use a tool",
			ToolCalls: []OpenAIToolCall{
				{
					ID:   "call-123",
					Type: "function",
					Function: OpenAIFunction{
						Name:      "get_weather",
						Arguments: `{"location": "Paris"}`,
					},
				},
			},
		},
		anthropic: AnthropicMessage{
			Role: "assistant",
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: "Let me use a tool",
				},
				{
					Type: "tool_use",
					ID:   "call-123",
					Name: "get_weather",
					Input: map[string]interface{}{
						"location": "Paris",
					},
				},
			},
		},
	},
	{
		name: "Tool result message",
		openai: OpenAIMessage{
			Role:       "tool",
			Content:    "The weather is sunny",
			ToolCallID: "call-123",
		},
		anthropic: AnthropicMessage{
			Role: "user",
			Content: []AnthropicContent{
				{
					Type:      "tool_result",
					ToolUseID: "call-123",
					Content:   "The weather is sunny",
				},
			},
		},
	},
}

// OpenAIFunction represents a function call in OpenAI format
type OpenAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// TestOpenAIToAnthropicConverter tests the OpenAI to Anthropic format conversion
func TestOpenAIToAnthropicConverter(t *testing.T) {
	converter := NewOpenAIToAnthropicConverter()

	t.Run("ConvertSimpleMessage", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:    "user",
			Content: "Hello!",
		}

		anthropicMsg, err := converter.ConvertMessage(openaiMsg)
		require.NoError(t, err)
		assert.Equal(t, "user", anthropicMsg.Role)
		assert.Equal(t, "Hello!", anthropicMsg.Content)
	})

	t.Run("ConvertSystemMessage", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:    "system",
			Content: "You are a helpful assistant.",
		}

		anthropicMsg, err := converter.ConvertMessage(openaiMsg)
		require.NoError(t, err)
		assert.Equal(t, "system", anthropicMsg.Role)
		assert.Equal(t, "You are a helpful assistant.", anthropicMsg.Content)
	})

	t.Run("ConvertAssistantMessage", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:    "assistant",
			Content: "I can help with that.",
		}

		anthropicMsg, err := converter.ConvertMessage(openaiMsg)
		require.NoError(t, err)
		assert.Equal(t, "assistant", anthropicMsg.Role)
		assert.Equal(t, "I can help with that.", anthropicMsg.Content)
	})

	t.Run("ConvertWithToolCalls", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:    "assistant",
			Content: "Let me use a tool",
			ToolCalls: []OpenAIToolCall{
				{
					ID:   "call-123",
					Type: "function",
					Function: OpenAIFunction{
						Name:      "get_weather",
						Arguments: `{"location": "London"}`,
					},
				},
			},
		}

		anthropicMsg, err := converter.ConvertMessage(openaiMsg)
		require.NoError(t, err)
		assert.Equal(t, "assistant", anthropicMsg.Role)
		assert.Contains(t, anthropicMsg.Content, "Let me use a tool")
	})

	t.Run("ConvertWithToolResult", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:       "tool",
			Content:    "Rainy, 15°C",
			ToolCallID: "call-123",
		}

		anthropicMsg, err := converter.ConvertMessage(openaiMsg)
		require.NoError(t, err)
		assert.Equal(t, "user", anthropicMsg.Role)
	})

	t.Run("TestVectorConversion", func(t *testing.T) {
		for _, vector := range openAIToAnthropicTestVectors {
			t.Run(vector.name, func(t *testing.T) {
				anthropicMsg, err := converter.ConvertMessage(vector.openai)
				require.NoError(t, err)
				assert.Equal(t, vector.anthropic.Role, anthropicMsg.Role)
				// Note: Content conversion may vary for complex types
			})
		}
	})

	t.Run("InvalidRole", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:    "invalid_role",
			Content: "Test content",
		}

		_, err := converter.ConvertMessage(openaiMsg)
		assert.Error(t, err)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		openaiMsg := OpenAIMessage{
			Role:    "user",
			Content: "",
		}

		anthropicMsg, err := converter.ConvertMessage(openaiMsg)
		require.NoError(t, err)
		assert.Equal(t, "", anthropicMsg.Content)
	})
}

// TestOpenAIToAnthropicBatchConversion tests batch conversion
func TestOpenAIToAnthropicBatchConversion(t *testing.T) {
	converter := NewOpenAIToAnthropicConverter()

	t.Run("ConvertMessageBatch", func(t *testing.T) {
		openaiMessages := []OpenAIMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
			{Role: "assistant", Content: "Hi there!"},
		}

		anthropicMessages, err := converter.ConvertMessages(openaiMessages)
		require.NoError(t, err)
		assert.Equal(t, 3, len(anthropicMessages))
		assert.Equal(t, "system", anthropicMessages[0].Role)
		assert.Equal(t, "user", anthropicMessages[1].Role)
		assert.Equal(t, "assistant", anthropicMessages[2].Role)
	})

	t.Run("ConvertEmptyBatch", func(t *testing.T) {
		openaiMessages := []OpenAIMessage{}
		anthropicMessages, err := converter.ConvertMessages(openaiMessages)
		require.NoError(t, err)
		assert.Equal(t, 0, len(anthropicMessages))
	})

	t.Run("ConvertWithInvalidMessage", func(t *testing.T) {
		openaiMessages := []OpenAIMessage{
			{Role: "user", Content: "Valid"},
			{Role: "invalid", Content: "Invalid"},
		}

		_, err := converter.ConvertMessages(openaiMessages)
		assert.Error(t, err)
	})
}

// TestOpenAIRequestConversion tests complete request conversion
func TestOpenAIRequestConversion(t *testing.T) {
	converter := NewOpenAIToAnthropicConverter()

	t.Run("ConvertChatRequest", func(t *testing.T) {
		openaiRequest := map[string]interface{}{
			"model": "gpt-4",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Hello!",
				},
			},
			"max_tokens": 100,
		}

		anthropicRequest, err := converter.ConvertRequest(openaiRequest)
		require.NoError(t, err)
		assert.Equal(t, "claude-3-haiku", anthropicRequest["model"])
	})

	t.Run("ConvertRequestWithTools", func(t *testing.T) {
		openaiRequest := map[string]interface{}{
			"model": "gpt-4-turbo",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Use the calculator",
				},
			},
			"tools": []map[string]interface{}{
				{
					"type": "function",
					"function": map[string]interface{}{
						"name":        "calculator",
						"description": "Perform calculations",
						"parameters": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"operation": map[string]interface{}{
									"type": "string",
								},
							},
						},
					},
				},
			},
		}

		anthropicRequest, err := converter.ConvertRequest(openaiRequest)
		require.NoError(t, err)
		assert.NotNil(t, anthropicRequest)
	})

	t.Run("ConvertRequestWithStream", func(t *testing.T) {
		openaiRequest := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]interface{}{
				{"role": "user", "content": "Hello"},
			},
			"stream": true,
		}

		anthropicRequest, err := converter.ConvertRequest(openaiRequest)
		require.NoError(t, err)
		assert.NotNil(t, anthropicRequest)
	})
}

// OpenAIToAnthropicConverter is the converter interface
type OpenAIToAnthropicConverter interface {
	ConvertMessage(msg OpenAIMessage) (AnthropicMessage, error)
	ConvertMessages(msgs []OpenAIMessage) ([]AnthropicMessage, error)
	ConvertRequest(req map[string]interface{}) (map[string]interface{}, error)
}

type openAIToAnthropicConverter struct{}

// NewOpenAIToAnthropicConverter creates a new converter
func NewOpenAIToAnthropicConverter() OpenAIToAnthropicConverter {
	return &openAIToAnthropicConverter{}
}

func (c *openAIToAnthropicConverter) ConvertMessage(msg OpenAIMessage) (AnthropicMessage, error) {
	// Implementation pending - test-first development
	return AnthropicMessage{}, nil
}

func (c *openAIToAnthropicConverter) ConvertMessages(msgs []OpenAIMessage) ([]AnthropicMessage, error) {
	// Implementation pending - test-first development
	return nil, nil
}

func (c *openAIToAnthropicConverter) ConvertRequest(req map[string]interface{}) (map[string]interface{}, error) {
	// Implementation pending - test-first development
	return nil, nil
}
