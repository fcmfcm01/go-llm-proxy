package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/converter"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
	"github.com/sirupsen/logrus"
)

// ProxyHandler handles proxying requests to LLM providers
type ProxyHandler struct {
	loadBalancer     LoadBalancer
	healthChecker    *HealthChecker
	converterFactory *converter.Factory
	httpClient       *http.Client
	logger           *logrus.Logger
	maxRetries       int
}

// ProxyConfig holds configuration for the proxy handler
type ProxyConfig struct {
	LoadBalancer     LoadBalancer
	HealthChecker    *HealthChecker
	ConverterFactory *converter.Factory
	Logger           *logrus.Logger
	MaxRetries       int
	RequestTimeout   time.Duration
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(config *ProxyConfig) *ProxyHandler {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 60 * time.Second
	}

	return &ProxyHandler{
		loadBalancer:     config.LoadBalancer,
		healthChecker:    config.HealthChecker,
		converterFactory: config.ConverterFactory,
		logger:           config.Logger,
		maxRetries:       config.MaxRetries,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ProxyRequest represents a proxy request
type ProxyRequest struct {
	Method  string
	Path    string
	Body    map[string]interface{}
	Headers map[string]string
}

// ProxyResponse represents a proxy response
type ProxyResponse struct {
	StatusCode int
	Body       map[string]interface{}
	Headers    map[string]string
	Provider   *models.Provider
	Latency    time.Duration
	Error      error
}

// HandleRequest handles a proxied request with failover
func (p *ProxyHandler) HandleRequest(ctx context.Context, req *ProxyRequest) (*ProxyResponse, error) {
	var lastErr error

	for attempt := 0; attempt < p.maxRetries; attempt++ {
		// Select a provider
		provider, err := p.loadBalancer.SelectProvider(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to select provider: %w", err)
		}

		if p.logger != nil {
			p.logger.WithFields(logrus.Fields{
				"provider": provider.ID,
				"attempt":  attempt + 1,
				"path":     req.Path,
			}).Debug("Attempting request to provider")
		}

		// Check provider health
		if p.healthChecker != nil && !p.healthChecker.CheckHealth(ctx, provider) {
			if p.logger != nil {
				p.logger.WithField("provider", provider.ID).Warn("Provider unhealthy, marking as failed")
			}
			p.loadBalancer.MarkProviderFailed(provider.ID)
			lastErr = fmt.Errorf("provider %s is unhealthy", provider.ID)
			continue
		}

		// Execute the request
		resp, err := p.executeRequest(ctx, provider, req)
		if err != nil {
			if p.logger != nil {
				p.logger.WithFields(logrus.Fields{
					"provider": provider.ID,
					"error":    err.Error(),
				}).Error("Request failed")
			}
			p.loadBalancer.MarkProviderFailed(provider.ID)
			lastErr = err
			continue
		}

		// Success - mark provider as healthy
		p.loadBalancer.MarkProviderHealthy(provider.ID)
		return resp, nil
	}

	// All retries failed
	return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
}

// executeRequest executes a single request to a provider
func (p *ProxyHandler) executeRequest(ctx context.Context, provider *models.Provider, req *ProxyRequest) (*ProxyResponse, error) {
	startTime := time.Now()

	// Convert request format for the provider
	convertedBody, err := p.converterFactory.ConvertRequest(provider, req.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// Serialize request body
	bodyBytes, err := json.Marshal(convertedBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Construct target URL
	targetURL := provider.APIURL + req.Path

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	if provider.APIKey != "" {
		// Set API key based on provider type
		if contains(provider.APIURL, "anthropic") {
			httpReq.Header.Set("x-api-key", provider.APIKey)
			httpReq.Header.Set("anthropic-version", "2023-06-01")
		} else if contains(provider.APIURL, "openai") {
			httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
		} else {
			// Default to Authorization header
			httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
		}
	}

	// Copy custom headers
	for key, value := range req.Headers {
		if key != "Authorization" && key != "x-api-key" {
			httpReq.Header.Set(key, value)
		}
	}

	// Execute request
	httpResp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		return &ProxyResponse{
			StatusCode: httpResp.StatusCode,
			Body: map[string]interface{}{
				"error": string(respBody),
			},
			Provider: provider,
			Latency:  time.Since(startTime),
			Error:    fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(respBody)),
		}, nil
	}

	// Parse response body
	var responseMap map[string]interface{}
	if err := json.Unmarshal(respBody, &responseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Convert response format back to OpenAI standard
	convertedResp, err := p.converterFactory.ConvertResponse(provider, responseMap)
	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	// Extract headers
	headers := make(map[string]string)
	for key := range httpResp.Header {
		headers[key] = httpResp.Header.Get(key)
	}

	return &ProxyResponse{
		StatusCode: httpResp.StatusCode,
		Body:       convertedResp,
		Headers:    headers,
		Provider:   provider,
		Latency:    time.Since(startTime),
		Error:      nil,
	}, nil
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	if len(s) == 0 || len(substr) == 0 {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
