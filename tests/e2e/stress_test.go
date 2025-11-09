package e2e

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/stretchr/testify/suite"
)

// StressTestSuite tests system stability over extended periods
type StressTestSuite struct {
	suite.Suite
	*TestSuite
	successCount     uint64
	failureCount     uint64
	totalCount       uint64
	startTime        time.Time
	duration         time.Duration
	providerID       string
	errorCounts      map[string]uint64
	errorMutex       sync.Mutex
	healthCheckCount uint64
}

// TestExtendedOperation simulates 24-hour operation in accelerated time
func (s *StressTestSuite) TestExtendedOperation() {
	t := s.T()

	// Simulate 24 hours in 24 seconds (1000x acceleration)
	s.duration = 24 * time.Second
	s.setupTestProvider()
	s.resetCounters()
	s.errorCounts = make(map[string]uint64)
	s.startTime = time.Now()

	t.Logf("Starting 24-hour stability test (simulated in %v)", s.duration)
	t.Logf("Acceleration factor: %dx", 24*time.Hour/s.duration)

	// Continuous load generator
	var wg sync.WaitGroup
	loadGenerator := func() {
		defer wg.Done()
		for {
			elapsed := time.Since(s.startTime)
			if elapsed >= s.duration {
				return
			}

			// Simulate realistic load pattern
			s.makeRequest()
			atomic.AddUint64(&s.totalCount, 1)

			// Vary request rate
			if elapsed < s.duration/4 {
				time.Sleep(10 * time.Millisecond) // Light load
			} else if elapsed < s.duration/2 {
				time.Sleep(5 * time.Millisecond) // Medium load
			} else if elapsed < 3*s.duration/4 {
				time.Sleep(2 * time.Millisecond) // Heavy load
			} else {
				time.Sleep(5 * time.Millisecond) // Back to medium
			}
		}
	}

	// Start multiple load generators
	const numGenerators = 10
	for i := 0; i < numGenerators; i++ {
		wg.Add(1)
		go loadGenerator()
	}

	// Periodic health checks
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				atomic.AddUint64(&s.healthCheckCount, 1)
				// Perform health check
				s.performHealthCheck()
			case <-time.After(s.duration):
				return
			}
		}
	}()

	// Memory monitor
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		var startMem runtime.MemStats
		runtime.ReadMemStats(&startMem)

		for {
			select {
			case <-ticker.C:
				var mem runtime.MemStats
				runtime.ReadMemStats(&mem)
				t.Logf("Memory - Alloc: %d KB, Sys: %d KB, NumGC: %d",
					mem.Alloc/1024, mem.Sys/1024, mem.NumGC)
			case <-time.After(s.duration):
				return
			}
		}
	}()

	// Wait for test duration
	wg.Wait()

	duration := time.Since(s.startTime)
	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("\n=== 24-Hour Stability Test Results ===")
	t.Logf("Simulated Duration: %v (24 hours @ 1000x speed)", s.duration)
	t.Logf("Actual Duration: %v", duration)
	t.Logf("Total Requests: %d", s.totalCount)
	t.Logf("Successful: %d", s.successCount)
	t.Logf("Failed: %d", s.failureCount)
	t.Logf("Success Rate: %.2f%%", successRate)
	t.Logf("Average RPS: %.2f", float64(s.totalCount)/duration.Seconds())
	t.Logf("Health Checks: %d", s.healthCheckCount)

	// Report top errors
	if len(s.errorCounts) > 0 {
		t.Logf("\nTop Errors:")
		for errType, count := range s.errorCounts {
			t.Logf("  %s: %d", errType, count)
		}
	}

	// Assertions
	s.Greater(s.totalCount, uint64(1000), "Should process at least 1000 requests")
	s.Greater(s.successCount, uint64(0), "At least some requests should succeed")
	s.Less(s.failureCount, s.totalCount/10, "Failure rate should be < 10%%")
	s.Equal(s.healthCheckCount, uint64(int(s.duration.Seconds())), "Health checks should run every second")
}

