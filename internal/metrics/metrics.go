package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_proxy_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_proxy_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	// Provider metrics
	ProviderRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_proxy_provider_requests_total",
			Help: "Total number of provider requests",
		},
		[]string{"provider", "status"},
	)

	ProviderRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_proxy_provider_request_duration_seconds",
			Help:    "Provider request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider"},
	)

	ProviderErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_proxy_provider_errors_total",
			Help: "Total number of provider errors",
		},
		[]string{"provider", "error_type"},
	)

	// Format conversion metrics
	ConversionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "llm_proxy_conversion_duration_seconds",
			Help:    "Format conversion duration in seconds",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
		},
		[]string{"from_format", "to_format"},
	)

	ConversionErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_proxy_conversion_errors_total",
			Help: "Total number of conversion errors",
		},
		[]string{"from_format", "to_format"},
	)

	// Auth metrics
	AuthAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "llm_proxy_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"result"},
	)

	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "llm_proxy_active_sessions",
			Help: "Number of active sessions",
		},
	)

	// System metrics
	ConfigReloads = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "llm_proxy_config_reloads_total",
			Help: "Total number of configuration reloads",
		},
	)

	HealthCheckStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "llm_proxy_health_check_status",
			Help: "Health check status (1=healthy, 0=unhealthy)",
		},
		[]string{"component"},
	)
)

// RecordHTTPRequest records an HTTP request metric
func RecordHTTPRequest(method, path, status string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)
}

// RecordProviderRequest records a provider request metric
func RecordProviderRequest(provider, status string, duration float64) {
	ProviderRequestsTotal.WithLabelValues(provider, status).Inc()
	ProviderRequestDuration.WithLabelValues(provider).Observe(duration)
}

// RecordProviderError records a provider error
func RecordProviderError(provider, errorType string) {
	ProviderErrors.WithLabelValues(provider, errorType).Inc()
}

// RecordConversion records a format conversion metric
func RecordConversion(fromFormat, toFormat string, duration float64, success bool) {
	ConversionDuration.WithLabelValues(fromFormat, toFormat).Observe(duration)
	if !success {
		ConversionErrors.WithLabelValues(fromFormat, toFormat).Inc()
	}
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(result string) {
	AuthAttempts.WithLabelValues(result).Inc()
}

// SetHealthStatus sets the health status for a component
func SetHealthStatus(component string, healthy bool) {
	status := 0.0
	if healthy {
		status = 1.0
	}
	HealthCheckStatus.WithLabelValues(component).Set(status)
}
