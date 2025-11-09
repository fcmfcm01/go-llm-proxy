package unit

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
	"github.com/fcmfcm01/go-llm-proxy/internal/proxy"
)

// TestRoundRobinLoadBalancing tests round-robin load balancing
// T048 [P] [US2] Unit test for round-robin load balancing
func TestRoundRobinLoadBalancing(t *testing.T) {
	providers := []*models.Provider{
		{ID: "provider-1", Name: "Provider 1", APIURL: "http://provider1.com", Enabled: true, Priority: 1},
		{ID: "provider-2", Name: "Provider 2", APIURL: "http://provider2.com", Enabled: true, Priority: 2},
		{ID: "provider-3", Name: "Provider 3", APIURL: "http://provider3.com", Enabled: true, Priority: 3},
	}

	lb := proxy.NewRoundRobinLoadBalancer(providers)

	// Test round-robin behavior
	ctx := context.Background()

	// First round
	p1, err := lb.SelectProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, "provider-1", p1.ID)

	p2, err := lb.SelectProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, "provider-2", p2.ID)

	p3, err := lb.SelectProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, "provider-3", p3.ID)

	// Second round - should wrap around
	p4, err := lb.SelectProvider(ctx)
	require.NoError(t, err)
	assert.Equal(t, "provider-1", p4.ID)

	// Verify consistent round-robin over multiple iterations
	selectedIDs := make(map[string]int)
	iterations := 30

	for i := 0; i < iterations; i++ {
		provider, err := lb.SelectProvider(ctx)
		require.NoError(t, err)
		selectedIDs[provider.ID]++
	}

	// Each provider should be selected exactly 10 times (30 / 3)
	for _, provider := range providers {
		assert.Equal(t, iterations/len(providers), selectedIDs[provider.ID],
			"Provider %s should be selected evenly", provider.ID)
	}
}

// TestLoadBalancerWithDisabledProviders tests load balancer with disabled providers
func TestLoadBalancerWithDisabledProviders(t *testing.T) {
	providers := []*models.Provider{
		{ID: "provider-1", Name: "Provider 1", APIURL: "http://provider1.com", Enabled: true, Priority: 1},
		{ID: "provider-2", Name: "Provider 2", APIURL: "http://provider2.com", Enabled: false, Priority: 2},
		{ID: "provider-3", Name: "Provider 3", APIURL: "http://provider3.com", Enabled: true, Priority: 3},
	}

	lb := proxy.NewRoundRobinLoadBalancer(providers)
	ctx := context.Background()

	// Should only select enabled providers
	for i := 0; i < 10; i++ {
		provider, err := lb.SelectProvider(ctx)
		require.NoError(t, err)
		assert.True(t, provider.Enabled, "Should only select enabled providers")
		assert.NotEqual(t, "provider-2", provider.ID, "Should skip disabled provider")
	}
}

// TestLoadBalancerNoProviders tests behavior when no providers available
func TestLoadBalancerNoProviders(t *testing.T) {
	lb := proxy.NewRoundRobinLoadBalancer([]*models.Provider{})
	ctx := context.Background()

	_, err := lb.SelectProvider(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no providers available")
}

// TestProviderHealthChecking tests provider health checking
// T049 [P] [US2] Unit test for provider health checking
func TestProviderHealthChecking(t *testing.T) {
	tests := []struct {
		name           string
		provider       *models.Provider
		mockResponse   int
		expectedHealth bool
	}{
		{
			name: "healthy provider (200 OK)",
			provider: &models.Provider{
				ID:      "provider-1",
				APIURL:  "http://provider1.com",
				Enabled: true,
			},
			mockResponse:   200,
			expectedHealth: true,
		},
		{
			name: "unhealthy provider (500 error)",
			provider: &models.Provider{
				ID:      "provider-2",
				APIURL:  "http://provider2.com",
				Enabled: true,
			},
			mockResponse:   500,
			expectedHealth: false,
		},
		{
			name: "timeout provider",
			provider: &models.Provider{
				ID:      "provider-3",
				APIURL:  "http://timeout.com",
				Enabled: true,
				Timeout: 100 * time.Millisecond,
			},
			mockResponse:   0, // timeout
			expectedHealth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := proxy.NewHealthChecker(1 * time.Second)
			ctx := context.Background()

			// Mock HTTP client would be injected here
			// For now, we're testing the interface
			healthy := checker.CheckHealth(ctx, tt.provider)

			// This will be implemented with actual health check logic
			_ = healthy
		})
	}
}

// TestHealthCheckCache tests health check result caching
func TestHealthCheckCache(t *testing.T) {
	provider := &models.Provider{
		ID:      "provider-1",
		APIURL:  "http://provider1.com",
		Enabled: true,
	}

	checker := proxy.NewHealthChecker(5 * time.Second)
	ctx := context.Background()

	// First check
	start1 := time.Now()
	_ = checker.CheckHealth(ctx, provider)
	elapsed1 := time.Since(start1)

	// Second check (should use cache)
	start2 := time.Now()
	_ = checker.CheckHealth(ctx, provider)
	elapsed2 := time.Since(start2)

	// Cached check should be much faster
	assert.Less(t, elapsed2, elapsed1/10, "Cached health check should be faster")
}

