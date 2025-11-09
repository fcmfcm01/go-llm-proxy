package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/example/go-llm-proxy/internal/config"
	"github.com/example/go-llm-proxy/internal/converter"
	"github.com/example/go-llm-proxy/internal/types"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// ProxyHandler handles proxying requests to LLM providers
type ProxyHandler struct {
	configManager  *config.ConfigManager
	converter      *converter.Converter
	httpClient     *http.Client
	logger         *logrus.Logger
	connectionPool *sync.Pool
}

// NewProxyHandler creates a new proxy handler
func NewProxyHandler(configManager *config.ConfigManager, converter *converter.Converter, logger *logrus.Logger) *ProxyHandler {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}

	return &ProxyHandler{
		configManager: configManager,
		converter:     converter,
		httpClient:    client,
		logger:        logger,
		connectionPool: &sync.Pool{
			New: func() interface{} {
				return make(map[string]interface{}, 10)
			},
		},
	}
}

// HandleChatCompletions handles chat completion requests
func (ph *ProxyHandler) HandleChatCompletions(c *gin.Context) {
	// Read request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		ph.logger.Error("Failed to read request body: ", err)
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "INVALID_REQUEST",
				Message: "Failed to read request body",
			},
		})
		return
	}

	// Parse request
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		ph.logger.Error("Failed to parse request: ", err)
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "INVALID_JSON",
				Message: "Invalid JSON in request body",
			},
		})
		return
	}

	// Detect format
	detector := &converter.FormatDetector{}
	format, confidence := detector.DetectFormat(req)

	if format == converter.Unknown {
		c.JSON(http.StatusBadRequest, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "UNKNOWN_FORMAT",
				Message: "Could not detect API format (Anthropic or OpenAI)",
			},
		})
		return
	}

	ph.logger.Infof("Detected format: %s (confidence: %.2f)", format, confidence)

	// Determine target provider and convert
	provider := ph.selectProvider(c.Request.Context(), req)
	if provider == nil {
		c.JSON(http.StatusServiceUnavailable, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "NO_PROVIDER",
				Message: "No available provider found",
			},
		})
		return
	}

	// Convert request if needed
	var targetFormat converter.Format
	var convertedBody []byte

	if strings.Contains(provider.URL, "openai.com") {
		targetFormat = converter.OpenAI
		if format == converter.Anthropic {
			convertedReq := ph.converter.ConvertRequest(req, converter.Anthropic, converter.OpenAI)
			convertedBody, _ = json.Marshal(convertedReq)
			ph.logger.Info("Converted Anthropic request to OpenAI")
		} else {
			convertedBody = body
		}
	} else {
		targetFormat = converter.Anthropic
		if format == converter.OpenAI {
			convertedReq := ph.converter.ConvertRequest(req, converter.OpenAI, converter.Anthropic)
			convertedBody, _ = json.Marshal(convertedReq)
			ph.logger.Info("Converted OpenAI request to Anthropic")
		} else {
			convertedBody = body
		}
	}

	// Forward request
	resp, err := ph.forwardRequest(provider, convertedBody)
	if err != nil {
		ph.logger.Error("Failed to forward request: ", err)
		c.JSON(http.StatusBadGateway, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "UPSTREAM_ERROR",
				Message: "Failed to forward request to provider",
			},
		})
		return
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		ph.logger.Error("Failed to read response body: ", err)
		c.JSON(http.StatusBadGateway, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "RESPONSE_READ_ERROR",
				Message: "Failed to read provider response",
			},
		})
		return
	}

	// Parse and convert response if needed
	var respData map[string]interface{}
	if err := json.Unmarshal(respBody, &respData); err == nil {
		var convertedResp map[string]interface{}
		if targetFormat == converter.OpenAI && format == converter.Anthropic {
			convertedResp = ph.converter.ConvertResponse(respData, converter.OpenAI, converter.Anthropic)
		} else if targetFormat == converter.Anthropic && format == converter.OpenAI {
			convertedResp = ph.converter.ConvertResponse(respData, converter.Anthropic, converter.OpenAI)
		} else {
			convertedResp = respData
		}

		convertedRespBytes, _ := json.Marshal(convertedResp)
		c.Data(resp.StatusCode, "application/json", convertedRespBytes)
	} else {
		// Return raw response if parsing fails
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), respBody)
	}
}

// selectProvider selects an available provider
func (ph *ProxyHandler) selectProvider(ctx interface{}, req map[string]interface{}) *config.Provider {
	providers := ph.configManager.GetProviders()

	// Find enabled provider
	for _, provider := range providers {
		if provider.Enabled {
			return &provider
		}
	}

	// Return default if no providers enabled
	defaultProvider := config.Provider{
		ID:      "default",
		Name:    "Default",
		URL:     "https://api.openai.com/v1",
		Enabled: true,
	}
	return &defaultProvider
}

// forwardRequest forwards a request to the provider
func (ph *ProxyHandler) forwardRequest(provider *config.Provider, body []byte) (*http.Response, error) {
	url := provider.URL + "/chat/completions"

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := ph.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	return resp, nil
}

// HandleModels handles model listing requests
func (ph *ProxyHandler) HandleModels(c *gin.Context) {
	provider := ph.selectProvider(c.Request.Context(), nil)
	if provider == nil {
		c.JSON(http.StatusServiceUnavailable, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "NO_PROVIDER",
				Message: "No available provider found",
			},
		})
		return
	}

	url := provider.URL + "/models"
	resp, err := ph.httpClient.Get(url)
	if err != nil {
		c.JSON(http.StatusBadGateway, types.APIResponse{
			Success: false,
			Error: &types.APIError{
				Code:    "UPSTREAM_ERROR",
				Message: "Failed to fetch models from provider",
			},
		})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	c.Data(resp.StatusCode, "application/json", body)
}

// HandleHealth handles health check requests
func (ph *ProxyHandler) HandleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, types.APIResponse{
		Success: true,
		Data: map[string]string{
			"status":  "healthy",
			"service": "go-llm-proxy",
		},
	})
}
