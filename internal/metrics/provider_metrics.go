package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// ProviderMetricsCollector collects and exposes metrics for all providers
type ProviderMetricsCollector struct {
	logger *logrus.Logger

	// Prometheus metrics
	requestDuration prometheus.Histogram
	requestCount    prometheus.Counter
	errorCount      prometheus.Counter
	activeRequests  prometheus.Gauge

	// In-memory metrics storage
	metrics map[string]*ProviderMetrics
	mu      sync.RWMutex
}

// ProviderMetrics holds in-memory metrics for a single provider
type ProviderMetrics struct {
	ProviderID        string        `json:"provider_id"`
	RequestCount      int64         `json:"request_count"`
	SuccessCount      int64         `json:"success_count"`
	ErrorCount        int64         `json:"error_count"`
	SuccessRate       float64       `json:"success_rate"`
	MinResponseTime   int64         `json:"min_response_time_ms"`
	MaxResponseTime   int64         `json:"max_response_time_ms"`
	AvgResponseTime   float64       `json:"avg_response_time_ms"`
	TotalResponseTime int64         `json:"total_response_time_ms"`
	Healthy           bool          `json:"healthy"`
	LastHealthCheck   time.Time     `json:"last_health_check"`
	MetricsByStatus   map[int]int64 `json:"metrics_by_status"`
	RecentDurations   []int64       `json:"-"`
}

// NewProviderMetricsCollector creates a new provider metrics collector
func NewProviderMetricsCollector(logger *logrus.Logger) *ProviderMetricsCollector {
	c := &ProviderMetricsCollector{
		logger:  logger,
		metrics: make(map[string]*ProviderMetrics),
	}

	// Initialize Prometheus metrics
	c.requestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "request_duration_seconds",
			Help:      "Duration of requests to providers",
			Buckets:   prometheus.DefBuckets,
		},
	)

	c.requestCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "requests_total",
			Help:      "Total number of requests to providers",
		},
	)

	c.errorCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "errors_total",
			Help:      "Total number of errors from providers",
		},
	)

	c.activeRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "active_requests",
			Help:      "Current number of active requests to providers",
		},
	)

	// Register metrics
	prometheus.MustRegister(c.requestDuration, c.requestCount, c.errorCount, c.activeRequests)

	return c
}

// RecordRequest records a request metric
func (c *ProviderMetricsCollector) RecordRequest(providerID string, duration time.Duration, statusCode int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get or create provider metrics
	if _, ok := c.metrics[providerID]; !ok {
		c.metrics[providerID] = &ProviderMetrics{
			ProviderID:      providerID,
			MetricsByStatus: make(map[int]int64),
			MinResponseTime: -1,
		}
	}

	m := c.metrics[providerID]
	durationMs := duration.Milliseconds()

	// Update counters
	m.RequestCount++
	m.MetricsByStatus[statusCode]++

	if statusCode >= 200 && statusCode < 300 {
		m.SuccessCount++
	} else {
		m.ErrorCount++
	}

	// Update response time stats
	if m.MinResponseTime == -1 || durationMs < m.MinResponseTime {
		m.MinResponseTime = durationMs
	}
	if durationMs > m.MaxResponseTime {
		m.MaxResponseTime = durationMs
	}

	m.TotalResponseTime += durationMs
	m.AvgResponseTime = float64(m.TotalResponseTime) / float64(m.RequestCount)

	// Keep recent durations for rolling avg
	m.RecentDurations = append(m.RecentDurations, durationMs)
	if len(m.RecentDurations) > 100 {
		m.RecentDurations = m.RecentDurations[1:]
	}

	// Update Prometheus metrics
	c.requestCount.Inc()
	if statusCode >= 400 {
		c.errorCount.Inc()
	}
	c.requestDuration.Observe(duration.Seconds())

	// Update active requests
	c.activeRequests.Inc()
	go func() {
		time.Sleep(duration)
		c.activeRequests.Dec()
	}()
}

// UpdateProviderHealth updates provider health status
func (c *ProviderMetricsCollector) UpdateProviderHealth(providerID string, healthy bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, ok := c.metrics[providerID]; !ok {
		c.metrics[providerID] = &ProviderMetrics{
			ProviderID:      providerID,
			MetricsByStatus: make(map[int]int64),
			MinResponseTime: -1,
		}
	}

	m := c.metrics[providerID]
	m.Healthy = healthy
	m.LastHealthCheck = time.Now()
}

// GetMetrics returns metrics for a specific provider
func (c *ProviderMetricsCollector) GetMetrics(providerID string) *ProviderMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if m, ok := c.metrics[providerID]; ok {
		return m
	}

	return nil
}

// GetAllMetrics returns metrics for all providers
func (c *ProviderMetricsCollector) GetAllMetrics() map[string]*ProviderMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[string]*ProviderMetrics)
	for k, v := range c.metrics {
		result[k] = v
	}

	return result
}

// ResetMetrics resets metrics for a specific provider
func (c *ProviderMetricsCollector) ResetMetrics(providerID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.metrics, providerID)
}

// GetHealthStatus returns health status for all providers
func (c *ProviderMetricsCollector) GetHealthStatus() map[string]bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status := make(map[string]bool)
	for providerID, m := range c.metrics {
		status[providerID] = m.Healthy
	}

	return status
}

// GetAggregatedStats returns aggregated statistics across all providers
func (c *ProviderMetricsCollector) GetAggregatedStats() *AggregatedProviderStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := &AggregatedProviderStats{
		TotalRequests:  0,
		TotalErrors:    0,
		AvgSuccessRate: 0,
		TotalProviders: len(c.metrics),
		HealthyCount:   0,
	}

	var totalSuccessRate float64
	var count int

	for _, m := range c.metrics {
		stats.TotalRequests += m.RequestCount
		stats.TotalErrors += m.ErrorCount

		if m.RequestCount > 0 {
			rate := (float64(m.SuccessCount) / float64(m.RequestCount)) * 100
			totalSuccessRate += rate
			count++
		}

		if m.Healthy {
			stats.HealthyCount++
		}
	}

	if count > 0 {
		stats.AvgSuccessRate = totalSuccessRate / float64(count)
	}

	return stats
}

// AggregatedProviderStats holds aggregated statistics
type AggregatedProviderStats struct {
	TotalRequests  int64   `json:"total_requests"`
	TotalErrors    int64   `json:"total_errors"`
	AvgSuccessRate float64 `json:"avg_success_rate"`
	TotalProviders int     `json:"total_providers"`
	HealthyCount   int     `json:"healthy_providers"`
}

// Close cleans up Prometheus metrics
func (c *ProviderMetricsCollector) Close() {
	prometheus.Unregister(c.requestDuration)
	prometheus.Unregister(c.requestCount)
	prometheus.Unregister(c.errorCount)
	prometheus.Unregister(c.activeRequests)
}
