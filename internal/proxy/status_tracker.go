package proxy

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HealthStatus represents the health status of a provider
type HealthStatus int

const (
	StatusHealthy HealthStatus = iota
	StatusUnhealthy
	StatusUnknown
)

func (s HealthStatus) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	case StatusUnknown:
		return "unknown"
	default:
		return "unknown"
	}
}

// ProviderStatus holds the real-time status information for a provider
type ProviderStatus struct {
	ProviderID        string        `json:"provider_id"`
	Status            HealthStatus  `json:"status"`
	LastCheck         time.Time     `json:"last_check"`
	ResponseTime      int64         `json:"response_time_ms"`
	ConsecutiveHits   int64         `json:"consecutive_hits"`
	ConsecutiveMisses int64         `json:"consecutive_misses"`
	TotalChecks       int64         `json:"total_checks"`
	SuccessCount      int64         `json:"success_count"`
	FailureCount      int64         `json:"failure_count"`
	Uptime            time.Duration `json:"uptime"`
	FirstCheck        time.Time     `json:"first_check"`
}

// StatusTracker tracks real-time health status of all providers
type StatusTracker struct {
	logger *logrus.Logger
	mu     sync.RWMutex

	// Provider status tracking
	providers map[string]*ProviderStatus

	// Configuration
	checkInterval    time.Duration
	timeout          time.Duration
	failureThreshold int64
	successThreshold int64
}

// NewStatusTracker creates a new status tracker
func NewStatusTracker(
	logger *logrus.Logger,
	checkInterval time.Duration,
	timeout time.Duration,
	failureThreshold int64,
	successThreshold int64,
) *StatusTracker {
	t := &StatusTracker{
		logger:           logger,
		providers:        make(map[string]*ProviderStatus),
		checkInterval:    checkInterval,
		timeout:          timeout,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
	}

	return t
}

// RegisterProvider registers a new provider for monitoring
func (t *StatusTracker) RegisterProvider(providerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.providers[providerID]; !ok {
		t.providers[providerID] = &ProviderStatus{
			ProviderID: providerID,
			Status:     StatusUnknown,
			FirstCheck: time.Now(),
		}
		t.logger.WithField("providerID", providerID).Info("Provider registered for monitoring")
	}
}

// UnregisterProvider removes a provider from monitoring
func (t *StatusTracker) UnregisterProvider(providerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.providers[providerID]; ok {
		delete(t.providers, providerID)
		t.logger.WithField("providerID", providerID).Info("Provider unregistered from monitoring")
	}
}

// RecordSuccess records a successful request
func (t *StatusTracker) RecordSuccess(providerID string, responseTime time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()

	status, ok := t.providers[providerID]
	if !ok {
		// Auto-register if not found
		t.RegisterProvider(providerID)
		status = t.providers[providerID]
	}

	now := time.Now()
	status.LastCheck = now
	status.ResponseTime = responseTime.Milliseconds()
	status.TotalChecks++
	status.SuccessCount++
	status.ConsecutiveHits++
	status.ConsecutiveMisses = 0

	// Determine status based on consecutive hits
	if status.ConsecutiveHits >= t.successThreshold {
		if status.Status != StatusHealthy {
			t.logger.WithField("providerID", providerID).Info("Provider marked as healthy")
		}
		status.Status = StatusHealthy
	}

	// Update uptime
	status.Uptime = now.Sub(status.FirstCheck)
}

// RecordFailure records a failed request
func (t *StatusTracker) RecordFailure(providerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	status, ok := t.providers[providerID]
	if !ok {
		// Auto-register if not found
		t.RegisterProvider(providerID)
		status = t.providers[providerID]
	}

	now := time.Now()
	status.LastCheck = now
	status.TotalChecks++
	status.FailureCount++
	status.ConsecutiveMisses++
	status.ConsecutiveHits = 0

	// Determine status based on consecutive misses
	if status.ConsecutiveMisses >= t.failureThreshold {
		if status.Status != StatusUnhealthy {
			t.logger.WithField("providerID", providerID).Warn("Provider marked as unhealthy")
		}
		status.Status = StatusUnhealthy
	}
}

