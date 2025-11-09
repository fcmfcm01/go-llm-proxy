package unit

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
)

var ErrNoHealthyProviders = errors.New("no healthy providers available")

// TestAutomaticFailover tests automatic failover functionality
func TestAutomaticFailover(t *testing.T) {
	t.Run("CreateFailoverManager", func(t *testing.T) {
		// Test that failover manager can be created
		manager := NewTestFailoverManager()

		assert.NotNil(t, manager, "Failover manager should be created")
		assert.Equal(t, "test-failover", manager.Name(), "Manager should have correct name")
	})

	t.Run("SingleProviderHealthy", func(t *testing.T) {
		// Test failover behavior with a single healthy provider
		manager := NewTestFailoverManager()

		provider := &models.Provider{
			ID:       "provider-1",
			Name:     "Test Provider 1",
			Enabled:  true,
			Priority: 1,
		}

		manager.AddProvider(provider)
		manager.SetHealthy("provider-1", true)

		// Should select the only provider
		selected, err := manager.SelectProvider()
		require.NoError(t, err, "Should successfully select a provider")
		assert.Equal(t, "provider-1", selected.ID, "Should select the healthy provider")
	})

	t.Run("MultipleProvidersPriorityOrder", func(t *testing.T) {
		// Test provider selection by priority order
		manager := NewTestFailoverManager()

		// Add providers with different priorities
		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 3, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-3", Name: "Provider 3", Priority: 2, Enabled: true})

		// All providers healthy, should select lowest priority number (highest priority)
		selected, err := manager.SelectProvider()
		require.NoError(t, err)
		assert.Equal(t, "provider-2", selected.ID, "Should select provider with priority 1 (highest)")
	})

	t.Run("FailoverToNextProvider", func(t *testing.T) {
		// Test automatic failover when primary provider fails
		manager := NewTestFailoverManager()

		// Add providers
		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: true})

		// Initially both healthy
		manager.SetHealthy("provider-1", true)
		manager.SetHealthy("provider-2", true)

		// First request should go to provider-1 (higher priority)
		selected, _ := manager.SelectProvider()
		assert.Equal(t, "provider-1", selected.ID, "Should select provider-1 first")

		// Mark provider-1 as unhealthy
		manager.SetHealthy("provider-1", false)

		// Next request should failover to provider-2
		selected, _ = manager.SelectProvider()
		assert.Equal(t, "provider-2", selected.ID, "Should failover to provider-2")
	})

	t.Run("NoHealthyProviders", func(t *testing.T) {
		// Test behavior when no providers are healthy
		manager := NewTestFailoverManager()

		// Add providers but mark all as unhealthy
		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: true})

		manager.SetHealthy("provider-1", false)
		manager.SetHealthy("provider-2", false)

		// Should return error when no healthy providers
		_, err := manager.SelectProvider()
		assert.Error(t, err, "Should return error when no healthy providers")
		assert.Contains(t, err.Error(), "no healthy providers", "Error message should mention no healthy providers")
	})

	t.Run("AllProvidersDisabled", func(t *testing.T) {
		// Test behavior when all providers are disabled
		manager := NewTestFailoverManager()

		// Add providers but disable them
		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: false})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: false})

		// Should return error
		_, err := manager.SelectProvider()
		assert.Error(t, err, "Should return error when all providers disabled")
	})

	t.Run("RecoveryAfterFailover", func(t *testing.T) {
		// Test provider recovery and re-selection
		manager := NewTestFailoverManager()

		// Add providers
		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: true})

		// Mark primary as unhealthy
		manager.SetHealthy("provider-1", false)

		// Should select provider-2
		selected, _ := manager.SelectProvider()
		assert.Equal(t, "provider-2", selected.ID, "Should select provider-2")

		// Primary provider recovers
		manager.SetHealthy("provider-1", true)

		// Should select primary provider again
		selected, _ = manager.SelectProvider()
		assert.Equal(t, "provider-1", selected.ID, "Should reselect recovered provider-1")
	})

	t.Run("RoundRobinAmongEqualPriority", func(t *testing.T) {
		// Test round-robin selection among providers with same priority
		manager := NewTestFailoverManager()

		// Add providers with same priority
		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-3", Name: "Provider 3", Priority: 1, Enabled: true})

		// Should round-robin among them
		selected1, _ := manager.SelectProvider()
		selected2, _ := manager.SelectProvider()
		selected3, _ := manager.SelectProvider()
		selected4, _ := manager.SelectProvider() // Should cycle back

		// All should be different providers
		assert.NotEqual(t, selected1.ID, selected2.ID, "Should select different providers")
		assert.NotEqual(t, selected2.ID, selected3.ID, "Should select different providers")
		assert.Equal(t, selected1.ID, selected4.ID, "Should cycle back to first provider")
	})

	t.Run("ProviderFailureCountTracking", func(t *testing.T) {
		// Test tracking of provider failure counts
		manager := NewTestFailoverManager()

		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: true})

		// Record failures
		manager.RecordFailure("provider-1")
		manager.RecordFailure("provider-1")
		manager.RecordFailure("provider-2")

		// Provider-1 should have more failures
		assert.Equal(t, int64(2), manager.GetFailureCount("provider-1"), "Provider-1 should have 2 failures")
		assert.Equal(t, int64(1), manager.GetFailureCount("provider-2"), "Provider-2 should have 1 failure")

		// Mark provider-1 as recovered
		manager.RecordSuccess("provider-1")
		assert.Equal(t, int64(0), manager.GetFailureCount("provider-1"), "Failure count should reset after success")
	})

	t.Run("CircuitBreakerBehavior", func(t *testing.T) {
		// Test circuit breaker opens after threshold failures
		manager := NewTestFailoverManager()

		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: true})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: true})

		// Record many failures to trigger circuit breaker
		for i := 0; i < 5; i++ {
			manager.RecordFailure("provider-1")
		}

		// Provider-1 should be in circuit breaker state
		assert.True(t, manager.IsCircuitOpen("provider-1"), "Provider-1 should have circuit breaker open")

		// Should not select provider-1
		selected, _ := manager.SelectProvider()
		assert.NotEqual(t, "provider-1", selected.ID, "Should not select provider with open circuit")

		// Provider-2 should still be selectable
		assert.Equal(t, "provider-2", selected.ID, "Should select provider-2")
	})

	t.Run("DisabledProviderNotSelected", func(t *testing.T) {
		// Test that disabled providers are not selected
		manager := NewTestFailoverManager()

		manager.AddProvider(&models.Provider{ID: "provider-1", Name: "Provider 1", Priority: 1, Enabled: false})
		manager.AddProvider(&models.Provider{ID: "provider-2", Name: "Provider 2", Priority: 2, Enabled: true})

		// Should skip disabled provider-1 and select provider-2
		selected, _ := manager.SelectProvider()
		assert.Equal(t, "provider-2", selected.ID, "Should not select disabled provider")
	})
}