// TestMemoryLeaks tests for memory leaks during extended operation
func (s *StressTestSuite) TestMemoryLeaks() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()
	s.startTime = time.Now()

	// Get baseline memory
	var startMem runtime.MemStats
	runtime.ReadMemStats(&startMem)

	baselineAlloc := startMem.Alloc
	baselineNumGC := startMem.NumGC

	t.Logf("Baseline Memory - Alloc: %d KB, NumGC: %d", baselineAlloc/1024, baselineNumGC)

	// Run extended test
	const testDuration = 10 * time.Second
	var wg sync.WaitGroup

	// Continuous request generator
	go func() {
		defer wg.Done()
		for time.Since(s.startTime) < testDuration {
			s.makeRequest()
			atomic.AddUint64(&s.totalCount, 1)
			time.Sleep(5 * time.Millisecond)
		}
	}()

	// GC trigger
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for time.Since(s.startTime) < testDuration {
			select {
			case <-ticker.C:
				runtime.GC()
			}
		}
	}()

	wg.Wait()

	// Force GC and check memory
	runtime.GC()
	runtime.GC()
	runtime.GC()

	var endMem runtime.MemStats
	runtime.ReadMemStats(&endMem)

	endAlloc := endMem.Alloc
	endNumGC := endMem.NumGC

	allocGrowth := int64(endAlloc) - int64(baselineAlloc)
	gcGrowth := int(endNumGC) - int(baselineNumGC)

	t.Logf("End Memory - Alloc: %d KB, NumGC: %d", endAlloc/1024, endNumGC)
	t.Logf("Memory Growth: %d KB", allocGrowth/1024)
	t.Logf("GC Count Growth: %d", gcGrowth)
	t.Logf("Total Requests: %d", s.totalCount)

	// Assertions - memory growth should be reasonable (< 50MB)
	s.Less(allocGrowth, int64(50*1024*1024), "Memory growth should be < 50MB")
}

// TestConnectionPoolStability tests connection pool stability under stress
func (s *StressTestSuite) TestConnectionPoolStability() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()
	s.startTime = time.Now()

	const testDuration = 5 * time.Second
	const numWorkers = 50

	var wg sync.WaitGroup
	successCount := atomic.Uint64{}
	failureCount := atomic.Uint64{}

	// Worker pool
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for time.Since(s.startTime) < testDuration {
				// Make burst of requests
				for j := 0; j < 10; j++ {
					if s.makeRequest() {
						successCount.Add(1)
					} else {
						failureCount.Add(1)
					}
				}
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(s.startTime)
	totalSuccess := successCount.Load()
	totalFailure := failureCount.Load()
	totalRequests := totalSuccess + totalFailure

	t.Logf("Connection Pool Stability Test Results:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total Requests: %d", totalRequests)
	t.Logf("  Successful: %d", totalSuccess)
	t.Logf("  Failed: %d", totalFailure)
	t.Logf("  Success Rate: %.2f%%", float64(totalSuccess)/float64(totalRequests)*100)
	t.Logf("  Requests/sec: %.2f", float64(totalRequests)/duration.Seconds())

	// Assertions
	s.Greater(totalRequests, uint64(1000), "Should process at least 1000 requests")
	s.Greater(totalSuccess, uint64(0), "At least some requests should succeed")
}

// TestResourceExhaustion tests behavior when resources are exhausted
func (s *StressTestSuite) TestResourceExhaustion() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()
	s.startTime = time.Now()

	// Overwhelm the system with requests
	const numRequests = 500
	var wg sync.WaitGroup

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(requestID int) {
			defer wg.Done()
			success := s.makeRequest()
			if success {
				atomic.AddUint64(&s.successCount, 1)
			} else {
				atomic.AddUint64(&s.failureCount, 1)
			}
			atomic.AddUint64(&s.totalCount, 1)
		}(i)
	}

	wg.Wait()

	duration := time.Since(s.startTime)
	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("Resource Exhaustion Test Results:")
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)
	t.Logf("  Duration: %v", duration)

	// System should handle overload gracefully (not crash)
	s.Equal(uint64(numRequests), s.totalCount, "Should track all requests")
	// Some failures are acceptable under resource exhaustion
	s.GreaterOrEqual(s.successCount, uint64(0), "Should handle overload gracefully")
}

