package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// AdminProviderListHandler handles listing all providers
type AdminProviderListHandler struct {
	providerService *services.ProviderService
	auditLogger     *logging.AuditLogger
	logger          *logrus.Logger
}

// NewAdminProviderListHandler creates a new provider list handler
func NewAdminProviderListHandler(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *AdminProviderListHandler {
	return &AdminProviderListHandler{
		providerService: providerService,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// Handle handles GET /admin/api/v1/providers
func (h *AdminProviderListHandler) Handle(c *gin.Context) {
	// Get all providers
	providers, err := h.providerService.ListProviders()
	if err != nil {
		h.logger.WithError(err).Error("Failed to list providers")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to list providers",
		})
		return
	}

	// Log the action
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderList, "", c.ClientIP(), "success", map[string]interface{}{
			"count": len(providers),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"providers": providers,
		"count":     len(providers),
	})
}

// HandleStatus handles GET /admin/api/v1/providers/status
func (h *AdminProviderListHandler) HandleStatus(c *gin.Context) {
	// Get provider status
	status, err := h.providerService.GetProviderStatus()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get provider status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get provider status",
		})
		return
	}

	// Log the action
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderStatus, "", c.ClientIP(), "success", nil)
	}

	c.JSON(http.StatusOK, status)
}
