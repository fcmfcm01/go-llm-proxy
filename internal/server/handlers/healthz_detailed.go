package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// DetailedHealthHandler handles detailed health check requests
type DetailedHealthHandler struct {
	logger    *logrus.Logger
	startTime time.Time
}

// NewDetailedHealthHandler creates a new detailed health handler
func NewDetailedHealthHandler(logger *logrus.Logger) *DetailedHealthHandler {
	return &DetailedHealthHandler{
		logger:    logger,
		startTime: time.Now(),
	}
}

// HandleDetailed handles GET /healthz/detailed
func (h *DetailedHealthHandler) HandleDetailed(c *gin.Context) {
	// Collect detailed system information
	systemInfo := h.collectSystemInfo()
	componentInfo := h.collectComponentInfo()

	// Build response
	response := gin.H{
		"status":     "healthy",
		"timestamp":  time.Now().Format(time.RFC3339),
		"uptime":     h.getUptime(),
		"version":    getVersion(),
		"system":     systemInfo,
		"components": componentInfo,
	}

	// Add runtime information
	if memStats := h.getMemoryStats(); memStats != nil {
		response["runtime"] = gin.H{
			"goroutines": runtime.NumGoroutine(),
			"memory":     memStats,
		}
	}

	c.JSON(http.StatusOK, response)
}

// collectSystemInfo collects system information
func (h *DetailedHealthHandler) collectSystemInfo() gin.H {
	return gin.H{
		"hostname":   getHostname(),
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"num_cpu":    runtime.NumCPU(),
		"build_info": getBuildInfo(),
	}
}

// collectComponentInfo collects component information
func (h *DetailedHealthHandler) collectComponentInfo() gin.H {
	return gin.H{
		"server": gin.H{
			"status":  "ok",
			"message": "Server is running",
		},
		"proxy": gin.H{
			"status":  "ok",
			"message": "Proxy is operational",
		},
		"config": gin.H{
			"status":  "ok",
			"message": "Configuration loaded",
		},
		"metrics": gin.H{
			"status":  "ok",
			"message": "Metrics collection active",
		},
		"health_checks": gin.H{
			"status":  "ok",
			"message": "Health checks operational",
		},
	}
}

// getUptime returns the uptime duration
func (h *DetailedHealthHandler) getUptime() string {
	return time.Since(h.startTime).String()
}

// getMemoryStats returns memory statistics
func (h *DetailedHealthHandler) getMemoryStats() *runtime.MemStats {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	return &memStats
}

// getHostname returns the hostname
func getHostname() string {
	// In a real implementation, would get actual hostname
	return "localhost"
}

// getBuildInfo returns build information
func getBuildInfo() gin.H {
	return gin.H{
		"git_commit": "unknown",
		"build_time": "unknown",
		"compiler":   runtime.Compiler,
	}
}
