package services

import (
	"fmt"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/internal/repository"
	"github.com/sirupsen/logrus"
)

// ProviderService provides business logic for provider management
type ProviderService struct {
	repo         *repository.ProviderRepository
	mappingRepo  *repository.ModelMappingRepository
	logger       *logrus.Logger
	reloadConfig func() error
}

// NewProviderService creates a new provider service
func NewProviderService(
	repo *repository.ProviderRepository,
	mappingRepo *repository.ModelMappingRepository,
	logger *logrus.Logger,
	reloadConfig func() error,
) *ProviderService {
	return &ProviderService{
		repo:         repo,
		mappingRepo:  mappingRepo,
		logger:       logger,
		reloadConfig: reloadConfig,
	}
}

// ListProviders returns all providers
func (s *ProviderService) ListProviders() ([]*models.Provider, error) {
	return s.repo.List()
}

// GetProvider returns a provider by ID
func (s *ProviderService) GetProvider(id string) (*models.Provider, error) {
	return s.repo.Get(id)
}

// CreateProvider creates a new provider
func (s *ProviderService) CreateProvider(provider *models.Provider) error {
	// Validate provider
	if err := provider.Validate(); err != nil {
		return fmt.Errorf("invalid provider: %w", err)
	}

	// Create provider
	if err := s.repo.Create(provider); err != nil {
		return fmt.Errorf("failed to create provider: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after provider creation")
		}
	}

	s.logger.WithField("providerID", provider.ID).Info("Provider created successfully")
	return nil
}

// UpdateProvider updates an existing provider
func (s *ProviderService) UpdateProvider(id string, updates map[string]interface{}) (*models.Provider, error) {
	// Apply updates
	updatedProvider, err := s.repo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update provider: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after provider update")
		}
	}

	s.logger.WithField("providerID", id).Info("Provider updated successfully")
	return updatedProvider, nil
}

// DeleteProvider deletes a provider
func (s *ProviderService) DeleteProvider(id string) error {
	// Get provider for logging
	provider, err := s.repo.Get(id)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	// Delete provider
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after provider deletion")
		}
	}

	s.logger.WithField("providerID", provider.ID).WithField("providerName", provider.Name).Info("Provider deleted successfully")
	return nil
}

// ToggleProvider toggles the enabled status of a provider
func (s *ProviderService) ToggleProvider(id string) (*models.Provider, error) {
	// Toggle provider
	provider, err := s.repo.ToggleEnabled(id)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle provider: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after provider toggle")
		}
	}

	s.logger.WithField("providerID", id).WithField("enabled", provider.Enabled).Info("Provider toggled successfully")
	return provider, nil
}

// ListModelMappings returns all model mappings
func (s *ProviderService) ListModelMappings() ([]*models.ModelMapping, error) {
	return s.mappingRepo.List()
}

// GetModelMapping returns a model mapping by ID
func (s *ProviderService) GetModelMapping(id string) (*models.ModelMapping, error) {
	return s.mappingRepo.Get(id)
}

// CreateModelMapping creates a new model mapping
func (s *ProviderService) CreateModelMapping(mapping *models.ModelMapping) error {
	// Create mapping
	if err := s.mappingRepo.Create(mapping); err != nil {
		return fmt.Errorf("failed to create model mapping: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after model mapping creation")
		}
	}

	s.logger.WithField("mappingID", mapping.ID).Info("Model mapping created successfully")
	return nil
}

// UpdateModelMapping updates an existing model mapping
func (s *ProviderService) UpdateModelMapping(id string, updates map[string]interface{}) (*models.ModelMapping, error) {
	// Update mapping
	mapping, err := s.mappingRepo.Update(id, updates)
	if err != nil {
		return nil, fmt.Errorf("failed to update model mapping: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after model mapping update")
		}
	}

	s.logger.WithField("mappingID", id).Info("Model mapping updated successfully")
	return mapping, nil
}

// DeleteModelMapping deletes a model mapping
func (s *ProviderService) DeleteModelMapping(id string) error {
	// Delete mapping
	if err := s.mappingRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete model mapping: %w", err)
	}

	// Reload configuration
	if s.reloadConfig != nil {
		if err := s.reloadConfig(); err != nil {
			s.logger.WithError(err).Error("Failed to reload configuration after model mapping deletion")
		}
	}

	s.logger.WithField("mappingID", id).Info("Model mapping deleted successfully")
	return nil
}

// GetProviderStatus returns the status of all providers
func (s *ProviderService) GetProviderStatus() (map[string]interface{}, error) {
	providers, err := s.repo.List()
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}

	status := map[string]interface{}{
		"total":     len(providers),
		"enabled":   0,
		"disabled":  0,
		"providers": make([]map[string]interface{}, 0, len(providers)),
	}

	for _, provider := range providers {
		providerStatus := map[string]interface{}{
			"id":       provider.ID,
			"name":     provider.Name,
			"enabled":  provider.Enabled,
			"priority": provider.Priority,
			"url":      provider.APIURL,
			"created":  provider.CreatedAt.Format(time.RFC3339),
			"updated":  provider.UpdatedAt.Format(time.RFC3339),
		}

		if provider.Enabled {
			status["enabled"] = status["enabled"].(int) + 1
		} else {
			status["disabled"] = status["disabled"].(int) + 1
		}

		status["providers"] = append(status["providers"].([]map[string]interface{}), providerStatus)
	}

	return status, nil
}
