package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// ConfigImportHandler handles configuration import requests
type ConfigImportHandler struct {
	logger        *logrus.Logger
	backupService *services.BackupService
	dataDir       string
}

// NewConfigImportHandler creates a new config import handler
func NewConfigImportHandler(logger *logrus.Logger, backupService *services.BackupService, dataDir string) *ConfigImportHandler {
	return &ConfigImportHandler{
		logger:        logger,
		backupService: backupService,
		dataDir:       dataDir,
	}
}

// HandleImport handles POST /admin/api/v1/config/import
func (h *ConfigImportHandler) HandleImport(c *gin.Context) {
	// Create a backup before importing
	h.logger.Info("Creating backup before import...")
	backupConfig := services.BackupConfig{
		IncludeProviders:     true,
		IncludeModelMappings: true,
		IncludeSettings:      true,
		Compression:          true,
		Timestamp:            time.Now(),
		Version:              "1.0.0",
	}

	_, err := h.backupService.CreateBackup(backupConfig)
	if err != nil {
		h.logger.Warnf("Failed to create pre-import backup: %v", err)
	}

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_FORM",
				"message": "Failed to parse form data",
			},
		})
		return
	}

	// Get backup file
	fileHeader, err := c.FormFile("backup")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "NO_FILE",
				"message": "No backup file provided",
			},
		})
		return
	}

	// Validate file extension
	if filepath.Ext(fileHeader.Filename) != ".zip" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_FILE",
				"message": "Backup file must be a zip archive",
			},
		})
		return
	}

	// Save uploaded file temporarily
	tempFile, err := os.CreateTemp("", "backup_*.zip")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "TEMP_FILE_ERROR",
				"message": "Failed to create temp file",
			},
		})
		return
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	if err := c.SaveUploadedFile(fileHeader, tempFile.Name()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "SAVE_ERROR",
				"message": "Failed to save uploaded file",
			},
		})
		return
	}

	// Import the backup
	result, err := h.importBackup(tempFile.Name())
	if err != nil {
		h.logger.Errorf("Failed to import backup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "IMPORT_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"message":   "Configuration imported successfully",
		"result":    result,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// ImportResult holds the result of an import operation
type ImportResult struct {
	Success     bool      `json:"success"`
	Imported    []string  `json:"imported"`
	Skipped     []string  `json:"skipped"`
	Timestamp   time.Time `json:"timestamp"`
	Duration    string    `json:"duration"`
	RecordCount int       `json:"record_count"`
}

// importBackup imports configuration from a backup file
func (h *ConfigImportHandler) importBackup(backupPath string) (*ImportResult, error) {
	startTime := time.Now()

	// Open the zip file
	zipFile, err := zip.OpenReader(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer zipFile.Close()

	result := &ImportResult{
		Success:   false,
		Imported:  []string{},
		Skipped:   []string{},
		Timestamp: startTime,
	}

	// Extract and import each file
	for _, file := range zipFile.File {
		if err := h.importFile(file); err != nil {
			h.logger.Warnf("Failed to import %s: %v", file.Name, err)
			result.Skipped = append(result.Skipped, file.Name)
		} else {
			result.Imported = append(result.Imported, file.Name)
		}
	}

	result.Duration = time.Since(startTime).String()
	result.RecordCount = len(result.Imported)
	result.Success = len(result.Skipped) == 0

	if result.Success {
		h.logger.Infof("Import completed successfully: %d files imported in %s",
			result.RecordCount, result.Duration)
	} else {
		h.logger.Warnf("Import completed with errors: %d imported, %d skipped",
			len(result.Imported), len(result.Skipped))
	}

	return result, nil
}

// importFile imports a single file from the backup
func (h *ConfigImportHandler) importFile(file *zip.File) error {
	// Skip metadata and non-JSON files
	if file.Name == "metadata.json" {
		return nil
	}

	if filepath.Ext(file.Name) != ".json" {
		return fmt.Errorf("unsupported file type: %s", file.Name)
	}

	// Open the file in the archive
	archiveFile, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file in archive: %w", err)
	}
	defer archiveFile.Close()

	// Read the content
	content, err := io.ReadAll(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Validate JSON
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Determine destination path
	destPath := filepath.Join(h.dataDir, filepath.Base(file.Name))
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write to destination
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	h.logger.Infof("Imported: %s", destPath)
	return nil
}
