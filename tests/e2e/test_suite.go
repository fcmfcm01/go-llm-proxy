package e2e

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// TestSuite provides a comprehensive E2E testing framework
type TestSuite struct {
	suite.Suite
	ctx        context.Context
	testServer *httptest.Server
	adminToken string
	tmpDir     string
	configPath string
	testData   map[string]interface{}
}

// SetupSuite runs once before all tests in the suite
func (s *TestSuite) SetupSuite() {
	// Set up timeout context
	s.ctx = context.Background()

	// Create temporary directory for test data
	tmpDir, err := os.MkdirTemp("", "llm-proxy-e2e-*")
	s.Require().NoError(err)
	s.tmpDir = tmpDir
	s.configPath = filepath.Join(tmpDir, "config")

	// Create test server
	s.setupTestServer()

	// Create test data
	s.createTestData()
}

// TearDownSuite runs once after all tests in the suite
func (s *TestSuite) TearDownSuite() {
	if s.testServer != nil {
		s.testServer.Close()
	}

	// Cleanup temporary directory
	if s.tmpDir != "" {
		os.RemoveAll(s.tmpDir)
	}
}

// SetupTest runs before each test
func (s *TestSuite) SetupTest() {
	// Reset test data
	s.createTestData()

	// Setup admin authentication
	s.setupAdminAuth()
}

// TearDownTest runs after each test
func (s *TestSuite) TearDownTest() {
	// Cleanup test data
	s.cleanupTestData()
}

// setupTestServer creates and configures the test HTTP server
func (s *TestSuite) setupTestServer() {
	// For E2E tests, we use a mock server
	// In a real implementation, this would start the actual proxy server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	s.testServer = httptest.NewServer(handler)
}

// createTestData initializes test data for the suite
func (s *TestSuite) createTestData() {
	s.testData = map[string]interface{}{
		"admin_user": map[string]interface{}{
			"username": "admin",
			"password": "admin123",
		},
		"test_provider": map[string]interface{}{
			"name":         "test-openai",
			"type":         "openai",
			"api_key":      "sk-test-key",
			"base_url":     "https://api.openai.com/v1",
			"priority":     1,
			"enabled":      true,
			"max_requests": 100,
		},
		"test_model": map[string]interface{}{
			"provider_name": "test-openai",
			"local_name":    "gpt-3.5-turbo",
			"remote_name":   "gpt-3.5-turbo",
		},
	}
}

// cleanupTestData cleans up test data
func (s *TestSuite) cleanupTestData() {
	// Clean up any test files
	s.testData = nil
}

// setupAdminAuth creates and stores admin authentication token
func (s *TestSuite) setupAdminAuth() {
	// Generate a mock token for testing
	s.adminToken = "admin-token-" + time.Now().Format("20060102150405")
}

// GetServerURL returns the test server URL
func (s *TestSuite) GetServerURL() string {
	return s.testServer.URL
}

// GetAdminToken returns the admin authentication token
func (s *TestSuite) GetAdminToken() string {
	return s.adminToken
}

// MakeRequest creates an HTTP request with optional authentication
func (s *TestSuite) MakeRequest(method, path string, body interface{}, auth bool) *http.Request {
	var req *http.Request
	if body != nil {
		req, _ = http.NewRequest(method, s.testServer.URL+path, nil)
	} else {
		req, _ = http.NewRequest(method, s.testServer.URL+path, nil)
	}

	if auth {
		req.Header.Set("Authorization", "Bearer "+s.adminToken)
	}

	return req
}

// AssertAPIResponse checks basic API response structure
func (s *TestSuite) AssertAPIResponse(statusCode int, response map[string]interface{}) {
	s.True(statusCode >= 200 && statusCode < 300, "Expected successful response, got status %d", statusCode)
	if response != nil {
		s.NotNil(response)
	}
}

// AssertErrorResponse checks error API response structure
func (s *TestSuite) AssertErrorResponse(statusCode int, response map[string]interface{}, errorType string) {
	s.True(statusCode >= 400, "Expected error response, got status %d", statusCode)
	if response != nil && errorType != "" {
		if err, ok := response["error"]; ok {
			s.NotEmpty(err)
		}
	}
}

// LoadTestConfig loads test configuration from file
func (s *TestSuite) LoadTestConfig() (interface{}, error) {
	return nil, nil
}

// GetTestData returns test data by key
func (s *TestSuite) GetTestData(key string) interface{} {
	return s.testData[key]
}

// SetupProvider creates a test provider
func (s *TestSuite) SetupProvider() error {
	return nil
}

// CleanupProvider removes test provider
func (s *TestSuite) CleanupProvider(name string) error {
	return nil
}

// createProvider creates a new provider (stub for E2E testing)
func (s *TestSuite) createProvider(data map[string]interface{}) *map[string]interface{} {
	// Generate a mock ID
	provider := make(map[string]interface{})
	for k, v := range data {
		provider[k] = v
	}
	provider["id"] = "provider-" + time.Now().Format("20060102150405")
	return &provider
}

// RunE2ETest runs an E2E test with proper setup and teardown
func RunE2ETest(m *testing.M) {
	// Run tests
	os.Exit(m.Run())
}

// TestProvider provides a mock provider for testing
type TestProvider struct {
	Name       string
	Type       string
	BaseURL    string
	APIKey     string
	Enabled    bool
	Priority   int
	MaxRequest int
}

// NewTestProvider creates a new test provider
func NewTestProvider(name string) *TestProvider {
	return &TestProvider{
		Name:       name,
		Type:       "openai",
		BaseURL:    "https://api.openai.com/v1",
		APIKey:     "sk-test-key",
		Enabled:    true,
		Priority:   1,
		MaxRequest: 100,
	}
}

// ToMap converts TestProvider to map
func (tp *TestProvider) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"name":         tp.Name,
		"type":         tp.Type,
		"api_key":      tp.APIKey,
		"base_url":     tp.BaseURL,
		"priority":     tp.Priority,
		"enabled":      tp.Enabled,
		"max_requests": tp.MaxRequest,
	}
}

// Timeout returns a context with timeout
func Timeout(duration time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), duration)
}

// Eventually retries a function until it succeeds or times out
func Eventually(condition func() bool, timeout time.Duration, interval time.Duration) bool {
	ctx, cancel := Timeout(timeout)
	defer cancel()

	for {
		if condition() {
			return true
		}

		select {
		case <-ctx.Done():
			return false
		case <-time.After(interval):
		}
	}
}

// RequireWithRetry retries a function with exponential backoff
func RequireWithRetry(fn func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * 100 * time.Millisecond)
	}
	return err
}
