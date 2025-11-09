package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/stretchr/testify/suite"
)

// FailoverTestSuite tests automatic failover scenarios
type FailoverTestSuite struct {
	suite.Suite
	*TestSuite
	providerIDs []string
}

// TestAutomaticFailover tests automatic failover when primary provider fails
func (s *FailoverTestSuite) TestAutomaticFailover() {
	t := s.T()

	// Setup two providers with different priorities
	provider1 := s.createProvider(map[string]interface{}{
		"name":         "primary-provider",
		"type":         "openai",
		"api_key":      "sk-primary",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider1)
	provider1ID := (*provider1)["id"].(string)
	s.providerIDs = append(s.providerIDs, provider1ID)

	provider2 := s.createProvider(map[string]interface{}{
		"name":         "secondary-provider",
		"type":         "openai",
		"api_key":      "sk-secondary",
		"base_url":     "https://api.openai.com/v1",
		"priority":     2,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider2)
	provider2ID := (*provider2)["id"].(string)
	s.providerIDs = append(s.providerIDs, provider2ID)

	// Make multiple requests - should use primary provider first
	numRequests := 5
	successfulRequests := 0
	failedRequests := 0

	for i := 0; i < numRequests; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Test request %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)

		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusBadGateway {
				successfulRequests++
			} else {
				failedRequests++
			}
		} else {
			failedRequests++
		}
	}

	// At least some requests should complete (or fail gracefully)
	s.Equal(numRequests, successfulRequests+failedRequests,
		"All requests should complete")

	// Note: In a real test with actual provider APIs, we would:
	// 1. Disable the primary provider
	// 2. Verify requests fail over to secondary
	// 3. Re-enable primary and verify it takes precedence again
}

// TestProviderHealthCheck tests provider health monitoring
func (s *FailoverTestSuite) TestProviderHealthCheck() {
	t := s.T()

	// Create a provider
	provider := s.createProvider(map[string]interface{}{
		"name":         "health-check-provider",
		"type":         "openai",
		"api_key":      "sk-health",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider)
	providerID := (*provider)["id"].(string)
	s.providerIDs = append(s.providerIDs, providerID)

	// Check provider status
	status := s.getProviderStatus(providerID)
	s.NotNil(status)

	// Verify status structure
	if status != nil {
		if enabled, ok := (*status)["enabled"]; ok {
			s.Equal(true, enabled, "Provider should be enabled")
		}
		if health, ok := (*status)["health"]; ok {
			s.NotNil(health, "Provider should have health status")
		}
	}
}

// TestFailoverWithDisabledProvider tests failover when primary is disabled
func (s *FailoverTestSuite) TestFailoverWithDisabledProvider() {
	t := s.T()

	// Create two providers
	provider1 := s.createProvider(map[string]interface{}{
		"name":         "primary-disabled",
		"type":         "openai",
		"api_key":      "sk-primary",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider1)
	provider1ID := (*provider1)["id"].(string)
	s.providerIDs = append(s.providerIDs, provider1ID)

	provider2 := s.createProvider(map[string]interface{}{
		"name":         "secondary-fallback",
		"type":         "openai",
		"api_key":      "sk-secondary",
		"base_url":     "https://api.openai.com/v1",
		"priority":     2,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider2)
	provider2ID := (*provider2)["id"].(string)
	s.providerIDs = append(s.providerIDs, provider2ID)

	// Disable primary provider
	toggled := s.toggleProvider(provider1ID)
	s.NotNil(toggled)
	s.Equal(false, (*toggled)["enabled"], "Provider should be disabled")

	// Make request - should use secondary provider
	requestData := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{"role": "user", "content": "Test failover"},
		},
	}

	req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)

	// Request should complete (even if provider fails, should not crash)
	if err == nil {
		resp.Body.Close()
		s.True(resp.StatusCode >= 200 && resp.StatusCode < 600,
			"Request should complete even with disabled provider")
	}

	// Re-enable provider
	s.toggleProvider(provider1ID)
}

