package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/example/go-llm-proxy/internal/converter"
	"github.com/example/go-llm-proxy/internal/models"
	"github.com/sirupsen/logrus"
)

// StreamingHandler handles streaming responses from LLM providers
type StreamingHandler struct {
	loadBalancer     LoadBalancer
	healthChecker    *HealthChecker
	converterFactory *converter.Factory
	httpClient       *http.Client
	logger           *logrus.Logger
}

// StreamingConfig holds configuration for the streaming handler
type StreamingConfig struct {
	LoadBalancer     LoadBalancer
	HealthChecker    *HealthChecker
	ConverterFactory *converter.Factory
	Logger           *logrus.Logger
	RequestTimeout   time.Duration
}

// NewStreamingHandler creates a new streaming handler
func NewStreamingHandler(config *StreamingConfig) *StreamingHandler {
	if config.RequestTimeout == 0 {
		config.RequestTimeout = 300 * time.Second // 5 minutes for streaming
	}

	return &StreamingHandler{
		loadBalancer:     config.LoadBalancer,
		healthChecker:    config.HealthChecker,
		converterFactory: config.ConverterFactory,
		logger:           config.Logger,
		httpClient: &http.Client{
			Timeout: config.RequestTimeout,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   10,
				IdleConnTimeout:       90 * time.Second,
				ResponseHeaderTimeout: 30 * time.Second,
			},
		},
	}
}

// StreamRequest handles a streaming request
func (s *StreamingHandler) StreamRequest(ctx context.Context, req *ProxyRequest, writer io.Writer) error {
	// Select a provider
	provider, err := s.loadBalancer.SelectProvider(ctx)
	if err != nil {
		return fmt.Errorf("failed to select provider: %w", err)
	}

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"provider": provider.ID,
			"path":     req.Path,
		}).Debug("Starting streaming request to provider")
	}

	// Check provider health
	if s.healthChecker != nil && !s.healthChecker.CheckHealth(ctx, provider) {
		s.loadBalancer.MarkProviderFailed(provider.ID)
		return fmt.Errorf("provider %s is unhealthy", provider.ID)
	}

	// Execute the streaming request
	return s.executeStreamingRequest(ctx, provider, req, writer)
}

// executeStreamingRequest executes a streaming request to a provider
func (s *StreamingHandler) executeStreamingRequest(ctx context.Context, provider *models.Provider, req *ProxyRequest, writer io.Writer) error {
	// Convert request format for the provider
	convertedBody, err := s.converterFactory.ConvertRequest(provider, req.Body)
	if err != nil {
		return fmt.Errorf("failed to convert request: %w", err)
	}

	// Ensure streaming is enabled
	convertedBody["stream"] = true

	// Serialize request body
	bodyBytes, err := json.Marshal(convertedBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Construct target URL
	targetURL := provider.APIURL + req.Path

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("Cache-Control", "no-cache")
	httpReq.Header.Set("Connection", "keep-alive")

	if provider.APIKey != "" {
		// Set API key based on provider type
		if contains(provider.APIURL, "anthropic") {
			httpReq.Header.Set("x-api-key", provider.APIKey)
			httpReq.Header.Set("anthropic-version", "2023-06-01")
		} else if contains(provider.APIURL, "openai") {
			httpReq.Header.Set("Authorization", "Bearer "+provider.APIKey)
		} else {
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
	httpResp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer httpResp.Body.Close()

	// Check for HTTP errors
	if httpResp.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResp.Body)
		return fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, string(body))
	}

	// Stream the response
	return s.streamResponse(ctx, provider, httpResp.Body, writer)
}

