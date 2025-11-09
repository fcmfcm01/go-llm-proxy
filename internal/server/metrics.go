package server

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Counter is a Prometheus counter
type Counter struct {
	metric prometheus.Counter
}

// Gauge is a Prometheus gauge
type Gauge struct {
	metric prometheus.Gauge
}

// Histogram is a Prometheus histogram
type Histogram struct {
	metric prometheus.Histogram
}

// Metrics holds all server metrics
type Metrics struct {
	mu sync.RWMutex

	// Request metrics
	TotalRequests   *Counter
	ActiveRequests  *Gauge
	RequestDuration *Histogram
	RequestSize     *Histogram
	ResponseSize    *Histogram

	// Error metrics
	Errors       *Counter
	ClientErrors *Counter
	ServerErrors *Counter

	// Response metrics by status code
	StatusCodes map[int]*Counter

	// HTTP methods
	Methods map[string]*Counter

	// Paths
	Paths map[string]*Counter

	// Latency buckets
	LatencyBuckets []float64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	latencyBuckets := []float64{
		0.001, // 1ms
		0.005, // 5ms
		0.01,  // 10ms
		0.025, // 25ms
		0.05,  // 50ms
		0.1,   // 100ms
		0.25,  // 250ms
		0.5,   // 500ms
		1.0,   // 1s
		2.5,   // 2.5s
		5.0,   // 5s
		10.0,  // 10s
	}

	return &Metrics{
		TotalRequests: &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "requests_total",
				Help:      "Total number of HTTP requests",
			}),
		},
		ActiveRequests: &Gauge{
			metric: promauto.NewGauge(prometheus.GaugeOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "requests_active",
				Help:      "Number of active HTTP requests",
			}),
		},
		RequestDuration: &Histogram{
			metric: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds",
				Buckets:   latencyBuckets,
			}),
		},
		RequestSize: &Histogram{
			metric: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "request_size_bytes",
				Help:      "Size of HTTP requests in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			}),
		},
		ResponseSize: &Histogram{
			metric: promauto.NewHistogram(prometheus.HistogramOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "response_size_bytes",
				Help:      "Size of HTTP responses in bytes",
				Buckets:   prometheus.ExponentialBuckets(100, 10, 7),
			}),
		},
		Errors: &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "errors_total",
				Help:      "Total number of HTTP errors",
			}),
		},
		ClientErrors: &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "client_errors_total",
				Help:      "Total number of HTTP client errors (4xx)",
			}),
		},
		ServerErrors: &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace: "llm_proxy",
				Subsystem: "server",
				Name:      "server_errors_total",
				Help:      "Total number of HTTP server errors (5xx)",
			}),
		},
		StatusCodes:    make(map[int]*Counter),
		Methods:        make(map[string]*Counter),
		Paths:          make(map[string]*Counter),
		LatencyBuckets: latencyBuckets,
	}
}

// Inc increments the counter
func (c *Counter) Inc() {
	c.metric.Inc()
}

// Dec decrements the gauge
func (g *Gauge) Dec() {
	g.metric.Dec()
}

// Inc increments the gauge
func (g *Gauge) Inc() {
	g.metric.Inc()
}

// Observe adds an observation to the histogram
func (h *Histogram) Observe(value float64) {
	h.metric.Observe(value)
}

// RecordRequest records a request
func (m *Metrics) RecordRequest(method, path string, statusCode int, duration time.Duration, requestSize, responseSize int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Total requests
	m.TotalRequests.Inc()

	// Request duration
	m.RequestDuration.Observe(duration.Seconds())

	// Request size
	if requestSize > 0 {
		m.RequestSize.Observe(float64(requestSize))
	}

	// Response size
	if responseSize > 0 {
		m.ResponseSize.Observe(float64(responseSize))
	}

	// Status code
	if counter, ok := m.StatusCodes[statusCode]; ok {
		counter.Inc()
	} else {
		counter = &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace:   "llm_proxy",
				Subsystem:   "server",
				Name:        "requests_by_status_code",
				Help:        "Number of HTTP requests by status code",
				ConstLabels: prometheus.Labels{"status_code": string(rune(statusCode))},
			}),
		}
		m.StatusCodes[statusCode] = counter
		counter.Inc()
	}

	// HTTP method
	if counter, ok := m.Methods[method]; ok {
		counter.Inc()
	} else {
		counter = &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace:   "llm_proxy",
				Subsystem:   "server",
				Name:        "requests_by_method",
				Help:        "Number of HTTP requests by method",
				ConstLabels: prometheus.Labels{"method": method},
			}),
		}
		m.Methods[method] = counter
		counter.Inc()
	}

	// Path
	if counter, ok := m.Paths[path]; ok {
		counter.Inc()
	} else {
		counter = &Counter{
			metric: promauto.NewCounter(prometheus.CounterOpts{
				Namespace:   "llm_proxy",
				Subsystem:   "server",
				Name:        "requests_by_path",
				Help:        "Number of HTTP requests by path",
				ConstLabels: prometheus.Labels{"path": path},
			}),
		}
		m.Paths[path] = counter
		counter.Inc()
	}

	// Errors
	if statusCode >= 400 && statusCode < 500 {
		m.ClientErrors.Inc()
	} else if statusCode >= 500 {
		m.ServerErrors.Inc()
	}
}

// setupMetricsRoutes sets up metrics endpoints
func (s *Server) setupMetricsRoutes() {
	s.router.GET("/metrics", gin.HandlerFunc(func(c *gin.Context) {
		// Use default Prometheus handler
		promhttp.Handler().ServeHTTP(c.Writer, c.Request)
	}))
}
