package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/proxy"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ChatCompletionsHandler handles chat completions requests
type ChatCompletionsHandler struct {
	proxyHandler     *proxy.ProxyHandler
	streamingHandler *proxy.StreamingHandler
	modelMapper      *proxy.ModelMapper
	proxyAuditLogger *logging.ProxyAuditLogger
	logger           *logrus.Logger
}

// ChatCompletionsConfig holds configuration for the handler
type ChatCompletionsConfig struct {
	ProxyHandler     *proxy.ProxyHandler
	StreamingHandler *proxy.StreamingHandler
	ModelMapper      *proxy.ModelMapper
	ProxyAuditLogger *logging.ProxyAuditLogger
	Logger           *logrus.Logger
}

// NewChatCompletionsHandler creates a new chat completions handler
func NewChatCompletionsHandler(config *ChatCompletionsConfig) *ChatCompletionsHandler {
	return &ChatCompletionsHandler{
		proxyHandler:     config.ProxyHandler,
		streamingHandler: config.StreamingHandler,
		modelMapper:      config.ModelMapper,
		proxyAuditLogger: config.ProxyAuditLogger,
		logger:           config.Logger,
	}
}

// HandleChatCompletions handles POST /v1/chat/completions
func (h *ChatCompletionsHandler) HandleChatCompletions(c *gin.Context) {
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
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/chat/completions", "failed to read request body", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	defer c.Request.Body.Close()

	// Parse request
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/chat/completions", "invalid JSON", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON in request body"})
		return
	}

	// Validate required fields
	if err := h.validateChatCompletionsRequest(requestBody); err != nil {
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/chat/completions", "validation failed", err)
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
		}).Info("Chat completions request received")
	}

	// Create proxy request
	proxyReq := &proxy.ProxyRequest{
		Method: "POST",
		Path:   "/v1/chat/completions",
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

// handleNonStreamingRequest handles non-streaming chat completion requests
func (h *ChatCompletionsHandler) handleNonStreamingRequest(
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
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/chat/completions", "proxy request failed", err)

		if h.proxyAuditLogger != nil {
			h.proxyAuditLogger.LogProxyError(
				requestID, sessionID, userID, username, c.ClientIP(),
				"POST", "/v1/chat/completions",
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
			// Determine if this was a provider failure
			provider := resp.Provider
			if provider != nil {
				h.proxyAuditLogger.LogProxyFailure(
					requestID, sessionID, userID, username, c.ClientIP(),
					"POST", "/v1/chat/completions",
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

	// Extract target model (may be mapped)
	targetModel := sourceModel
	if modelVal, ok := resp.Body["model"].(string); ok {
		targetModel = modelVal
	}

	if h.proxyAuditLogger != nil {
		h.proxyAuditLogger.LogProxySuccess(
			requestID, sessionID, userID, username, c.ClientIP(),
			"POST", "/v1/chat/completions",
			resp.Provider, sourceModel, targetModel,
			duration, requestSize, responseSize,
			resp.StatusCode, false,
		)
	}

	// Return response
	c.JSON(resp.StatusCode, resp.Body)
}

// handleStreamingRequest handles streaming chat completion requests
func (h *ChatCompletionsHandler) handleStreamingRequest(
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
		h.logError(requestID, sessionID, userID, username, c.ClientIP(), "POST", "/v1/chat/completions", "streaming request failed", err)

		if h.proxyAuditLogger != nil {
			h.proxyAuditLogger.LogProxyError(
				requestID, sessionID, userID, username, c.ClientIP(),
				"POST", "/v1/chat/completions",
				err.Error(), "streaming_error",
				duration,
			)
		}

		// For streaming, we may have already sent headers, so we can't change status
		// Write an error event instead
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
			"POST", "/v1/chat/completions",
			&models.Provider{ID: "streaming", Name: "streaming"}, // Would need to track actual provider
			sourceModel, sourceModel,
			duration, requestSize, 0,
			http.StatusOK, true,
		)
	}
}

// validateChatCompletionsRequest validates the chat completions request
func (h *ChatCompletionsHandler) validateChatCompletionsRequest(req map[string]interface{}) error {
	// Check model
	model, ok := req["model"].(string)
	if !ok || model == "" {
		return fmt.Errorf("model is required")
	}

	// Check messages
	messages, ok := req["messages"].([]interface{})
	if !ok || len(messages) == 0 {
		return fmt.Errorf("messages is required and must be a non-empty array")
	}

	// Validate each message
	for i, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			return fmt.Errorf("message %d is not a valid object", i)
		}

		// Check role
		role, ok := msgMap["role"].(string)
		if !ok || role == "" {
			return fmt.Errorf("message %d: role is required", i)
		}

		if role != "system" && role != "user" && role != "assistant" {
			return fmt.Errorf("message %d: role must be system, user, or assistant", i)
		}

		// Check content
		content := msgMap["content"]
		if content == nil {
			return fmt.Errorf("message %d: content is required", i)
		}

		// Content can be string or array
		switch content.(type) {
		case string:
			// Valid
		case []interface{}:
			// Valid (multimodal content)
		default:
			return fmt.Errorf("message %d: content must be string or array", i)
		}
	}

	return nil
}

// extractSessionInfo extracts session information from context
func (h *ChatCompletionsHandler) extractSessionInfo(c *gin.Context) (*models.Session, string, string) {
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
func (h *ChatCompletionsHandler) logError(requestID, sessionID, userID, username, ip, method, path, message string, err error) {
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

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return fmt.Sprintf("req_%d", time.Now().UnixNano())
}
