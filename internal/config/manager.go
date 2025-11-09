package config

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ConfigManager manages application configuration
type ConfigManager struct {
	config   *Config
	filePath string
	mutex    sync.RWMutex
	logger   *logrus.Logger
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(filePath string, logger *logrus.Logger) *ConfigManager {
	return &ConfigManager{
		filePath: filePath,
		logger:   logger,
		config:   &Config{},
	}
}

// Load loads configuration from file
func (cm *ConfigManager) Load() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cm.logger.Info("Config file does not exist, using defaults")
			return nil
		}
		return err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	cm.config = &config
	cm.logger.Info("Configuration loaded successfully")
	return nil
}

// Save saves configuration to file
func (cm *ConfigManager) Save() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(cm.filePath, data, 0644); err != nil {
		return err
	}

	cm.logger.Info("Configuration saved successfully")
	return nil
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() *Config {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.config
}

// GetProviders returns all providers
func (cm *ConfigManager) GetProviders() []Provider {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.config.Providers
}

// AddProvider adds a new provider
func (cm *ConfigManager) AddProvider(p Provider) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now().Unix()
	p.CreatedAt = now
	p.UpdatedAt = now

	cm.config.Providers = append(cm.config.Providers, p)
	return cm.Save()
}

// UpdateProvider updates an existing provider
func (cm *ConfigManager) UpdateProvider(id string, p Provider) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for i, provider := range cm.config.Providers {
		if provider.ID == id {
			p.CreatedAt = provider.CreatedAt
			p.UpdatedAt = time.Now().Unix()
			cm.config.Providers[i] = p
			return cm.Save()
		}
	}

	return nil
}

// DeleteProvider deletes a provider
func (cm *ConfigManager) DeleteProvider(id string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for i, provider := range cm.config.Providers {
		if provider.ID == id {
			cm.config.Providers = append(cm.config.Providers[:i], cm.config.Providers[i+1:]...)
			return cm.Save()
		}
	}

	return nil
}

// ToggleProvider toggles provider enabled status
func (cm *ConfigManager) ToggleProvider(id string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	for i, provider := range cm.config.Providers {
		if provider.ID == id {
			provider.Enabled = !provider.Enabled
			provider.UpdatedAt = time.Now().Unix()
			cm.config.Providers[i] = provider
			return cm.Save()
		}
	}

	return nil
}
