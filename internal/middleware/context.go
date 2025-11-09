package middleware

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey contextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey contextKey = "user_id"
	// UsernameKey is the context key for username
	UsernameKey contextKey = "username"
)

// RequestID middleware adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(string(RequestIDKey), requestID)
		c.Header("X-Request-ID", requestID)

		// Add to context
		ctx := context.WithValue(c.Request.Context(), RequestIDKey, requestID)
		c.Request = c.Request.WithContext(ctx)

		c.Next()
	}
}

// GetRequestID retrieves the request ID from context
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUserIDFromContext retrieves the user ID from context
func GetUserIDFromContext(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// GetUsername retrieves the username from context
func GetUsername(ctx context.Context) string {
	if username, ok := ctx.Value(UsernameKey).(string); ok {
		return username
	}
	return ""
}

// SetUserContext sets user information in the context
func SetUserContext(c *gin.Context, userID, username string) {
	c.Set(string(UserIDKey), userID)
	c.Set(string(UsernameKey), username)

	ctx := c.Request.Context()
	ctx = context.WithValue(ctx, UserIDKey, userID)
	ctx = context.WithValue(ctx, UsernameKey, username)
	c.Request = c.Request.WithContext(ctx)
}
