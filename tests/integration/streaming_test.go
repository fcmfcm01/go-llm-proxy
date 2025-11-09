package integration

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StreamEvent represents a streaming event from the LLM
type StreamEvent struct {
	EventType string      `json:"event_type"`
	Content   string      `json:"content"`
	Done      bool        `json:"done"`
	Index     int         `json:"index,omitempty"`
	Model     string      `json:"model,omitempty"`
	Usage     *UsageStats `json:"usage,omitempty"`
}

// UsageStats represents token usage statistics
type UsageStats struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// TestStreamingResponse tests streaming response handling
func TestStreamingResponse(t *testing.T) {
	proxy := NewTestProxy()

	t.Run("BasicStreaming", func(t *testing.T) {
		// Create a streaming response from upstream provider
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// Send SSE events
			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("Streaming not supported")
			}

			// Event 1
			w.Write([]byte("data: {\"event_type\":\"content\",\"content\":\"Hello\"}\n\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)

			// Event 2
			w.Write([]byte("data: {\"event_type\":\"content\",\"content\":\" World\"}\n\n"))
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)

			// Event 3 - Done
			w.Write([]byte("data: {\"event_type\":\"done\",\"content\":\"\",\"done\":true}\n\n"))
			flusher.Flush()
		}))
		defer upstream.Close()

		// Make request to proxy with streaming
		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Say hello"}],
			"stream": true
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))

		// Parse streaming response
		events, err := parseStreamEvents(rec.Body)
		require.NoError(t, err)
		assert.True(t, len(events) > 0)

		// Verify events
		var hasContentEvent, hasDoneEvent bool
		for _, event := range events {
			if event.EventType == "content" {
				hasContentEvent = true
			}
			if event.EventType == "done" {
				hasDoneEvent = true
			}
		}
		assert.True(t, hasContentEvent, "Should have content events")
		assert.True(t, hasDoneEvent, "Should have done event")
	})

	t.Run("StreamingWithFunctionCalls", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("Streaming not supported")
			}

			// Stream partial function call
			w.Write([]byte("data: {\"event_type\":\"tool_call\",\"content\":\"{\"name\":\"get_weather\"}\"}\n\n"))
			flusher.Flush()

			// Stream result
			w.Write([]byte("data: {\"event_type\":\"content\",\"content\":\"Weather is sunny\"}\n\n"))
			flusher.Flush()

			w.Write([]byte("data: {\"event_type\":\"done\"}\n\n"))
			flusher.Flush()
		}))
		defer upstream.Close()

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-4",
			"messages": [{"role": "user", "content": "What's the weather?"}],
			"stream": true
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		events, err := parseStreamEvents(rec.Body)
		require.NoError(t, err)
		assert.True(t, len(events) > 0)
	})

	t.Run("StreamingWithError", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)

			// Send partial data then error
			w.Write([]byte("data: {\"event_type\":\"content\",\"content\":\"Partial\"}\n\n"))
			flusher.Flush()

			// Send error event
			w.Write([]byte("data: {\"event_type\":\"error\",\"content\":\"Provider error\"}\n\n"))
			flusher.Flush()

			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		}))
		defer upstream.Close()

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Test"}],
			"stream": true
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should still return 200 for streaming, errors are in the stream
		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("NonStreamingRequest", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "Non-streaming response"
					}
				}]
			}`))
		}))
		defer upstream.Close()

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Hello"}]
		}`))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.NotEqual(t, "text/event-stream", rec.Header().Get("Content-Type"))
	})

	t.Run("StreamingCancellation", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)

			// Simulate slow response
			for i := 0; i < 10; i++ {
				w.Write([]byte(`data: {"event_type":"content","content":"Chunk ` + string(rune(i)) + `"}\n\n`))
				flusher.Flush()
				time.Sleep(100 * time.Millisecond)
			}
		}))
		defer upstream.Close()

		req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
			"model": "gpt-3.5-turbo",
			"messages": [{"role": "user", "content": "Stream"}],
			"stream": true
		}`))
		req.Header.Set("Content-Type", "application/json")

		// Cancel request after short delay
		ctx := req.Context()
		ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		defer cancel()
		*req = *req.WithContext(ctx)

		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)

		// Should handle cancellation gracefully
		assert.True(t, rec.Code == http.StatusOK || rec.Code == http.StatusRequestTimeout)
	})

	t.Run("MultipleConcurrentStreams", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher := w.(http.Flusher)

			w.Write([]byte(`data: {"event_type":"content","content":"Response"}\n\n`))
			flusher.Flush()

			w.Write([]byte(`data: {"event_type":"done"}\n\n`))
			flusher.Flush()
		}))
		defer upstream.Close()

		// Make multiple concurrent streaming requests
		for i := 0; i < 5; i++ {
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{
				"model": "gpt-3.5-turbo",
				"messages": [{"role": "user", "content": "Stream `+string(rune(i))+`"}],
				"stream": true
			}`))
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			go proxy.ServeHTTP(rec, req)

			// Verify each stream
			assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
		}
	})
}

// parseStreamEvents parses SSE event stream from response body
func parseStreamEvents(body *bytes.Buffer) ([]StreamEvent, error) {
	var events []StreamEvent
	data := body.String()

	// Simple SSE parser - split by "\n\n"
	parts := bytes.Split([]byte(data), []byte("\n\n"))
	for _, part := range parts {
		if len(part) == 0 {
			continue
		}

		// Extract "data: " prefixed content
		lines := bytes.Split(part, []byte("\n"))
		for _, line := range lines {
			if bytes.HasPrefix(line, []byte("data: ")) {
				eventData := bytes.TrimPrefix(line, []byte("data: "))
				// Parse event (simplified)
				event := StreamEvent{
					EventType: "content",
					Content:   string(eventData),
				}
				events = append(events, event)
			}
		}
	}

	return events, nil
}

// TestProxy is a test proxy instance
type TestProxy struct{}

func NewTestProxy() *TestProxy {
	return &TestProxy{}
}

func (p *TestProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implementation pending - test-first development
}