// GetStatus returns the current status of a provider
func (t *StatusTracker) GetStatus(providerID string) *ProviderStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if status, ok := t.providers[providerID]; ok {
		return status
	}

	return nil
}

// GetAllStatuses returns status for all providers
func (t *StatusTracker) GetAllStatuses() map[string]*ProviderStatus {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]*ProviderStatus)
	for k, v := range t.providers {
		result[k] = v
	}

	return result
}

// GetHealthyProviders returns IDs of healthy providers
func (t *StatusTracker) GetHealthyProviders() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var healthy []string
	for providerID, status := range t.providers {
		if status.Status == StatusHealthy {
			healthy = append(healthy, providerID)
		}
	}

	return healthy
}

// GetUnhealthyProviders returns IDs of unhealthy providers
func (t *StatusTracker) GetUnhealthyProviders() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var unhealthy []string
	for providerID, status := range t.providers {
		if status.Status == StatusUnhealthy {
			unhealthy = append(unhealthy, providerID)
		}
	}

	return unhealthy
}

// IsHealthy checks if a provider is healthy
func (t *StatusTracker) IsHealthy(providerID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if status, ok := t.providers[providerID]; ok {
		return status.Status == StatusHealthy
	}

	return false
}

// ForceStatus forces a provider's status (for manual control)
func (t *StatusTracker) ForceStatus(providerID string, status HealthStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, ok := t.providers[providerID]; !ok {
		t.RegisterProvider(providerID)
	}

	t.providers[providerID].Status = status
	t.providers[providerID].LastCheck = time.Now()

	t.logger.WithField("providerID", providerID).WithField("status", status.String()).
		Info("Provider status manually set")
}

// GetStats returns statistics about all providers
func (t *StatusTracker) GetStats() ProviderStats {
	t.mu.RLock()
	defer t.mu.RUnlock()

	stats := ProviderStats{
		TotalProviders: len(t.providers),
		HealthyCount:   0,
		UnhealthyCount: 0,
		UnknownCount:   0,
	}

	for _, status := range t.providers {
		switch status.Status {
		case StatusHealthy:
			stats.HealthyCount++
		case StatusUnhealthy:
			stats.UnhealthyCount++
		case StatusUnknown:
			stats.UnknownCount++
		}
	}

	return stats
}

// ProviderStats holds aggregated statistics
type ProviderStats struct {
	TotalProviders int `json:"total_providers"`
	HealthyCount   int `json:"healthy_providers"`
	UnhealthyCount int `json:"unhealthy_providers"`
	UnknownCount   int `json:"unknown_providers"`
}

// StartPeriodicCheck starts periodic health checks (optional background task)
func (t *StatusTracker) StartPeriodicCheck(checkFunc func(providerID string) (bool, time.Duration)) {
	go func() {
		ticker := time.NewTicker(t.checkInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				t.mu.RLock()
				providerIDs := make([]string, 0, len(t.providers))
				for providerID := range t.providers {
					providerIDs = append(providerIDs, providerID)
				}
				t.mu.RUnlock()

				for _, providerID := range providerIDs {
					healthy, responseTime := checkFunc(providerID)
					if healthy {
						t.RecordSuccess(providerID, responseTime)
					} else {
						t.RecordFailure(providerID)
					}
				}
			}
		}
	}()
}

// Reset resets status for a specific provider
func (t *StatusTracker) Reset(providerID string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if status, ok := t.providers[providerID]; ok {
		status.Status = StatusUnknown
		status.ConsecutiveHits = 0
		status.ConsecutiveMisses = 0
		status.SuccessCount = 0
		status.FailureCount = 0
		status.TotalChecks = 0
		status.FirstCheck = time.Now()
		status.Uptime = 0

		t.logger.WithField("providerID", providerID).Info("Provider status reset")
	}
}
