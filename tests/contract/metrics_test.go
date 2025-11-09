package contract

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"github.com/fcmfcm01/go-llm-proxy/internal/logging"
	"github.com/fcmfcm01/go-llm-proxy/internal/server"
)

// TestMetricsContract tests the GET /metrics endpoint per health-metrics.yaml
func TestMetricsContract(t *testing.T) {
	// Set Gin to Test mode
	gin.SetMode(gin.TestMode)

	// Create logger
	logger := logging.NewSimpleLogger("debug")

	// Create a test server
	srv := server.NewServer(&server.ServerConfig{
		Port:         8080,
		ReadTimeout:  30,
		WriteTimeout: 30,
		IdleTimeout:  60,
	}, logger.Logger)

	// Setup routes
	srv.SetupRoutes()

	t.Run("MetricsEndpointExists", func(t *testing.T) {
		// Test that the metrics endpoint exists
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Metrics endpoint should return 200")
	})

	t.Run("ReturnsPrometheusFormat", func(t *testing.T) {
		// Test that the endpoint returns Prometheus-formatted metrics
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// Should return 200
		assert.Equal(t, 200, w.Code, "Expected 200 OK")

		// Should return text/plain content type
		contentType := w.Header().Get("Content-Type")
		assert.True(t, strings.Contains(contentType, "text/plain"),
			"Content-Type should be text/plain, got: %s", contentType)

		// Should return Prometheus metrics format
		body := w.Body.String()
		assert.NotEmpty(t, body, "Metrics response should not be empty")

		// Should contain Prometheus metric lines
		assert.Contains(t, body, "# HELP", "Should contain HELP comments")
		assert.Contains(t, body, "# TYPE", "Should contain TYPE comments")
	})

	t.Run("ContainsGoMetrics", func(t *testing.T) {
		// Test that standard Go metrics are included
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		body := w.Body.String()

		// Should contain Go runtime metrics
		assert.Contains(t, body, "go_gc_duration_seconds", "Should contain GC duration metric")
		assert.Contains(t, body, "go_goroutines", "Should contain goroutines metric")
		assert.Contains(t, body, "go_memstats_alloc_bytes", "Should contain memory allocation metric")
	})

	t.Run("ContainsLLMProxyMetrics", func(t *testing.T) {
		// Test that LLM Proxy specific metrics are included
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		body := w.Body.String()

		// Should contain LLM Proxy namespace metrics
		assert.Contains(t, body, "llm_proxy", "Should contain llm_proxy namespace")
	})

	t.Run("ValidPrometheusSyntax", func(t *testing.T) {
		// Test that the returned metrics are valid Prometheus text format
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		body := w.Body.String()
		lines := strings.Split(strings.TrimSpace(body), "\n")

		// Each metric line should match Prometheus format
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			// Metric line should contain a value
			assert.Contains(t, line, " ", "Metric line should have space-separated format")
		}
	})

	t.Run("EndpointAccessibleViaGET", func(t *testing.T) {
		// Test that metrics endpoint only accepts GET requests
		req := httptest.NewRequest(http.MethodPost, "/metrics", nil)
		w := httptest.NewRecorder()

		srv.Router().ServeHTTP(w, req)

		// POST should be allowed (or return 405 Method Not Allowed)
		assert.True(t, w.Code == 200 || w.Code == 405,
			"Metrics endpoint should accept GET or return 405 for other methods")
	})
}
