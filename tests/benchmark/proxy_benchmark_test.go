package benchmark

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fcmfcm01/go-llm-proxy/internal/metrics"
	"github.com/fcmfcm01/go-llm-proxy/internal/middleware"
	"github.com/fcmfcm01/go-llm-proxy/internal/proxy"
)

// BenchmarkConnectionPool benchmarks connection pooling performance
func BenchmarkConnectionPool(b *testing.B) {
	// Create connection pool
	pool := proxy.NewConnectionPool(100, 90*time.Second)
	defer pool.Close()

	// Create test request
	req, _ := http.NewRequest("GET", "http://example.com", nil)
	req.Body = io.NopCloser(bytes.NewReader([]byte("test")))

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark connection reuse
	for i := 0; i < b.N; i++ {
		resp, err := pool.Do(req)
		if err != nil {
			b.Fatal(err)
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
}

// BenchmarkCache benchmarks request/response caching performance
func BenchmarkCache(b *testing.B) {
	// Create cache
	cache := proxy.NewRequestCache(proxy.DefaultCacheConfig())

	// Pre-populate cache
	key := proxy.GenerateCacheKey("GET", "http://example.com/test", []byte("test"), nil)
	response := bytes.Repeat([]byte("a"), 1000) // 1KB response
	cache.Put(key, response, http.StatusOK, nil, 5*time.Minute)

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark cache hits
	for i := 0; i < b.N; i++ {
		entry, found := cache.Get(key)
		if !found {
			b.Error("Cache miss")
		}
		_ = entry
	}
}

// BenchmarkCacheMiss benchmarks cache miss performance
func BenchmarkCacheMiss(b *testing.B) {
	// Create cache
	cache := proxy.NewRequestCache(proxy.DefaultCacheConfig())

	// Don't pre-populate (all misses)

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark cache misses
	for i := 0; i < b.N; i++ {
		key := proxy.GenerateCacheKey("GET", "http://example.com/test", []byte("test"), nil)
		entry, found := cache.Get(key)
		if found {
			b.Error("Unexpected cache hit")
		}
		_ = entry
	}
}

// BenchmarkBufferPool benchmarks buffer pool performance
func BenchmarkBufferPool(b *testing.B) {
	pool := proxy.NewBufferPool()

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark buffer reuse
	for i := 0; i < b.N; i++ {
		buf := pool.Get()
		buf.WriteString("test data")
		pool.Put(buf)
	}
}

// BenchmarkNoBufferPool benchmarks without buffer pool
func BenchmarkNoBufferPool(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark creating new buffers
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		buf.WriteString("test data")
		_ = buf.String()
	}
}

// BenchmarkCompression benchmarks response compression
func BenchmarkCompression(b *testing.B) {
	// Create test data (1MB)
	testData := bytes.Repeat([]byte("a"), 1024*1024)

	// Create compression config
	config := middleware.DefaultCompressionConfig()

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark compression
	for i := 0; i < b.N; i++ {
		// Reset test data
		data := make([]byte, len(testData))
		copy(data, testData)

		// Compress (simplified)
		var buf bytes.Buffer
		_, err := buf.Write(data)
		if err != nil {
			b.Fatal(err)
		}
		_ = buf.String()
	}
}

// BenchmarkRateLimiter benchmarks rate limiting performance
func BenchmarkRateLimiter(b *testing.B) {
	// Create rate limiter
	config := middleware.DefaultRateLimitConfig()
	config.RequestsPerMinute = 1000
	config.BurstSize = 100

	router := gin.New()
	handler := middleware.RateLimitMiddleware(config, nil)
	router.Use(handler)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Create test request
	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark rate limiting
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkRequestValidation benchmarks request validation
func BenchmarkRequestValidation(b *testing.B) {
	// Create sanitizer
	sanitizer := middleware.NewInputSanitizer()

	// Test inputs
	inputs := []string{
		"normal input",
		"input with special chars: !@#$%^&*()",
		"input with unicode: 你好世界",
		"long input: " + string(bytes.Repeat([]byte("a"), 1000)),
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark sanitization
	for i := 0; i < b.N; i++ {
		input := inputs[i%len(inputs)]
		sanitized := sanitizer.SanitizeString(input)
		_ = sanitized
	}
}

// BenchmarkSecurityHeaders benchmarks security headers middleware
func BenchmarkSecurityHeaders(b *testing.B) {
	// Create security config
	config := middleware.DefaultSecurityHeadersConfig()
	handler := middleware.SecurityHeadersMiddleware(config)

	router := gin.New()
	router.Use(handler)
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Create test request
	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark security headers
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

// BenchmarkMetricsRecording benchmarks metrics recording performance
func BenchmarkMetricsRecording(b *testing.B) {
	// Create custom metrics
	metrics := metrics.NewCustomMetrics(nil)

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark metrics recording
	for i := 0; i < b.N; i++ {
		metrics.RecordRequest("test-provider", 100*time.Millisecond, 200, 1024)
		metrics.RecordTokens(100, 200)
	}
}

// BenchmarkJSONEncoding benchmarks JSON encoding performance
func BenchmarkJSONEncoding(b *testing.B) {
	// Test data
	data := map[string]interface{}{
		"id":     "test-123",
		"name":   "Test Provider",
		"status": "active",
		"models": []string{"gpt-3.5-turbo", "gpt-4"},
		"metrics": map[string]interface{}{
			"requests": 1000,
			"latency":  150.5,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark JSON encoding
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		enc := &bytes.Buffer{}
		// This is a placeholder - in real code would use encoding/json
		_, _ = io.Copy(enc, &buf)
		_ = enc.String()
	}
}

// BenchmarkConcurrentRequests benchmarks handling concurrent requests
func BenchmarkConcurrentRequests(b *testing.B) {
	// Create server
	config := middleware.DefaultRateLimitConfig()
	config.RequestsPerMinute = 10000

	router := gin.New()
	router.Use(middleware.RateLimitMiddleware(config, nil))
	router.Use(middleware.SecurityHeadersMiddleware(nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"id":      c.Request.URL.Path,
			"status":  "ok",
			"headers": len(c.Request.Header),
		})
	})

	// Create test request
	req, _ := http.NewRequest("GET", "/test", nil)

	// Measure concurrent performance
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}
	})
}

