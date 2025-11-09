package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/services"
)

// AdminProviderDeleteHandler handles deleting providers
type AdminProviderDeleteHandler struct {
	providerService *services.ProviderService
	auditLogger     *logging.AuditLogger
	logger          *logrus.Logger
}

// NewAdminProviderDeleteHandler creates a new provider delete handler
func NewAdminProviderDeleteHandler(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *AdminProviderDeleteHandler {
	return &AdminProviderDeleteHandler{
		providerService: providerService,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// Handle handles DELETE /admin/api/v1/providers/:id
func (h *AdminProviderDeleteHandler) Handle(c *gin.Context) {
	providerID := c.Param("id")
	if providerID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "provider ID is required",
		})
		return
	}

	// Delete provider
	if err := h.providerService.DeleteProvider(providerID); err != nil {
		h.logger.WithError(err).Error("Failed to delete provider")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderDelete, providerID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to delete provider: " + err.Error(),
		})
		return
	}

	// Log successful deletion
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderDelete, providerID, c.ClientIP(), "success", nil)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "provider deleted successfully",
	})
}
