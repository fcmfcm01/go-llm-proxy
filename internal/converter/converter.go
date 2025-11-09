package converter

import (
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"
)

// Converter handles format conversion between Anthropic and OpenAI APIs
type Converter struct {
	modelMapping map[string]string
	logger       *logrus.Logger
}

// NewConverter creates a new converter instance
func NewConverter(modelMapping map[string]string, logger *logrus.Logger) *Converter {
	return &Converter{
		modelMapping: modelMapping,
		logger:       logger,
	}
}

// ConvertRequest converts a request from one format to another
func (c *Converter) ConvertRequest(data map[string]interface{}, source, target Format) map[string]interface{} {
	if source == target || source == Unknown || target == Unknown {
		return data
	}

	switch source {
	case Anthropic:
		if target == OpenAI {
			return c.anthropicToOpenAI(data)
		}
	case OpenAI:
		if target == Anthropic {
			return c.openaiToAnthropic(data)
		}
	}

	return data
}

// ConvertResponse converts a response from one format to another
func (c *Converter) ConvertResponse(data map[string]interface{}, source, target Format) map[string]interface{} {
	if source == target || source == Unknown || target == Unknown {
		return data
	}

	switch source {
	case OpenAI:
		if target == Anthropic {
			return c.openaiToAnthropicResponse(data)
		}
	case Anthropic:
		if target == OpenAI {
			return c.anthropicToOpenAIResponse(data)
		}
	}

	return data
}

// anthropicToOpenAI converts Anthropic request to OpenAI format
func (c *Converter) anthropicToOpenAI(req map[string]interface{}) map[string]interface{} {
	openAIReq := make(map[string]interface{})

	// Map model
	if model, ok := req["model"].(string); ok {
		openAIReq["model"] = c.mapModel(model)
	}

	// Convert messages
	if anthropicMsgs, ok := req["messages"].([]interface{}); ok {
		messages := make([]map[string]interface{}, 0, len(anthropicMsgs))

		// Extract system message if present
		var systemMessage string
		if system, ok := req["system"].(string); ok {
			systemMessage = system
		}

		// Convert messages
		for _, msg := range anthropicMsgs {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				newMsg := make(map[string]interface{})
				if role, ok := msgMap["role"].(string); ok {
					newMsg["role"] = role
				}
				if content, ok := msgMap["content"].(string); ok {
					newMsg["content"] = content
				}
				messages = append(messages, newMsg)
			}
		}

		// Add system message if it existed
		if systemMessage != "" && len(messages) > 0 {
			messages = prependSystemMessage(messages, systemMessage)
		}

		openAIReq["messages"] = messages
	}

	// Map common parameters
	c.mapCommonParameters(req, openAIReq)

	return openAIReq
}

// openaiToAnthropic converts OpenAI request to Anthropic format
func (c *Converter) openaiToAnthropic(req map[string]interface{}) map[string]interface{} {
	anthropicReq := make(map[string]interface{})

	// Map model
	if model, ok := req["model"].(string); ok {
		anthropicReq["model"] = c.mapModel(model)
	}

	// Convert messages
	if openAIMsgs, ok := req["messages"].([]interface{}); ok {
		messages := make([]map[string]interface{}, 0, len(openAIMsgs))
		var systemMessage string

		// Extract and convert system message
		for _, msg := range openAIMsgs {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok {
					if role == "system" {
						if content, ok := msgMap["content"].(string); ok {
							systemMessage = content
						}
						continue
					}

					newMsg := make(map[string]interface{})
					newMsg["role"] = role
					if content, ok := msgMap["content"].(string); ok {
						newMsg["content"] = content
					}
					messages = append(messages, newMsg)
				}
			}
		}

		anthropicReq["messages"] = messages
		if systemMessage != "" {
			anthropicReq["system"] = systemMessage
		}
	}

	// Add Anthropic version
	anthropicReq["anthropic_version"] = "v3-20240307"

	// Map common parameters
	c.mapCommonParameters(req, anthropicReq)

	return anthropicReq
}

// mapModel maps a model name using the configured mapping
func (c *Converter) mapModel(model string) string {
	if mapped, ok := c.modelMapping[model]; ok {
		return mapped
	}
	return model
}

// mapCommonParameters maps parameters common to both formats
func (c *Converter) mapCommonParameters(source, target map[string]interface{}) {
	commonParams := []string{"max_tokens", "temperature", "top_p", "stream", "stop"}

	for _, param := range commonParams {
		if val, ok := source[param]; ok && val != nil {
			target[param] = val
		}
	}
}

// prependSystemMessage adds system message to the beginning of messages
func prependSystemMessage(messages []map[string]interface{}, system string) []map[string]interface{} {
	systemMsg := map[string]interface{}{
		"role":    "system",
		"content": system,
	}
	result := make([]map[string]interface{}, 0, len(messages)+1)
	result = append(result, systemMsg)
	result = append(result, messages...)
	return result
}

// openaiToAnthropicResponse converts OpenAI response to Anthropic format
func (c *Converter) openaiToAnthropicResponse(resp map[string]interface{}) map[string]interface{} {
	anthropicResp := make(map[string]interface{})

	if choices, ok := resp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				content := make([]map[string]interface{}, 1)
				content[0] = map[string]interface{}{
					"type": "text",
				}
				if text, ok := message["content"].(string); ok {
					content[0]["text"] = text
				}
				anthropicResp["content"] = content
			}

			if finishReason, ok := choice["finish_reason"].(string); ok {
				anthropicResp["stop_reason"] = finishReason
			}
		}
	}

	return anthropicResp
}

// anthropicToOpenAIResponse converts Anthropic response to OpenAI format
func (c *Converter) anthropicToOpenAIResponse(resp map[string]interface{}) map[string]interface{} {
	openAIResp := make(map[string]interface{})
	openAIResp["choices"] = []interface{}{}

	if content, ok := resp["content"].([]interface{}); ok && len(content) > 0 {
		var text string
		if firstContent, ok := content[0].(map[string]interface{}); ok {
			if t, ok := firstContent["text"].(string); ok {
				text = t
			}
		}

		choice := map[string]interface{}{
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": text,
			},
			"finish_reason": "stop",
		}

		if stopReason, ok := resp["stop_reason"].(string); ok {
			choice["finish_reason"] = stopReason
		}

		openAIResp["choices"] = []interface{}{choice}
	}

	return openAIResp
}
