package e2e

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/stretchr/testify/suite"
)

// ProviderLifecycleTestSuite tests the complete provider configuration lifecycle
type ProviderLifecycleTestSuite struct {
	suite.Suite
	*TestSuite
	providerID string
}

// TestProviderLifecycle tests complete CRUD operations for providers
func (s *ProviderLifecycleTestSuite) TestProviderLifecycle() {
	t := s.T()

	// Step 1: List providers (should be empty initially)
	providers := s.listProviders()
	s.NotNil(providers)
	s.Equal(0, len(*providers), "Initially should have no providers")

	// Step 2: Create a new provider
	providerData := map[string]interface{}{
		"name":         "test-openai",
		"type":         "openai",
		"api_key":      "sk-test123",
		"base_url":     "https://api.openai.com/v1",
		"priority":     1,
		"enabled":      true,
		"max_requests": 100,
	}
	provider := s.createProvider(providerData)
	s.NotNil(provider)
	s.Equal("test-openai", (*provider)["name"])
	s.Equal("openai", (*provider)["type"])
	s.NotEmpty((*provider)["id"])

	// Store provider ID for later use
	if id, ok := (*provider)["id"].(string); ok {
		s.providerID = id
	}

	// Step 3: Verify provider exists in list
	providers = s.listProviders()
	s.Equal(1, len(*providers), "Should have 1 provider after creation")
	s.Equal("test-openai", (*providers)[0]["name"])

	// Step 4: Get specific provider by ID
	fetchedProvider := s.getProvider(s.providerID)
	s.NotNil(fetchedProvider)
	s.Equal("test-openai", (*fetchedProvider)["name"])
	s.Equal("sk-test123", (*fetchedProvider)["api_key"])

	// Step 5: Update provider
	updateData := map[string]interface{}{
		"name":         "test-openai-updated",
		"api_key":      "sk-test456",
		"priority":     2,
		"enabled":      false,
		"max_requests": 200,
	}
	updatedProvider := s.updateProvider(s.providerID, updateData)
	s.NotNil(updatedProvider)
	s.Equal("test-openai-updated", (*updatedProvider)["name"])
	s.Equal("sk-test456", (*updatedProvider)["api_key"])
	s.Equal(2.0, (*updatedProvider)["priority"])
	s.Equal(false, (*updatedProvider)["enabled"])

	// Step 6: Verify update persisted
	fetchedProvider = s.getProvider(s.providerID)
	s.Equal("test-openai-updated", (*fetchedProvider)["name"])
	s.Equal("sk-test456", (*fetchedProvider)["api_key"])

	// Step 7: Toggle provider enabled status
	toggledProvider := s.toggleProvider(s.providerID)
	s.NotNil(toggledProvider)
	s.Equal(true, (*toggledProvider)["enabled"], "Provider should be enabled after toggle")

	// Step 8: Verify toggle persisted
	fetchedProvider = s.getProvider(s.providerID)
	s.Equal(true, (*fetchedProvider)["enabled"])

	// Step 9: Delete provider
	s.deleteProvider(s.providerID)

	// Step 10: Verify provider is deleted
	providers = s.listProviders()
	s.Equal(0, len(*providers), "Should have 0 providers after deletion")
}

// TestMultipleProviders tests managing multiple providers
func (s *ProviderLifecycleTestSuite) TestMultipleProviders() {
	t := s.T()

	// Create 3 providers
	providers := []map[string]interface{}{
		{
			"name":         "provider-1",
			"type":         "openai",
			"api_key":      "sk-key1",
			"base_url":     "https://api.openai.com/v1",
			"priority":     1,
			"enabled":      true,
			"max_requests": 100,
		},
		{
			"name":         "provider-2",
			"type":         "anthropic",
			"api_key":      "sk-ant-key1",
			"base_url":     "https://api.anthropic.com",
			"priority":     2,
			"enabled":      true,
			"max_requests": 200,
		},
		{
			"name":         "provider-3",
			"type":         "openai",
			"api_key":      "sk-key2",
			"base_url":     "https://api.openai.com/v1",
			"priority":     3,
			"enabled":      false,
			"max_requests": 50,
		},
	}

	// Create all providers
	createdProviders := make([]map[string]interface{}, len(providers))
	for i, p := range providers {
		created := s.createProvider(p)
		s.NotNil(created)
		createdProviders[i] = *created
	}

	// Verify all providers are in the list
	allProviders := s.listProviders()
	s.Equal(3, len(*allProviders), "Should have 3 providers")

	// Verify providers are sorted by priority
	providerList := *allProviders
	for i := 0; i < len(providerList)-1; i++ {
		priority1 := providerList[i]["priority"].(float64)
		priority2 := providerList[i+1]["priority"].(float64)
		s.True(priority1 <= priority2, "Providers should be sorted by priority")
	}

	// Update provider 2
	updateData := map[string]interface{}{
		"priority": 1,
		"enabled":  false,
	}
	updated := s.updateProvider(createdProviders[1]["id"].(string), updateData)
	s.NotNil(updated)
	s.Equal(1.0, (*updated)["priority"])
	s.Equal(false, (*updated)["enabled"])

	// Delete provider 1
	s.deleteProvider(createdProviders[0]["id"].(string))

	// Verify 2 providers remain
	allProviders = s.listProviders()
	s.Equal(2, len(*allProviders), "Should have 2 providers after deletion")
}

