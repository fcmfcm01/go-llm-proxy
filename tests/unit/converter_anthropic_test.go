package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Anthropic format conversion test vectors
// Based on FR-006 requirement for predefined test vectors
var anthropicToOpenAITestVectors = []struct {
	name      string
	anthropic AnthropicMessage
	openai    OpenAIMessage
}{
	{
		name: "Simple user message",
		anthropic: AnthropicMessage{
			Role:    "user",
			Content: "Hello, world!",
		},
		openai: OpenAIMessage{
			Role:    "user",
			Content: "Hello, world!",
		},
	},
	{
		name: "System message conversion",
		anthropic: AnthropicMessage{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
		openai: OpenAIMessage{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
	},
	{
		name: "Assistant message with content",
		anthropic: AnthropicMessage{
			Role:    "assistant",
			Content: "I can help you with that.",
		},
		openai: OpenAIMessage{
			Role:    "assistant",
			Content: "I can help you with that.",
		},
	},
	{
		name: "Message with tool_use",
		anthropic: AnthropicMessage{
			Role: "assistant",
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: "Let me use a tool",
				},
				{
					Type: "tool_use",
					ID:   "tool-1",
					Name: "get_weather",
					Input: map[string]interface{}{
						"location": "New York",
					},
				},
			},
		},
		openai: OpenAIMessage{
			Role:      "assistant",
			Content:   "Let me use a tool",
			ToolCalls: []OpenAIToolCall{},
		},
	},
	{
		name: "Message with tool_result",
		anthropic: AnthropicMessage{
			Role: "user",
			Content: []AnthropicContent{
				{
					Type:      "tool_result",
					ToolUseID: "tool-1",
					Content:   "The weather is sunny",
				},
			},
		},
		openai: OpenAIMessage{
			Role:       "tool",
			Content:    "The weather is sunny",
			ToolCallID: "tool-1",
		},
	},
}