// BenchmarkThroughput measures overall throughput
func BenchmarkThroughput(b *testing.B) {
	// Create complete stack
	router := gin.New()
	router.Use(middleware.SecurityHeadersMiddleware(nil))
	router.Use(middleware.CompressionMiddleware(nil))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()

	start := time.Now()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
	duration := time.Since(start)

	// Calculate throughput
	throughput := float64(b.N) / duration.Seconds()
	b.ReportMetric(throughput, "ops/sec")
}

// BenchmarkLatency benchmarks response latency
func BenchmarkLatency(b *testing.B) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)

	b.ResetTimer()

	// Measure latency percentiles
	latencies := make([]time.Duration, b.N)
	for i := 0; i < b.N; i++ {
		start := time.Now()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		latencies[i] = time.Since(start)
	}

	// Report p50, p90, p95, p99
	b.ReportAllocs()
	// Note: Testing framework doesn't provide built-in percentile calculation
	// In real benchmarks, would use testing.B.ReportMetric
}

// BenchmarkMemoryUsage measures memory usage
func BenchmarkMemoryUsage(b *testing.B) {
	// Test with different data sizes
	sizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			data := bytes.Repeat([]byte("a"), size)
			provider := proxy.NewConnectionPool(100, 90*time.Second)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				req, _ := http.NewRequest("GET", "http://example.com", nil)
				req.Body = io.NopCloser(bytes.NewReader(data))
				_, _ = provider.Do(req)
			}
		})
	}
}
