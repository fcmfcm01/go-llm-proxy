package proxy

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/example/go-llm-proxy/internal/models"
)

// HealthChecker performs health checks on providers
type HealthChecker struct {
	httpClient    *http.Client
	checkInterval time.Duration
	cache         map[string]*healthStatus
	mu            sync.RWMutex
}

// healthStatus stores the health status of a provider
type healthStatus struct {
	healthy    bool
	lastCheck  time.Time
	lastError  error
	checkCount int
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(checkInterval time.Duration) *HealthChecker {
	return &HealthChecker{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		checkInterval: checkInterval,
		cache:         make(map[string]*healthStatus),
	}
}

// CheckHealth checks if a provider is healthy
func (h *HealthChecker) CheckHealth(ctx context.Context, provider *models.Provider) bool {
	h.mu.RLock()
	status, exists := h.cache[provider.ID]
	h.mu.RUnlock()

	// Use cached result if recent
	if exists && time.Since(status.lastCheck) < h.checkInterval {
		return status.healthy
	}

	// Perform actual health check
	healthy := h.performHealthCheck(ctx, provider)

	// Update cache
	h.mu.Lock()
	h.cache[provider.ID] = &healthStatus{
		healthy:    healthy,
		lastCheck:  time.Now(),
		checkCount: status.checkCount + 1,
	}
	h.mu.Unlock()

	return healthy
}

// performHealthCheck performs the actual HTTP health check
func (h *HealthChecker) performHealthCheck(ctx context.Context, provider *models.Provider) bool {
	// Use provider timeout if specified, otherwise use default
	timeout := 5 * time.Second
	if provider.Timeout > 0 {
		timeout = provider.Timeout
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Construct health check URL
	healthURL := provider.APIURL
	if healthURL == "" {
		return false
	}

	// Add health check endpoint if needed
	// Most APIs respond to their base URL
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, healthURL, nil)
	if err != nil {
		h.updateError(provider.ID, err)
		return false
	}

	// Send request
	resp, err := h.httpClient.Do(req)
	if err != nil {
		h.updateError(provider.ID, err)
		return false
	}
	defer resp.Body.Close()

	// Consider 2xx and 401/403 as healthy (auth errors mean service is up)
	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300 ||
		resp.StatusCode == http.StatusUnauthorized ||
		resp.StatusCode == http.StatusForbidden

	if !healthy {
		h.updateError(provider.ID, fmt.Errorf("unhealthy status code: %d", resp.StatusCode))
	}

	return healthy
}

// GetStatus returns the current health status of a provider
func (h *HealthChecker) GetStatus(providerID string) *ProviderHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status, exists := h.cache[providerID]
	if !exists {
		return &ProviderHealth{
			Healthy:   false,
			LastCheck: time.Time{},
			Message:   "not checked",
		}
	}

	health := &ProviderHealth{
		Healthy:    status.healthy,
		LastCheck:  status.lastCheck,
		CheckCount: status.checkCount,
	}

	if status.lastError != nil {
		health.Message = status.lastError.Error()
	} else {
		health.Message = "healthy"
	}

	return health
}

// GetAllStatus returns health status for all providers
func (h *HealthChecker) GetAllStatus() map[string]*ProviderHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*ProviderHealth)
	for id, status := range h.cache {
		health := &ProviderHealth{
			Healthy:    status.healthy,
			LastCheck:  status.lastCheck,
			CheckCount: status.checkCount,
		}

		if status.lastError != nil {
			health.Message = status.lastError.Error()
		} else {
			health.Message = "healthy"
		}

		result[id] = health
	}

	return result
}

// StartPeriodicChecks starts periodic health checks for providers
func (h *HealthChecker) StartPeriodicChecks(ctx context.Context, providers []*models.Provider) {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			h.checkAllProviders(ctx, providers)
		case <-ctx.Done():
			return
		}
	}
}

// checkAllProviders checks all providers
func (h *HealthChecker) checkAllProviders(ctx context.Context, providers []*models.Provider) {
	var wg sync.WaitGroup

	for _, provider := range providers {
		if !provider.Enabled {
			continue
		}

		wg.Add(1)
		go func(p *models.Provider) {
			defer wg.Done()
			h.CheckHealth(ctx, p)
		}(provider)
	}

	wg.Wait()
}

// updateError updates the error for a provider
func (h *HealthChecker) updateError(providerID string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if status, exists := h.cache[providerID]; exists {
		status.lastError = err
	}
}

// ClearCache clears the health check cache
func (h *HealthChecker) ClearCache() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cache = make(map[string]*healthStatus)
}

// ProviderHealth represents the health status of a provider
type ProviderHealth struct {
	Healthy    bool      `json:"healthy"`
	LastCheck  time.Time `json:"last_check"`
	CheckCount int       `json:"check_count"`
	Message    string    `json:"message"`
}
