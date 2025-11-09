package services

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// BackupService handles configuration backup and restore operations
type BackupService struct {
	logger          *logrus.Logger
	backupDir       string
	dataDir         string
	maxBackups      int
	compressionType string
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	IncludeProviders     bool      `json:"include_providers"`
	IncludeModelMappings bool      `json:"include_model_mappings"`
	IncludeSettings      bool      `json:"include_settings"`
	Compression          bool      `json:"compression"`
	Timestamp            time.Time `json:"timestamp"`
	Version              string    `json:"version"`
}

// BackupResult holds the result of a backup operation
type BackupResult struct {
	Success      bool      `json:"success"`
	BackupPath   string    `json:"backup_path"`
	BackupSize   int64     `json:"backup_size_bytes"`
	Timestamp    time.Time `json:"timestamp"`
	Duration     string    `json:"duration"`
	RecordCount  int       `json:"record_count"`
	Error        string    `json:"error,omitempty"`
}

// NewBackupService creates a new backup service
func NewBackupService(logger *logrus.Logger, backupDir, dataDir string) *BackupService {
	return &BackupService{
		logger:          logger,
		backupDir:       backupDir,
		dataDir:         dataDir,
		maxBackups:      10, // Keep up to 10 backups
		compressionType: "zip",
	}
}

// CreateBackup creates a backup of configuration data
func (b *BackupService) CreateBackup(config BackupConfig) (*BackupResult, error) {
	startTime := time.Now()
	timestamp := time.Now().Format("20060102_150405")
	backupName := fmt.Sprintf("backup_%s", timestamp)

	var backupPath string
	var err error

	if config.Compression {
		backupPath = filepath.Join(b.backupDir, backupName+".zip")
		err = b.createCompressedBackup(backupPath, config)
	} else {
		backupPath = filepath.Join(b.backupDir, backupName)
		err = b.createUncompressedBackup(backupPath, config)
	}

	if err != nil {
		return &BackupResult{
			Success:   false,
			Timestamp: startTime,
			Error:     err.Error(),
		}, err
	}

	// Get file size
	backupInfo, err := os.Stat(backupPath)
	if err != nil {
		return &BackupResult{
			Success:   false,
			Timestamp: startTime,
			Error:     err.Error(),
		}, err
	}

	// Count records
	recordCount, err := b.countRecords(config)
	if err != nil {
		b.logger.Warnf("Failed to count records: %v", err)
		recordCount = 0
	}

	// Clean up old backups
	b.cleanupOldBackups()

	result := &BackupResult{
		Success:      true,
		BackupPath:   backupPath,
		BackupSize:   backupInfo.Size(),
		Timestamp:    startTime,
		Duration:     time.Since(startTime).String(),
		RecordCount:  recordCount,
	}

	b.logger.Infof("Backup created successfully: %s (size: %d bytes, duration: %s)",
		backupPath, backupInfo.Size(), result.Duration)

	return result, nil
}

// createCompressedBackup creates a compressed backup
func (b *BackupService) createCompressedBackup(backupPath string, config BackupConfig) error {
	// Create zip file
	zipFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer zipFile.Close()

	writer := zip.NewWriter(zipFile)
	defer writer.Close()

	// Add metadata file
	if err := b.addMetadata(writer, config); err != nil {
		return err
	}

	// Add providers
	if config.IncludeProviders {
		if err := b.addFileToZip(writer, filepath.Join(b.dataDir, "providers.json"), "providers.json"); err != nil {
			return err
		}
	}

	// Add model mappings
	if config.IncludeModelMappings {
		if err := b.addFileToZip(writer, filepath.Join(b.dataDir, "model_mappings.json"), "model_mappings.json"); err != nil {
			return err
		}
	}

	// Add settings
	if config.IncludeSettings {
		if err := b.addFileToZip(writer, filepath.Join(b.dataDir, "settings.json"), "settings.json"); err != nil {
			return err
		}
	}

	return nil
}

