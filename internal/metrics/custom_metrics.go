package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// CustomMetrics holds all custom Prometheus metrics for the proxy
type CustomMetrics struct {
	// Request metrics
	TotalRequests          prometheus.Counter
	TotalErrors            prometheus.Counter
	RequestDuration        prometheus.Histogram
	RequestInFlight        prometheus.Gauge

	// Provider-specific metrics
	ProviderRequests       *prometheus.CounterVec
	ProviderDuration       *prometheus.HistogramVec
	ProviderErrors         *prometheus.CounterVec
	ProviderSuccessRate    *prometheus.GaugeVec

	// Response metrics
	ResponseSize           prometheus.Histogram
	PromptTokens           prometheus.Histogram
	CompletionTokens       prometheus.Histogram
	TotalTokens            prometheus.Histogram

	// Business metrics
	ActiveUsers            prometheus.Gauge
	LoadBalancedRequests   prometheus.Counter
	FailedOverRequests     prometheus.Counter
	RetriedRequests        prometheus.Counter

	// Health metrics
	HealthyProviders       prometheus.Gauge
	UnhealthyProviders     prometheus.Gauge
	ProviderHealthChecks   *prometheus.CounterVec
	ProviderResponseTime   *prometheus.HistogramVec
}

// NewCustomMetrics creates a new custom metrics collector
func NewCustomMetrics(logger *logrus.Logger) *CustomMetrics {
	m := &CustomMetrics{}

	// Total requests
	m.TotalRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Name:      "requests_total",
			Help:      "Total number of requests processed",
		},
	)

	// Total errors
	m.TotalErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Name:      "errors_total",
			Help:      "Total number of errors",
		},
	)

	// Request duration
	m.RequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Name:      "request_duration_seconds",
			Help:      "Duration of requests in seconds",
			Buckets:   prometheus.DefBuckets,
		},
	)

	// Requests in flight
	m.RequestInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "llm_proxy",
			Name:      "requests_in_flight",
			Help:      "Current number of requests being processed",
		},
	)

	// Provider requests by status
	m.ProviderRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "requests_total",
			Help:      "Total requests to each provider",
		},
		[]string{"provider", "status_code"},
	)

	// Provider duration
	m.ProviderDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "request_duration_seconds",
			Help:      "Request duration by provider",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"provider"},
	)

	// Provider errors
	m.ProviderErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "errors_total",
			Help:      "Total errors by provider",
		},
		[]string{"provider", "error_type"},
	)

	// Provider success rate
	m.ProviderSuccessRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "llm_proxy",
			Subsystem: "provider",
			Name:      "success_rate",
			Help:      "Success rate by provider (0-100)",
		},
		[]string{"provider"},
	)

	// Response size
	m.ResponseSize = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Name:      "response_size_bytes",
			Help:      "Size of responses in bytes",
			Buckets:   prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to ~5MB
		},
	)

	// Token usage metrics
	m.PromptTokens = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Name:      "prompt_tokens",
			Help:      "Number of prompt tokens",
			Buckets:   prometheus.ExponentialBuckets(10, 2, 15),
		},
	)

	m.CompletionTokens = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Name:      "completion_tokens",
			Help:      "Number of completion tokens",
			Buckets:   prometheus.ExponentialBuckets(10, 2, 15),
		},
	)

	m.TotalTokens = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Name:      "total_tokens",
			Help:      "Total number of tokens (prompt + completion)",
			Buckets:   prometheus.ExponentialBuckets(10, 2, 16),
		},
	)

	// Business metrics
	m.ActiveUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "llm_proxy",
			Name:      "active_users",
			Help:      "Number of active users",
		},
	)

	m.LoadBalancedRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Name:      "load_balanced_requests_total",
			Help:      "Total number of load balanced requests",
		},
	)

	m.FailedOverRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Name:      "failed_over_requests_total",
			Help:      "Total number of requests that failed over to backup providers",
		},
	)

	m.RetriedRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Name:      "retried_requests_total",
			Help:      "Total number of retried requests",
		},
	)

	// Health metrics
	m.HealthyProviders = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "llm_proxy",
			Subsystem: "health",
			Name:      "healthy_providers",
			Help:      "Number of healthy providers",
		},
	)

	m.UnhealthyProviders = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "llm_proxy",
			Subsystem: "health",
			Name:      "unhealthy_providers",
			Help:      "Number of unhealthy providers",
		},
	)

	m.ProviderHealthChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "llm_proxy",
			Subsystem: "health",
			Name:      "checks_total",
			Help:      "Total health checks performed",
		},
		[]string{"provider", "result"},
	)

	m.ProviderResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "llm_proxy",
			Subsystem: "health",
			Name:      "response_time_seconds",
			Help:      "Provider response times for health checks",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"provider"},
	)

	return m
}

