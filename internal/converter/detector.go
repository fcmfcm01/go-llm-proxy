package converter

import "fmt"

// Format represents the API format type
type Format int

const (
	Unknown Format = iota
	Anthropic
	OpenAI
)

func (f Format) String() string {
	return [...]string{"Unknown", "Anthropic", "OpenAI"}[f]
}

// FormatDetector detects the API format based on request structure
type FormatDetector struct{}

// DetectFormat detects the format of a request
func (fd *FormatDetector) DetectFormat(data map[string]interface{}) (Format, float64) {
	if data == nil {
		return Unknown, 0.0
	}

	anthropicScore := fd.detectAnthropicFeatures(data)
	openAIScore := fd.detectOpenAIFeatures(data)

	totalFeatures := float64(anthropicScore + openAIScore)
	if totalFeatures == 0 {
		return Unknown, 0.0
	}

	if anthropicScore > openAIScore {
		return Anthropic, float64(anthropicScore) / totalFeatures
	} else if openAIScore > anthropicScore {
		return OpenAI, float64(openAIScore) / totalFeatures
	}

	return Unknown, 0.0
}

func (fd *FormatDetector) detectAnthropicFeatures(data map[string]interface{}) int {
	score := 0

	// Check for Anthropic-specific fields
	if _, ok := data["system"]; ok {
		score += 2
	}

	if _, ok := data["anthropic_version"]; ok {
		score += 3
	}

	if messages, ok := data["messages"].([]interface{}); ok {
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok {
					if role == "user" || role == "assistant" {
						score += 1
					}
				}
			}
		}
	}

	return score
}

func (fd *FormatDetector) detectOpenAIFeatures(data map[string]interface{}) int {
	score := 0

	// Check for OpenAI-specific fields
	if model, ok := data["model"].(string); ok && model != "" {
		score += 2
	}

	if messages, ok := data["messages"].([]interface{}); ok {
		hasValidRoles := false
		for _, msg := range messages {
			if msgMap, ok := msg.(map[string]interface{}); ok {
				if role, ok := msgMap["role"].(string); ok {
					if role == "system" || role == "user" || role == "assistant" {
						score += 1
						hasValidRoles = true
					}
				}
			}
		}
		if hasValidRoles {
			score += 1
		}
	}

	// Check for common OpenAI parameters
	if _, ok := data["max_tokens"]; ok {
		score += 1
	}

	if _, ok := data["temperature"]; ok {
		score += 1
	}

	if _, ok := data["top_p"]; ok {
		score += 1
	}

	return score
}

// ValidateFormat checks if the data matches the expected format
func (fd *FormatDetector) ValidateFormat(data map[string]interface{}, expected Format) error {
	detected, confidence := fd.DetectFormat(data)

	if detected != expected && confidence > 0.5 {
		return fmt.Errorf("format mismatch: detected %s (confidence: %.2f), expected %s",
			detected.String(), confidence, expected.String())
	}

	return nil
}
