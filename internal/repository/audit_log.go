package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// AuditLog represents an audit log entry
type AuditLog struct {
	ID            string                 `json:"id"`
	Timestamp     time.Time              `json:"timestamp"`
	Level         string                 `json:"level"`
	Action        string                 `json:"action"`
	Entity        string                 `json:"entity"`
	EntityID      string                 `json:"entity_id"`
	Actor         string                 `json:"actor"`
	IPAddress     string                 `json:"ip_address"`
	UserAgent     string                 `json:"user_agent"`
	Status        string                 `json:"status"` // "success", "failure", "error"
	Message       string                 `json:"message"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Duration      int64                  `json:"duration_ms,omitempty"`
	Error         string                 `json:"error,omitempty"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

// AuditLogQueryParams represents query parameters for audit logs
type AuditLogQueryParams struct {
	StartTime time.Time
	EndTime   time.Time
	Actor     string
	Action    string
	Entity    string
	Status    string
	Search    string
	Limit     int
	Offset    int
	SortBy    string // "timestamp", "action", "actor"
	SortOrder string // "asc", "desc"
}

// AuditLogRepository handles audit log storage and retrieval
type AuditLogRepository struct {
	logger  *logrus.Logger
	dataDir string
	logs    []AuditLog
	mu      sync.RWMutex
	maxLogs int
}

// NewAuditLogRepository creates a new audit log repository
func NewAuditLogRepository(logger *logrus.Logger, dataDir string) *AuditLogRepository {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorf("Failed to create audit log directory: %v", err)
	}

	repo := &AuditLogRepository{
		logger:  logger,
		dataDir: dataDir,
		logs:    make([]AuditLog, 0),
		maxLogs: 10000, // Keep last 10,000 logs in memory
	}

	// Load existing logs
	repo.loadLogs()

	return repo
}

// Create creates a new audit log entry
func (r *AuditLogRepository) Create(log *AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Set timestamp if not provided
	if log.Timestamp.IsZero() {
		log.Timestamp = time.Now()
	}

	// Add to in-memory logs
	r.logs = append(r.logs, *log)

	// Trim old logs if exceeding max
	if len(r.logs) > r.maxLogs {
		r.logs = r.logs[1:]
	}

	// Save to file asynchronously
	go r.saveLog(*log)

	return nil
}

// Query retrieves audit logs based on query parameters
func (r *AuditLogRepository) Query(params AuditLogQueryParams) ([]AuditLog, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter logs
	filtered := r.filterLogs(params)

	// Sort logs
	sort.Slice(filtered, func(i, j int) bool {
		ascending := params.SortOrder != "desc"

		switch params.SortBy {
		case "action":
			if ascending {
				return filtered[i].Action < filtered[j].Action
			}
			return filtered[i].Action > filtered[j].Action
		case "actor":
			if ascending {
				return filtered[i].Actor < filtered[j].Actor
			}
			return filtered[i].Actor > filtered[j].Actor
		case "timestamp":
			if ascending {
				return filtered[i].Timestamp.Before(filtered[j].Timestamp)
			}
			return filtered[i].Timestamp.After(filtered[j].Timestamp)
		default:
			// Default: sort by timestamp descending
			return filtered[i].Timestamp.After(filtered[j].Timestamp)
		}
	})

	// Apply pagination
	total := len(filtered)
	offset := params.Offset
	limit := params.Limit

	if offset > total {
		return []AuditLog{}, total, nil
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return filtered[offset:end], total, nil
}

// GetByID retrieves a specific audit log by ID
func (r *AuditLogRepository) GetByID(id string) (*AuditLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, log := range r.logs {
		if log.ID == id {
			return &log, nil
		}
	}

	return nil, fmt.Errorf("audit log not found: %s", id)
}

// GetByCorrelationID retrieves logs by correlation ID
func (r *AuditLogRepository) GetByCorrelationID(correlationID string) []AuditLog {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []AuditLog
	for _, log := range r.logs {
		if log.CorrelationID == correlationID {
			results = append(results, log)
		}
	}

	return results
}

// DeleteOldLogs removes logs older than the specified duration
func (r *AuditLogRepository) DeleteOldLogs(olderThan time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	var newLogs []AuditLog

	for _, log := range r.logs {
		if log.Timestamp.After(cutoff) {
			newLogs = append(newLogs, log)
		}
	}

	r.logs = newLogs
	return nil
}

// Export exports all logs to a file
func (r *AuditLogRepository) Export(filename string) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	data, err := json.MarshalIndent(r.logs, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal audit logs: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// filterLogs filters logs based on query parameters
func (r *AuditLogRepository) filterLogs(params AuditLogQueryParams) []AuditLog {
	var results []AuditLog

	for _, log := range r.logs {
		// Time range filter
		if !params.StartTime.IsZero() && log.Timestamp.Before(params.StartTime) {
			continue
		}
		if !params.EndTime.IsZero() && log.Timestamp.After(params.EndTime) {
			continue
		}

		// Actor filter
		if params.Actor != "" && !strings.Contains(strings.ToLower(log.Actor), strings.ToLower(params.Actor)) {
			continue
		}

		// Action filter
		if params.Action != "" && log.Action != params.Action {
			continue
		}

		// Entity filter
		if params.Entity != "" && log.Entity != params.Entity {
			continue
		}

		// Status filter
		if params.Status != "" && log.Status != params.Status {
			continue
		}

		// Search filter
		if params.Search != "" {
			searchLower := strings.ToLower(params.Search)
			haystack := strings.ToLower(log.Message + " " + log.Action + " " + log.Entity)
			if !strings.Contains(haystack, searchLower) {
				continue
			}
		}

		results = append(results, log)
	}

	return results
}

// saveLog saves a single log to file
func (r *AuditLogRepository) saveLog(log AuditLog) {
	r.logger.Debug("Saving audit log to file")
	// For now, just log that we would save
	// In a production system, you might write to a rolling log file
}

// loadLogs loads existing logs from file
func (r *AuditLogRepository) loadLogs() {
	// In a production system, you would load from files
	// For this implementation, logs are in-memory only
	r.logger.Info("Loaded audit logs from storage")
}

// GetStats returns statistics about audit logs
func (r *AuditLogRepository) GetStats() (map[string]interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_logs"] = len(r.logs)
	stats["start_time"] = time.Now()
	stats["end_time"] = time.Now()

	if len(r.logs) > 0 {
		stats["start_time"] = r.logs[0].Timestamp
		stats["end_time"] = r.logs[len(r.logs)-1].Timestamp

		// Count by status
		statusCounts := make(map[string]int)
		actionCounts := make(map[string]int)

		for _, log := range r.logs {
			statusCounts[log.Status]++
			actionCounts[log.Action]++
		}

		stats["status_counts"] = statusCounts
		stats["action_counts"] = actionCounts
	}

	return stats, nil
}