// streamResponse reads and forwards streaming chunks
func (s *StreamingHandler) streamResponse(ctx context.Context, provider *models.Provider, reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(scanSSE)

	for scanner.Scan() {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}

		// Parse SSE event
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream end
		if data == "[DONE]" {
			// Write final event
			if _, err := fmt.Fprintf(writer, "data: [DONE]\n\n"); err != nil {
				return fmt.Errorf("failed to write stream end: %w", err)
			}
			if flusher, ok := writer.(http.Flusher); ok {
				flusher.Flush()
			}
			break
		}

		// Parse JSON chunk
		var chunk map[string]interface{}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			if s.logger != nil {
				s.logger.WithError(err).Warn("Failed to parse streaming chunk")
			}
			continue
		}

		// Convert chunk format if needed
		convertedChunk, err := s.convertStreamingChunk(provider, chunk)
		if err != nil {
			if s.logger != nil {
				s.logger.WithError(err).Warn("Failed to convert streaming chunk")
			}
			continue
		}

		// Serialize and write chunk
		chunkBytes, err := json.Marshal(convertedChunk)
		if err != nil {
			return fmt.Errorf("failed to marshal chunk: %w", err)
		}

		if _, err := fmt.Fprintf(writer, "data: %s\n\n", string(chunkBytes)); err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}

		// Flush to ensure immediate delivery
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner error: %w", err)
	}

	return nil
}

// convertStreamingChunk converts a streaming chunk between formats
func (s *StreamingHandler) convertStreamingChunk(provider *models.Provider, chunk map[string]interface{}) (map[string]interface{}, error) {
	// Detect provider type
	providerType := detectProviderType(provider)

	// For Anthropic providers, convert to OpenAI format
	if providerType == "anthropic" {
		return s.convertAnthropicStreamChunk(chunk)
	}

	// For OpenAI and others, pass through
	return chunk, nil
}

// convertAnthropicStreamChunk converts an Anthropic streaming chunk to OpenAI format
func (s *StreamingHandler) convertAnthropicStreamChunk(chunk map[string]interface{}) (map[string]interface{}, error) {
	// Anthropic streaming format differs from OpenAI
	// This is a simplified conversion - full implementation would handle all event types

	eventType, _ := chunk["type"].(string)

	switch eventType {
	case "message_start":
		// Convert to OpenAI chat.completion.chunk format
		return map[string]interface{}{
			"id":      chunk["message"].(map[string]interface{})["id"],
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   chunk["message"].(map[string]interface{})["model"],
			"choices": []interface{}{
				map[string]interface{}{
					"index": 0,
					"delta": map[string]interface{}{
						"role": "assistant",
					},
					"finish_reason": nil,
				},
			},
		}, nil

	case "content_block_delta":
		// Extract text from delta
		delta, _ := chunk["delta"].(map[string]interface{})
		text, _ := delta["text"].(string)

		return map[string]interface{}{
			"id":      chunk["id"],
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "unknown", // Model info not in delta events
			"choices": []interface{}{
				map[string]interface{}{
					"index": 0,
					"delta": map[string]interface{}{
						"content": text,
					},
					"finish_reason": nil,
				},
			},
		}, nil

	case "message_delta":
		// Handle finish reason
		delta, _ := chunk["delta"].(map[string]interface{})
		stopReason, _ := delta["stop_reason"].(string)

		finishReason := ""
		if stopReason == "end_turn" {
			finishReason = "stop"
		} else if stopReason == "max_tokens" {
			finishReason = "length"
		}

		return map[string]interface{}{
			"id":      chunk["id"],
			"object":  "chat.completion.chunk",
			"created": time.Now().Unix(),
			"model":   "unknown",
			"choices": []interface{}{
				map[string]interface{}{
					"index":         0,
					"delta":         map[string]interface{}{},
					"finish_reason": finishReason,
				},
			},
		}, nil

	default:
		// Pass through other event types
		return chunk, nil
	}
}

// scanSSE is a custom scanner split function for Server-Sent Events
func scanSSE(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Look for double newline (event boundary)
	if i := bytes.Index(data, []byte("\n\n")); i >= 0 {
		return i + 2, data[0:i], nil
	}

	// Single newline for individual lines
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		return i + 1, data[0:i], nil
	}

	// If at EOF, return remaining data
	if atEOF {
		return len(data), data, nil
	}

	// Request more data
	return 0, nil, nil
}

// Helper function from factory.go
func detectProviderType(provider *models.Provider) string {
	if provider == nil {
		return "openai"
	}

	apiURL := provider.APIURL
	if contains(apiURL, "anthropic") || contains(apiURL, "claude") {
		return "anthropic"
	}
	if contains(apiURL, "openai") {
		return "openai"
	}

	return "openai"
}
