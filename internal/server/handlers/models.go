package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ModelsHandler handles models list requests
type ModelsHandler struct {
	loadBalancer     proxy.LoadBalancer
	modelMapper      *proxy.ModelMapper
	proxyAuditLogger *logging.ProxyAuditLogger
	logger           *logrus.Logger
}

// ModelsConfig holds configuration for the handler
type ModelsConfig struct {
	LoadBalancer     proxy.LoadBalancer
	ModelMapper      *proxy.ModelMapper
	ProxyAuditLogger *logging.ProxyAuditLogger
	Logger           *logrus.Logger
}

// NewModelsHandler creates a new models handler
func NewModelsHandler(config *ModelsConfig) *ModelsHandler {
	return &ModelsHandler{
		loadBalancer:     config.LoadBalancer,
		modelMapper:      config.ModelMapper,
		proxyAuditLogger: config.ProxyAuditLogger,
		logger:           config.Logger,
	}
}

// Model represents a model in the OpenAI format
type Model struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// ModelsResponse represents the response for /v1/models
type ModelsResponse struct {
	Object string  `json:"object"`
	Data   []Model `json:"data"`
}

// HandleModels handles GET /v1/models
func (h *ModelsHandler) HandleModels(c *gin.Context) {
	requestID := generateRequestID()

	// Extract session information
	session, userID, username := h.extractSessionInfo(c)
	sessionID := ""
	if session != nil {
		sessionID = session.ID
	}

	// Log request
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"session_id": sessionID,
			"user_id":    userID,
			"username":   username,
		}).Info("Models list request received")
	}

	// Get all providers
	providers := h.loadBalancer.GetProviders()

	// Collect unique models from all enabled providers
	modelSet := make(map[string]*Model)

	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}

		// Get supported models for this provider
		supportedModels := h.getSupportedModelsForProvider(provider)

		for _, modelID := range supportedModels {
			if _, exists := modelSet[modelID]; !exists {
				modelSet[modelID] = &Model{
					ID:      modelID,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: provider.Name,
				}
			}
		}
	}

	// Convert map to slice
	modelsList := make([]Model, 0, len(modelSet))
	for _, model := range modelSet {
		modelsList = append(modelsList, *model)
	}

	// Create response
	response := ModelsResponse{
		Object: "list",
		Data:   modelsList,
	}

	// Log success
	if h.proxyAuditLogger != nil {
		// Use the underlying audit logger
		// h.proxyAuditLogger.auditLogger.LogEvent(event) // would need to expose this
	}

	// Return response
	c.JSON(http.StatusOK, response)
}

// HandleModelDetails handles GET /v1/models/:model
func (h *ModelsHandler) HandleModelDetails(c *gin.Context) {
	requestID := generateRequestID()
	modelID := c.Param("model")

	// Extract session information
	session, userID, username := h.extractSessionInfo(c)
	sessionID := ""
	if session != nil {
		sessionID = session.ID
	}

	// Log request
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"session_id": sessionID,
			"user_id":    userID,
			"username":   username,
			"model":      modelID,
		}).Info("Model details request received")
	}

	// Get all providers
	providers := h.loadBalancer.GetProviders()

	// Find the model in any enabled provider
	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}

		supportedModels := h.getSupportedModelsForProvider(provider)
		for _, supportedModel := range supportedModels {
			if supportedModel == modelID {
				// Found the model
				model := Model{
					ID:      modelID,
					Object:  "model",
					Created: time.Now().Unix(),
					OwnedBy: provider.Name,
				}

				c.JSON(http.StatusOK, model)
				return
			}
		}
	}

	// Model not found
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"model":      modelID,
		}).Warn("Model not found")
	}

	c.JSON(http.StatusNotFound, gin.H{
		"error": map[string]interface{}{
			"message": "The model '" + modelID + "' does not exist",
			"type":    "invalid_request_error",
			"param":   nil,
			"code":    "model_not_found",
		},
	})
}

// getSupportedModelsForProvider returns supported models for a provider
func (h *ModelsHandler) getSupportedModelsForProvider(provider *models.Provider) []string {
	// If model mapper is available, use it
	if h.modelMapper != nil {
		return h.modelMapper.GetSupportedModels(provider)
	}

	// Otherwise return default models based on provider type
	return h.getDefaultModelsForProvider(provider)
}

// getDefaultModelsForProvider returns default models for a provider
func (h *ModelsHandler) getDefaultModelsForProvider(provider *models.Provider) []string {
	// Detect provider type from URL
	apiURL := provider.APIURL

	if contains(apiURL, "openai") {
		return []string{
			"gpt-4-turbo",
			"gpt-4",
			"gpt-3.5-turbo",
			"text-embedding-3-large",
			"text-embedding-3-small",
			"text-embedding-ada-002",
		}
	}

	if contains(apiURL, "anthropic") || contains(apiURL, "claude") {
		return []string{
			"claude-3-5-sonnet-20241022",
			"claude-3-opus-20240229",
			"claude-3-sonnet-20240229",
			"claude-3-haiku-20240307",
		}
	}

	// Unknown provider - return empty list
	return []string{}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

// extractSessionInfo extracts session information from context
func (h *ModelsHandler) extractSessionInfo(c *gin.Context) (*models.Session, string, string) {
	sessionVal, exists := c.Get("session")
	if !exists {
		return nil, "", ""
	}

	session, ok := sessionVal.(*models.Session)
	if !ok {
		return nil, "", ""
	}

	return session, session.UserID, session.Username
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d_%s", time.Now().UnixNano(), time.Now().Format("20060102"))
}
