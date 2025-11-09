package metrics

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// PrometheusExporter handles Prometheus metrics export
type PrometheusExporter struct {
	logger          *logrus.Logger
	metricsRegistry *prometheus.Registry
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(logger *logrus.Logger) *PrometheusExporter {
	// Create a custom registry
	registry := prometheus.NewRegistry()

	// Register default Go metrics
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	return &PrometheusExporter{
		logger:          logger,
		metricsRegistry: registry,
	}
}

// Handler returns the HTTP handler for Prometheus metrics
func (e *PrometheusExporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.metricsRegistry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// RegisterMetrics registers additional metrics to the exporter
func (e *PrometheusExporter) RegisterMetrics(collectors ...prometheus.Collector) {
	for _, collector := range collectors {
		if err := e.metricsRegistry.Register(collector); err != nil {
			e.logger.WithError(err).Error("Failed to register Prometheus metric")
		}
	}
}

// HTTPHandler creates a Gin handler for metrics endpoint
func (e *PrometheusExporter) HTTPHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		e.Handler().ServeHTTP(c.Writer, c.Request)
	}
}
