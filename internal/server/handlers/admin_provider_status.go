package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/services"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
)

// AdminProviderStatusHandler handles provider status requests
type AdminProviderStatusHandler struct {
	providerService *services.ProviderService
	logger          *logrus.Logger
	auditLogger     *logging.AuditLogger
}

// NewAdminProviderStatusHandler creates a new provider status handler
func NewAdminProviderStatusHandler(
	providerService *services.ProviderService,
	logger *logrus.Logger,
	auditLogger *logging.AuditLogger,
) *AdminProviderStatusHandler {
	return &AdminProviderStatusHandler{
		providerService: providerService,
		logger:          logger,
		auditLogger:     auditLogger,
	}
}

// HandleStatus handles GET /admin/api/v1/providers/status
func (h *AdminProviderStatusHandler) HandleStatus(c *gin.Context) {
	// Get provider status from service
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

// HandleDetailedStatus handles GET /admin/api/v1/providers/status/detailed
func (h *AdminProviderStatusHandler) HandleDetailedStatus(c *gin.Context) {
	// Get basic status
	status, err := h.providerService.GetProviderStatus()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get provider status")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get provider status",
		})
		return
	}

	// Add detailed metrics (if available from monitoring service)
	detailedStatus := map[string]interface{}{
		"total":     status["total"],
		"enabled":   status["enabled"],
		"disabled":  status["disabled"],
		"providers": status["providers"],
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// Log the action
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderStatus, "detailed", c.ClientIP(), "success", nil)
	}

	c.JSON(http.StatusOK, detailedStatus)
}
