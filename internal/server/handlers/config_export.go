package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// ConfigExportHandler handles configuration export requests
type ConfigExportHandler struct {
	logger        *logrus.Logger
	backupService *services.BackupService
}

// NewConfigExportHandler creates a new config export handler
func NewConfigExportHandler(logger *logrus.Logger, backupService *services.BackupService) *ConfigExportHandler {
	return &ConfigExportHandler{
		logger:        logger,
		backupService: backupService,
	}
}

// HandleExport handles GET /admin/api/v1/config/export
func (h *ConfigExportHandler) HandleExport(c *gin.Context) {
	// Parse query parameters
	includeProviders := c.DefaultQuery("include_providers", "true") == "true"
	includeModelMappings := c.DefaultQuery("include_model_mappings", "true") == "true"
	includeSettings := c.DefaultQuery("include_settings", "true") == "true"
	compression := c.DefaultQuery("compression", "true") == "true"

	// Create backup config
	config := services.BackupConfig{
		IncludeProviders:     includeProviders,
		IncludeModelMappings: includeModelMappings,
		IncludeSettings:      includeSettings,
		Compression:          compression,
		Timestamp:            time.Now(),
		Version:              "1.0.0",
	}

	// Create backup
	result, err := h.backupService.CreateBackup(config)
	if err != nil {
		h.logger.Errorf("Failed to create backup: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "BACKUP_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	// Check if client wants to download the file
	download := c.DefaultQuery("download", "false") == "true"
	if download {
		// Set headers for file download
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Disposition", "attachment; filename=\""+result.BackupPath+"\"")
		c.Header("Content-Length", string(rune(result.BackupSize)))

		// Stream the file to the client
		c.File(result.BackupPath)
	} else {
		// Return backup information
		c.JSON(http.StatusOK, gin.H{
			"status":    "success",
			"message":   "Configuration exported successfully",
			"backup":    result,
			"timestamp": time.Now().Format(time.RFC3339),
		})
	}
}

// HandleListBackups handles GET /admin/api/v1/config/backups
func (h *ConfigExportHandler) HandleListBackups(c *gin.Context) {
	backups, err := h.backupService.ListBackups()
	if err != nil {
		h.logger.Errorf("Failed to list backups: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "LIST_BACKUPS_FAILED",
				"message": err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "success",
		"backups":   backups,
		"count":     len(backups),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
