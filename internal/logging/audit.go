package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AuditLogger provides audit logging functionality
type AuditLogger struct {
	logger *logrus.Logger
	file   *os.File
	mu     sync.Mutex
	quit   chan bool
}

// AuditEvent represents an audit log entry
type AuditEvent struct {
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	EventID      string                 `json:"event_id"`
	UserID       string                 `json:"user_id"`
	Username     string                 `json:"username,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	ResourceID   string                 `json:"resource_id,omitempty"`
	Result       string                 `json:"result"` // success, failure, error
	Reason       string                 `json:"reason,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	RequestID    string                 `json:"request_id,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	Duration     time.Duration          `json:"duration,omitempty"`
	RequestSize  int64                  `json:"request_size,omitempty"`
	ResponseSize int64                  `json:"response_size,omitempty"`
}

// NewSimpleAuditLogger creates a new audit logger with simple configuration
func NewSimpleAuditLogger(logger *Logger) *AuditLogger {
	// For testing, we'll use a null logger
	return &AuditLogger{
		logger: logger.Logger,
	}
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(filePath string, retention time.Duration, maxSize, maxBackups int, compress bool) (*AuditLogger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(filePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, err
	}

	// Open audit log file
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return nil, err
	}

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
		PrettyPrint:     false,
	})
	logger.SetOutput(f)

	auditLogger := &AuditLogger{
		logger: logger,
		file:   f,
		quit:   make(chan bool),
	}

	// Start cleanup goroutine
	go auditLogger.cleanup(retention)

	return auditLogger, nil
}

// LogEvent logs an audit event
func (a *AuditLogger) LogEvent(event *AuditEvent) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Log the event
	a.logger.WithFields(logrus.Fields{
		"event_type":    event.EventType,
		"event_id":      event.EventID,
		"user_id":       event.UserID,
		"username":      event.Username,
		"ip_address":    event.IPAddress,
		"action":        event.Action,
		"resource":      event.Resource,
		"resource_id":   event.ResourceID,
		"result":        event.Result,
		"reason":        event.Reason,
		"metadata":      event.Metadata,
		"request_id":    event.RequestID,
		"session_id":    event.SessionID,
		"duration":      event.Duration,
		"request_size":  event.RequestSize,
		"response_size": event.ResponseSize,
	}).Info("audit_event")
}

// LogAuthentication logs an authentication event
func (a *AuditLogger) LogAuthentication(userID, username, ipAddress, action, result, reason string) {
	event := &AuditEvent{
		EventType: "authentication",
		EventID:   generateEventID(),
		UserID:    userID,
		Username:  username,
		IPAddress: ipAddress,
		Action:    action, // login, logout, password_change
		Resource:  "auth",
		Result:    result, // success, failure
		Reason:    reason,
	}
	a.LogEvent(event)
}

// LogAuthEvent logs an authentication event with optional metadata
func (a *AuditLogger) LogAuthEvent(ctx interface{}, eventType, username, ipAddress, result string, metadata map[string]interface{}) {
	event := &AuditEvent{
		EventType: eventType,
		EventID:   generateEventID(),
		Username:  username,
		IPAddress: ipAddress,
		Resource:  "auth",
		Result:    result,
		Metadata:  metadata,
		Timestamp: time.Now(),
	}
	a.LogEvent(event)
}

// Audit event constants
const (
	AuditEventAuthLogin   = "auth.login"
	AuditEventAuthLogout  = "auth.logout"
	AuditEventAuthRefresh = "auth.refresh"

	// Provider management events
	AuditEventProviderList   = "provider.list"
	AuditEventProviderStatus = "provider.status"
	AuditEventProviderCreate = "provider.create"
	AuditEventProviderUpdate = "provider.update"
	AuditEventProviderDelete = "provider.delete"
	AuditEventProviderToggle = "provider.toggle"

	// Model mapping events
	AuditEventMappingList   = "mapping.list"
	AuditEventMappingCreate = "mapping.create"
	AuditEventMappingUpdate = "mapping.update"
	AuditEventMappingDelete = "mapping.delete"
)

// LogAuthorization logs an authorization event
func (a *AuditLogger) LogAuthorization(userID, username, ipAddress, action, resource, resourceID, result, reason string) {
	event := &AuditEvent{
		EventType:  "authorization",
		EventID:    generateEventID(),
		UserID:     userID,
		Username:   username,
		IPAddress:  ipAddress,
		Action:     action, // access, modify, delete
		Resource:   resource,
		ResourceID: resourceID,
		Result:     result, // success, failure, denied
		Reason:     reason,
	}
	a.LogEvent(event)
}

// LogDataAccess logs a data access event
func (a *AuditLogger) LogDataAccess(userID, username, ipAddress, action, resource, resourceID, result string, metadata map[string]interface{}) {
	event := &AuditEvent{
		EventType:  "data_access",
		EventID:    generateEventID(),
		UserID:     userID,
		Username:   username,
		IPAddress:  ipAddress,
		Action:     action, // create, read, update, delete
		Resource:   resource,
		ResourceID: resourceID,
		Result:     result, // success, failure
		Metadata:   metadata,
	}
	a.LogEvent(event)
}

// LogSystem logs a system event
func (a *AuditLogger) LogSystem(eventType, action, result, reason string, metadata map[string]interface{}) {
	event := &AuditEvent{
		EventType: eventType, // system_startup, system_shutdown, config_change
		EventID:   generateEventID(),
		Action:    action,
		Resource:  "system",
		Result:    result, // success, failure
		Reason:    reason,
		Metadata:  metadata,
	}
	a.LogEvent(event)
}

// LogProxy logs a proxy request event
func (a *AuditLogger) LogProxy(requestID, userID, ipAddress, action, provider, model, result string, duration time.Duration, requestSize, responseSize int64) {
	event := &AuditEvent{
		EventType:    "proxy_request",
		EventID:      generateEventID(),
		UserID:       userID,
		IPAddress:    ipAddress,
		Action:       action, // chat_completions, embeddings
		Resource:     "proxy",
		ResourceID:   provider + "/" + model,
		Result:       result, // success, failure
		RequestID:    requestID,
		Duration:     duration,
		RequestSize:  requestSize,
		ResponseSize: responseSize,
	}
	a.LogEvent(event)
}

// LogAdminEvent logs an admin event
func (a *AuditLogger) LogAdminEvent(ctx interface{}, eventType, resourceID, ipAddress, result string, metadata map[string]interface{}) {
	event := &AuditEvent{
		EventType:  eventType,
		EventID:    generateEventID(),
		IPAddress:  ipAddress,
		Resource:   "admin",
		ResourceID: resourceID,
		Result:     result,
		Metadata:   metadata,
		Timestamp:  time.Now(),
	}
	a.LogEvent(event)
}

// cleanup removes old audit log files based on retention policy
func (a *AuditLogger) cleanup(retention time.Duration) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.performCleanup(retention)
		case <-a.quit:
			return
		}
	}
}

// performCleanup performs the actual cleanup
func (a *AuditLogger) performCleanup(retention time.Duration) {
	// This would be implemented to remove files older than retention
	// For now, it's a placeholder
}

// Close closes the audit logger
func (a *AuditLogger) Close() error {
	a.quit <- true

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.file != nil {
		return a.file.Close()
	}
	return nil
}

// GetAuditLogger returns the underlying logrus logger
func (a *AuditLogger) GetAuditLogger() *logrus.Logger {
	return a.logger
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return fmt.Sprintf("audit_%d_%s", time.Now().UnixNano(), time.Now().Format("20060102"))
}
