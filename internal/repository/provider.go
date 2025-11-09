package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
	"github.com/sirupsen/logrus"
)

// ProviderRepository provides filesystem-based storage for providers
type ProviderRepository struct {
	dataDir   string
	logger    *logrus.Logger
	mu        sync.RWMutex
	providers map[string]*models.Provider
}

// NewProviderRepository creates a new provider repository
func NewProviderRepository(dataDir string, logger *logrus.Logger) (*ProviderRepository, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	repo := &ProviderRepository{
		dataDir:   dataDir,
		logger:    logger,
		providers: make(map[string]*models.Provider),
	}

	// Load existing providers
	if err := repo.load(); err != nil {
		repo.logger.WithError(err).Warning("Failed to load providers, starting with empty set")
	}

	return repo, nil
}

// List returns all providers
func (r *ProviderRepository) List() ([]*models.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]*models.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	// Sort by ID for consistency
	sort.Slice(providers, func(i, j int) bool {
		return providers[i].ID < providers[j].ID
	})

	return providers, nil
}

// Get returns a provider by ID
func (r *ProviderRepository) Get(id string) (*models.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", id)
	}

	return provider, nil
}

// Create creates a new provider
func (r *ProviderRepository) Create(provider *models.Provider) error {
	// Validate
	if provider.ID == "" {
		return fmt.Errorf("provider ID is required")
	}

	if provider.Name == "" {
		return fmt.Errorf("provider name is required")
	}

	if provider.APIURL == "" {
		return fmt.Errorf("provider API URL is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates
	if _, exists := r.providers[provider.ID]; exists {
		return fmt.Errorf("provider already exists: %s", provider.ID)
	}

	// Set timestamps
	now := time.Now()
	provider.CreatedAt = now
	provider.UpdatedAt = now

	// Store
	r.providers[provider.ID] = provider

	// Persist
	if err := r.save(); err != nil {
		// Rollback
		delete(r.providers, provider.ID)
		return fmt.Errorf("failed to save provider: %w", err)
	}

	r.logger.WithField("providerID", provider.ID).Info("Provider created")
	return nil
}

// Update updates an existing provider
func (r *ProviderRepository) Update(id string, updates map[string]interface{}) (*models.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", id)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok && name != "" {
		provider.Name = name
	}

	if apiURL, ok := updates["api_url"].(string); ok && apiURL != "" {
		provider.APIURL = apiURL
	}

	if enabled, ok := updates["enabled"].(bool); ok {
		provider.Enabled = enabled
	}

	if priority, ok := updates["priority"].(int); ok && priority > 0 {
		provider.Priority = priority
	}

	if timeout, ok := updates["timeout"].(time.Duration); ok && timeout > 0 {
		provider.Timeout = timeout
	}

	if maxRetries, ok := updates["max_retries"].(int); ok && maxRetries >= 0 {
		provider.MaxRetries = maxRetries
	}

	// Update timestamp
	provider.UpdatedAt = time.Now()

	// Persist
	if err := r.save(); err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}

	r.logger.WithField("providerID", provider.ID).Info("Provider updated")
	return provider, nil
}

// Delete deletes a provider
func (r *ProviderRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[id]; !exists {
		return fmt.Errorf("provider not found: %s", id)
	}

	// Delete
	delete(r.providers, id)

	// Persist
	if err := r.save(); err != nil {
		return fmt.Errorf("failed to save providers: %w", err)
	}

	r.logger.WithField("providerID", id).Info("Provider deleted")
	return nil
}

// ToggleEnabled toggles the enabled status of a provider
func (r *ProviderRepository) ToggleEnabled(id string) (*models.Provider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	provider, exists := r.providers[id]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", id)
	}

	provider.Enabled = !provider.Enabled
	provider.UpdatedAt = time.Now()

	// Persist
	if err := r.save(); err != nil {
		return nil, fmt.Errorf("failed to save provider: %w", err)
	}

	r.logger.WithField("providerID", provider.ID).WithField("enabled", provider.Enabled).Info("Provider toggled")
	return provider, nil
}

// load loads providers from disk
func (r *ProviderRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	providersPath := filepath.Join(r.dataDir, "providers.json")

	// Check if file exists
	if _, err := os.Stat(providersPath); os.IsNotExist(err) {
		// No providers file yet, create empty
		return r.save()
	}

	// Read file
	data, err := os.ReadFile(providersPath)
	if err != nil {
		return fmt.Errorf("failed to read providers file: %w", err)
	}

	// Parse JSON
	var providers []*models.Provider
	if err := json.Unmarshal(data, &providers); err != nil {
		return fmt.Errorf("failed to parse providers file: %w", err)
	}

	// Build map
	for _, provider := range providers {
		r.providers[provider.ID] = provider
	}

	r.logger.WithField("count", len(providers)).Info("Providers loaded")
	return nil
}

// save saves providers to disk
func (r *ProviderRepository) save() error {
	providersPath := filepath.Join(r.dataDir, "providers.json")

	// Convert to slice
	providers := make([]*models.Provider, 0, len(r.providers))
	for _, provider := range r.providers {
		providers = append(providers, provider)
	}

	// Serialize
	data, err := json.MarshalIndent(providers, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize providers: %w", err)
	}

	// Write file
	if err := os.WriteFile(providersPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write providers file: %w", err)
	}

	return nil
}