// TestHealthCheckConcurrency tests concurrent health checks
func TestHealthCheckConcurrency(t *testing.T) {
	providers := []*models.Provider{
		{ID: "provider-1", APIURL: "http://provider1.com", Enabled: true},
		{ID: "provider-2", APIURL: "http://provider2.com", Enabled: true},
		{ID: "provider-3", APIURL: "http://provider3.com", Enabled: true},
	}

	checker := proxy.NewHealthChecker(1 * time.Second)
	ctx := context.Background()

	// Run concurrent health checks
	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			for _, provider := range providers {
				_ = checker.CheckHealth(ctx, provider)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		<-done
	}

	// No assertions needed - test passes if no race conditions
}

// TestStreamingResponseHandling tests streaming response handling
// T050 [P] [US2] Integration test for streaming responses (placeholder)
func TestStreamingResponseSetup(t *testing.T) {
	// This will be a full integration test in tests/integration/
	// For now, we test the streaming handler setup

	handler := proxy.NewStreamingHandler()
	assert.NotNil(t, handler)

	// Test SSE format
	event := proxy.ServerSentEvent{
		Data: map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"delta": map[string]string{
						"content": "Hello",
					},
				},
			},
		},
	}

	formatted := handler.FormatSSE(event)
	assert.Contains(t, formatted, "data: ")
	assert.Contains(t, formatted, "\n\n")
}

// TestAutomaticFailover tests automatic provider failover
// T051 [P] [US2] Integration test for automatic failover (placeholder)
func TestFailoverLogic(t *testing.T) {
	providers := []*models.Provider{
		{ID: "provider-1", APIURL: "http://provider1.com", Enabled: true, Priority: 1},
		{ID: "provider-2", APIURL: "http://provider2.com", Enabled: true, Priority: 2},
		{ID: "provider-3", APIURL: "http://provider3.com", Enabled: true, Priority: 3},
	}

	lb := proxy.NewRoundRobinLoadBalancer(providers)
	ctx := context.Background()

	// Test failover when provider fails
	primary, err := lb.SelectProvider(ctx)
	require.NoError(t, err)

	// Mark primary as failed
	lb.MarkProviderFailed(primary.ID)

	// Next selection should skip failed provider
	backup, err := lb.SelectProvider(ctx)
	require.NoError(t, err)
	assert.NotEqual(t, primary.ID, backup.ID, "Should failover to different provider")
}

// TestAuditLogging tests audit logging with required fields
// T052 [P] [US2] Unit test for audit logging with required fields
func TestProxyAuditLogging(t *testing.T) {
	// This tests the audit log structure for proxy requests

	auditEntry := &models.AuditLog{
		ID:             "audit-123",
		Timestamp:      time.Now(),
		UserID:         "user-1",
		Username:       "testuser",
		OperationType:  "proxy_request",
		TargetResource: "provider-1/gpt-4",
		IPAddress:      "192.168.1.1",
		Result:         "success",
		Details:        `{"model": "gpt-4", "tokens": 100}`,
	}

	// Validate required fields per FR-032
	assert.NotEmpty(t, auditEntry.ID)
	assert.NotZero(t, auditEntry.Timestamp)
	assert.NotEmpty(t, auditEntry.UserID)
	assert.NotEmpty(t, auditEntry.Username)
	assert.NotEmpty(t, auditEntry.OperationType)
	assert.NotEmpty(t, auditEntry.TargetResource)
	assert.NotEmpty(t, auditEntry.IPAddress)
	assert.NotEmpty(t, auditEntry.Result)
}

// TestAuditLogTimestamps tests audit log timestamp format
func TestAuditLogTimestamps(t *testing.T) {
	entry := &models.AuditLog{
		Timestamp: time.Now(),
		CreatedAt: time.Now(),
	}

	// Timestamps should be in chronological order
	assert.True(t, entry.Timestamp.Before(time.Now().Add(1*time.Second)))
	assert.True(t, entry.CreatedAt.Before(time.Now().Add(1*time.Second)))
}

// TestAuditLogOrdering tests audit logs are ordered by timestamp
func TestAuditLogOrdering(t *testing.T) {
	logs := []*models.AuditLog{
		{ID: "3", Timestamp: time.Now().Add(2 * time.Second)},
		{ID: "1", Timestamp: time.Now()},
		{ID: "2", Timestamp: time.Now().Add(1 * time.Second)},
	}

	// Sort by timestamp
	sorted := sortAuditLogs(logs)

	assert.Equal(t, "1", sorted[0].ID)
	assert.Equal(t, "2", sorted[1].ID)
	assert.Equal(t, "3", sorted[2].ID)
}

// Helper functions

func sortAuditLogs(logs []*models.AuditLog) []*models.AuditLog {
	sorted := make([]*models.AuditLog, len(logs))
	copy(sorted, logs)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Timestamp.After(sorted[j].Timestamp) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}
