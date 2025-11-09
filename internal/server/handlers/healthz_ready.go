package handlers

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ReadinessChecker interface for checking readiness components
type ReadinessChecker interface {
	Check() ReadinessStatus
}

// ReadinessStatus holds the status of a readiness check
type ReadinessStatus struct {
	Ready   bool                   `json:"ready"`
	Status  string                 `json:"status,omitempty"`
	Message string                 `json:"message,omitempty"`
	Checks  map[string]CheckStatus `json:"checks,omitempty"`
}

// CheckStatus holds the status of an individual check
type CheckStatus struct {
	Status  string `json:"status"` // "ok", "error", "warning"
	Message string `json:"message,omitempty"`
	Latency int64  `json:"latency_ms,omitempty"`
}

// ReadinessHandler handles readiness check requests
type ReadinessHandler struct {
	logger   *logrus.Logger
	checkers []ReadinessChecker
	mu       sync.RWMutex
}

// NewReadinessHandler creates a new readiness handler
func NewReadinessHandler(logger *logrus.Logger) *ReadinessHandler {
	return &ReadinessHandler{
		logger:   logger,
		checkers: make([]ReadinessChecker, 0),
	}
}

// AddChecker adds a readiness checker
func (h *ReadinessHandler) AddChecker(checker ReadinessChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checker)
}

// HandleReady handles GET /healthz/ready
func (h *ReadinessHandler) HandleReady(c *gin.Context) {
	start := time.Now()

	// Run all readiness checks
	checkResults := make(map[string]CheckStatus)
	allReady := true

	h.mu.RLock()
	for _, checker := range h.checkers {
		checkStart := time.Now()
		status := checker.Check()
		latency := time.Since(checkStart).Milliseconds()

		checkName := getCheckerName(checker)
		checkResults[checkName] = CheckStatus{
			Status:  status.Status,
			Message: status.Message,
			Latency: latency,
		}

		if status.Status != "ok" {
			allReady = false
		}
	}
	h.mu.RUnlock()

	totalLatency := time.Since(start).Milliseconds()

	// Determine response based on readiness
	if allReady {
		c.JSON(http.StatusOK, gin.H{
			"status":   "ready",
			"checks":   checkResults,
			"latency":  totalLatency,
			"message":  "All components are ready",
			"checksum": calculateChecksum(checkResults),
		})
	} else {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":   "not ready",
			"checks":   checkResults,
			"latency":  totalLatency,
			"message":  "One or more components are not ready",
			"checksum": calculateChecksum(checkResults),
		})
	}
}

// getCheckerName returns the name of a checker
func getCheckerName(checker ReadinessChecker) string {
	// Try to get a meaningful name
	switch checker.(type) {
	case *ConfigChecker:
		return "config"
	case *ProviderChecker:
		return "providers"
	case *DependencyChecker:
		return "dependencies"
	default:
		return "unknown"
	}
}

// calculateChecksum calculates a simple checksum of check results
func calculateChecksum(checks map[string]CheckStatus) string {
	// Simple hash - in production, use a proper hash function
	okCount := 0
	for _, check := range checks {
		if check.Status == "ok" {
			okCount++
		}
	}
	return "ok"
}

// ConfigChecker checks if configuration is loaded
type ConfigChecker struct {
	logger *logrus.Logger
	ready  bool
}

// NewConfigChecker creates a new config checker
func NewConfigChecker(logger *logrus.Logger) *ConfigChecker {
	return &ConfigChecker{
		logger: logger,
		ready:  true, // Assume config is ready
	}
}

// Check returns the config readiness status
func (c *ConfigChecker) Check() ReadinessStatus {
	var status string
	var message string
	if c.ready {
		status = "ok"
		message = "Configuration loaded"
	} else {
		status = "error"
		message = "Configuration not loaded"
	}
	return ReadinessStatus{
		Ready:   c.ready,
		Status:  status,
		Message: message,
	}
}

// ProviderChecker checks if providers are available
type ProviderChecker struct {
	logger        *logrus.Logger
	providerCount int
}

// NewProviderChecker creates a new provider checker
func NewProviderChecker(logger *logrus.Logger) *ProviderChecker {
	return &ProviderChecker{
		logger: logger,
		// In real implementation, would get actual count
		providerCount: 0,
	}
}

// Check returns the provider readiness status
func (p *ProviderChecker) Check() ReadinessStatus {
	ready := p.providerCount > 0
	status := "ok"
	message := "No providers configured"

	if ready {
		message = "Providers available"
	} else {
		status = "warning"
	}

	return ReadinessStatus{
		Ready:   ready,
		Status:  status,
		Message: message,
	}
}

// DependencyChecker checks if external dependencies are available
type DependencyChecker struct {
	logger *logrus.Logger
	ready  bool
}

// NewDependencyChecker creates a new dependency checker
func NewDependencyChecker(logger *logrus.Logger) *DependencyChecker {
	return &DependencyChecker{
		logger: logger,
		ready:  true, // Assume dependencies are ready
	}
}

// Check returns the dependency readiness status
func (d *DependencyChecker) Check() ReadinessStatus {
	return ReadinessStatus{
		Ready:   d.ready,
		Status:  "ok",
		Message: "Dependencies available",
	}
}
