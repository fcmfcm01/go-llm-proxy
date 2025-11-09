package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/services"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
)

// AdminProviderCreateHandler handles creating new providers
type AdminProviderCreateHandler struct {
	providerService *services.ProviderService
	auditLogger     *logging.AuditLogger
	logger          *logrus.Logger
}

// NewAdminProviderCreateHandler creates a new provider create handler
func NewAdminProviderCreateHandler(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *AdminProviderCreateHandler {
	return &AdminProviderCreateHandler{
		providerService: providerService,
		auditLogger:     auditLogger,
		logger:          logger,
	}
}

// Handle handles POST /admin/api/v1/providers
func (h *AdminProviderCreateHandler) Handle(c *gin.Context) {
	var provider models.Provider

	// Bind JSON request
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request: " + err.Error(),
		})
		return
	}

	// Create provider
	if err := h.providerService.CreateProvider(&provider); err != nil {
		h.logger.WithError(err).Error("Failed to create provider")

		// Log the failed action
		if h.auditLogger != nil {
			h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderCreate, provider.ID, c.ClientIP(), "failure", map[string]interface{}{
				"error": err.Error(),
			})
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to create provider: " + err.Error(),
		})
		return
	}

	// Log successful creation
	if h.auditLogger != nil {
		h.auditLogger.LogAdminEvent(c.Request.Context(), logging.AuditEventProviderCreate, provider.ID, c.ClientIP(), "success", map[string]interface{}{
			"name": provider.Name,
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "provider created successfully",
		"provider": provider,
	})
}
