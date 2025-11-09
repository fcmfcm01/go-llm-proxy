package integration

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAutomaticFailover tests automatic failover to backup providers
func TestAutomaticFailover(t *testing.T) {
	proxy := NewTestProxy()

	t.Run("FailoverOnProviderFailure", func(t *testing.T) {
		// Create multiple upstream servers
		primaryDown := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal server error"}`))
		}))
		defer primaryDown.Close()

		backupUp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "Response from backup provider"
					}
				}]
			}`))
		}))
		defer backupUp.Close()

		// Configure providers
		providers := []Provider{
			{ID: "primary", URL: primaryDown.URL, Priority: 1, Enabled: true},
			{ID: "backup", URL: backupUp.URL, Priority: 2, Enabled: true},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test failover"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should succeed by failing over to backup
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "backup provider")
	})

	t.Run("FailoverWithRetry", func(t *testing.T) {
		callCount := 0
		flaky := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount <= 2 {
				// Fail first 2 attempts
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			// Succeed on 3rd attempt
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Success after retry"}
				}]
			}`))
		}))
		defer flaky.Close()

		providers := []Provider{
			{ID: "flaky", URL: flaky.URL, Priority: 1, Enabled: true, MaxRetries: 3},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test retry"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should succeed after retries
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.GreaterOrEqual(t, callCount, 2, "Should have retried")
	})

	t.Run("AllProvidersDown", func(t *testing.T) {
		down1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer down1.Close()

		down2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadGateway)
		}))
		defer down2.Close()

		providers := []Provider{
			{ID: "p1", URL: down1.URL, Priority: 1, Enabled: true},
			{ID: "p2", URL: down2.URL, Priority: 2, Enabled: true},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should return 502 Bad Gateway when all providers are down
		assert.True(t, rec.Code >= 500)
	})

	t.Run("FailoverChain", func(t *testing.T) {
		// Create a chain of failing providers
		providers := make([]*httptest.Server, 3)
		for i := 0; i < 3; i++ {
			providers[i] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer providers[i].Close()
		}

		// Last provider succeeds
		lastProvider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Final provider success"}
				}]
			}`))
		}))
		defer lastProvider.Close()

		allProviders := []Provider{
			{ID: "p1", URL: providers[0].URL, Priority: 1, Enabled: true},
			{ID: "p2", URL: providers[1].URL, Priority: 2, Enabled: true},
			{ID: "p3", URL: providers[2].URL, Priority: 3, Enabled: true},
			{ID: "p4", URL: lastProvider.URL, Priority: 4, Enabled: true},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test chain"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should eventually succeed
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Final provider success")
	})

	t.Run("FailoverWithTimeout", func(t *testing.T) {
		slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Simulate slow response that times out
			time.Sleep(10 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer slow.Close()

		fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Fast response"}
				}]
			}`))
		}))
		defer fast.Close()

		providers := []Provider{
			{ID: "slow", URL: slow.URL, Priority: 1, Enabled: true, Timeout: 1 * time.Second},
			{ID: "fast", URL: fast.URL, Priority: 2, Enabled: true, Timeout: 5 * time.Second},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test timeout"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should succeed by failing over to fast provider
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Fast response")
	})

	t.Run("FailoverRateLimit", func(t *testing.T) {
		rateLimited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Rate limit exceeded"}`))
		}))
		defer rateLimited.Close()

		normal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Normal response"}
				}]
			}`))
		}))
		defer normal.Close()

		providers := []Provider{
			{ID: "limited", URL: rateLimited.URL, Priority: 1, Enabled: true},
			{ID: "normal", URL: normal.URL, Priority: 2, Enabled: true},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test rate limit"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should failover when rate limited
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Normal response")
	})

	t.Run("FailoverPartialResponse", func(t *testing.T) {
		partial := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Close connection mid-response
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Write([]byte("HTTP/1.1 200 OK\r\n"))
				conn.Write([]byte("Content-Type: application/json\r\n"))
				conn.Write([]byte("\r\n"))
				conn.Write([]byte(`{"choices": [{"message": {"role":"`))
				conn.Close()
			}
		}))
		defer partial.Close()

		complete := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Complete response"}
				}]
			}`))
		}))
		defer complete.Close()

		providers := []Provider{
			{ID: "partial", URL: partial.URL, Priority: 1, Enabled: true},
			{ID: "complete", URL: complete.URL, Priority: 2, Enabled: true},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test partial"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should failover when connection fails
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Complete response")
	})

	t.Run("FailoverCache", func(t *testing.T) {
		down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer down.Close()

		cache := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return cached response
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write([]byte(`{
				"choices": [{
					"message": {"role": "assistant", "content": "Cached response"}
				}]
			}`))
		}))
		defer cache.Close()

		providers := []Provider{
			{ID: "down", URL: down.URL, Priority: 1, Enabled: true},
			{ID: "cache", URL: cache.URL, Priority: 2, Enabled: true},
		}

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test cache"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should use cache when provider is down
		assert.Equal(t, http.StatusOK, rec.Code)
	})
}
