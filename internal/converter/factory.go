package converter

import (
	"fmt"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
)

// Converter defines the interface for format conversion
type Converter interface {
	ConvertRequest(providerType string, req map[string]interface{}) (map[string]interface{}, error)
	ConvertResponse(providerType string, resp map[string]interface{}) (map[string]interface{}, error)
}

// Factory creates converters for different provider types
type Factory struct {
	converters map[string]interface{}
}

// NewFactory creates a new converter factory
func NewFactory() *Factory {
	return &Factory{
		converters: map[string]interface{}{
			"anthropic": NewAnthropicConverter(),
			"openai":    NewOpenAIConverter(),
		},
	}
}

// GetConverter returns the appropriate converter for a provider
func (f *Factory) GetConverter(provider *models.Provider) (interface{}, error) {
	// Determine provider type from URL or configuration
	providerType := detectProviderType(provider)

	converter, exists := f.converters[providerType]
	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	return converter, nil
}

// ConvertRequest converts a request for a specific provider
func (f *Factory) ConvertRequest(provider *models.Provider, req map[string]interface{}) (map[string]interface{}, error) {
	providerType := detectProviderType(provider)

	switch providerType {
	case "anthropic":
		converter := NewAnthropicConverter()
		return converter.ConvertFromOpenAI(req)
	case "openai":
		converter := NewOpenAIConverter()
		return converter.ConvertRequest(req)
	default:
		// Pass through for unknown providers
		return req, nil
	}
}

// ConvertResponse converts a response from a specific provider
func (f *Factory) ConvertResponse(provider *models.Provider, resp map[string]interface{}) (map[string]interface{}, error) {
	providerType := detectProviderType(provider)

	switch providerType {
	case "anthropic":
		converter := NewAnthropicConverter()
		return converter.ConvertResponseToOpenAI(resp)
	case "openai":
		converter := NewOpenAIConverter()
		return converter.ConvertResponse(resp)
	default:
		// Pass through for unknown providers
		return resp, nil
	}
}

// detectProviderType determines the provider type from configuration
func detectProviderType(provider *models.Provider) string {
	if provider == nil {
		return "openai"
	}

	// Check URL patterns
	apiURL := provider.APIURL
	if contains(apiURL, "anthropic") || contains(apiURL, "claude") {
		return "anthropic"
	}
	if contains(apiURL, "openai") {
		return "openai"
	}

	// Check model name patterns
	// This would be enhanced with actual provider metadata

	// Default to OpenAI format (most common)
	return "openai"
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
