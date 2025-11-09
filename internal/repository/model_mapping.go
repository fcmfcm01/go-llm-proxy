package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
	"github.com/sirupsen/logrus"
)

// ModelMappingRepository provides filesystem-based storage for model mappings
type ModelMappingRepository struct {
	dataDir  string
	logger   *logrus.Logger
	mu       sync.RWMutex
	mappings map[string]*models.ModelMapping
}

// NewModelMappingRepository creates a new model mapping repository
func NewModelMappingRepository(dataDir string, logger *logrus.Logger) (*ModelMappingRepository, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	repo := &ModelMappingRepository{
		dataDir:  dataDir,
		logger:   logger,
		mappings: make(map[string]*models.ModelMapping),
	}

	// Load existing mappings
	if err := repo.load(); err != nil {
		repo.logger.WithError(err).Warning("Failed to load model mappings, starting with empty set")
	}

	return repo, nil
}

// List returns all model mappings
func (r *ModelMappingRepository) List() ([]*models.ModelMapping, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mappings := make([]*models.ModelMapping, 0, len(r.mappings))
	for _, mapping := range r.mappings {
		mappings = append(mappings, mapping)
	}

	// Sort by ID for consistency
	sort.Slice(mappings, func(i, j int) bool {
		return mappings[i].ID < mappings[j].ID
	})

	return mappings, nil
}

// Get returns a model mapping by ID
func (r *ModelMappingRepository) Get(id string) (*models.ModelMapping, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mapping, exists := r.mappings[id]
	if !exists {
		return nil, fmt.Errorf("model mapping not found: %s", id)
	}

	return mapping, nil
}

// Create creates a new model mapping
func (r *ModelMappingRepository) Create(mapping *models.ModelMapping) error {
	// Validate
	if mapping.SourceProvider == "" {
		return fmt.Errorf("source provider is required")
	}

	if mapping.SourceModel == "" {
		return fmt.Errorf("source model is required")
	}

	if mapping.TargetProvider == "" {
		return fmt.Errorf("target provider is required")
	}

	if mapping.TargetModel == "" {
		return fmt.Errorf("target model is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for duplicates (same source)
	for _, m := range r.mappings {
		if m.SourceProvider == mapping.SourceProvider && m.SourceModel == mapping.SourceModel {
			return fmt.Errorf("mapping already exists for source: %s/%s", mapping.SourceProvider, mapping.SourceModel)
		}
	}

	// Generate ID if not set
	if mapping.ID == "" {
		mapping.ID = fmt.Sprintf("%s_%s_%s_%d", mapping.SourceProvider, mapping.SourceModel, mapping.TargetModel, time.Now().Unix())
	}

	// Set timestamps
	now := time.Now()
	mapping.CreatedAt = now
	mapping.UpdatedAt = now

	// Store
	r.mappings[mapping.ID] = mapping

	// Persist
	if err := r.save(); err != nil {
		// Rollback
		delete(r.mappings, mapping.ID)
		return fmt.Errorf("failed to save model mapping: %w", err)
	}

	r.logger.WithField("mappingID", mapping.ID).Info("Model mapping created")
	return nil
}

// Update updates an existing model mapping
func (r *ModelMappingRepository) Update(id string, updates map[string]interface{}) (*models.ModelMapping, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	mapping, exists := r.mappings[id]
	if !exists {
		return nil, fmt.Errorf("model mapping not found: %s", id)
	}

	// Apply updates
	if sourceProvider, ok := updates["source_provider"].(string); ok && sourceProvider != "" {
		mapping.SourceProvider = sourceProvider
	}

	if sourceModel, ok := updates["source_model"].(string); ok && sourceModel != "" {
		mapping.SourceModel = sourceModel
	}

	if targetProvider, ok := updates["target_provider"].(string); ok && targetProvider != "" {
		mapping.TargetProvider = targetProvider
	}

	if targetModel, ok := updates["target_model"].(string); ok && targetModel != "" {
		mapping.TargetModel = targetModel
	}

	// Update timestamp
	mapping.UpdatedAt = time.Now()

	// Persist
	if err := r.save(); err != nil {
		return nil, fmt.Errorf("failed to save model mapping: %w", err)
	}

	r.logger.WithField("mappingID", mapping.ID).Info("Model mapping updated")
	return mapping, nil
}

// Delete deletes a model mapping
func (r *ModelMappingRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.mappings[id]; !exists {
		return fmt.Errorf("model mapping not found: %s", id)
	}

	// Delete
	delete(r.mappings, id)

	// Persist
	if err := r.save(); err != nil {
		return fmt.Errorf("failed to save model mappings: %w", err)
	}

	r.logger.WithField("mappingID", id).Info("Model mapping deleted")
	return nil
}

// load loads model mappings from disk
func (r *ModelMappingRepository) load() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	mappingsPath := filepath.Join(r.dataDir, "model_mappings.json")

	// Check if file exists
	if _, err := os.Stat(mappingsPath); os.IsNotExist(err) {
		// No mappings file yet, create empty
		return r.save()
	}

	// Read file
	data, err := os.ReadFile(mappingsPath)
	if err != nil {
		return fmt.Errorf("failed to read model mappings file: %w", err)
	}

	// Parse JSON
	var mappings []*models.ModelMapping
	if err := json.Unmarshal(data, &mappings); err != nil {
		return fmt.Errorf("failed to parse model mappings file: %w", err)
	}

	// Build map
	for _, mapping := range mappings {
		r.mappings[mapping.ID] = mapping
	}

	r.logger.WithField("count", len(mappings)).Info("Model mappings loaded")
	return nil
}

// save saves model mappings to disk
func (r *ModelMappingRepository) save() error {
	mappingsPath := filepath.Join(r.dataDir, "model_mappings.json")

	// Convert to slice
	mappings := make([]*models.ModelMapping, 0, len(r.mappings))
	for _, mapping := range r.mappings {
		mappings = append(mappings, mapping)
	}

	// Serialize
	data, err := json.MarshalIndent(mappings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize model mappings: %w", err)
	}

	// Write file
	if err := os.WriteFile(mappingsPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write model mappings file: %w", err)
	}

	return nil
}
