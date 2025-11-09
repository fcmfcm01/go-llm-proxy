package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// MetricsHandler handles Prometheus metrics endpoint requests
type MetricsHandler struct {
	logger *logrus.Logger
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(logger *logrus.Logger) *MetricsHandler {
	return &MetricsHandler{
		logger: logger,
	}
}

// HandleMetrics handles GET /metrics
func (h *MetricsHandler) HandleMetrics(c *gin.Context) {
	// Prometheus metrics endpoint
	// This uses the Prometheus HTTP handler which formats metrics
	// according to the Prometheus text format
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

// HandleMetricsConfig handles GET /metrics/config (metadata about metrics)
func (h *MetricsHandler) HandleMetricsConfig(c *gin.Context) {
	// Return metadata about available metrics
	c.JSON(http.StatusOK, gin.H{
		"metrics": gin.H{
			"llm_proxy_requests_total": gin.H{
				"type": "counter",
				"help": "Total number of requests processed",
			},
			"llm_proxy_errors_total": gin.H{
				"type": "counter",
				"help": "Total number of errors",
			},
			"llm_proxy_request_duration_seconds": gin.H{
				"type": "histogram",
				"help": "Duration of requests in seconds",
			},
			"llm_proxy_requests_in_flight": gin.H{
				"type": "gauge",
				"help": "Current number of requests being processed",
			},
			"llm_proxy_provider_requests_total": gin.H{
				"type": "counter",
				"help": "Total requests to each provider",
				"labels": []string{"provider", "status_code"},
			},
			"llm_proxy_provider_request_duration_seconds": gin.H{
				"type": "histogram",
				"help": "Request duration by provider",
				"labels": []string{"provider"},
			},
			"llm_proxy_provider_errors_total": gin.H{
				"type": "counter",
				"help": "Total errors by provider",
				"labels": []string{"provider", "error_type"},
			},
			"llm_proxy_response_size_bytes": gin.H{
				"type": "histogram",
				"help": "Size of responses in bytes",
			},
			"llm_proxy_prompt_tokens": gin.H{
				"type": "histogram",
				"help": "Number of prompt tokens",
			},
			"llm_proxy_completion_tokens": gin.H{
				"type": "histogram",
				"help": "Number of completion tokens",
			},
			"llm_proxy_total_tokens": gin.H{
				"type": "histogram",
				"help": "Total number of tokens (prompt + completion)",
			},
			"llm_proxy_active_users": gin.H{
				"type": "gauge",
				"help": "Number of active users",
			},
			"llm_proxy_load_balanced_requests_total": gin.H{
				"type": "counter",
				"help": "Total number of load balanced requests",
			},
			"llm_proxy_failed_over_requests_total": gin.H{
				"type": "counter",
				"help": "Total number of requests that failed over to backup providers",
			},
			"llm_proxy_retried_requests_total": gin.H{
				"type": "counter",
				"help": "Total number of retried requests",
			},
			"llm_proxy_health_healthy_providers": gin.H{
				"type": "gauge",
				"help": "Number of healthy providers",
			},
			"llm_proxy_health_unhealthy_providers": gin.H{
				"type": "gauge",
				"help": "Number of unhealthy providers",
			},
			"llm_proxy_health_checks_total": gin.H{
				"type": "counter",
				"help": "Total health checks performed",
				"labels": []string{"provider", "result"},
			},
			"llm_proxy_health_response_time_seconds": gin.H{
				"type": "histogram",
				"help": "Provider response times for health checks",
				"labels": []string{"provider"},
			},
		},
	})
}
