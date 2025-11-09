package server

import (
	"net/http"

	"github.com/example/go-llm-proxy/internal/config"
	"github.com/example/go-llm-proxy/internal/proxy"
	"github.com/example/go-llm-proxy/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler manages HTTP request handlers
type Handler struct {
	configManager *config.ConfigManager
	proxyHandler  *proxy.ProxyHandler
	logger        *logrus.Logger
}

// NewHandler creates a new HTTP handler
func NewHandler(configManager *config.ConfigManager, proxyHandler *proxy.ProxyHandler, logger *logrus.Logger) *Handler {
	return &Handler{
		configManager: configManager,
		proxyHandler:  proxyHandler,
		logger:        logger,
	}
}

// GetProviders returns all providers
func (h *Handler) GetProviders(c *gin.Context) {
	providers := h.configManager.GetProviders()
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    providers,
	})
}

// AddProvider adds a new provider
func (h *Handler) AddProvider(c *gin.Context) {
	var provider config.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	if err := h.configManager.AddProvider(provider); err != nil {
		c.JSON(http.StatusInternalServerError, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "SAVE_ERROR",
				Message: "Failed to save provider",
			},
		})
		return
	}

	c.JSON(http.StatusCreated, types.APIResponse{
		Success: true,
		Data:    provider,
		Message: "Provider added successfully",
	})
}

// UpdateProvider updates an existing provider
func (h *Handler) UpdateProvider(c *gin.Context) {
	id := c.Param("id")
	var provider config.Provider
	if err := c.ShouldBindJSON(&provider); err != nil {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "VALIDATION_ERROR",
				Message: err.Error(),
			},
		})
		return
	}

	if err := h.configManager.UpdateProvider(id, provider); err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "NOT_FOUND",
				Message: "Provider not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    provider,
		Message: "Provider updated successfully",
	})
}

// DeleteProvider deletes a provider
func (h *Handler) DeleteProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.configManager.DeleteProvider(id); err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "NOT_FOUND",
				Message: "Provider not found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Message: "Provider deleted successfully",
	})
}

// ToggleProvider toggles provider enabled status
func (h *Handler) ToggleProvider(c *gin.Context) {
	id := c.Param("id")
	if err := h.configManager.ToggleProvider(id); err != nil {
		c.JSON(http.StatusNotFound, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "NOT_FOUND",
				Message: "Provider not found",
			},
		})
		return
	}

	providers := h.configManager.GetProviders()
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data:    providers,
		Message: "Provider toggled successfully",
	})
}