// Register registers all metrics with Prometheus
func (m *CustomMetrics) Register(registry prometheus.Registerer) {
	registry.MustRegister(
		m.TotalRequests,
		m.TotalErrors,
		m.RequestDuration,
		m.RequestInFlight,
		m.ProviderRequests,
		m.ProviderDuration,
		m.ProviderErrors,
		m.ProviderSuccessRate,
		m.ResponseSize,
		m.PromptTokens,
		m.CompletionTokens,
		m.TotalTokens,
		m.ActiveUsers,
		m.LoadBalancedRequests,
		m.FailedOverRequests,
		m.RetriedRequests,
		m.HealthyProviders,
		m.UnhealthyProviders,
		m.ProviderHealthChecks,
		m.ProviderResponseTime,
	)
}

// RecordRequest records a request metric
func (m *CustomMetrics) RecordRequest(provider string, duration time.Duration, statusCode int, responseSize int) {
	m.TotalRequests.Inc()
	m.RequestDuration.Observe(duration.Seconds())
	m.RequestInFlight.Inc()

	// Record provider-specific metrics
	m.ProviderRequests.WithLabelValues(provider, getStatusCodeClass(statusCode)).Inc()
	m.ProviderDuration.WithLabelValues(provider).Observe(duration.Seconds())

	if responseSize > 0 {
		m.ResponseSize.Observe(float64(responseSize))
	}

	// Track errors
	if statusCode >= 400 {
		m.TotalErrors.Inc()
		m.ProviderErrors.WithLabelValues(provider, getErrorClass(statusCode)).Inc()
	}

	// Simulate decrement after request completes
	go func() {
		time.Sleep(duration)
		m.RequestInFlight.Dec()
	}()
}

// RecordLoadBalancedRequest records a load balanced request
func (m *CustomMetrics) RecordLoadBalancedRequest(fromProvider, toProvider string) {
	m.LoadBalancedRequests.Inc()
}

// RecordFailedOverRequest records a failed over request
func (m *CustomMetrics) RecordFailedOverRequest(provider string) {
	m.FailedOverRequests.Inc()
}

// RecordRetriedRequest records a retried request
func (m *CustomMetrics) RecordRetriedRequest(provider string) {
	m.RetriedRequests.Inc()
}

// RecordTokens records token usage
func (m *CustomMetrics) RecordTokens(promptTokens, completionTokens int) {
	if promptTokens > 0 {
		m.PromptTokens.Observe(float64(promptTokens))
	}
	if completionTokens > 0 {
		m.CompletionTokens.Observe(float64(completionTokens))
	}
	if promptTokens > 0 || completionTokens > 0 {
		m.TotalTokens.Observe(float64(promptTokens + completionTokens))
	}
}

// RecordProviderHealth records provider health status
func (m *CustomMetrics) RecordProviderHealth(provider string, healthy bool, responseTime time.Duration) {
	m.ProviderHealthChecks.WithLabelValues(provider, func() string {
		if healthy {
			return "healthy"
		}
		return "unhealthy"
	}()).Inc()

	if responseTime > 0 {
		m.ProviderResponseTime.WithLabelValues(provider).Observe(responseTime.Seconds())
	}
}

// UpdateProviderCounts updates the count of healthy/unhealthy providers
func (m *CustomMetrics) UpdateProviderCounts(healthyCount, unhealthyCount int) {
	m.HealthyProviders.Set(float64(healthyCount))
	m.UnhealthyProviders.Set(float64(unhealthyCount))
}

// getStatusCodeClass returns a status code class (2xx, 4xx, 5xx)
func getStatusCodeClass(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "5xx"
	case statusCode >= 400:
		return "4xx"
	case statusCode >= 300:
		return "3xx"
	case statusCode >= 200:
		return "2xx"
	case statusCode >= 100:
		return "1xx"
	default:
		return "unknown"
	}
}

// getErrorClass returns an error class based on status code
func getErrorClass(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "server_error"
	case statusCode >= 400:
		return "client_error"
	default:
		return "other"
	}
}
