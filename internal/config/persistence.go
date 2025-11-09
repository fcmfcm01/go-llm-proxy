package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

// Persistence handles filesystem-based configuration persistence
type Persistence struct {
	configPath string
	mu         sync.RWMutex
}

// NewPersistence creates a new Persistence instance
func NewPersistence(configPath string) *Persistence {
	return &Persistence{
		configPath: configPath,
	}
}

// Save saves configuration to filesystem atomically
func (p *Persistence) Save(config interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to temporary file
	tmpPath := p.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, p.configPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename config: %w", err)
	}

	return nil
}

// Load loads configuration from filesystem
func (p *Persistence) Load(config interface{}) error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// Backup creates a backup of the current configuration
func (p *Persistence) Backup() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	backupPath := fmt.Sprintf("%s.backup.%d", p.configPath, os.Getpid())

	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return fmt.Errorf("failed to read config for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// Restore restores configuration from a backup
func (p *Persistence) Restore(backupPath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !filepath.IsAbs(backupPath) {
		backupPath = filepath.Join(filepath.Dir(p.configPath), backupPath)
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if err := os.WriteFile(p.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to restore config: %w", err)
	}

	return nil
}
