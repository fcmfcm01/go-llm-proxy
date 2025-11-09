package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/metrics"
)

// TestMetricAccuracy tests the accuracy of metric collection
func TestMetricAccuracy(t *testing.T) {
	t.Run("RequestCountAccuracy", func(t *testing.T) {
		// Test that request counts are accurate
		collector := metrics.NewProviderMetricsCollector(nil)

		providerID := "test-provider"
		numRequests := 100

		// Record multiple requests
		for i := 0; i < numRequests; i++ {
			collector.RecordRequest(providerID, 50*time.Millisecond, 200)
		}

		// Verify exact count
		metrics := collector.GetMetrics(providerID)
		require.NotNil(t, metrics, "Metrics should be available")
		assert.Equal(t, int64(numRequests), metrics.RequestCount,
			"Request count should be exactly %d", numRequests)
	})

	t.Run("SuccessRateAccuracy", func(t *testing.T) {
		// Test that success rates are calculated accurately
		collector := metrics.NewProviderMetricsCollector(nil)

		providerID := "test-provider"

		// Record 80 successful and 20 failed requests
		for i := 0; i < 80; i++ {
			collector.RecordRequest(providerID, 50*time.Millisecond, 200)
		}
		for i := 0; i < 20; i++ {
			collector.RecordRequest(providerID, 50*time.Millisecond, 500)
		}

		metrics := collector.GetMetrics(providerID)
		require.NotNil(t, metrics)

		// Success rate should be 80%
		assert.Equal(t, float64(80.0), metrics.SuccessRate, 0.01,
			"Success rate should be 80%%")
		assert.Equal(t, int64(80), metrics.SuccessCount)
		assert.Equal(t, int64(20), metrics.ErrorCount)
	})

	t.Run("ResponseTimeAccuracy", func(t *testing.T) {
		// Test that response time metrics are accurate
		collector := metrics.NewProviderMetricsCollector(nil)

		providerID := "test-provider"

		// Record requests with specific durations
		durations := []int64{100, 200, 300, 400, 500} // ms
		for _, d := range durations {
			collector.RecordRequest(providerID, time.Duration(d)*time.Millisecond, 200)
		}

		metrics := collector.GetMetrics(providerID)
		require.NotNil(t, metrics)

		// Verify min, max, and average
		assert.Equal(t, int64(100), metrics.MinResponseTime, "Min should be 100ms")
		assert.Equal(t, int64(500), metrics.MaxResponseTime, "Max should be 500ms")
		assert.Equal(t, int64(300), metrics.AvgResponseTime, 1, "Average should be 300ms")
		assert.Equal(t, int64(1500), metrics.TotalResponseTime, "Total should be 1500ms")
	})

	t.Run("MultipleProvidersIndependent", func(t *testing.T) {
		// Test that metrics for different providers are independent
		collector := metrics.NewProviderMetricsCollector(nil)

		// Provider 1: 100% success
		for i := 0; i < 10; i++ {
			collector.RecordRequest("provider-1", 100*time.Millisecond, 200)
		}

		// Provider 2: 50% success
		for i := 0; i < 10; i++ {
			status := 200
			if i%2 == 0 {
				status = 500
			}
			collector.RecordRequest("provider-2", 200*time.Millisecond, status)
		}

		// Provider 3: 0% success
		for i := 0; i < 10; i++ {
			collector.RecordRequest("provider-3", 300*time.Millisecond, 500)
		}

		// Verify each provider has correct metrics
		p1 := collector.GetMetrics("provider-1")
		require.NotNil(t, p1)
		assert.Equal(t, int64(10), p1.RequestCount)
		assert.Equal(t, float64(100.0), p1.SuccessRate, 0.01)

		p2 := collector.GetMetrics("provider-2")
		require.NotNil(t, p2)
		assert.Equal(t, int64(10), p2.RequestCount)
		assert.Equal(t, float64(50.0), p2.SuccessRate, 0.01)

		p3 := collector.GetMetrics("provider-3")
		require.NotNil(t, p3)
		assert.Equal(t, int64(10), p3.RequestCount)
		assert.Equal(t, float64(0.0), p3.SuccessRate, 0.01)
	})

	t.Run("StatusCodeBreakdown", func(t *testing.T) {
		// Test that metrics are correctly broken down by status code
		collector := metrics.NewProviderMetricsCollector(nil)

		providerID := "test-provider"

		// Record requests with different status codes
		collector.RecordRequest(providerID, 50*time.Millisecond, 200) // 2xx
		collector.RecordRequest(providerID, 50*time.Millisecond, 200)
		collector.RecordRequest(providerID, 50*time.Millisecond, 201)
		collector.RecordRequest(providerID, 50*time.Millisecond, 400) // 4xx
		collector.RecordRequest(providerID, 50*time.Millisecond, 400)
		collector.RecordRequest(providerID, 50*time.Millisecond, 500) // 5xx
		collector.RecordRequest(providerID, 50*time.Millisecond, 500)
		collector.RecordRequest(providerID, 50*time.Millisecond, 500)

		metrics := collector.GetMetrics(providerID)
		require.NotNil(t, metrics)

		// Verify status code breakdown
		assert.Contains(t, metrics.MetricsByStatus, 200)
		assert.Contains(t, metrics.MetricsByStatus, 201)
		assert.Contains(t, metrics.MetricsByStatus, 400)
		assert.Contains(t, metrics.MetricsByStatus, 500)

		assert.Equal(t, int64(2), metrics.MetricsByStatus[200])
		assert.Equal(t, int64(1), metrics.MetricsByStatus[201])
		assert.Equal(t, int64(2), metrics.MetricsByStatus[400])
		assert.Equal(t, int64(3), metrics.MetricsByStatus[500])
	})

	t.Run("HealthStatusTracking", func(t *testing.T) {
		// Test that health status is tracked accurately
		collector := metrics.NewProviderMetricsCollector(nil)

		providerID := "test-provider"

		// Initially should not be tracked
		metrics := collector.GetMetrics(providerID)
		assert.Nil(t, metrics, "No metrics initially")

		// Update health status
		collector.UpdateProviderHealth(providerID, true)
		metrics = collector.GetMetrics(providerID)
		require.NotNil(t, metrics)
		assert.True(t, metrics.Healthy, "Provider should be healthy")

		// Update to unhealthy
		collector.UpdateProviderHealth(providerID, false)
		metrics = collector.GetMetrics(providerID)
		assert.False(t, metrics.Healthy, "Provider should be unhealthy")
	})

	t.Run("MetricsResetAccuracy", func(t *testing.T) {
		// Test that metrics reset is accurate
		collector := metrics.NewProviderMetricsCollector(nil)

		providerID := "test-provider"

		// Record some requests
		for i := 0; i < 10; i++ {
			collector.RecordRequest(providerID, 100*time.Millisecond, 200)
		}

		// Verify metrics exist
		metrics := collector.GetMetrics(providerID)
		require.NotNil(t, metrics)
		assert.Equal(t, int64(10), metrics.RequestCount)

		// Reset metrics
		collector.ResetMetrics(providerID)

		// Verify metrics are reset
		metrics = collector.GetMetrics(providerID)
		assert.NotNil(t, metrics, "Metrics struct should still exist")
		assert.Equal(t, int64(0), metrics.RequestCount, "Request count should be 0")
		assert.Equal(t, int64(0), metrics.SuccessCount, "Success count should be 0")
		assert.Equal(t, int64(0), metrics.ErrorCount, "Error count should be 0")
		assert.Equal(t, float64(0), metrics.SuccessRate, "Success rate should be 0")
	})

	t.Run("GetAllMetricsAccuracy", func(t *testing.T) {
		// Test that GetAllMetrics returns accurate data
		collector := metrics.NewProviderMetricsCollector(nil)

		// Add metrics for multiple providers
		collector.RecordRequest("provider-1", 100*time.Millisecond, 200)
		collector.RecordRequest("provider-2", 200*time.Millisecond, 500)
		collector.RecordRequest("provider-3", 300*time.Millisecond, 200)

		allMetrics := collector.GetAllMetrics()

		// Should have exactly 3 providers
		assert.Equal(t, 3, len(allMetrics), "Should have 3 providers")

		// Verify each provider is present
		assert.Contains(t, allMetrics, "provider-1")
		assert.Contains(t, allMetrics, "provider-2")
		assert.Contains(t, allMetrics, "provider-3")

		// Verify each provider has correct data
		assert.Equal(t, int64(1), allMetrics["provider-1"].RequestCount)
		assert.Equal(t, int64(1), allMetrics["provider-2"].RequestCount)
		assert.Equal(t, int64(1), allMetrics["provider-3"].RequestCount)
	})

	t.Run("AggregatedStatsAccuracy", func(t *testing.T) {
		// Test that aggregated statistics are accurate
		collector := metrics.NewProviderMetricsCollector(nil)

		// Add provider 1: 10 requests, 100% success
		for i := 0; i < 10; i++ {
			collector.RecordRequest("provider-1", 100*time.Millisecond, 200)
		}

		// Add provider 2: 10 requests, 50% success
		for i := 0; i < 10; i++ {
			status := 200
			if i%2 == 0 {
				status = 500
			}
			collector.RecordRequest("provider-2", 200*time.Millisecond, status)
		}

		stats := collector.GetAggregatedStats()

		// Verify aggregated stats
		assert.Equal(t, int64(20), stats.TotalRequests, "Total requests should be 20")
		assert.Equal(t, int64(15), stats.TotalErrors, "Total errors should be 15")
		assert.Equal(t, 2, stats.TotalProviders, "Should have 2 providers")

		// Average success rate should be (100% + 50%) / 2 = 75%
		assert.Equal(t, float64(75.0), stats.AvgSuccessRate, 0.01,
			"Average success rate should be 75%%")
	})
}
