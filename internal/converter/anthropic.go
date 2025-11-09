package converter

import (
	"fmt"
	"strings"
)

// AnthropicConverter handles conversion between Anthropic and OpenAI formats
type AnthropicConverter struct {
	modelMapping map[string]string
}

// NewAnthropicConverter creates a new Anthropic converter
func NewAnthropicConverter() *AnthropicConverter {
	return &AnthropicConverter{
		modelMapping: map[string]string{
			"gpt-4":          "claude-2",
			"gpt-3.5-turbo":  "claude-instant-1",
			"claude-2":       "gpt-4",
			"claude-instant": "gpt-3.5-turbo",
		},
	}
}

// ConvertToOpenAI converts Anthropic format request to OpenAI format
func (c *AnthropicConverter) ConvertToOpenAI(anthropicReq map[string]interface{}) (map[string]interface{}, error) {
	openAIReq := make(map[string]interface{})

	// Copy model
	if model, ok := anthropicReq["model"].(string); ok {
		openAIReq["model"] = model
	}

	// Handle system parameter - convert to system message
	messages := []interface{}{}
	if system, ok := anthropicReq["system"].(string); ok && system != "" {
		messages = append(messages, map[string]interface{}{
			"role":    "system",
			"content": system,
		})
	}

	// Copy messages
	if anthropicMessages, ok := anthropicReq["messages"].([]interface{}); ok {
		messages = append(messages, anthropicMessages...)
	} else if anthropicMessages, ok := anthropicReq["messages"].([]map[string]interface{}); ok {
		for _, msg := range anthropicMessages {
			messages = append(messages, msg)
		}
	}

	openAIReq["messages"] = messages

	// Copy optional parameters
	copyIfPresent(anthropicReq, openAIReq, "temperature")
	copyIfPresent(anthropicReq, openAIReq, "top_p")
	copyIfPresent(anthropicReq, openAIReq, "max_tokens")
	copyIfPresent(anthropicReq, openAIReq, "stream")
	copyIfPresent(anthropicReq, openAIReq, "stop")

	// Map top_k to n (best_of parameter)
	if topK, ok := anthropicReq["top_k"].(float64); ok {
		openAIReq["n"] = int(topK)
	}

	return openAIReq, nil
}

// ConvertFromOpenAI converts OpenAI format request to Anthropic format
func (c *AnthropicConverter) ConvertFromOpenAI(openAIReq map[string]interface{}) (map[string]interface{}, error) {
	anthropicReq := make(map[string]interface{})

	// Map model
	if model, ok := openAIReq["model"].(string); ok {
		if mapped, exists := c.modelMapping[model]; exists {
			anthropicReq["model"] = mapped
		} else {
			anthropicReq["model"] = model
		}
	}

	// Extract system messages and regular messages
	var systemMessages []string
	var regularMessages []interface{}

	if messages, ok := openAIReq["messages"].([]interface{}); ok {
		for _, msg := range messages {
			msgMap, ok := msg.(map[string]interface{})
			if !ok {
				continue
			}

			role, _ := msgMap["role"].(string)
			content, _ := msgMap["content"].(string)

			if role == "system" {
				systemMessages = append(systemMessages, content)
			} else {
				regularMessages = append(regularMessages, msg)
			}
		}
	}

	// Merge system messages into system parameter
	if len(systemMessages) > 0 {
		anthropicReq["system"] = strings.Join(systemMessages, "\n")
	}

	anthropicReq["messages"] = regularMessages

	// Copy optional parameters
	copyIfPresent(openAIReq, anthropicReq, "temperature")
	copyIfPresent(openAIReq, anthropicReq, "top_p")
	copyIfPresent(openAIReq, anthropicReq, "max_tokens")
	copyIfPresent(openAIReq, anthropicReq, "stream")
	copyIfPresent(openAIReq, anthropicReq, "stop")

	// Check for unsupported features
	if _, ok := openAIReq["functions"]; ok {
		return nil, fmt.Errorf("function calling is not supported by Anthropic")
	}
	if _, ok := openAIReq["function_call"]; ok {
		return nil, fmt.Errorf("function calling is not supported by Anthropic")
	}

	return anthropicReq, nil
}

