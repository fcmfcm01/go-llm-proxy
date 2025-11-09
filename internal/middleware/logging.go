package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// responseWriter wraps gin.ResponseWriter to capture response data
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// ProxyLoggingMiddleware creates comprehensive logging middleware for proxy operations
func ProxyLoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip logging for health checks
		if c.Request.URL.Path == "/healthz" || c.Request.URL.Path == "/healthz/ready" || c.Request.URL.Path == "/healthz/live" {
			c.Next()
			return
		}

		// Record start time
		startTime := time.Now()

		// Extract request information
		method := c.Request.Method
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()
		requestID := c.GetHeader("X-Request-ID")

		// Log request
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"method":     method,
				"path":       path,
				"query":      query,
				"client_ip":  clientIP,
				"user_agent": userAgent,
				"request_id": requestID,
			}).Info("Incoming request")
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer([]byte{}),
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Record end time and calculate duration
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		// Extract session/user info if available
		userID := ""
		username := ""
		sessionID := ""

		if sessionVal, exists := c.Get("session"); exists {
			if session, ok := sessionVal.(interface {
				GetUserID() string
				GetUsername() string
				GetID() string
			}); ok {
				userID = session.GetUserID()
				username = session.GetUsername()
				sessionID = session.GetID()
			}
		}

		// Log response
		if logger != nil {
			logEntry := logger.WithFields(logrus.Fields{
				"method":        method,
				"path":          path,
				"status_code":   statusCode,
				"duration_ms":   duration.Milliseconds(),
				"client_ip":     clientIP,
				"request_id":    requestID,
				"response_size": writer.body.Len(),
			})

			// Add user info if available
			if userID != "" {
				logEntry = logEntry.WithFields(logrus.Fields{
					"user_id":    userID,
					"username":   username,
					"session_id": sessionID,
				})
			}

			// Log at appropriate level based on status code
			if statusCode >= 500 {
				logEntry.Error("Request completed with server error")
			} else if statusCode >= 400 {
				logEntry.Warn("Request completed with client error")
			} else {
				logEntry.Info("Request completed successfully")
			}
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID already exists
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			// Generate new request ID
			requestID = generateRequestID()
		}

		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

// generateRequestID generates a unique request ID
func generateRequestID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(result)
}

// DetailedProxyLoggingMiddleware creates very detailed logging for debugging
func DetailedProxyLoggingMiddleware(logger *logrus.Logger, logRequestBody, logResponseBody bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Read request body if logging is enabled
		var requestBody []byte
		if logRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			// Restore body for downstream handlers
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// Log detailed request
		if logger != nil {
			logEntry := logger.WithFields(logrus.Fields{
				"method":         c.Request.Method,
				"path":           c.Request.URL.Path,
				"query":          c.Request.URL.RawQuery,
				"client_ip":      c.ClientIP(),
				"user_agent":     c.Request.UserAgent(),
				"content_length": c.Request.ContentLength,
				"headers":        c.Request.Header,
			})

			if logRequestBody && len(requestBody) > 0 {
				logEntry = logEntry.WithField("request_body", string(requestBody))
			}

			logEntry.Debug("Detailed request")
		}

		// Capture response
		writer := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer([]byte{}),
		}
		c.Writer = writer

		// Process request
		c.Next()

		// Log detailed response
		duration := time.Since(startTime)
		if logger != nil {
			logEntry := logger.WithFields(logrus.Fields{
				"method":           c.Request.Method,
				"path":             c.Request.URL.Path,
				"status_code":      writer.Status(),
				"duration_ms":      duration.Milliseconds(),
				"response_size":    writer.body.Len(),
				"response_headers": writer.Header(),
			})

			if logResponseBody && writer.body.Len() > 0 {
				// Only log first 1000 chars to avoid huge logs
				responseBody := writer.body.String()
				if len(responseBody) > 1000 {
					responseBody = responseBody[:1000] + "... (truncated)"
				}
				logEntry = logEntry.WithField("response_body", responseBody)
			}

			logEntry.Debug("Detailed response")
		}
	}
}

// PerformanceLoggingMiddleware logs performance metrics
func PerformanceLoggingMiddleware(logger *logrus.Logger, slowThreshold time.Duration) gin.HandlerFunc {
	if slowThreshold == 0 {
		slowThreshold = 1 * time.Second
	}

	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Log if request was slow
		if duration > slowThreshold {
			if logger != nil {
				logger.WithFields(logrus.Fields{
					"method":       c.Request.Method,
					"path":         c.Request.URL.Path,
					"duration_ms":  duration.Milliseconds(),
					"threshold_ms": slowThreshold.Milliseconds(),
					"status_code":  c.Writer.Status(),
				}).Warn("Slow request detected")
			}
		}
	}
}

// ErrorLoggingMiddleware logs errors with full context
func ErrorLoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Process request
		c.Next()

		// Check for errors
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				if logger != nil {
					logger.WithFields(logrus.Fields{
						"method":      c.Request.Method,
						"path":        c.Request.URL.Path,
						"client_ip":   c.ClientIP(),
						"error_type":  err.Type,
						"error":       err.Error(),
						"status_code": c.Writer.Status(),
					}).Error("Request error")
				}
			}
		}
	}
}

// MetricsLoggingMiddleware logs metrics in a structured format
func MetricsLoggingMiddleware(logger *logrus.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		// Calculate metrics
		duration := time.Since(startTime)
		statusCode := c.Writer.Status()

		// Log metrics
		if logger != nil {
			logger.WithFields(logrus.Fields{
				"metric":         "http_request",
				"method":         c.Request.Method,
				"path":           c.Request.URL.Path,
				"status_code":    statusCode,
				"duration_ms":    duration.Milliseconds(),
				"content_length": c.Request.ContentLength,
				"timestamp":      startTime.Unix(),
			}).Info("Request metrics")
		}
	}
}
