package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// AdminModelMappingsHandler handles model mapping operations
type AdminModelMappingsHandler struct {
	providerService *services.ProviderService
	auditLogger     *logging.AuditLogger
	logger          *logrus.Logger
}

// NewAdminModelMappingsHandler creates a new model mappings handler
func NewAdminModelMappingsHandler(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *AdminModelMappingsHandler {
	return &AdminModelMappingsHandler{
		providerService: providerService,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// List handles GET /admin/api/v1/model-mappings
func (h *AdminModelMappingsHandler) List(c *gin.Context) {
	// Get all model mappings
	mappings, err := h.providerService.ListModelMappings()
	if err != nil {
		h.logger.WithError(err).Error("Failed to list model mappings")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list model mappings",
		})
		return
	}

	// Log the action
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingList, "", c.ClientIP(), "success", map[string]interface{}{
			"count": len(mappings),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"mappings": mappings,
		"count":    len(mappings),
	})
}

// Create handles POST /admin/api/v1/model-mappings
func (h *AdminModelMappingsHandler) Create(c *gin.Context) {
	var mapping models.ModelMapping

	// Bind JSON request
	if err := c.ShouldBindJSON(&mapping); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Create mapping
	if err := h.providerService.CreateModelMapping(&mapping); err != nil {
		h.logger.WithError(err).Error("Failed to create model mapping")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingCreate, mapping.ID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create model mapping: " + err.Error(),
		})
		return
	}

	// Log successful creation
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingCreate, mapping.ID, c.ClientIP(), "success", map[string]interface{}{
			"source": mapping.SourceProvider + "/" + mapping.SourceModel,
			"target": mapping.TargetProvider + "/" + mapping.TargetModel,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "model mapping created successfully",
		"mapping": mapping,
	})
}

// Update handles PUT /admin/api/v1/model-mappings/:id
func (h *AdminModelMappingsHandler) Update(c *gin.Context) {
	mappingID := c.Param("id")
	if mappingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "mapping ID is required",
		})
		return
	}

	var updates map[string]interface{}

	// Bind JSON request
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Update mapping
	updatedMapping, err := h.providerService.UpdateModelMapping(mappingID, updates)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update model mapping")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingUpdate, mappingID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update model mapping: " + err.Error(),
		})
		return
	}

	// Log successful update
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingUpdate, mappingID, c.ClientIP(), "success", map[string]interface{}{
			"updates": updates,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "model mapping updated successfully",
		"mapping": updatedMapping,
	})
}

// Delete handles DELETE /admin/api/v1/model-mappings/:id
func (h *AdminModelMappingsHandler) Delete(c *gin.Context) {
	mappingID := c.Param("id")
	if mappingID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "mapping ID is required",
		})
		return
	}

	// Delete mapping
	if err := h.providerService.DeleteModelMapping(mappingID); err != nil {
		h.logger.WithError(err).Error("Failed to delete model mapping")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingDelete, mappingID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete model mapping: " + err.Error(),
		})
		return
	}

	// Log successful deletion
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventMappingDelete, mappingID, c.ClientIP(), "success", nil)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "model mapping deleted successfully",
	})
}
