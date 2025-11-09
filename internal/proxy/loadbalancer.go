package proxy

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/example/go-llm-proxy/internal/models"
)

// LoadBalancer defines the interface for load balancing
type LoadBalancer interface {
	SelectProvider(ctx context.Context) (*models.Provider, error)
	MarkProviderFailed(providerID string)
	MarkProviderHealthy(providerID string)
	GetProviders() []*models.Provider
}

// RoundRobinLoadBalancer implements round-robin load balancing
type RoundRobinLoadBalancer struct {
	providers       []*models.Provider
	currentIndex    uint32
	failedProviders map[string]bool
	mu              sync.RWMutex
}

// NewRoundRobinLoadBalancer creates a new round-robin load balancer
func NewRoundRobinLoadBalancer(providers []*models.Provider) *RoundRobinLoadBalancer {
	return &RoundRobinLoadBalancer{
		providers:       providers,
		currentIndex:    0,
		failedProviders: make(map[string]bool),
	}
}

// SelectProvider selects the next provider using round-robin
func (lb *RoundRobinLoadBalancer) SelectProvider(ctx context.Context) (*models.Provider, error) {
	lb.mu.RLock()
	enabledProviders := lb.getEnabledProviders()
	lb.mu.RUnlock()

	if len(enabledProviders) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Try to find a healthy provider
	maxAttempts := len(enabledProviders)
	for i := 0; i < maxAttempts; i++ {
		// Atomic increment and get
		idx := atomic.AddUint32(&lb.currentIndex, 1) - 1
		provider := enabledProviders[idx%uint32(len(enabledProviders))]

		// Check if provider is failed
		lb.mu.RLock()
		isFailed := lb.failedProviders[provider.ID]
		lb.mu.RUnlock()

		if !isFailed {
			return provider, nil
		}
	}

	// All providers are failed, return the next one anyway
	idx := atomic.AddUint32(&lb.currentIndex, 1) - 1
	return enabledProviders[idx%uint32(len(enabledProviders))], nil
}

// MarkProviderFailed marks a provider as failed
func (lb *RoundRobinLoadBalancer) MarkProviderFailed(providerID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.failedProviders[providerID] = true
}

// MarkProviderHealthy marks a provider as healthy
func (lb *RoundRobinLoadBalancer) MarkProviderHealthy(providerID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	delete(lb.failedProviders, providerID)
}

// GetProviders returns all providers
func (lb *RoundRobinLoadBalancer) GetProviders() []*models.Provider {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	providers := make([]*models.Provider, len(lb.providers))
	copy(providers, lb.providers)
	return providers
}

// UpdateProviders updates the provider list
func (lb *RoundRobinLoadBalancer) UpdateProviders(providers []*models.Provider) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.providers = providers
}

// getEnabledProviders returns only enabled providers
func (lb *RoundRobinLoadBalancer) getEnabledProviders() []*models.Provider {
	var enabled []*models.Provider
	for _, p := range lb.providers {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

// WeightedLoadBalancer implements weighted load balancing based on priority
type WeightedLoadBalancer struct {
	providers       []*models.Provider
	weights         map[string]int
	failedProviders map[string]bool
	mu              sync.RWMutex
}

// NewWeightedLoadBalancer creates a weighted load balancer
func NewWeightedLoadBalancer(providers []*models.Provider) *WeightedLoadBalancer {
	lb := &WeightedLoadBalancer{
		providers:       providers,
		weights:         make(map[string]int),
		failedProviders: make(map[string]bool),
	}

	// Calculate weights based on priority (lower priority = higher weight)
	for _, p := range providers {
		if p.Priority > 0 {
			// Invert priority: priority 1 gets highest weight
			lb.weights[p.ID] = 100 - p.Priority
		} else {
			lb.weights[p.ID] = 50 // default weight
		}
	}

	return lb
}

// SelectProvider selects a provider based on weights
func (lb *WeightedLoadBalancer) SelectProvider(ctx context.Context) (*models.Provider, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	enabledProviders := lb.getEnabledProviders()
	if len(enabledProviders) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	// Calculate total weight of healthy providers
	totalWeight := 0
	for _, p := range enabledProviders {
		if !lb.failedProviders[p.ID] {
			totalWeight += lb.weights[p.ID]
		}
	}

	if totalWeight == 0 {
		// All providers failed, return first enabled one
		return enabledProviders[0], nil
	}

	// Select based on weight (simplified - would use random selection in production)
	// For now, return highest weighted provider
	var selected *models.Provider
	maxWeight := 0
	for _, p := range enabledProviders {
		if !lb.failedProviders[p.ID] && lb.weights[p.ID] > maxWeight {
			maxWeight = lb.weights[p.ID]
			selected = p
		}
	}

	if selected == nil {
		return enabledProviders[0], nil
	}

	return selected, nil
}

// MarkProviderFailed marks a provider as failed
func (lb *WeightedLoadBalancer) MarkProviderFailed(providerID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	lb.failedProviders[providerID] = true
}

// MarkProviderHealthy marks a provider as healthy
func (lb *WeightedLoadBalancer) MarkProviderHealthy(providerID string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()
	delete(lb.failedProviders, providerID)
}

// GetProviders returns all providers
func (lb *WeightedLoadBalancer) GetProviders() []*models.Provider {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	providers := make([]*models.Provider, len(lb.providers))
	copy(providers, lb.providers)
	return providers
}

// getEnabledProviders returns only enabled providers
func (lb *WeightedLoadBalancer) getEnabledProviders() []*models.Provider {
	var enabled []*models.Provider
	for _, p := range lb.providers {
		if p.Enabled {
			enabled = append(enabled, p)
		}
	}
	return enabled
}
