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

// CompletionsHandler handles legacy completions requests
type CompletionsHandler struct {
	proxyHandler     *proxy.ProxyHandler
	streamingHandler *proxy.StreamingHandler
	modelMapper      *proxy.ModelMapper
	proxyAuditLogger *logging.ProxyAuditLogger
	logger           *logrus.Logger
}

// CompletionsConfig holds configuration for the handler
type CompletionsConfig struct {
	ProxyHandler     *proxy.ProxyHandler
	StreamingHandler *proxy.StreamingHandler
	ModelMapper      *proxy.ModelMapper
	ProxyAuditLogger *logging.ProxyAuditLogger
	Logger           *logrus.Logger
}

// NewCompletionsHandler creates a new completions handler
func NewCompletionsHandler(config *CompletionsConfig) *CompletionsHandler {
	return &CompletionsHandler{
		proxyHandler:     config.ProxyHandler,
		streamingHandler: config.StreamingHandler,
		modelMapper:      config.ModelMapper,
		proxyAuditLogger: config.ProxyAuditLogger,
		logger:           config.Logger,
	}
}

// HandleCompletions handles POST /v1/completions
func (h *CompletionsHandler) HandleCompletions(c *gin.Context) {
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
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/completions", "failed to read request body", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	defer c.Request.Body.Close()

	// Parse request
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/completions", "invalid JSON", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON in request body"})
		return
	}

	// Validate required fields
	if err := h.validateCompletionsRequest(requestBody); err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/completions", "validation failed", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract model and streaming flag
	sourceModel, _ := requestBody["model"].(string)
	isStreaming, _ := requestBody["stream"].(bool)

	// Log request
	if h.logger != nil {
		h.logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"session_id": sessionID,
			"user_id":    userID,
			"username":   username,
			"model":      sourceModel,
			"streaming":  isStreaming,
		}).Info("Completions request received")
	}

	// Create proxy request
	proxyReq := &proxy.ProxyRequest{
		Method: "POST",
		Path:   "/v1/completions",
		Body:   requestBody,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Handle streaming or non-streaming
	if isStreaming {
		h.handleStreamingRequest(c, proxyReq, requestID, sessionID, userID, username, sourceModel, startTime, int64(len(bodyBytes)))
	} else {
		h.handleNonStreamingRequest(c, proxyReq, requestID, sessionID, userID, username, sourceModel, startTime, int64(len(bodyBytes)))
	}
}

// handleNonStreamingRequest handles non-streaming completion requests
func (h *CompletionsHandler) handleNonStreamingRequest(
	c *gin.Context,
	proxyReq *proxy.ProxyRequest,
	requestID, sessionID, userID, username, sourceModel string,
	startTime time.Time,
	requestSize int64,
) {
	// Execute proxy request
	resp, err := h.proxyHandler.HandleRequest(c.Request.Context(), proxyReq)
	if err != nil {
		duration := time.Since(startTime)
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/completions", "proxy request failed", err)

		if h.proxyAuditLogger != nil {
			h.proxyAuditLogger.LogProxyError(
				requestID, sessionID, userID, username, c.ClientIP(),
				"POST", "/v1/completions",
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
					"POST", "/v1/completions",
					provider, sourceModel,
					duration, requestSize,
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
			"POST", "/v1/completions",
			resp.Provider, sourceModel, targetModel,
			duration, requestSize, responseSize,
			resp.StatusCode, false,
		)
	}

	// Return response
	c.JSON(resp.StatusCode, resp.Body)
}

// handleStreamingRequest handles streaming completion requests
func (h *CompletionsHandler) handleStreamingRequest(
	c *gin.Context,
	proxyReq *proxy.ProxyRequest,
	requestID, sessionID, userID, username, sourceModel string,
	startTime time.Time,
	requestSize int64,
) {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// Get response writer
	writer := c.Writer

	// Execute streaming request
	err := h.streamingHandler.StreamRequest(c.Request.Context(), proxyReq, writer)
	duration := time.Since(startTime)

	if err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/completions", "streaming request failed", err)

		if h.proxyAuditLogger != nil {
			h.proxyAuditLogger.LogProxyError(
				requestID, sessionID, userID, username, c.ClientIP(),
				"POST", "/v1/completions",
				err.Error(), "streaming_error",
				duration,
			)
		}

		// Write error event
		fmt.Fprintf(writer, "data: {\"error\": \"%s\"}\n\n", err.Error())
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	// Success - log audit event
	if h.proxyAuditLogger != nil {
		h.proxyAuditLogger.LogProxySuccess(
			requestID, sessionID, userID, username, c.ClientIP(),
			"POST", "/v1/completions",
			&models.Provider{ID: "streaming", Name: "streaming"},
			sourceModel, sourceModel,
			duration, requestSize, 0,
			http.StatusOK, true,
		)
	}
}

// validateCompletionsRequest validates the completions request
func (h *CompletionsHandler) validateCompletionsRequest(req map[string]interface{}) error {
	// Check model
	model, ok := req["model"].(string)
	if !ok || model == "" {
		return fmt.Errorf("model is required")
	}

	// Check prompt
	prompt := req["prompt"]
	if prompt == nil {
		return fmt.Errorf("prompt is required")
	}

	// Prompt can be string or array of strings
	switch p := prompt.(type) {
	case string:
		if p == "" {
			return fmt.Errorf("prompt cannot be empty")
		}
	case []interface{}:
		if len(p) == 0 {
			return fmt.Errorf("prompt array cannot be empty")
		}
		for i, item := range p {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("prompt[%d] must be a string", i)
			}
		}
	default:
		return fmt.Errorf("prompt must be string or array of strings")
	}

	// Validate optional parameters
	if maxTokens, ok := req["max_tokens"]; ok {
		if mt, ok := maxTokens.(float64); ok {
			if mt < 1 {
				return fmt.Errorf("max_tokens must be at least 1")
			}
		}
	}

	if temperature, ok := req["temperature"]; ok {
		if temp, ok := temperature.(float64); ok {
			if temp < 0 || temp > 2 {
				return fmt.Errorf("temperature must be between 0 and 2")
			}
		}
	}

	return nil
}

// extractSessionInfo extracts session information from context
func (h *CompletionsHandler) extractSessionInfo(c *gin.Context) (*models.Session, string, string) {
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
func (h *CompletionsHandler) logError(requestID, sessionID, userID, username, ip, method, path, message string, err error) {
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
