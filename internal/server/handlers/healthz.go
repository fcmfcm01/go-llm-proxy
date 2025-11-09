package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// HealthzHandler handles basic health check requests
type HealthzHandler struct {
	logger *logrus.Logger
	startTime time.Time
}

// NewHealthzHandler creates a new health check handler
func NewHealthzHandler(logger *logrus.Logger) *HealthzHandler {
	return &HealthzHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// HandleHealthz handles GET /healthz
func (h *HealthzHandler) HandleHealthz(c *gin.Context) {
	// Basic health check - just verify the server is running
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"uptime":  h.getUptime(),
		"version": getVersion(),
		"runtime": gin.Mode(),
	})
}

// HandleLive handles GET /healthz/live (liveness probe)
func (h *HealthzHandler) HandleLive(c *gin.Context) {
	// Liveness probe - simple check that the process is running
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// getUptime returns the uptime duration
func (h *HealthzHandler) getUptime() string {
	uptime := time.Since(h.startTime)
	// Return formatted uptime string
	return time.Duration(uptime).String()
}

// getVersion returns the application version
func getVersion() string {
	// In a real implementation, this would come from build info
	return "1.0.0"
}