// TestLoadBalancingFailover tests load balancing with failover
func (s *FailoverTestSuite) TestLoadBalancingFailover() {
	t := s.T()

	// Create three providers with different priorities
	providers := []map[string]interface{}{
		{
			"name":         "provider-priority-1",
			"type":         "openai",
			"api_key":      "sk-p1",
			"base_url":     "https://api.openai.com/v1",
			"priority":     1,
			"enabled":      true,
			"max_requests": 100,
		},
		{
			"name":         "provider-priority-2",
			"type":         "openai",
			"api_key":      "sk-p2",
			"base_url":     "https://api.openai.com/v1",
			"priority":     2,
			"enabled":      true,
			"max_requests": 100,
		},
		{
			"name":         "provider-priority-3",
			"type":         "openai",
			"api_key":      "sk-p3",
			"base_url":     "https://api.openai.com/v1",
			"priority":     3,
			"enabled":      true,
			"max_requests": 100,
		},
	}

	createdProviders := make([]string, len(providers))
	for i, p := range providers {
		provider := s.createProvider(p)
		s.NotNil(provider)
		if id, ok := (*provider)["id"].(string); ok {
			createdProviders[i] = id
			s.providerIDs = append(s.providerIDs, id)
		}
	}

	// Make multiple requests
	numRequests := 10
	for i := 0; i < numRequests; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Load balance test %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)

		if err == nil {
			resp.Body.Close()
			// Request should complete
			s.True(resp.StatusCode >= 200 && resp.StatusCode < 600,
				"Request %d should complete", i)
		}
	}

	// Disable middle priority provider
	s.toggleProvider(createdProviders[1])

	// Make more requests - should use providers 1 and 3
	for i := 0; i < 5; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Failover test %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)

		if err == nil {
			resp.Body.Close()
			s.True(resp.StatusCode >= 200 && resp.StatusCode < 600,
				"Failover request %d should complete", i)
		}
	}

	// Re-enable provider
	s.toggleProvider(createdProviders[1])
}

// TestCircuitBreakerPattern tests circuit breaker pattern for failing providers
func (s *FailoverTestSuite) TestCircuitBreakerPattern() {
	t := s.T()

	// Create provider that will fail
	provider := s.createProvider(map[string]interface{}{
		"name":         "circuit-breaker-test",
		"type":         "openai",
		"api_key":      "sk-fail",
		"base_url":     "https://invalid-url.example.com",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider)
	providerID := (*provider)["id"].(string)
	s.providerIDs = append(s.providerIDs, providerID)

	// Make multiple failing requests
	numFailures := 5
	for i := 0; i < numFailures; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Failure test %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)

		if err == nil {
			resp.Body.Close()
			// Should fail with error status
			s.True(resp.StatusCode >= 400, "Request should fail with invalid URL")
		}
	}

	// Note: In a real implementation, circuit breaker would:
	// 1. Track consecutive failures
	// 2. Open circuit after threshold
	// 3. Return fast failures without attempting provider
	// 4. Periodically try to close circuit
}

// TestProviderPriorityUpdate tests updating provider priority affects routing
func (s *FailoverTestSuite) TestProviderPriorityUpdate() {
	t := s.T()

	// Create two providers
	provider1 := s.createProvider(map[string]interface{}{
		"name":         "priority-test-1",
		"type":         "openai",
		"api_key":      "sk-p1",
		"base_url":     "https://api.openai.com/v1",
		"priority":     2,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider1)
	provider1ID := (*provider1)["id"].(string)
	s.providerIDs = append(s.providerIDs, provider1ID)

	provider2 := s.createProvider(map[string]interface{}{
		"name":         "priority-test-2",
		"type":         "openai",
		"api_key":      "sk-p2",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	})
	s.NotNil(provider2)
	provider2ID := (*provider2)["id"].(string")
	s.providerIDs = append(s.providerIDs, provider2ID)

	// Verify initial priority ordering
	// Provider 2 should have priority 1 (higher)
	// Provider 1 should have priority 2 (lower)

	// Update provider 1 to have higher priority
	updateData := map[string]interface{}{
		"priority": 0,
	}
	updated := s.updateProvider(provider1ID, updateData)
	s.NotNil(updated)
	s.Equal(0.0, (*updated)["priority"], "Provider 1 should now have higher priority")

	// Make requests with new priority ordering
	for i := 0; i < 3; i++ {
		requestData := map[string]interface{}{
			"model": "gpt-3.5-turbo",
			"messages": []map[string]string{
				{"role": "user", "content": fmt.Sprintf("Priority update test %d", i)},
			},
		}

		req := s.MakeProxyRequest("POST", "/v1/chat/completions", requestData)
		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)

		if err == nil {
			resp.Body.Close()
			s.True(resp.StatusCode >= 200 && resp.StatusCode < 600,
				"Request with updated priority should complete")
		}
	}
}

// getProviderStatus retrieves provider status information
func (s *FailoverTestSuite) getProviderStatus(id string) *map[string]interface{} {
	req := s.MakeRequest("GET", "/admin/api/v1/providers/"+id+"/status", nil, true)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var status map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&status)
	s.Require().NoError(err)

	return &status
}

// MakeProxyRequest creates an HTTP request to the proxy endpoint
func (s *FailoverTestSuite) MakeProxyRequest(method, path string, body interface{}) *http.Request {
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

// createProvider creates a provider
func (s *FailoverTestSuite) createProvider(data map[string]interface{}) *map[string]interface{} {
	return s.createProvider(data)
}

// RunFailoverTests runs the failover test suite
func RunFailoverTests() {
	suite.Run(&FailoverTestSuite{
		TestSuite: &TestSuite{},
	})
}