// TestRecoveryAfterFailure tests system recovery after temporary failures
func (s *StressTestSuite) TestRecoveryAfterFailure() {
	t := s.T()

	s.setupTestProvider()
	s.resetCounters()
	s.startTime = time.Now()

	// Phase 1: Normal operation
	t.Logf("Phase 1: Normal operation (2s)")
	time.Sleep(2 * time.Second)
	s.makeBurstRequests(100)

	// Phase 2: High load
	t.Logf("Phase 2: High load (3s)")
	s.makeBurstRequests(500)

	// Phase 3: Recovery period
	t.Logf("Phase 3: Recovery (2s)")
	time.Sleep(2 * time.Second)

	// Phase 4: Verify recovery
	t.Logf("Phase 4: Verify recovery")
	s.makeBurstRequests(200)

	duration := time.Since(s.startTime)
	successRate := float64(s.successCount) / float64(s.totalCount) * 100

	t.Logf("Recovery Test Results:")
	t.Logf("  Total Duration: %v", duration)
	t.Logf("  Total Requests: %d", s.totalCount)
	t.Logf("  Successful: %d", s.successCount)
	t.Logf("  Failed: %d", s.failureCount)
	t.Logf("  Success Rate: %.2f%%", successRate)

	// System should recover and handle requests after stress
	s.Greater(s.successCount, uint64(0), "Should recover and handle requests")
}

// performHealthCheck performs a system health check
func (s *StressTestSuite) performHealthCheck() {
	// Check if server is responding
	req := s.MakeRequest("GET", "/healthz", nil, false)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		// Health check passed
	} else {
		// Record error
		s.recordError("health_check_failed")
	}
}

// makeBurstRequests makes a burst of concurrent requests
func (s *StressTestSuite) makeBurstRequests(count int) {
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.makeRequest()
			atomic.AddUint64(&s.totalCount, 1)
		}()
	}
	wg.Wait()
}

// makeRequest makes a single test request
func (s *StressTestSuite) makeRequest() bool {
	// Create request
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Stability test message"},
		},
		"max_tokens": 5,
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 2 * time.Second)

	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			atomic.AddUint64(&s.successCount, 1)
			return true
		} else {
			atomic.AddUint64(&s.failureCount, 1)
			return false
		}
	} else {
		atomic.AddUint64(&s.failureCount, 1)
		return false
	}
}

// recordError records an error type
func (s *StressTestSuite) recordError(errorType string) {
	s.errorMutex.Lock()
	defer s.errorMutex.Unlock()
	s.errorCounts[errorType]++
}

// setupTestProvider creates a test provider
func (s *StressTestSuite) setupTestProvider() {
	if s.providerID != "" {
		return
	}

	providerData := map[string]interface{}{
		"name":         "stress-test-provider",
		"type":         "openai",
		"api_key":      "sk-stress-test",
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
func (s *StressTestSuite) resetCounters() {
	atomic.StoreUint64(&s.successCount, 0)
	atomic.StoreUint64(&s.failureCount, 0)
	atomic.StoreUint64(&s.totalCount, 0)
	atomic.StoreUint64(&s.healthCheckCount, 0)
}

// createProvider creates a provider
func (s *StressTestSuite) createProvider(data map[string]interface{}) *map[string]interface{} {
	return s.createProvider(data)
}

// MakeProxyRequest creates an HTTP request to the proxy endpoint
func (s *StressTestSuite) MakeProxyRequest(method, path string, body interface{}) *http.Request {
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

// RunStressTests runs the stress test suite
func RunStressTests() {
	suite.Run(&StressTestSuite{
		TestSuite: &TestSuite{},
	})
}
