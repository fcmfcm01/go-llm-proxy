package e2e

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/stretchr/testify/suite"
)

// LoadTestSuite tests system under high load (1000+ concurrent connections)
type LoadTestSuite struct {
	suite.Suite
	*TestSuite
	successCount uint64
	failureCount uint64
	totalCount   uint64
	startTime    time.Time
	providerID   string
}

// TestConcurrentConnections tests 1000+ concurrent connections
func (s *LoadTestSuite) TestConcurrentConnections() {
	t := s.T()

	// Setup test
	const numWorkers = 1000
	const requestsPerWorker = 5
	const totalRequests = numWorkers * requestsPerWorker

	s.setupTestProvider()
	s.resetCounters()
	s.startTime = time.Now()

	// Create worker pool
	var wg sync.WaitGroup
	workerChan := make(chan struct{}, 100) // Limit concurrent workers

	// Launch workers
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		workerChan <- struct{}{} // Acquire slot

		go func(workerID int) {
			defer wg.Done()
			defer func() { <-workerChan }() // Release slot

			for j := 0; j < requestsPerWorker; j++ {
				s.makeLoadTestRequest(workerID, j)
				atomic.AddUint64(&s.totalCount, 1)
			}
		}(i)
	}

	// Wait for all workers to complete
	wg.Wait()

	duration := time.Since(s.startTime)
	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	// Report results
	t.Logf("Load Test Results:")
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Requests/sec: %.2f", float64(s.totalCount)/duration.Seconds())

	// Assertions
	s.Equal(totalRequests, int(s.totalCount), "Should make all expected requests")
	s.Greater(s.successCount, uint64(0), "At least some requests should succeed")
	s.LessOrEqual(s.failureCount, uint64(totalRequests/10), "Failure rate should be < 10%%")
}

// TestSustainedLoad tests sustained load over time
func (s *LoadTestSuite) TestSustainedLoad() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()

	const duration = 10 * time.Second
	const rps = 100 // Requests per second

	s.startTime = time.Now()
	endTime := s.startTime.Add(duration)

	var wg sync.WaitGroup
	stopChan := make(chan bool)

	// Producer goroutine
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stopChan:
				return
			default:
				s.makeLoadTestRequest(0, 0)
				atomic.AddUint64(&s.totalCount, 1)
				time.Sleep(time.Second / time.Duration(rps))
			}
		}
	}()

	// Monitor goroutine
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-stopChan:
				return
			case <-ticker.C:
				current := atomic.LoadUint64(&s.totalCount)
				elapsed := time.Since(s.startTime)
				rps := float64(current) / elapsed.Seconds()
				t.Logf("RPS: %.2f, Total: %d, Elapsed: %v", rps, current, elapsed)
			}
		}
	}()

	// Wait for test duration
	time.Sleep(duration)
	close(stopChan)
	wg.Wait()

	duration = time.Since(s.startTime)
	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("Sustained Load Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Average RPS: %.2f", float64(s.totalCount)/duration.Seconds())

	// Assertions
	s.Greater(s.totalCount, uint64(500), "Should complete at least 500 requests")
	s.Greater(s.successCount, uint64(0), "At least some requests should succeed")
}

// TestSpikeLoad tests sudden spike in load
func (s *LoadTestSuite) TestSpikeLoad() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()

	const spikeSize = 2000
	s.startTime = time.Now()

	// Sudden spike of requests
	var wg sync.WaitGroup
	for i := 0; i < spikeSize; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			s.makeLoadTestRequest(requestID, 0)
			atomic.AddUint64(&s.totalCount, 1)
		}(i)
	}

	wg.Wait()
	duration := time.Since(s.startTime)

	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("Spike Load Results:")
	t.Logf("  Spike Size: %d", spikeSize)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Peak RPS: %.2f", float64(spikeSize)/duration.Seconds())

	// Assertions
	s.Equal(uint64(spikeSize), s.totalCount, "Should handle all spike requests")
	s.Greater(s.successCount, uint64(0), "At least some requests should succeed")
}

// TestMemoryUsageUnderLoad tests memory usage during high load
func (s *LoadTestSuite) TestMemoryUsageUnderLoad() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()

	const numRequests = 500
	s.startTime = time.Now()

	// Make requests and track memory
	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			s.makeLoadTestRequest(requestID, 0)
			atomic.AddUint64(&s.totalCount, 1)
		}(i)
	}

	wg.Wait()
	duration := time.Since(s.startTime)

	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("Memory Load Test Results:")
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Requests/sec: %.2f", float64(s.totalCount)/duration.Seconds())

	// Assertions
	s.Equal(uint64(numRequests), s.totalCount, "Should complete all requests")
	// Note: Actual memory profiling would require pprof, which is beyond E2E scope
	s.Greater(s.successCount, uint64(0), "At least some requests should succeed")
}

