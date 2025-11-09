package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/internal/services"
)

// ProviderServiceWrapper wraps the provider service for handler functions
type ProviderServiceWrapper struct {
	ProviderService *services.ProviderService
	AuditLogger     *logging.AuditLogger
	Logger          *logrus.Logger
}

// NewProviderServiceWrapper creates a new provider service wrapper
func NewProviderServiceWrapper(
	providerService *services.ProviderService,
	auditLogger *logging.AuditLogger,
	logger *logrus.Logger,
) *ProviderServiceWrapper {
	return &ProviderServiceWrapper{
		ProviderService: providerService,
		AuditLogger:     auditLogger,
		Logger:          logger,
	}
}

// HandleListProviders handles GET /admin/api/v1/providers
func (w *ProviderServiceWrapper) HandleListProviders(c *gin.Context) {
	providers, err := w.ProviderService.ListProviders()
	if err != nil {
		w.Logger.WithError(err).Error("Failed to list providers")
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
func (w *ProviderServiceWrapper) HandleCreateProvider(c *gin.Context) {
	var provider models.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	if err := w.ProviderService.CreateProvider(&provider); err != nil {
		w.Logger.WithError(err).Error("Failed to create provider")
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
func (w *ProviderServiceWrapper) HandleGetProvider(c *gin.Context) {
	id := c.Param("id")
	provider, err := w.ProviderService.GetProvider(id)
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
func (w *ProviderServiceWrapper) HandleUpdateProvider(c *gin.Context) {
	id := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid request body",
		})
		return
	}

	provider, err := w.ProviderService.UpdateProvider(id, updates)
	if err != nil {
		w.Logger.WithError(err).Error("Failed to update provider")
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
func (w *ProviderServiceWrapper) HandleDeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if err := w.ProviderService.DeleteProvider(id); err != nil {
		w.Logger.WithError(err).Error("Failed to delete provider")
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
func (w *ProviderServiceWrapper) HandleToggleProvider(c *gin.Context) {
	id := c.Param("id")
	provider, err := w.ProviderService.GetProvider(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "provider not found",
		})
		return
	}

	updates := map[string]interface{}{
		"enabled": !provider.Enabled,
	}
	provider, err = w.ProviderService.UpdateProvider(id, updates)
	if err != nil {
		w.Logger.WithError(err).Error("Failed to toggle provider")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to toggle provider",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": provider,
	})
}
