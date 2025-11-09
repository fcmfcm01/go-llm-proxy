package proxy

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
)

// ModelMapper handles model name mapping between providers
type ModelMapper struct {
	mappings map[string]*models.ModelMapping
	mu       sync.RWMutex
}

// NewModelMapper creates a new model mapper
func NewModelMapper(mappings []*models.ModelMapping) *ModelMapper {
	mapper := &ModelMapper{
		mappings: make(map[string]*models.ModelMapping),
	}

	// Index mappings by source model
	for _, mapping := range mappings {
		mapper.mappings[mapping.SourceModel] = mapping
	}

	return mapper
}

// MapModel maps a source model name to a target model for a provider
func (m *ModelMapper) MapModel(sourceModel string, provider *models.Provider) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Look for exact match
	if mapping, exists := m.mappings[sourceModel]; exists {
		// Check if mapping applies to this provider
		if mapping.SourceProvider == "" || mapping.SourceProvider == provider.ID {
			return mapping.TargetModel, nil
		}
	}

	// Look for prefix match (e.g., gpt-4* -> claude-3-opus)
	for source, mapping := range m.mappings {
		if strings.HasSuffix(source, "*") {
			prefix := strings.TrimSuffix(source, "*")
			if strings.HasPrefix(sourceModel, prefix) {
				if mapping.SourceProvider == "" || mapping.SourceProvider == provider.ID {
					return mapping.TargetModel, nil
				}
			}
		}
	}

	// Check if model needs mapping based on provider type
	providerType := detectProviderType(provider)

	// Auto-map common models
	if autoMapped := m.autoMapModel(sourceModel, providerType); autoMapped != "" {
		return autoMapped, nil
	}

	// No mapping found - pass through original model name
	return sourceModel, nil
}

// autoMapModel provides automatic mapping for common model names
func (m *ModelMapper) autoMapModel(sourceModel, providerType string) string {
	// Normalize model name
	normalized := strings.ToLower(sourceModel)

	// OpenAI to Anthropic mappings
	if providerType == "anthropic" {
		switch {
		case strings.Contains(normalized, "gpt-4-turbo"), strings.Contains(normalized, "gpt-4o"):
			return "claude-3-5-sonnet-20241022"
		case strings.Contains(normalized, "gpt-4"):
			return "claude-3-opus-20240229"
		case strings.Contains(normalized, "gpt-3.5-turbo"):
			return "claude-3-sonnet-20240229"
		}
	}

	// Anthropic to OpenAI mappings
	if providerType == "openai" {
		switch {
		case strings.Contains(normalized, "claude-3-opus"):
			return "gpt-4-turbo"
		case strings.Contains(normalized, "claude-3-sonnet"):
			return "gpt-3.5-turbo"
		case strings.Contains(normalized, "claude-3-haiku"):
			return "gpt-3.5-turbo"
		}
	}

	// No automatic mapping
	return ""
}

// ReverseMapModel maps a target model back to source model
func (m *ModelMapper) ReverseMapModel(targetModel string, provider *models.Provider) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Look for reverse mapping
	for source, mapping := range m.mappings {
		if mapping.TargetModel == targetModel {
			if mapping.SourceProvider == "" || mapping.SourceProvider == provider.ID {
				return source
			}
		}
	}

	// No reverse mapping found - return original
	return targetModel
}

// AddMapping adds a new model mapping
func (m *ModelMapper) AddMapping(mapping *models.ModelMapping) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := mapping.SourceModel
	if mapping.SourceProvider != "" {
		key = fmt.Sprintf("%s:%s", mapping.SourceProvider, mapping.SourceModel)
	}

	m.mappings[key] = mapping
}

// RemoveMapping removes a model mapping
func (m *ModelMapper) RemoveMapping(sourceModel, providerID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := sourceModel
	if providerID != "" {
		key = fmt.Sprintf("%s:%s", providerID, sourceModel)
	}

	delete(m.mappings, key)
}

// GetMapping returns a specific mapping
func (m *ModelMapper) GetMapping(sourceModel, providerID string) (*models.ModelMapping, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := sourceModel
	if providerID != "" {
		key = fmt.Sprintf("%s:%s", providerID, sourceModel)
	}

	mapping, exists := m.mappings[key]
	return mapping, exists
}

// GetAllMappings returns all model mappings
func (m *ModelMapper) GetAllMappings() []*models.ModelMapping {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*models.ModelMapping, 0, len(m.mappings))
	for _, mapping := range m.mappings {
		result = append(result, mapping)
	}

	return result
}

// UpdateMappings replaces all mappings
func (m *ModelMapper) UpdateMappings(mappings []*models.ModelMapping) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.mappings = make(map[string]*models.ModelMapping)
	for _, mapping := range mappings {
		key := mapping.SourceModel
		if mapping.SourceProvider != "" {
			key = fmt.Sprintf("%s:%s", mapping.SourceProvider, mapping.SourceModel)
		}
		m.mappings[key] = mapping
	}
}

// MapRequestModel maps the model in a request
func (m *ModelMapper) MapRequestModel(req map[string]interface{}, provider *models.Provider) error {
	// Get the model from request
	sourceModel, ok := req["model"].(string)
	if !ok || sourceModel == "" {
		return fmt.Errorf("missing or invalid model in request")
	}

	// Map the model
	targetModel, err := m.MapModel(sourceModel, provider)
	if err != nil {
		return fmt.Errorf("failed to map model: %w", err)
	}

	// Update request with mapped model
	req["model"] = targetModel

	return nil
}

// MapResponseModel maps the model in a response back to source
func (m *ModelMapper) MapResponseModel(resp map[string]interface{}, provider *models.Provider) {
	// Get the model from response
	targetModel, ok := resp["model"].(string)
	if !ok || targetModel == "" {
		return
	}

	// Reverse map the model
	sourceModel := m.ReverseMapModel(targetModel, provider)

	// Update response with original model name
	resp["model"] = sourceModel
}

// ValidateMapping validates a model mapping
func ValidateMapping(mapping *models.ModelMapping) error {
	if mapping.SourceModel == "" {
		return fmt.Errorf("source model cannot be empty")
	}
	if mapping.TargetModel == "" {
		return fmt.Errorf("target model cannot be empty")
	}
	return nil
}

// GetSupportedModels returns a list of supported models for a provider
func (m *ModelMapper) GetSupportedModels(provider *models.Provider) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	models := make(map[string]bool)

	// Add all mapped models for this provider
	for source, mapping := range m.mappings {
		if mapping.SourceProvider == "" || mapping.SourceProvider == provider.ID {
			models[source] = true
		}
	}

	// Add common models based on provider type
	providerType := detectProviderType(provider)
	if providerType == "openai" {
		models["gpt-4-turbo"] = true
		models["gpt-4"] = true
		models["gpt-3.5-turbo"] = true
	} else if providerType == "anthropic" {
		models["claude-3-5-sonnet-20241022"] = true
		models["claude-3-opus-20240229"] = true
		models["claude-3-sonnet-20240229"] = true
		models["claude-3-haiku-20240307"] = true
	}

	// Convert to slice
	result := make([]string, 0, len(models))
	for model := range models {
		result = append(result, model)
	}

	return result
}
