package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ValidationConfig holds configuration for request validation
type ValidationConfig struct {
	MaxRequestSize  int64 // Maximum request body size in bytes
	MaxMessageCount int   // Maximum number of messages in chat completions
	MaxPromptLength int   // Maximum prompt length in characters
	Logger          *logrus.Logger
}

// DefaultValidationConfig returns default validation configuration
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		MaxRequestSize:  10 * 1024 * 1024, // 10MB
		MaxMessageCount: 100,              // 100 messages
		MaxPromptLength: 100000,           // 100k characters
	}
}

// ValidationMiddleware creates a request validation middleware
func ValidationMiddleware(config *ValidationConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultValidationConfig()
	}

	return func(c *gin.Context) {
		// Skip validation for GET requests
		if c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		// Validate content type for POST/PUT requests
		if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut {
			contentType := c.GetHeader("Content-Type")
			if contentType != "application/json" && contentType != "application/json; charset=utf-8" {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error": "Content-Type must be application/json",
				})
				c.Abort()
				return
			}
		}

		// Validate request size
		if c.Request.ContentLength > config.MaxRequestSize {
			if config.Logger != nil {
				config.Logger.WithFields(logrus.Fields{
					"content_length": c.Request.ContentLength,
					"max_size":       config.MaxRequestSize,
					"path":           c.Request.URL.Path,
				}).Warn("Request too large")
			}

			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": fmt.Sprintf("Request body too large. Maximum size: %d bytes", config.MaxRequestSize),
			})
			c.Abort()
			return
		}

		// For proxy endpoints, perform additional validation
		if isProxyEndpoint(c.Request.URL.Path) {
			if err := validateProxyRequest(c, config); err != nil {
				if config.Logger != nil {
					config.Logger.WithFields(logrus.Fields{
						"error": err.Error(),
						"path":  c.Request.URL.Path,
					}).Warn("Request validation failed")
				}

				c.JSON(http.StatusBadRequest, gin.H{
					"error": err.Error(),
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// isProxyEndpoint checks if the path is a proxy endpoint
func isProxyEndpoint(path string) bool {
	proxyPaths := []string{
		"/v1/chat/completions",
		"/v1/completions",
		"/v1/embeddings",
	}

	for _, proxyPath := range proxyPaths {
		if path == proxyPath {
			return true
		}
	}

	return false
}

// validateProxyRequest validates proxy-specific request fields
func validateProxyRequest(c *gin.Context, config *ValidationConfig) error {
	// Read request body
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: %w", err)
	}

	// Restore body for downstream handlers
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Parse JSON
	var requestBody map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate based on endpoint
	path := c.Request.URL.Path

	switch path {
	case "/v1/chat/completions":
		return validateChatCompletionsRequest(requestBody, config)
	case "/v1/completions":
		return validateCompletionsRequest(requestBody, config)
	case "/v1/embeddings":
		return validateEmbeddingsRequest(requestBody, config)
	}

	return nil
}

// validateChatCompletionsRequest validates chat completions request
func validateChatCompletionsRequest(req map[string]interface{}, config *ValidationConfig) error {
	// Check messages count
	messages, ok := req["messages"].([]interface{})
	if ok && len(messages) > config.MaxMessageCount {
		return fmt.Errorf("too many messages: %d (max: %d)", len(messages), config.MaxMessageCount)
	}

	// Validate message content length
	if ok {
		for i, msg := range messages {
			msgMap, ok := msg.(map[string]interface{})
			if !ok {
				continue
			}

			content, ok := msgMap["content"].(string)
			if ok && len(content) > config.MaxPromptLength {
				return fmt.Errorf("message %d content too long: %d chars (max: %d)", i, len(content), config.MaxPromptLength)
			}
		}
	}

	// Validate max_tokens if present
	if maxTokens, ok := req["max_tokens"].(float64); ok {
		if maxTokens < 1 || maxTokens > 100000 {
			return fmt.Errorf("max_tokens must be between 1 and 100000")
		}
	}

	// Validate temperature if present
	if temperature, ok := req["temperature"].(float64); ok {
		if temperature < 0 || temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
	}

	// Validate top_p if present
	if topP, ok := req["top_p"].(float64); ok {
		if topP < 0 || topP > 1 {
			return fmt.Errorf("top_p must be between 0 and 1")
		}
	}

	// Validate n if present
	if n, ok := req["n"].(float64); ok {
		if n < 1 || n > 10 {
			return fmt.Errorf("n must be between 1 and 10")
		}
	}

	return nil
}

// validateCompletionsRequest validates completions request
func validateCompletionsRequest(req map[string]interface{}, config *ValidationConfig) error {
	// Validate prompt length
	prompt := req["prompt"]
	if promptStr, ok := prompt.(string); ok {
		if len(promptStr) > config.MaxPromptLength {
			return fmt.Errorf("prompt too long: %d chars (max: %d)", len(promptStr), config.MaxPromptLength)
		}
	} else if promptArray, ok := prompt.([]interface{}); ok {
		for i, p := range promptArray {
			if pStr, ok := p.(string); ok && len(pStr) > config.MaxPromptLength {
				return fmt.Errorf("prompt[%d] too long: %d chars (max: %d)", i, len(pStr), config.MaxPromptLength)
			}
		}
	}

	// Validate max_tokens if present
	if maxTokens, ok := req["max_tokens"].(float64); ok {
		if maxTokens < 1 || maxTokens > 100000 {
			return fmt.Errorf("max_tokens must be between 1 and 100000")
		}
	}

	// Validate temperature if present
	if temperature, ok := req["temperature"].(float64); ok {
		if temperature < 0 || temperature > 2 {
			return fmt.Errorf("temperature must be between 0 and 2")
		}
	}

	return nil
}

// validateEmbeddingsRequest validates embeddings request
func validateEmbeddingsRequest(req map[string]interface{}, config *ValidationConfig) error {
	// Validate input length
	input := req["input"]
	if inputStr, ok := input.(string); ok {
		if len(inputStr) > config.MaxPromptLength {
			return fmt.Errorf("input too long: %d chars (max: %d)", len(inputStr), config.MaxPromptLength)
		}
	} else if inputArray, ok := input.([]interface{}); ok {
		if len(inputArray) > 100 {
			return fmt.Errorf("too many inputs: %d (max: 100)", len(inputArray))
		}

		for i, inp := range inputArray {
			if inpStr, ok := inp.(string); ok && len(inpStr) > config.MaxPromptLength {
				return fmt.Errorf("input[%d] too long: %d chars (max: %d)", i, len(inpStr), config.MaxPromptLength)
			}
		}
	}

	return nil
}

// ContentTypeMiddleware ensures Content-Type is set correctly
func ContentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// For JSON responses, ensure Content-Type is set
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware(allowedOrigins []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		allowed := false
		if len(allowedOrigins) == 0 || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
			allowed = true
		} else {
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}
		}

		if allowed {
			if origin != "" {
				c.Header("Access-Control-Allow-Origin", origin)
			} else {
				c.Header("Access-Control-Allow-Origin", "*")
			}
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Session-ID")
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Max-Age", "3600")
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