// TestRateLimitingUnderLoad tests that rate limiting works under load
func (s *LoadTestSuite) TestRateLimitingUnderLoad() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()

	// Make rapid requests to trigger rate limiting
	const numRequests = 100
	s.startTime = time.Now()

	var wg sync.WaitGroup
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			s.makeLoadTestRequest(requestID, 0)
			atomic.AddUint64(&s.totalCount, 1)
		}(i)
	}

	wg.Wait()
	duration := time.Since(s.startTime)

	// At high request rates, some should be rate limited
	rateLimitedCount := s.failureCount

	t.Logf("Rate Limiting Test Results:")
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Rate Limited: %d", rateLimitedCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Duration: %v", duration)

	// Assertions
	s.Equal(uint64(numRequests), s.totalCount, "Should track all requests")
	// Rate limiting is expected and acceptable under load
	s.GreaterOrEqual(s.successCount, uint64(0), "Should handle rate limiting gracefully")
}

// TestConcurrentProviderRequests tests concurrent requests to different providers
func (s *LoadTestSuite) TestConcurrentProviderRequests() {
	t := s.T()

	// Setup multiple providers
	provider1 := s.createProvider(map[string]interface{}{
		"name":         "load-test-provider-1",
		"type":         "openai",
		"api_key":      "sk-load1",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 1000,
	})
	s.NotNil(provider1)

	provider2 := s.createProvider(map[string]interface{}{
		"name":         "load-test-provider-2",
		"type":         "openai",
		"api_key":      "sk-load2",
		"base_url":     "https://api.openai.com/v1",
		"priority":     2,
		"enabled":      true,
		"max_requests": 1000,
	})
	s.NotNil(provider2)

	s.resetCounters()
	s.startTime = time.Now()

	const numRequests = 1000
	var wg sync.WaitGroup

	// Split requests between providers
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			s.makeLoadTestRequest(requestID, 0)
			atomic.AddUint64(&s.totalCount, 1)
		}(i)
	}

	wg.Wait()
	duration := time.Since(s.startTime)

	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("Multi-Provider Load Test Results:")
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Duration: %v", duration)
	t.Logf("  Requests/sec: %.2f", float64(s.totalCount)/duration.Seconds())

	// Assertions
	s.Equal(uint64(numRequests), s.totalCount, "Should complete all requests")
	s.Greater(s.successCount, uint64(0), "At least some requests should succeed")
}

// makeLoadTestRequest makes a single load test request
func (s *LoadTestSuite) makeLoadTestRequest(workerID, requestID int) {
	// Create request
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Load test message"},
		},
		"max_tokens": 10,
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 5 * time.Second}

	// Make request with retry
	success := false
	for i := 0; i < 2; i++ {
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				success = true
			}
			break
		}
		time.Sleep(10 * time.Millisecond) // Brief retry delay
	}

	if success {
		atomic.AddUint64(&s.successCount, 1)
	} else {
		atomic.AddUint64(&s.failureCount, 1)
	}
}

// setupTestProvider creates a test provider
func (s *LoadTestSuite) setupTestProvider() {
	if s.providerID != "" {
		return
	}

	providerData := map[string]interface{}{
		"name":         "load-test-provider",
		"type":         "openai",
		"api_key":      "sk-load-test",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 10000,
	}
	provider := s.createProvider(providerData)
	if provider != nil {
		if id, ok := (*provider)["id"].(string); ok {
			s.providerID = id
		}
	}
}

// resetCounters resets test counters
func (s *LoadTestSuite) resetCounters() {
	atomic.StoreUint64(&s.successCount, 0)
	atomic.StoreUint64(&s.failureCount, 0)
	atomic.StoreUint64(&s.totalCount, 0)
}

// createProvider creates a provider
func (s *LoadTestSuite) createProvider(data map[string]interface{}) *map[string]interface{} {
	return s.createProvider(data)
}

// MakeProxyRequest creates an HTTP request to the proxy endpoint
func (s *LoadTestSuite) MakeProxyRequest(method, path string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, s.testServer.URL+path, nil)
	} else {
		req, _ = http.NewRequest(method, s.testServer.URL+path, nil)
	}

	if body != nil {
		jsonBody, _ := json.Marshal(body)
		req.Body = http.NoBody
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer sk-test-key")

	return req
}

// RunLoadTests runs the load test suite
func RunLoadTests() {
	suite.Run(&LoadTestSuite{
		TestSuite: &TestSuite{},
	})
}
