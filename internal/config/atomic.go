package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// AtomicWriter provides atomic write operations for configuration files
type AtomicWriter struct {
	mu     sync.Mutex
	config *Config
	path   string
	tmpDir string
}

// NewAtomicWriter creates a new atomic writer
func NewAtomicWriter(config *Config, configPath string) *AtomicWriter {
	return &AtomicWriter{
		config: config,
		path:   configPath,
		tmpDir: filepath.Dir(configPath) + "/.tmp",
	}
}

// Write writes the configuration atomically
func (w *AtomicWriter) Write() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create temp directory if it doesn't exist
	if err := os.MkdirAll(w.tmpDir, 0700); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Generate temporary file path
	tmpFile := filepath.Join(w.tmpDir, fmt.Sprintf("config_%d.tmp", time.Now().UnixNano()))

	// Marshal configuration
	configPath := w.path
	viper.SetConfigFile(configPath)

	// Write to temporary file
	if err := viper.WriteConfigAs(tmpFile); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Get file info
	tmpFileInfo, err := os.Stat(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to stat temp file: %w", err)
	}

	// Ensure target directory exists
	targetDir := filepath.Dir(configPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// If target file exists, check if it needs update
	if _, err := os.Stat(configPath); err == nil {
		// Get target file info
		targetFileInfo, err := os.Stat(configPath)
		if err == nil {
			// Skip write if files are identical
			if tmpFileInfo.Size() == targetFileInfo.Size() {
				os.Remove(tmpFile)
				return nil
			}
		}
	}

	// Perform atomic rename
	if err := os.Rename(tmpFile, configPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(configPath, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// Backup creates a backup of the current configuration
func (w *AtomicWriter) Backup() (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, err := os.Stat(w.path); os.IsNotExist(err) {
		return "", nil
	}

	// Generate backup path
	backupPath := w.path + ".bak." + time.Now().Format("20060102_150405")

	// Read current configuration
	input, err := os.Open(w.path)
	if err != nil {
		return "", fmt.Errorf("failed to open config file: %w", err)
	}
	defer input.Close()

	// Write backup
	output, err := os.Create(backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return "", fmt.Errorf("failed to copy config to backup: %w", err)
	}

	return backupPath, nil
}

// Restore restores configuration from a backup
func (w *AtomicWriter) Restore(backupPath string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file does not exist: %s", backupPath)
	}

	// Read backup
	input, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer input.Close()

	// Ensure target directory exists
	targetDir := filepath.Dir(w.path)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write to temporary file first
	tmpFile := w.path + ".tmp." + time.Now().Format("20060102_150405")
	output, err := os.Create(tmpFile)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		return fmt.Errorf("failed to copy backup to temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, w.path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(w.path, 0600); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// CleanupTempFiles removes temporary files
func (w *AtomicWriter) CleanupTempFiles() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// List all files in temp directory
	files, err := os.ReadDir(w.tmpDir)
	if err != nil {
		// Directory might not exist
		return nil
	}

	// Remove old temp files (older than 1 hour)
	cutoff := time.Now().Add(-time.Hour)
	for _, file := range files {
		if !file.IsDir() && file.Name() != "." {
			filePath := filepath.Join(w.tmpDir, file.Name())
			fileInfo, err := file.Info()
			if err != nil {
				continue
			}
			if fileInfo.ModTime().Before(cutoff) {
				os.Remove(filePath)
			}
		}
	}

	return nil
}
