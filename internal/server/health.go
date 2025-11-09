package server

import (
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthChecker provides health check functionality
type HealthChecker struct {
	mu           sync.RWMutex
	startTime    time.Time
	checks       map[string]HealthCheck
	dependencies map[string]Dependency
}

// HealthCheck represents a health check
type HealthCheck struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"` // healthy, degraded, unhealthy
	Duration time.Duration `json:"duration"`
	Error    string        `json:"error,omitempty"`
}

// Dependency represents an external dependency
type Dependency struct {
	Name    string        `json:"name"`
	URL     string        `json:"url"`
	Timeout time.Duration `json:"timeout"`
}

// DetailedHealth represents detailed health information
type DetailedHealth struct {
	Status       string                `json:"status"`
	Timestamp    time.Time             `json:"timestamp"`
	Uptime       string                `json:"uptime"`
	Version      string                `json:"version"`
	BuildInfo    BuildInfo             `json:"build_info"`
	Checks       []HealthCheck         `json:"checks"`
	Dependencies map[string]Dependency `json:"dependencies"`
	Memory       MemoryStats           `json:"memory"`
	CPU          CPUStats              `json:"cpu"`
	Goroutines   int                   `json:"goroutines"`
	GC           GCStats               `json:"gc"`
}

// BuildInfo represents build information
type BuildInfo struct {
	GoVersion string `json:"go_version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

// MemoryStats represents memory statistics
type MemoryStats struct {
	Alloc      uint64 `json:"alloc"`
	TotalAlloc uint64 `json:"total_alloc"`
	Sys        uint64 `json:"sys"`
	NumGC      uint32 `json:"num_gc"`
}

// CPUStats represents CPU statistics
type CPUStats struct {
	NumCPU       int `json:"num_cpu"`
	NumGoroutine int `json:"num_goroutine"`
}

// GCStats represents garbage collection statistics
type GCStats struct {
	LastGC     time.Duration `json:"last_gc"`
	PauseTotal time.Duration `json:"pause_total"`
	NumGC      uint32        `json:"num_gc"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		startTime:    time.Now(),
		checks:       make(map[string]HealthCheck),
		dependencies: make(map[string]Dependency),
	}
}

// setupHealthRoutes sets up health check routes
func (s *Server) setupHealthRoutes() {
	health := NewHealthChecker()

	// Basic health check
	s.router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"timestamp": time.Now(),
		})
	})

	// Ready check
	s.router.GET("/healthz/ready", func(c *gin.Context) {
		// Check if service is ready to receive traffic
		// This could check dependencies, database connections, etc.
		c.JSON(http.StatusOK, gin.H{
			"status":    "ready",
			"timestamp": time.Now(),
		})
	})

	// Live check
	s.router.GET("/healthz/live", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "alive",
			"timestamp": time.Now(),
		})
	})

	// Detailed health check
	s.router.GET("/healthz/detailed", func(c *gin.Context) {
		detailed := health.GetDetailedHealth()
		c.JSON(http.StatusOK, detailed)
	})
}

// GetDetailedHealth returns detailed health information
func (h *HealthChecker) GetDetailedHealth() DetailedHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	uptime := time.Since(h.startTime)

	// Get GC statistics
	var gcPauseTotal uint64
	var lastGC uint64
	var numGC uint32
	if m.NumGC > 0 {
		lastGC = m.LastGC
		gcPauseTotal = m.PauseTotal
		numGC = m.NumGC
	}

	return DetailedHealth{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    uptime.String(),
		Version:   "1.0.0",
		BuildInfo: BuildInfo{
			GoVersion: runtime.Version(),
			GitCommit: "unknown",
			BuildTime: "unknown",
		},
		Checks:       h.getChecks(),
		Dependencies: h.dependencies,
		Memory: MemoryStats{
			Alloc:      m.Alloc,
			TotalAlloc: m.TotalAlloc,
			Sys:        m.Sys,
			NumGC:      m.NumGC,
		},
		CPU: CPUStats{
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
		},
		GC: GCStats{
			LastGC:     time.Duration(lastGC),
			PauseTotal: time.Duration(gcPauseTotal),
			NumGC:      numGC,
		},
	}
}

// getChecks returns all health checks
func (h *HealthChecker) getChecks() []HealthCheck {
	checks := make([]HealthCheck, 0, len(h.checks))
	for _, check := range h.checks {
		checks = append(checks, check)
	}
	return checks
}