// createUncompressedBackup creates an uncompressed backup
func (b *BackupService) createUncompressedBackup(backupPath string, config BackupConfig) error {
	// Create backup directory
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Write metadata
	metadataPath := filepath.Join(backupPath, "metadata.json")
	if err := b.writeMetadata(metadataPath, config); err != nil {
		return err
	}

	// Copy providers
	if config.IncludeProviders {
		if err := b.copyFile(filepath.Join(b.dataDir, "providers.json"), filepath.Join(backupPath, "providers.json")); err != nil {
			return err
		}
	}

	// Copy model mappings
	if config.IncludeModelMappings {
		if err := b.copyFile(filepath.Join(b.dataDir, "model_mappings.json"), filepath.Join(backupPath, "model_mappings.json")); err != nil {
			return err
		}
	}

	// Copy settings
	if config.IncludeSettings {
		if err := b.copyFile(filepath.Join(b.dataDir, "settings.json"), filepath.Join(backupPath, "settings.json")); err != nil {
			return err
		}
	}

	return nil
}

// addMetadata adds metadata to the backup
func (b *BackupService) addMetadata(writer *zip.Writer, config BackupConfig) error {
	_, err := writer.Create("metadata.json")
	return err
}

// writeMetadata writes metadata to a file
func (b *BackupService) writeMetadata(path string, config BackupConfig) error {
	config.Timestamp = time.Now()
	config.Version = "1.0.0"

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// addFileToZip adds a file to zip archive
func (b *BackupService) addFileToZip(writer *zip.Writer, filePath, zipPath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		// File might not exist, skip
		if os.IsNotExist(err) {
			b.logger.Warnf("File not found, skipping: %s", filePath)
			return nil
		}
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create zip header: %w", err)
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	zipWriter, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := io.Copy(zipWriter, file); err != nil {
		return fmt.Errorf("failed to write to zip: %w", err)
	}

	return nil
}

// copyFile copies a file
func (b *BackupService) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		// File might not exist, skip
		if os.IsNotExist(err) {
			b.logger.Warnf("File not found, skipping: %s", src)
			return nil
		}
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination directory
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// countRecords counts the number of records in the backup
func (b *BackupService) countRecords(config BackupConfig) (int, error) {
	count := 0

	if config.IncludeProviders {
		if data, err := os.ReadFile(filepath.Join(b.dataDir, "providers.json")); err == nil {
			var providers []interface{}
			if json.Unmarshal(data, &providers) == nil {
				count += len(providers)
			}
		}
	}

	if config.IncludeModelMappings {
		if data, err := os.ReadFile(filepath.Join(b.dataDir, "model_mappings.json")); err == nil {
			var mappings []interface{}
			if json.Unmarshal(data, &mappings) == nil {
				count += len(mappings)
			}
		}
	}

	return count, nil
}

// ListBackups lists all available backups
func (b *BackupService) ListBackups() ([]BackupResult, error) {
	backups, err := os.ReadDir(b.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	var results []BackupResult
	for _, entry := range backups {
		if entry.IsDir() || filepath.Ext(entry.Name()) == ".zip" {
			backupPath := filepath.Join(b.backupDir, entry.Name())
			info, err := os.Stat(backupPath)
			if err != nil {
				continue
			}

			results = append(results, BackupResult{
				BackupPath: backupPath,
				BackupSize: info.Size(),
				Timestamp:  info.ModTime(),
			})
		}
	}

	return results, nil
}

// cleanupOldBackups removes old backups beyond the maximum
func (b *BackupService) cleanupOldBackups() {
	backups, err := b.ListBackups()
	if err != nil {
		b.logger.Errorf("Failed to list backups for cleanup: %v", err)
		return
	}

	if len(backups) <= b.maxBackups {
		return
	}

	// Sort by timestamp (oldest first)
	for i := 0; i < len(backups)-b.maxBackups; i++ {
		backup := backups[i]
		if err := os.RemoveAll(backup.BackupPath); err != nil {
			b.logger.Errorf("Failed to remove old backup %s: %v", backup.BackupPath, err)
		} else {
			b.logger.Infof("Removed old backup: %s", backup.BackupPath)
		}
	}
}

// RestoreBackup restores configuration from a backup
func (b *BackupService) RestoreBackup(backupPath string) error {
	b.logger.Infof("Starting restore from: %s", backupPath)

	// TODO: Implement restore logic
	// For now, just log that we would restore
	b.logger.Info("Restore functionality to be implemented")

	return nil
}
