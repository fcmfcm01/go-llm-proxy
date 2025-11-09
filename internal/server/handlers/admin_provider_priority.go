package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// AdminProviderPriorityHandler handles provider priority updates
type AdminProviderPriorityHandler struct {
	providerService *services.ProviderService
	logger          *logrus.Logger
	auditLogger     *logging.AuditLogger
}

// NewAdminProviderPriorityHandler creates a new provider priority handler
func NewAdminProviderPriorityHandler(
	providerService *services.ProviderService,
	logger *logrus.Logger,
	auditLogger *logging.AuditLogger,
) *AdminProviderPriorityHandler {
	return &AdminProviderPriorityHandler{
		providerService: providerService,
		logger:          logger,
		auditLogger:     auditLogger,
	}
}

// HandleUpdatePriority handles POST /admin/api/v1/providers/:id/priority
func (h *AdminProviderPriorityHandler) HandleUpdatePriority(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provider ID is required",
		})
		return
	}

	// Parse request body
	var request struct {
		Priority int `json:"priority" binding:"required,min=1,max=100"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Update provider priority
	updates := map[string]interface{}{
		"priority": request.Priority,
	}

	updatedProvider, err := h.providerService.UpdateProvider(providerID, updates)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update provider priority")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderUpdate, providerID, c.ClientIP(), "failure", map[string]interface{}{
				"error":     err.Error(),
				"field":     "priority",
				"new_value": request.Priority,
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update provider priority: " + err.Error(),
		})
		return
	}

	// Log successful update
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderUpdate, providerID, c.ClientIP(), "success", map[string]interface{}{
			"field":     "priority",
			"new_value": request.Priority,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "provider priority updated successfully",
		"provider": updatedProvider,
	})
}