// ConvertResponseToOpenAI converts Anthropic response to OpenAI format
func (c *AnthropicConverter) ConvertResponseToOpenAI(anthropicResp map[string]interface{}) (map[string]interface{}, error) {
	openAIResp := make(map[string]interface{})

	// Set object type
	openAIResp["object"] = "chat.completion"

	// Copy ID
	if id, ok := anthropicResp["id"].(string); ok {
		openAIResp["id"] = id
	}

	// Set created timestamp (use current time if not provided)
	if created, ok := anthropicResp["created"].(int64); ok {
		openAIResp["created"] = created
	} else {
		openAIResp["created"] = getCurrentTimestamp()
	}

	// Copy model
	if model, ok := anthropicResp["model"].(string); ok {
		openAIResp["model"] = model
	}

	// Convert content to choices format
	choices := []interface{}{}

	if content, ok := anthropicResp["content"].([]interface{}); ok {
		// Extract text from content array
		var text strings.Builder
		for _, item := range content {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if itemType, _ := itemMap["type"].(string); itemType == "text" {
					if textContent, ok := itemMap["text"].(string); ok {
						text.WriteString(textContent)
					}
				}
			}
		}

		// Map finish reason
		finishReason := "stop"
		if stopReason, ok := anthropicResp["stop_reason"].(string); ok {
			finishReason = mapAnthropicStopReason(stopReason)
		}

		choice := map[string]interface{}{
			"index": 0,
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": text.String(),
			},
			"finish_reason": finishReason,
		}
		choices = append(choices, choice)
	}

	openAIResp["choices"] = choices

	// Add usage information if available
	if usage, ok := anthropicResp["usage"].(map[string]interface{}); ok {
		openAIResp["usage"] = usage
	}

	return openAIResp, nil
}

// ConvertResponseFromOpenAI converts OpenAI response to Anthropic format
func (c *AnthropicConverter) ConvertResponseFromOpenAI(openAIResp map[string]interface{}) (map[string]interface{}, error) {
	anthropicResp := make(map[string]interface{})

	// Copy ID
	if id, ok := openAIResp["id"].(string); ok {
		anthropicResp["id"] = id
	}

	// Set type
	anthropicResp["type"] = "message"
	anthropicResp["role"] = "assistant"

	// Copy model
	if model, ok := openAIResp["model"].(string); ok {
		anthropicResp["model"] = model
	}

	// Convert choices to content format
	content := []interface{}{}

	if choices, ok := openAIResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := firstChoice["message"].(map[string]interface{}); ok {
				if text, ok := message["content"].(string); ok {
					content = append(content, map[string]interface{}{
						"type": "text",
						"text": text,
					})
				}
			}

			// Map finish reason
			if finishReason, ok := firstChoice["finish_reason"].(string); ok {
				anthropicResp["stop_reason"] = mapOpenAIFinishReason(finishReason)
			}
		}
	}

	anthropicResp["content"] = content

	// Copy usage information
	if usage, ok := openAIResp["usage"].(map[string]interface{}); ok {
		anthropicResp["usage"] = usage
	}

	return anthropicResp, nil
}

// Helper functions

func copyIfPresent(src, dst map[string]interface{}, key string) {
	if value, ok := src[key]; ok {
		dst[key] = value
	}
}

func mapAnthropicStopReason(stopReason string) string {
	switch stopReason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	default:
		return stopReason
	}
}

func mapOpenAIFinishReason(finishReason string) string {
	switch finishReason {
	case "stop":
		return "end_turn"
	case "length":
		return "max_tokens"
	default:
		return finishReason
	}
}

func getCurrentTimestamp() int64 {
	// This would return current Unix timestamp
	// For now, return a placeholder
	return 1234567890
}
