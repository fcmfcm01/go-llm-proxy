package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/services"
)

// AdminProviderToggleHandler handles toggling provider status
type AdminProviderToggleHandler struct {
	providerService *services.ProviderService
	auditLogger     *logging.AuditLogger
	logger          *logrus.Logger
}

// NewAdminProviderToggleHandler creates a new provider toggle handler
func NewAdminProviderToggleHandler(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *AdminProviderToggleHandler {
	return &AdminProviderToggleHandler{
		providerService: providerService,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// Handle handles POST /admin/api/v1/providers/:id/toggle
func (h *AdminProviderToggleHandler) Handle(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provider ID is required",
		})
		return
	}

	// Toggle provider
	updatedProvider, err := h.providerService.ToggleProvider(providerID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to toggle provider")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderToggle, providerID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to toggle provider: " + err.Error(),
		})
		return
	}

	// Log successful toggle
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderToggle, providerID, c.ClientIP(), "success", map[string]interface{}{
			"enabled": updatedProvider.Enabled,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":  "provider toggled successfully",
		"provider": updatedProvider,
	})
}
