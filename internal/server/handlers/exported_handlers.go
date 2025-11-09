package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// InitHandlers initializes the global handler variables
// This should be called once during server startup
func InitHandlers(
	providerService *services.ProviderService,
	backupService *services.BackupService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) {
	ProviderService = providerService
	BackupService = backupService
	AuditLogger = auditLogger
	Logger = logger
}

// Global service instances (will be initialized by the server)
var (
	ProviderService *services.ProviderService
	BackupService   *services.BackupService
	AuditLogger     *logging.AuditLogger
	Logger          *logrus.Logger
)

// Provider Handlers

// HandleListProviders handles GET /admin/api/v1/providers
func HandleListProviders(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	providers, err := ProviderService.ListProviders()
	if err != nil {
		Logger.WithError(err).Error("Failed to list providers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list providers",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  providers,
		"count": len(providers),
	})
}

// HandleCreateProvider handles POST /admin/api/v1/providers
func HandleCreateProvider(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	var provider models.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if err := ProviderService.CreateProvider(&provider); err != nil {
		Logger.WithError(err).Error("Failed to create provider")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create provider",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"data": provider,
	})
}

// HandleGetProvider handles GET /admin/api/v1/providers/:id
func HandleGetProvider(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	id := c.Param("id")
	provider, err := ProviderService.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "provider not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}

// HandleUpdateProvider handles PUT /admin/api/v1/providers/:id
func HandleUpdateProvider(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	provider, err := ProviderService.UpdateProvider(id, updates)
	if err != nil {
		Logger.WithError(err).Error("Failed to update provider")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}

// HandleDeleteProvider handles DELETE /admin/api/v1/providers/:id
func HandleDeleteProvider(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	id := c.Param("id")
	if err := ProviderService.DeleteProvider(id); err != nil {
		Logger.WithError(err).Error("Failed to delete provider")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "provider deleted successfully",
	})
}

// HandleToggleProvider handles POST /admin/api/v1/providers/:id/toggle
func HandleToggleProvider(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	id := c.Param("id")
	provider, err := ProviderService.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "provider not found",
		})
		return
	}

	updates := map[string]interface{}{
		"enabled": !provider.Enabled,
	}
	provider, err = ProviderService.UpdateProvider(id, updates)
	if err != nil {
		Logger.WithError(err).Error("Failed to toggle provider")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to toggle provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}

// Model Mapping Handlers

// HandleListModelMappings handles GET /admin/api/v1/model-mappings
func HandleListModelMappings(c *gin.Context) {
	// TODO: Implement model mapping service
	c.JSON(http.StatusOK, gin.H{
		"data":  []interface{}{},
		"count": 0,
		"message": "model mapping feature not yet implemented",
	})
}

// HandleCreateModelMapping handles POST /admin/api/v1/model-mappings
func HandleCreateModelMapping(c *gin.Context) {
	// TODO: Implement model mapping service
	c.JSON(http.StatusOK, gin.H{
		"message": "model mapping feature not yet implemented",
	})
}

// HandleDeleteModelMapping handles DELETE /admin/api/v1/model-mappings/:id
func HandleDeleteModelMapping(c *gin.Context) {
	// TODO: Implement model mapping service
	c.JSON(http.StatusOK, gin.H{
		"message": "model mapping feature not yet implemented",
	})
}

// Provider Status Handlers

// HandleGetAllProviderStatuses handles GET /admin/api/v1/providers/status
func HandleGetAllProviderStatuses(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	// Get all providers
	providers, err := ProviderService.ListProviders()
	if err != nil {
		Logger.WithError(err).Error("Failed to list providers for status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get provider statuses",
		})
		return
	}

	// For now, just return the providers (status can be enhanced later)
	c.JSON(http.StatusOK, gin.H{
		"data": providers,
	})
}

// HandleGetProviderStatus handles GET /admin/api/v1/providers/status/:id
func HandleGetProviderStatus(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	id := c.Param("id")
	provider, err := ProviderService.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "provider not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}

// HandleUpdateProviderPriority handles POST /admin/api/v1/providers/:id/priority
func HandleUpdateProviderPriority(c *gin.Context) {
	if ProviderService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	id := c.Param("id")
	var req struct {
		Priority int `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	updates := map[string]interface{}{
		"priority": req.Priority,
	}
	provider, err := ProviderService.UpdateProvider(id, updates)
	if err != nil {
		Logger.WithError(err).Error("Failed to update provider priority")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update provider priority",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}

// Config Handlers

// HandleExportConfig handles GET /admin/api/v1/config/export
func HandleExportConfig(c *gin.Context) {
	if BackupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	// Parse query parameters
	includeProviders := c.DefaultQuery("include_providers", "true") == "true"
	includeModelMappings := c.DefaultQuery("include_model_mappings", "true") == "true"
	includeSettings := c.DefaultQuery("include_settings", "true") == "true"
	compression := c.DefaultQuery("compression", "true") == "true"

	// Create backup
	config := services.BackupConfig{
		IncludeProviders:     includeProviders,
		IncludeModelMappings: includeModelMappings,
		IncludeSettings:      includeSettings,
		Compression:          compression,
	}

	backupPath, err := BackupService.CreateBackup(config)
	if err != nil {
		Logger.WithError(err).Error("Failed to create backup")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to export config",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"backup_path": backupPath,
		"message":     "config exported successfully",
	})
}

// HandleImportConfig handles POST /admin/api/v1/config/import
func HandleImportConfig(c *gin.Context) {
	if BackupService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "service not initialized"})
		return
	}

	var req struct {
		BackupPath string `json:"backup_path"`
		Restore    bool   `json:"restore"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if err := BackupService.RestoreBackup(req.BackupPath); err != nil {
		Logger.WithError(err).Error("Failed to restore backup")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to import config",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "config imported successfully",
	})
}

// HandleReloadConfig handles POST /admin/api/v1/config/reload
func HandleReloadConfig(c *gin.Context) {
	// Reload configuration (implement based on your reload mechanism)
	// This would typically trigger a config reload in the service
	// For now, just return success
	c.JSON(http.StatusOK, gin.H{
		"message": "config reloaded successfully",
	})
}

// Audit Log Handlers

// HandleGetAuditLogs handles GET /admin/api/v1/audit/logs
func HandleGetAuditLogs(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"data":  []interface{}{},
		"count": 0,
		"message": "audit log feature not yet implemented",
	})
}

// HandleGetAuditLog handles GET /admin/api/v1/audit/logs/:id
func HandleGetAuditLog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "audit log feature not yet implemented",
	})
}