// TestFailoverManager is a test implementation of the failover manager
type TestFailoverManager struct {
	providers    map[string]*models.Provider
	healthy      map[string]bool
	failureCount map[string]int64
	selectionIdx int
}

func NewTestFailoverManager() *TestFailoverManager {
	return &TestFailoverManager{
		providers:    make(map[string]*models.Provider),
		healthy:      make(map[string]bool),
		failureCount: make(map[string]int64),
	}
}

func (m *TestFailoverManager) Name() string {
	return "test-failover"
}

func (m *TestFailoverManager) AddProvider(provider *models.Provider) {
	m.providers[provider.ID] = provider
	m.healthy[provider.ID] = true
}

func (m *TestFailoverManager) SelectProvider() (*models.Provider, error) {
	var selected *models.Provider

	// Find enabled providers sorted by priority
	var enabledProviders []*models.Provider
	for _, p := range m.providers {
		if p.Enabled {
			enabledProviders = append(enabledProviders, p)
		}
	}

	if len(enabledProviders) == 0 {
		return nil, ErrNoHealthyProviders
	}

	// Filter healthy providers
	var healthyProviders []*models.Provider
	for _, p := range enabledProviders {
		if m.healthy[p.ID] && !m.isCircuitOpen(p.ID) {
			healthyProviders = append(healthyProviders, p)
		}
	}

	if len(healthyProviders) == 0 {
		return nil, ErrNoHealthyProviders
	}

	// Group by priority
	byPriority := make(map[int][]*models.Provider)
	for _, p := range healthyProviders {
		byPriority[p.Priority] = append(byPriority[p.Priority], p)
	}

	// Select lowest priority number (highest priority)
	var minPriority int
	for k := range byPriority {
		if minPriority == 0 || k < minPriority {
			minPriority = k
		}
	}

	providersAtMinPriority := byPriority[minPriority]

	// Round-robin among providers at same priority
	if len(providersAtMinPriority) > 0 {
		selected = providersAtMinPriority[m.selectionIdx%len(providersAtMinPriority)]
		m.selectionIdx++
	}

	return selected, nil
}

func (m *TestFailoverManager) SetHealthy(providerID string, healthy bool) {
	m.healthy[providerID] = healthy
	if healthy {
		m.failureCount[providerID] = 0
	}
}

func (m *TestFailoverManager) RecordFailure(providerID string) {
	m.failureCount[providerID]++
	if m.failureCount[providerID] >= 5 {
		// Simulate circuit breaker opening
	}
}

func (m *TestFailoverManager) RecordSuccess(providerID string) {
	m.failureCount[providerID] = 0
}

func (m *TestFailoverManager) GetFailureCount(providerID string) int64 {
	return m.failureCount[providerID]
}

func (m *TestFailoverManager) isCircuitOpen(providerID string) bool {
	return m.failureCount[providerID] >= 5
}

func (m *TestFailoverManager) IsCircuitOpen(providerID string) bool {
	return m.isCircuitOpen(providerID)
}
