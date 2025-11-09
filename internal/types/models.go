package types

// Provider represents an LLM provider configuration
type Provider struct {
	ID        string `json:"id" binding:"required"`
	Name      string `json:"name" binding:"required"`
	URL       string `json:"url" binding:"required,url"`
	Enabled   bool   `json:"enabled"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

// ModelMapping represents a model name mapping
type ModelMapping struct {
	SourceModel string `json:"source_model" binding:"required"`
	TargetModel string `json:"target_model" binding:"required"`
	CreatedAt   int64  `json:"created_at"`
}

// ChatCompletionRequest represents a chat completion request
type ChatCompletionRequest struct {
	Model       string        `json:"model" binding:"required"`
	Messages    []ChatMessage `json:"messages"`
	System      string        `json:"system"`
	MaxTokens   *int          `json:"max_tokens"`
	Temperature *float64      `json:"temperature"`
	TopP        *float64      `json:"top_p"`
	Stream      bool          `json:"stream"`
	Stop        []string      `json:"stop"`
}

// ChatMessage represents a single chat message
type ChatMessage struct {
	Role    string `json:"role" binding:"required,oneof=user assistant system"`
	Content string `json:"content" binding:"required"`
	Name    string `json:"name"`
}

// APIResponse represents a standardized API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents an API error
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Format represents API format types
type Format int

const (
	Unknown Format = iota
	Anthropic
	OpenAI
)

func (f Format) String() string {
	return [...]string{"Unknown", "Anthropic", "OpenAI"}[f]
}
