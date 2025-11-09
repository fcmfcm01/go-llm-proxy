package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/example/go-llm-proxy/internal/logging"
	"github.com/example/go-llm-proxy/internal/models"
	"github.com/example/go-llm-proxy/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// EmbeddingsHandler handles embeddings requests
type EmbeddingsHandler struct {
	proxyHandler     *proxy.ProxyHandler
	modelMapper      *proxy.ModelMapper
	proxyAuditLogger *logging.ProxyAuditLogger
	logger           *logrus.Logger
}

// EmbeddingsConfig holds configuration for the handler
type EmbeddingsConfig struct {
	ProxyHandler     *proxy.ProxyHandler
	ModelMapper      *proxy.ModelMapper
	ProxyAuditLogger *logging.ProxyAuditLogger
	Logger           *logrus.Logger
}

// NewEmbeddingsHandler creates a new embeddings handler
func NewEmbeddingsHandler(config *EmbeddingsConfig) *EmbeddingsHandler {
	return &EmbeddingsHandler{
		proxyHandler:     config.ProxyHandler,
		modelMapper:      config.ModelMapper,
		proxyAuditLogger: config.ProxyAuditLogger,
		logger:           config.Logger,
	}
}

// HandleEmbeddings handles POST /v1/embeddings
func (h *EmbeddingsHandler) HandleEmbeddings(c *gin.Context) {
	startTime := time.Now()
	requestID := generateRequestID()

	// Extract session information
	session, userID, username := h.extractSessionInfo(c)
	sessionID := ""
	if session != nil {
		sessionID = session.ID
	}

	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/embeddings", "failed to read request body", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	defer c.Request.Body.Close()

	// Parse request
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/embeddings", "invalid JSON", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON in request body"})
		return
	}

	// Validate required fields
	if err := h.validateEmbeddingsRequest(requestBody); err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/embeddings", "validation failed", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract model
	sourceModel, _ := requestBody["model"].(string)

	// Log request
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"session_id": sessionID,
			"user_id":    userID,
			"username":   username,
			"model":      sourceModel,
		}).Info("Embeddings request received")
	}

	// Create proxy request
	proxyReq := &proxy.ProxyRequest{
		Method: "POST",
		Path:   "/v1/embeddings",
		Body:   requestBody,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Execute proxy request (embeddings don't support streaming)
	resp, err := h.proxyHandler.HandleRequest(c.Request.Context(), proxyReq)
	if err != nil {
		duration := time.Since(startTime)
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/embeddings", "proxy request failed", err)

		if h.proxyAuditLogger != nil {
			h.proxyAuditLogger.LogProxyError(
				requestID, sessionID, userID, username, c.ClientIP(),
				"POST", "/v1/embeddings",
				err.Error(), "proxy_error",
				duration,
			)
		}

		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to proxy request", "details": err.Error()})
		return
	}

	// Check for provider-level errors
	if resp.Error != nil {
		duration := time.Since(startTime)

		if h.proxyAuditLogger != nil {
			provider := resp.Provider
			if provider != nil {
				h.proxyAuditLogger.LogProxyFailure(
					requestID, sessionID, userID, username, c.ClientIP(),
					"POST", "/v1/embeddings",
					provider, sourceModel,
					duration, int64(len(bodyBytes)),
					resp.Error, 0, false,
				)
			}
		}

		c.JSON(resp.StatusCode, resp.Body)
		return
	}

	// Success - log audit event
	duration := time.Since(startTime)
	responseSize := int64(len(fmt.Sprintf("%v", resp.Body)))

	// Extract target model
	targetModel := sourceModel
	if modelVal, ok := resp.Body["model"].(string); ok {
		targetModel = modelVal
	}

	if h.proxyAuditLogger != nil {
		h.proxyAuditLogger.LogProxySuccess(
			requestID, sessionID, userID, username, c.ClientIP(),
			"POST", "/v1/embeddings",
			resp.Provider, sourceModel, targetModel,
			duration, int64(len(bodyBytes)), responseSize,
			resp.StatusCode, false,
		)
	}

	// Return response
	c.JSON(resp.StatusCode, resp.Body)
}

// validateEmbeddingsRequest validates the embeddings request
func (h *EmbeddingsHandler) validateEmbeddingsRequest(req map[string]interface{}) error {
	// Check model
	model, ok := req["model"].(string)
	if !ok || model == "" {
		return fmt.Errorf("model is required")
	}

	// Check input
	input := req["input"]
	if input == nil {
		return fmt.Errorf("input is required")
	}

	// Input can be string or array of strings
	switch inp := input.(type) {
	case string:
		if inp == "" {
			return fmt.Errorf("input cannot be empty")
		}
	case []interface{}:
		if len(inp) == 0 {
			return fmt.Errorf("input array cannot be empty")
		}
		for i, item := range inp {
			switch it := item.(type) {
			case string:
				if it == "" {
					return fmt.Errorf("input[%d] cannot be empty", i)
				}
			case []interface{}:
				// Token array format - validate it's not empty
				if len(it) == 0 {
					return fmt.Errorf("input[%d] token array cannot be empty", i)
				}
			default:
				return fmt.Errorf("input[%d] must be string or token array", i)
			}
		}
	default:
		return fmt.Errorf("input must be string or array")
	}

	// Validate encoding format if present
	if encodingFormat, ok := req["encoding_format"]; ok {
		if format, ok := encodingFormat.(string); ok {
			if format != "float" && format != "base64" {
				return fmt.Errorf("encoding_format must be 'float' or 'base64'")
			}
		}
	}

	// Validate dimensions if present (for models that support it)
	if dimensions, ok := req["dimensions"]; ok {
		if dim, ok := dimensions.(float64); ok {
			if dim < 1 {
				return fmt.Errorf("dimensions must be at least 1")
			}
		}
	}

	return nil
}

// extractSessionInfo extracts session information from context
func (h *EmbeddingsHandler) extractSessionInfo(c *gin.Context) (*models.Session, string, string) {
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

// logError logs an error with context
func (h *EmbeddingsHandler) logError(requestID, sessionID, userID, username, ip, method, path, message string, err error) {
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"session_id": sessionID,
			"user_id":    userID,
			"username":   username,
			"ip":         ip,
			"method":     method,
			"path":       path,
			"error":      err.Error(),
		}).Error(message)
	}
}
