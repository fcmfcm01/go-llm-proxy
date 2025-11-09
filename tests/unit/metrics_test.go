package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/internal/models"
)

// TestProviderMetricsCollector tests the provider metrics collection functionality
func TestProviderMetricsCollector(t *testing.T) {
	t.Run("CreateMetricsCollector", func(t *testing.T) {
		// Test that metrics collector can be created
		collector := NewTestMetricsCollector()

		assert.NotNil(t, collector, "Metrics collector should be created")
		assert.Equal(t, "test-collector", collector.Name(), "Collector should have correct name")
	})

	t.Run("RecordProviderRequest", func(t *testing.T) {
		// Test recording a provider request
		collector := NewTestMetricsCollector()

		providerID := "test-provider"
		duration := 150 * time.Millisecond
		statusCode := 200

		collector.RecordRequest(providerID, duration, statusCode)

		// Verify metrics were recorded
		metrics := collector.GetMetrics(providerID)
		assert.NotNil(t, metrics, "Metrics should be available")
		assert.Equal(t, int64(1), metrics.RequestCount, "Request count should be 1")
	})

	t.Run("CalculateSuccessRate", func(t *testing.T) {
		// Test success rate calculation
		collector := NewTestMetricsCollector()

		providerID := "test-provider"

		// Record successful requests
		for i := 0; i < 8; i++ {
			collector.RecordRequest(providerID, 100*time.Millisecond, 200)
		}

		// Record failed requests
		for i := 0; i < 2; i++ {
			collector.RecordRequest(providerID, 200*time.Millisecond, 500)
		}

		metrics := collector.GetMetrics(providerID)
		assert.Equal(t, int64(10), metrics.RequestCount, "Total requests should be 10")
		assert.Equal(t, float64(80.0), metrics.SuccessRate, "Success rate should be 80%")
	})

	t.Run("TrackResponseTime", func(t *testing.T) {
		// Test response time tracking
		collector := NewTestMetricsCollector()

		providerID := "test-provider"

		// Record requests with different durations
		durations := []time.Duration{100, 200, 300, 150, 250}
		for _, d := range durations {
			collector.RecordRequest(providerID, d, 200)
		}

		metrics := collector.GetMetrics(providerID)
		assert.Equal(t, int64(5), metrics.RequestCount, "Should have 5 requests")
		assert.Equal(t, int64(100), metrics.MinResponseTime, "Min response time should be 100ms")
		assert.Equal(t, int64(300), metrics.MaxResponseTime, "Max response time should be 300ms")
	})

	t.Run("MultipleProviders", func(t *testing.T) {
		// Test metrics collection for multiple providers
		collector := NewTestMetricsCollector()

		// Record for provider 1
		collector.RecordRequest("provider-1", 100*time.Millisecond, 200)
		collector.RecordRequest("provider-1", 200*time.Millisecond, 200)

		// Record for provider 2
		collector.RecordRequest("provider-2", 150*time.Millisecond, 500)
		collector.RecordRequest("provider-2", 250*time.Millisecond, 200)

		// Verify provider 1
		metrics1 := collector.GetMetrics("provider-1")
		assert.Equal(t, int64(2), metrics1.RequestCount, "Provider 1 should have 2 requests")
		assert.Equal(t, float64(100.0), metrics1.SuccessRate, "Provider 1 should have 100% success rate")

		// Verify provider 2
		metrics2 := collector.GetMetrics("provider-2")
		assert.Equal(t, int64(2), metrics2.RequestCount, "Provider 2 should have 2 requests")
		assert.Equal(t, float64(50.0), metrics2.SuccessRate, "Provider 2 should have 50% success rate")
	})

	t.Run("UpdateProviderStatus", func(t *testing.T) {
		// Test provider status updates
		collector := NewTestMetricsCollector()

		providerID := "test-provider"

		// Initially healthy
		collector.UpdateProviderStatus(providerID, true)
		assert.True(t, collector.IsHealthy(providerID), "Provider should be healthy")

		// Mark as unhealthy
		collector.UpdateProviderStatus(providerID, false)
		assert.False(t, collector.IsHealthy(providerID), "Provider should be unhealthy")

		// Mark as healthy again
		collector.UpdateProviderStatus(providerID, true)
		assert.True(t, collector.IsHealthy(providerID), "Provider should be healthy again")
	})

	t.Run("ResetMetrics", func(t *testing.T) {
		// Test metrics reset
		collector := NewTestMetricsCollector()

		providerID := "test-provider"
		collector.RecordRequest(providerID, 100*time.Millisecond, 200)
		collector.RecordRequest(providerID, 200*time.Millisecond, 200)

		// Verify metrics exist
		metrics := collector.GetMetrics(providerID)
		require.NotNil(t, metrics, "Metrics should exist")
		assert.Equal(t, int64(2), metrics.RequestCount, "Should have 2 requests")

		// Reset metrics
		collector.ResetMetrics(providerID)

		// Verify metrics are reset
		metrics = collector.GetMetrics(providerID)
		assert.Equal(t, int64(0), metrics.RequestCount, "Request count should be 0 after reset")
	})

	t.Run("GetAllProviderMetrics", func(t *testing.T) {
		// Test retrieving all provider metrics
		collector := NewTestMetricsCollector()

		// Record metrics for multiple providers
		collector.RecordRequest("provider-1", 100*time.Millisecond, 200)
		collector.RecordRequest("provider-2", 200*time.Millisecond, 500)
		collector.RecordRequest("provider-3", 300*time.Millisecond, 200)

		// Get all metrics
		allMetrics := collector.GetAllMetrics()

		assert.Equal(t, 3, len(allMetrics), "Should have metrics for 3 providers")
		assert.Contains(t, allMetrics, "provider-1", "Should contain provider-1")
		assert.Contains(t, allMetrics, "provider-2", "Should contain provider-2")
		assert.Contains(t, allMetrics, "provider-3", "Should contain provider-3")
	})
}