// AnthropicMessage represents an Anthropic message
type AnthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// AnthropicContent represents content items in Anthropic messages
type AnthropicContent struct {
	Type      string                 `json:"type"`
	Text      string                 `json:"text,omitempty"`
	ID        string                 `json:"id,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Input     map[string]interface{} `json:"input,omitempty"`
	ToolUseID string                 `json:"tool_use_id,omitempty"`
	Content   string                 `json:"content,omitempty"`
}

// OpenAIMessage represents an OpenAI message
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

// OpenAIToolCall represents a tool call in OpenAI format
type OpenAIToolCall struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// TestAnthropicToOpenAIConverter tests the Anthropic to OpenAI format conversion
func TestAnthropicToOpenAIConverter(t *testing.T) {
	converter := NewAnthropicToOpenAIConverter()

	t.Run("ConvertSimpleMessage", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role:    "user",
			Content: "Hello!",
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.Equal(t, "user", openaiMsg.Role)
		assert.Equal(t, "Hello!", openaiMsg.Content)
	})

	t.Run("ConvertSystemMessage", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role:    "system",
			Content: "You are a helpful assistant.",
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.Equal(t, "system", openaiMsg.Role)
		assert.Equal(t, "You are a helpful assistant.", openaiMsg.Content)
	})

	t.Run("ConvertAssistantMessage", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role:    "assistant",
			Content: "I can help with that.",
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.Equal(t, "assistant", openaiMsg.Role)
		assert.Equal(t, "I can help with that.", openaiMsg.Content)
	})

	t.Run("ConvertWithToolUse", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role: "assistant",
			Content: []AnthropicContent{
				{
					Type: "text",
					Text: "Let me use a tool",
				},
				{
					Type: "tool_use",
					ID:   "tool-123",
					Name: "get_weather",
					Input: map[string]interface{}{
						"location": "London",
					},
				},
			},
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.Equal(t, "assistant", openaiMsg.Role)
		assert.Contains(t, openaiMsg.Content, "Let me use a tool")
	})

	t.Run("ConvertWithToolResult", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role: "user",
			Content: []AnthropicContent{
				{
					Type:      "tool_result",
					ToolUseID: "tool-123",
					Content:   "Sunny, 22°C",
				},
			},
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.Equal(t, "tool", openaiMsg.Role)
		assert.Equal(t, "Sunny, 22°C", openaiMsg.Content)
		assert.Equal(t, "tool-123", openaiMsg.ToolCallID)
	})

	t.Run("TestVectorConversion", func(t *testing.T) {
		for _, vector := range anthropicToOpenAITestVectors {
			t.Run(vector.name, func(t *testing.T) {
				openaiMsg, err := converter.ConvertMessage(vector.anthropic)
				require.NoError(t, err)
				assert.Equal(t, vector.openai.Role, openaiMsg.Role)
				// Note: Content conversion may vary for complex types
			})
		}
	})

	t.Run("InvalidRole", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role:    "invalid_role",
			Content: "Test content",
		}

		_, err := converter.ConvertMessage(athropicMsg)
		assert.Error(t, err)
	})

	t.Run("EmptyContent", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role:    "user",
			Content: "",
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.Equal(t, "", openaiMsg.Content)
	})

	t.Run("NilContent", func(t *testing.T) {
		anthropicMsg := AnthropicMessage{
			Role:    "user",
			Content: nil,
		}

		openaiMsg, err := converter.ConvertMessage(athropicMsg)
		require.NoError(t, err)
		assert.NotNil(t, openaiMsg)
	})
}

// TestAnthropicToOpenAIBatchConversion tests batch conversion of multiple messages
func TestAnthropicToOpenAIBatchConversion(t *testing.T) {
	converter := NewAnthropicToOpenAIConverter()

	t.Run("ConvertMessageBatch", func(t *testing.T) {
		anthropicMessages := []AnthropicMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello!"},
			{Role: "assistant", Content: "Hi there! How can I help?"},
		}

		openaiMessages, err := converter.ConvertMessages(athropicMessages)
		require.NoError(t, err)
		assert.Equal(t, 3, len(openaiMessages))
		assert.Equal(t, "system", openaiMessages[0].Role)
		assert.Equal(t, "user", openaiMessages[1].Role)
		assert.Equal(t, "assistant", openaiMessages[2].Role)
	})

	t.Run("ConvertEmptyBatch", func(t *testing.T) {
		anthropicMessages := []AnthropicMessage{}
		openaiMessages, err := converter.ConvertMessages(athropicMessages)
		require.NoError(t, err)
		assert.Equal(t, 0, len(openaiMessages))
	})

	t.Run("ConvertWithInvalidMessage", func(t *testing.T) {
		anthropicMessages := []AnthropicMessage{
			{Role: "user", Content: "Valid message"},
			{Role: "invalid", Content: "Invalid message"},
		}

		_, err := converter.ConvertMessages(athropicMessages)
		assert.Error(t, err)
	})
}

// TestAnthropicRequestConversion tests complete request conversion
func TestAnthropicRequestConversion(t *testing.T) {
	converter := NewAnthropicToOpenAIConverter()

	t.Run("ConvertChatRequest", func(t *testing.T) {
		anthropicRequest := map[string]interface{}{
			"model": "claude-3-haiku",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Hello!",
				},
			},
			"max_tokens": 100,
		}

		openaiRequest, err := converter.ConvertRequest(athropicRequest)
		require.NoError(t, err)
		assert.Equal(t, "gpt-3.5-turbo", openaiRequest["model"])
	})

	t.Run("ConvertRequestWithSystemMessage", func(t *testing.T) {
		anthropicRequest := map[string]interface{}{
			"model": "claude-3-sonnet",
			"messages": []map[string]interface{}{
				{
					"role":    "system",
					"content": "You are a helpful assistant.",
				},
				{
					"role":    "user",
					"content": "What can you do?",
				},
			},
		}

		openaiRequest, err := converter.ConvertRequest(aanthropicRequest)
		require.NoError(t, err)
		assert.NotNil(t, openaiRequest["messages"])
	})

	t.Run("ConvertRequestWithTools", func(t *testing.T) {
		anthropicRequest := map[string]interface{}{
			"model": "claude-3-opus",
			"messages": []map[string]interface{}{
				{
					"role":    "user",
					"content": "Use the weather tool",
				},
			},
			"tools": []map[string]interface{}{
				{
					"name":        "get_weather",
					"description": "Get weather information",
					"input_schema": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type": "string",
							},
						},
					},
				},
			},
		}

		openaiRequest, err := converter.ConvertRequest(aanthropicRequest)
		require.NoError(t, err)
		assert.NotNil(t, openaiRequest)
	})
}

// AnthropicToOpenAIConverter is the converter interface
type AnthropicToOpenAIConverter interface {
	ConvertMessage(msg AnthropicMessage) (OpenAIMessage, error)
	ConvertMessages(msgs []AnthropicMessage) ([]OpenAIMessage, error)
	ConvertRequest(req map[string]interface{}) (map[string]interface{}, error)
}

type anthropicToOpenAIConverter struct{}

// NewAnthropicToOpenAIConverter creates a new converter
func NewAnthropicToOpenAIConverter() AnthropicToOpenAIConverter {
	return &anthropicToOpenAIConverter{}
}

func (c *anthropicToOpenAIConverter) ConvertMessage(msg AnthropicMessage) (OpenAIMessage, error) {
	// Implementation pending - test-first development
	return OpenAIMessage{}, nil
}

func (c *anthropicToOpenAIConverter) ConvertMessages(msgs []AnthropicMessage) ([]OpenAIMessage, error) {
	// Implementation pending - test-first development
	return nil, nil
}

func (c *anthropicToOpenAIConverter) ConvertRequest(req map[string]interface{}) (map[string]interface{}, error) {
	// Implementation pending - test-first development
	return nil, nil
}
