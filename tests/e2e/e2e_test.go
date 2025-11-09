package e2e

import (
	"testing"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/tests/testutil"
)

// TestAll runs all E2E test suites
func TestAll(t *testing.T) {
	// Run all E2E test suites

	// Test Suite 1: Authentication Flow
	t.Run("AuthFlow", func(t *testing.T) {
		RunAuthFlowTests()
	})

	// Test Suite 2: Provider Lifecycle
	t.Run("ProviderLifecycle", func(t *testing.T) {
		RunProviderLifecycleTests()
	})

	// Test Suite 3: Proxy Request Flow
	t.Run("ProxyFlow", func(t *testing.T) {
		RunProxyFlowTests()
	})

	// Test Suite 4: Streaming Responses
	t.Run("Streaming", func(t *testing.T) {
		RunStreamingTests()
	})

	// Test Suite 5: Failover Scenarios
	t.Run("Failover", func(t *testing.T) {
		RunFailoverTests()
	})

	// Test Suite 6: Load Testing
	t.Run("Load", func(t *testing.T) {
		RunLoadTests()
	})

	// Test Suite 7: Stress Testing
	t.Run("Stress", func(t *testing.T) {
		RunStressTests()
	})

	// Test Suite 8: Regression Testing
	t.Run("Regression", func(t *testing.T) {
		RunRegressionTests()
	})
}

// BenchmarkAll runs all E2E benchmark tests
func BenchmarkAll(b *testing.B) {
	// Run benchmarks

	b.Run("ConnectionPool", func(b *testing.B) {
		// Benchmark connection pooling
	})

	b.Run("Cache", func(b *testing.B) {
		// Benchmark caching
	})

	b.Run("Throughput", func(b *testing.B) {
		// Benchmark throughput
	})
}

// TestAuthFlow runs only authentication flow tests
func TestAuthFlow(t *testing.T) {
	RunAuthFlowTests()
}

// TestProviderLifecycle runs only provider lifecycle tests
func TestProviderLifecycle(t *testing.T) {
	RunProviderLifecycleTests()
}

// TestProxyFlow runs only proxy flow tests
func TestProxyFlow(t *testing.T) {
	RunProxyFlowTests()
}

// TestStreaming runs only streaming tests
func TestStreaming(t *testing.T) {
	RunStreamingTests()
}

// TestFailover runs only failover tests
func TestFailover(t *testing.T) {
	RunFailoverTests()
}

// TestLoad runs only load tests
func TestLoad(t *testing.T) {
	RunLoadTests()
}

// TestStress runs only stress tests
func TestStress(t *testing.T) {
	RunStressTests()
}

// TestRegression runs only regression tests
func TestRegression(t *testing.T) {
	RunRegressionTests()
}
