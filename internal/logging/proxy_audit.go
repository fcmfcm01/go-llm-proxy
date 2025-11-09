package logging

import (
	"encoding/json"
	"time"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/models"
)

// ProxyAuditEvent represents an audit event for proxy requests
type ProxyAuditEvent struct {
	// Basic audit fields
	Timestamp time.Time `json:"timestamp"`
	EventID   string    `json:"event_id"`
	RequestID string    `json:"request_id"`
	SessionID string    `json:"session_id,omitempty"`

	// User information
	UserID    string `json:"user_id,omitempty"`
	Username  string `json:"username,omitempty"`
	IPAddress string `json:"ip_address"`
	UserAgent string `json:"user_agent,omitempty"`

	// Request information
	Method      string `json:"method"`
	Path        string `json:"path"`
	RequestSize int64  `json:"request_size"`

	// Model information
	SourceModel string `json:"source_model"`
	TargetModel string `json:"target_model,omitempty"`
	ModelMapped bool   `json:"model_mapped"`

	// Provider information
	ProviderID       string `json:"provider_id"`
	ProviderName     string `json:"provider_name,omitempty"`
	ProviderType     string `json:"provider_type,omitempty"`
	AttemptNumber    int    `json:"attempt_number,omitempty"`
	FailoverOccurred bool   `json:"failover_occurred,omitempty"`

	// Response information
	StatusCode   int    `json:"status_code"`
	ResponseSize int64  `json:"response_size"`
	Result       string `json:"result"` // success, failure, error

	// Performance metrics
	Duration           time.Duration `json:"duration_ms"`
	ConversionDuration time.Duration `json:"conversion_duration_ms,omitempty"`
	ProviderDuration   time.Duration `json:"provider_duration_ms,omitempty"`

	// Token usage (if available)
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`

	// Streaming information
	IsStreaming  bool `json:"is_streaming"`
	StreamChunks int  `json:"stream_chunks,omitempty"`

	// Error information
	Error        string `json:"error,omitempty"`
	ErrorType    string `json:"error_type,omitempty"`
	ErrorDetails string `json:"error_details,omitempty"`

	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProxyAuditLogger provides enhanced audit logging for proxy requests
type ProxyAuditLogger struct {
	auditLogger *AuditLogger
}

// NewProxyAuditLogger creates a new proxy audit logger
func NewProxyAuditLogger(auditLogger *AuditLogger) *ProxyAuditLogger {
	return &ProxyAuditLogger{
		auditLogger: auditLogger,
	}
}

// LogProxyRequest logs a complete proxy request
func (p *ProxyAuditLogger) LogProxyRequest(event *ProxyAuditEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.EventID == "" {
		event.EventID = generateEventID()
	}

	// Convert to generic audit event
	auditEvent := &AuditEvent{
		Timestamp:    event.Timestamp,
		EventType:    "proxy_request",
		EventID:      event.EventID,
		UserID:       event.UserID,
		Username:     event.Username,
		IPAddress:    event.IPAddress,
		Action:       event.Method + " " + event.Path,
		Resource:     "proxy",
		ResourceID:   event.ProviderID,
		Result:       event.Result,
		RequestID:    event.RequestID,
		SessionID:    event.SessionID,
		Duration:     event.Duration,
		RequestSize:  event.RequestSize,
		ResponseSize: event.ResponseSize,
		Metadata:     p.buildMetadata(event),
	}

	p.auditLogger.LogEvent(auditEvent)
}

// buildMetadata builds detailed metadata for the audit event
func (p *ProxyAuditLogger) buildMetadata(event *ProxyAuditEvent) map[string]interface{} {
	metadata := make(map[string]interface{})

	// Model information
	metadata["source_model"] = event.SourceModel
	if event.TargetModel != "" {
		metadata["target_model"] = event.TargetModel
		metadata["model_mapped"] = event.ModelMapped
	}

	// Provider information
	if event.ProviderName != "" {
		metadata["provider_name"] = event.ProviderName
	}
	if event.ProviderType != "" {
		metadata["provider_type"] = event.ProviderType
	}
	if event.AttemptNumber > 0 {
		metadata["attempt_number"] = event.AttemptNumber
	}
	if event.FailoverOccurred {
		metadata["failover_occurred"] = true
	}

	// Performance metrics
	if event.ConversionDuration > 0 {
		metadata["conversion_duration_ms"] = event.ConversionDuration.Milliseconds()
	}
	if event.ProviderDuration > 0 {
		metadata["provider_duration_ms"] = event.ProviderDuration.Milliseconds()
	}

	// Token usage
	if event.TotalTokens > 0 {
		metadata["token_usage"] = map[string]int{
			"prompt_tokens":     event.PromptTokens,
			"completion_tokens": event.CompletionTokens,
			"total_tokens":      event.TotalTokens,
		}
	}

	// Streaming information
	metadata["is_streaming"] = event.IsStreaming
	if event.StreamChunks > 0 {
		metadata["stream_chunks"] = event.StreamChunks
	}

	// HTTP information
	metadata["status_code"] = event.StatusCode
	if event.UserAgent != "" {
		metadata["user_agent"] = event.UserAgent
	}

	// Error information
	if event.Error != "" {
		metadata["error"] = event.Error
		if event.ErrorType != "" {
			metadata["error_type"] = event.ErrorType
		}
		if event.ErrorDetails != "" {
			metadata["error_details"] = event.ErrorDetails
		}
	}

	// Merge additional metadata
	if event.Metadata != nil {
		for k, v := range event.Metadata {
			metadata[k] = v
		}
	}

	return metadata
}

// LogProxySuccess logs a successful proxy request
func (p *ProxyAuditLogger) LogProxySuccess(
	requestID, sessionID, userID, username, ipAddress string,
	method, path string,
	provider *models.Provider,
	sourceModel, targetModel string,
	duration time.Duration,
	requestSize, responseSize int64,
	statusCode int,
	isStreaming bool,
) {
	event := &ProxyAuditEvent{
		RequestID:    requestID,
		SessionID:    sessionID,
		UserID:       userID,
		Username:     username,
		IPAddress:    ipAddress,
		Method:       method,
		Path:         path,
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		SourceModel:  sourceModel,
		TargetModel:  targetModel,
		ModelMapped:  sourceModel != targetModel,
		Duration:     duration,
		RequestSize:  requestSize,
		ResponseSize: responseSize,
		StatusCode:   statusCode,
		Result:       "success",
		IsStreaming:  isStreaming,
	}

	p.LogProxyRequest(event)
}

// LogProxyFailure logs a failed proxy request
func (p *ProxyAuditLogger) LogProxyFailure(
	requestID, sessionID, userID, username, ipAddress string,
	method, path string,
	provider *models.Provider,
	sourceModel string,
	duration time.Duration,
	requestSize int64,
	err error,
	attemptNumber int,
	failoverOccurred bool,
) {
	event := &ProxyAuditEvent{
		RequestID:        requestID,
		SessionID:        sessionID,
		UserID:           userID,
		Username:         username,
		IPAddress:        ipAddress,
		Method:           method,
		Path:             path,
		ProviderID:       provider.ID,
		ProviderName:     provider.Name,
		SourceModel:      sourceModel,
		Duration:         duration,
		RequestSize:      requestSize,
		StatusCode:       500,
		Result:           "failure",
		Error:            err.Error(),
		AttemptNumber:    attemptNumber,
		FailoverOccurred: failoverOccurred,
	}

	p.LogProxyRequest(event)
}

// LogProxyError logs a proxy request error
func (p *ProxyAuditLogger) LogProxyError(
	requestID, sessionID, userID, username, ipAddress string,
	method, path string,
	errorMsg, errorType string,
	duration time.Duration,
) {
	event := &ProxyAuditEvent{
		RequestID:  requestID,
		SessionID:  sessionID,
		UserID:     userID,
		Username:   username,
		IPAddress:  ipAddress,
		Method:     method,
		Path:       path,
		Duration:   duration,
		StatusCode: 500,
		Result:     "error",
		Error:      errorMsg,
		ErrorType:  errorType,
	}

	p.LogProxyRequest(event)
}

// LogModelMapping logs a model mapping event
func (p *ProxyAuditLogger) LogModelMapping(
	requestID string,
	sourceModel, targetModel string,
	provider *models.Provider,
	success bool,
) {
	result := "success"
	if !success {
		result = "failure"
	}

	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "model_mapping",
		EventID:    generateEventID(),
		RequestID:  requestID,
		Resource:   "model_mapper",
		ResourceID: provider.ID,
		Result:     result,
		Metadata: map[string]interface{}{
			"source_model":  sourceModel,
			"target_model":  targetModel,
			"provider_id":   provider.ID,
			"provider_name": provider.Name,
		},
	}

	p.auditLogger.LogEvent(event)
}

// LogProviderHealth logs a provider health check event
func (p *ProxyAuditLogger) LogProviderHealth(
	providerID, providerName string,
	healthy bool,
	checkDuration time.Duration,
	errorMsg string,
) {
	result := "healthy"
	if !healthy {
		result = "unhealthy"
	}

	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "provider_health_check",
		EventID:    generateEventID(),
		Resource:   "health_checker",
		ResourceID: providerID,
		Result:     result,
		Duration:   checkDuration,
		Metadata: map[string]interface{}{
			"provider_name": providerName,
			"healthy":       healthy,
		},
	}

	if errorMsg != "" {
		event.Metadata["error"] = errorMsg
	}

	p.auditLogger.LogEvent(event)
}

// LogLoadBalancerEvent logs a load balancer selection event
func (p *ProxyAuditLogger) LogLoadBalancerEvent(
	requestID string,
	selectedProvider *models.Provider,
	attemptNumber int,
	totalProviders int,
) {
	event := &AuditEvent{
		Timestamp:  time.Now(),
		EventType:  "load_balancer_selection",
		EventID:    generateEventID(),
		RequestID:  requestID,
		Resource:   "load_balancer",
		ResourceID: selectedProvider.ID,
		Result:     "success",
		Metadata: map[string]interface{}{
			"provider_id":     selectedProvider.ID,
			"provider_name":   selectedProvider.Name,
			"attempt_number":  attemptNumber,
			"total_providers": totalProviders,
		},
	}

	p.auditLogger.LogEvent(event)
}

// ExtractTokenUsage extracts token usage from response
func ExtractTokenUsage(response map[string]interface{}) (prompt, completion, total int) {
	usage, ok := response["usage"].(map[string]interface{})
	if !ok {
		return 0, 0, 0
	}

	if p, ok := usage["prompt_tokens"].(float64); ok {
		prompt = int(p)
	}
	if c, ok := usage["completion_tokens"].(float64); ok {
		completion = int(c)
	}
	if t, ok := usage["total_tokens"].(float64); ok {
		total = int(t)
	}

	return
}

// MarshalProxyEvent marshals a proxy event to JSON
func MarshalProxyEvent(event *ProxyAuditEvent) ([]byte, error) {
	return json.Marshal(event)
}

// UnmarshalProxyEvent unmarshals a proxy event from JSON
func UnmarshalProxyEvent(data []byte) (*ProxyAuditEvent, error) {
	var event ProxyAuditEvent
	err := json.Unmarshal(data, &event)
	return &event, err
}
