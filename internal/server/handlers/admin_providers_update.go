package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/services"
)

// AdminProviderUpdateHandler handles updating providers
type AdminProviderUpdateHandler struct {
	providerService *services.ProviderService
	auditLogger     *logging.AuditLogger
	logger          *logrus.Logger
}

// NewAdminProviderUpdateHandler creates a new provider update handler
func NewAdminProviderUpdateHandler(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *AdminProviderUpdateHandler {
	return &AdminProviderUpdateHandler{
		providerService: providerService,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// Handle handles PUT /admin/api/v1/providers/:id
func (h *AdminProviderUpdateHandler) Handle(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provider ID is required",
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

	// Update provider
	updatedProvider, err := h.providerService.UpdateProvider(providerID, updates)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update provider")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderUpdate, providerID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to update provider: " + err.Error(),
		})
		return
	}

	// Log successful update
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderUpdate, providerID, c.ClientIP(), "success", map[string]interface{}{
			"updates": updates,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "provider updated successfully",
		"provider": updatedProvider,
	})
}