// TestProviderValidation tests provider validation
func (s *ProviderLifecycleTestSuite) TestProviderValidation() {
	t := s.T()

	testCases := []struct {
		name        string
		data        map[string]interface{}
		expectError bool
		desc        string
	}{
		{
			name: "valid-provider",
			data: map[string]interface{}{
				"name":     "valid-provider",
				"type":     "openai",
				"api_key":  "sk-valid",
				"base_url": "https://api.openai.com/v1",
				"priority": 1,
			},
			expectError: false,
			desc:        "valid provider should be created",
		},
		{
			name: "",
			data: map[string]interface{}{
				"name":         "",
				"type":         "openai",
				"api_key":      "sk-key",
				"base_url":     "https://api.openai.com/v1",
				"priority":     1,
				"enabled":      true,
				"max_requests": 100,
			},
			expectError: true,
			desc:        "empty name should cause error",
		},
		{
			name: "missing-type",
			data: map[string]interface{}{
				"name":     "provider-missing-type",
				"api_key":  "sk-key",
				"base_url": "https://api.openai.com/v1",
				"priority": 1,
			},
			expectError: true,
			desc:        "missing type should cause error",
		},
		{
			name: "missing-api-key",
			data: map[string]interface{}{
				"name":     "provider-missing-key",
				"type":     "openai",
				"base_url": "https://api.openai.com/v1",
				"priority": 1,
			},
			expectError: true,
			desc:        "missing API key should cause error",
		},
		{
			name: "invalid-type",
			data: map[string]interface{}{
				"name":     "provider-invalid-type",
				"type":     "invalid-type",
				"api_key":  "sk-key",
				"base_url": "https://api.openai.com/v1",
				"priority": 1,
			},
			expectError: true,
			desc:        "invalid provider type should cause error",
		},
		{
			name: "negative-priority",
			data: map[string]interface{}{
				"name":     "provider-negative",
				"type":     "openai",
				"api_key":  "sk-key",
				"base_url": "https://api.openai.com/v1",
				"priority": -1,
			},
			expectError: true,
			desc:        "negative priority should cause error",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			provider := s.createProvider(tc.data)
			if tc.expectError {
				s.Nil(provider, "Provider creation should fail: %s", tc.desc)
			} else {
				s.NotNil(provider, "Provider should be created: %s", tc.desc)
				// Clean up
				if provider != nil && (*provider)["id"] != nil {
					s.deleteProvider((*provider)["id"].(string))
				}
			}
		})
	}
}

// TestProviderHotReload tests that provider changes take effect without restart
func (s *ProviderLifecycleTestSuite) TestProviderHotReload() {
	t := s.T()

	// Create a provider
	providerData := map[string]interface{}{
		"name":     "test-hot-reload",
		"type":     "openai",
		"api_key":  "sk-hot-reload",
		"base_url": "https://api.openai.com/v1",
		"priority": 1,
		"enabled":  true,
	}
	provider := s.createProvider(providerData)
	s.NotNil(provider)
	providerID := (*provider)["id"].(string)

	// Update the provider
	updateData := map[string]interface{}{
		"enabled": false,
	}
	updated := s.updateProvider(providerID, updateData)
	s.NotNil(updated)
	s.Equal(false, (*updated)["enabled"])

	// Verify change is immediately effective
	fetched := s.getProvider(providerID)
	s.Equal(false, (*fetched)["enabled"], "Change should be immediately effective")

	// Clean up
	s.deleteProvider(providerID)
}

// listProviders returns the list of all providers
func (s *ProviderLifecycleTestSuite) listProviders() *[]map[string]interface{} {
	req := s.MakeRequest("GET", "/admin/api/v1/providers", nil, true)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var providers []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&providers)
	s.Require().NoError(err)

	return &providers
}

// createProvider creates a new provider
func (s *ProviderLifecycleTestSuite) createProvider(data map[string]interface{}) *map[string]interface{} {
	body, _ := json.Marshal(data)
	req := s.MakeRequest("POST", "/admin/api/v1/providers", nil, true)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil
	}

	var provider map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&provider)
	s.Require().NoError(err)

	return &provider
}

// getProvider retrieves a specific provider by ID
func (s *ProviderLifecycleTestSuite) getProvider(id string) *map[string]interface{} {
	req := s.MakeRequest("GET", "/admin/api/v1/providers/"+id, nil, true)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var provider map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&provider)
	s.Require().NoError(err)

	return &provider
}

// updateProvider updates an existing provider
func (s *ProviderLifecycleTestSuite) updateProvider(id string, data map[string]interface{}) *map[string]interface{} {
	body, _ := json.Marshal(data)
	req := s.MakeRequest("PUT", "/admin/api/v1/providers/"+id, nil, true)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var provider map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&provider)
	s.Require().NoError(err)

	return &provider
}

// toggleProvider toggles provider enabled status
func (s *ProviderLifecycleTestSuite) toggleProvider(id string) *map[string]interface{} {
	req := s.MakeRequest("POST", "/admin/api/v1/providers/"+id+"/toggle", nil, true)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var provider map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&provider)
	s.Require().NoError(err)

	return &provider
}

// deleteProvider deletes a provider
func (s *ProviderLifecycleTestSuite) deleteProvider(id string) {
	req := s.MakeRequest("DELETE", "/admin/api/v1/providers/"+id, nil, true)
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()

	s.Equal(http.StatusNoContent, resp.StatusCode, "Provider should be deleted")
}

// RunProviderLifecycleTests runs the provider lifecycle test suite
func RunProviderLifecycleTests() {
	suite.Run(&ProviderLifecycleTestSuite{
		TestSuite: &TestSuite{},
	})
}