// TestMetricsCollector is a test implementation of the metrics collector
type TestMetricsCollector struct {
	metrics map[string]*ProviderMetrics
}

type ProviderMetrics struct {
	RequestCount    int64   `json:"request_count"`
	SuccessCount    int64   `json:"success_count"`
	ErrorCount      int64   `json:"error_count"`
	SuccessRate     float64 `json:"success_rate"`
	MinResponseTime int64   `json:"min_response_time_ms"`
	MaxResponseTime int64   `json:"max_response_time_ms"`
	AvgResponseTime float64 `json:"avg_response_time_ms"`
	Healthy         bool    `json:"healthy"`
}

// NewTestMetricsCollector creates a new test metrics collector
func NewTestMetricsCollector() *TestMetricsCollector {
	return &TestMetricsCollector{
		metrics: make(map[string]*ProviderMetrics),
	}
}

// Name returns the collector name
func (c *TestMetricsCollector) Name() string {
	return "test-collector"
}

// RecordRequest records a request metric
func (c *TestMetricsCollector) RecordRequest(providerID string, duration time.Duration, statusCode int) {
	if _, ok := c.metrics[providerID]; !ok {
		c.metrics[providerID] = &ProviderMetrics{
			MinResponseTime: -1,
		}
	}

	metrics := c.metrics[providerID]
	metrics.RequestCount++

	if statusCode >= 200 && statusCode < 300 {
		metrics.SuccessCount++
	} else {
		metrics.ErrorCount++
	}

	// Update response time stats
	durationMs := duration.Milliseconds()
	if metrics.MinResponseTime == -1 || durationMs < metrics.MinResponseTime {
		metrics.MinResponseTime = durationMs
	}
	if durationMs > metrics.MaxResponseTime {
		metrics.MaxResponseTime = durationMs
	}

	// Calculate success rate
	if metrics.RequestCount > 0 {
		metrics.SuccessRate = (float64(metrics.SuccessCount) / float64(metrics.RequestCount)) * 100
	}
}

// GetMetrics gets metrics for a provider
func (c *TestMetricsCollector) GetMetrics(providerID string) *ProviderMetrics {
	return c.metrics[providerID]
}

// UpdateProviderStatus updates provider health status
func (c *TestMetricsCollector) UpdateProviderStatus(providerID string, healthy bool) {
	if _, ok := c.metrics[providerID]; !ok {
		c.metrics[providerID] = &ProviderMetrics{}
	}
	c.metrics[providerID].Healthy = healthy
}

// IsHealthy checks if provider is healthy
func (c *TestMetricsCollector) IsHealthy(providerID string) bool {
	if metrics, ok := c.metrics[providerID]; ok {
		return metrics.Healthy
	}
	return false
}

// ResetMetrics resets metrics for a provider
func (c *TestMetricsCollector) ResetMetrics(providerID string) {
	c.metrics[providerID] = &ProviderMetrics{
		MinResponseTime: -1,
	}
}

// GetAllMetrics gets all provider metrics
func (c *TestMetricsCollector) GetAllMetrics() map[string]*ProviderMetrics {
	return c.metrics
}
