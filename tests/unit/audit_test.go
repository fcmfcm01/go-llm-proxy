package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AuditLogEntry represents an audit log entry as per FR-032 requirement
type AuditLogEntry struct {
	ID             string    `json:"id"`
	Timestamp      time.Time `json:"timestamp"`
	UserID         string    `json:"user_id"`
	Username       string    `json:"username"`
	OperationType  string    `json:"operation_type"`
	TargetResource string    `json:"target_resource"`
	IPAddress      string    `json:"ip_address"`
	Result         string    `json:"result"`
	Details        string    `json:"details,omitempty"`
}

// TestAuditLogger tests audit logging functionality
func TestAuditLogger(t *testing.T) {
	logger := NewAuditLogger()

	t.Run("LogSuccessfulRequest", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "user-123",
			Username:       "testuser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/chat/completions",
			IPAddress:      "192.168.1.100",
			Result:         "success",
			Details:        "Model: gpt-4, Tokens: 100",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogFailedRequest", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "user-456",
			Username:       "testuser2",
			OperationType:  "proxy_request",
			TargetResource: "/v1/chat/completions",
			IPAddress:      "192.168.1.101",
			Result:         "failure",
			Details:        "Provider timeout after 30s",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogProviderConfigChange", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "admin-789",
			Username:       "admin",
			OperationType:  "update",
			TargetResource: "provider/openai",
			IPAddress:      "192.168.1.102",
			Result:         "success",
			Details:        "Updated API URL from old to new",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogAuthentication", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "user-123",
			Username:       "testuser",
			OperationType:  "login",
			TargetResource: "auth/session",
			IPAddress:      "192.168.1.100",
			Result:         "success",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogFailedAuthentication", func(t *testing.T) {
		entry := AuditLogEntry{
			Username:       "invaliduser",
			OperationType:  "login",
			TargetResource: "auth/session",
			IPAddress:      "192.168.1.200",
			Result:         "failure",
			Details:        "Invalid password",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogLogout", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "user-123",
			Username:       "testuser",
			OperationType:  "logout",
			TargetResource: "auth/session",
			IPAddress:      "192.168.1.100",
			Result:         "success",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogProviderToggle", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "admin-789",
			Username:       "admin",
			OperationType:  "toggle",
			TargetResource: "provider/anthropic",
			IPAddress:      "192.168.1.102",
			Result:         "success",
			Details:        "Provider disabled",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogModelMappingChange", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "admin-789",
			Username:       "admin",
			OperationType:  "create",
			TargetResource: "model_mapping/claude-to-gpt4",
			IPAddress:      "192.168.1.102",
			Result:         "success",
			Details:        "Created mapping: claude-3-sonnet -> gpt-4",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("LogConfigBackup", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "admin-789",
			Username:       "admin",
			OperationType:  "backup",
			TargetResource: "config/system",
			IPAddress:      "192.168.1.102",
			Result:         "success",
			Details:        "Backup created: config-backup-20241108.tar.gz",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("RequiredFields", func(t *testing.T) {
		// Per FR-032, all fields are required except Details
		entry := AuditLogEntry{
			// Missing UserID - should still log but track as system
			Username:       "admin",
			OperationType:  "system_operation",
			TargetResource: "config/reload",
			IPAddress:      "192.168.1.102",
			Result:         "success",
		}

		err := logger.Log(entry)
		require.NoError(t, err)
	})

	t.Run("OperationTypeEnum", func(t *testing.T) {
		validTypes := []string{
			"create", "update", "delete", "toggle",
			"login", "logout",
			"proxy_request", "proxy_response",
			"backup", "restore",
		}

		for _, opType := range validTypes {
			t.Run("ValidType"+opType, func(t *testing.T) {
				entry := AuditLogEntry{
					UserID:         "user-123",
					Username:       "testuser",
					OperationType:  opType,
					TargetResource: "test",
					IPAddress:      "192.168.1.100",
					Result:         "success",
				}

				err := logger.Log(entry)
				assert.NoError(t, err)
			})
		}
	})

	t.Run("ResultEnum", func(t *testing.T) {
		// Per data model, Result must be: success or failure
		t.Run("Success", func(t *testing.T) {
			entry := AuditLogEntry{
				Username:       "testuser",
				OperationType:  "proxy_request",
				TargetResource: "/v1/chat/completions",
				IPAddress:      "192.168.1.100",
				Result:         "success",
			}

			err := logger.Log(entry)
			assert.NoError(t, err)
		})

		t.Run("Failure", func(t *testing.T) {
			entry := AuditLogEntry{
				Username:       "testuser",
				OperationType:  "proxy_request",
				TargetResource: "/v1/chat/completions",
				IPAddress:      "192.168.1.100",
				Result:         "failure",
			}

			err := logger.Log(entry)
			assert.NoError(t, err)
		})
	})

	t.Run("TimestampFormat", func(t *testing.T) {
		entry := AuditLogEntry{
			Username:       "testuser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/models",
			IPAddress:      "192.168.1.100",
			Result:         "success",
			Timestamp:      time.Now(),
		}

		err := logger.Log(entry)
		require.NoError(t, err)
		// Should be in RFC3339 format
	})
}

// TestAuditLogCompleteness tests audit log completeness per SC-009
func TestAuditLogCompleteness(t *testing.T) {
	logger := NewAuditLogger()

	t.Run("AllFieldsLogged", func(t *testing.T) {
		entry := AuditLogEntry{
			ID:             "audit-123",
			Timestamp:      time.Now(),
			UserID:         "user-123",
			Username:       "testuser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/chat/completions",
			IPAddress:      "192.168.1.100",
			Result:         "success",
			Details:        "Additional details",
		}

		logged, err := logger.LogAndRetrieve(entry)
		require.NoError(t, err)

		// Verify all required fields are present
		assert.NotEmpty(t, logged.ID)
		assert.NotZero(t, logged.Timestamp)
		assert.NotEmpty(t, logged.OperationType)
		assert.NotEmpty(t, logged.TargetResource)
		assert.NotEmpty(t, logged.IPAddress)
		assert.NotEmpty(t, logged.Result)
	})

	t.Run("ChronologicalOrder", func(t *testing.T) {
		// Logs should be ordered chronologically by Timestamp
		entries := []AuditLogEntry{
			{
				Username:       "user1",
				OperationType:  "proxy_request",
				TargetResource: "/v1/models",
				IPAddress:      "192.168.1.100",
				Result:         "success",
				Timestamp:      time.Now().Add(-2 * time.Second),
			},
			{
				Username:       "user2",
				OperationType:  "proxy_request",
				TargetResource: "/v1/completions",
				IPAddress:      "192.168.1.101",
				Result:         "success",
				Timestamp:      time.Now().Add(-1 * time.Second),
			},
			{
				Username:       "user3",
				OperationType:  "proxy_request",
				TargetResource: "/v1/embeddings",
				IPAddress:      "192.168.1.102",
				Result:         "success",
				Timestamp:      time.Now(),
			},
		}

		for _, entry := range entries {
			logger.Log(entry)
		}

		logs := logger.GetRecentLogs(10)
		assert.Equal(t, 3, len(logs))

		// Verify chronological order
		for i := 1; i < len(logs); i++ {
			assert.True(t, logs[i-1].Timestamp.Before(logs[i].Timestamp) ||
				logs[i-1].Timestamp.Equal(logs[i].Timestamp))
		}
	})

	t.Run("RetentionPolicy", func(t *testing.T) {
		// Per research.md, 30-day retention
		logger := NewAuditLoggerWithRetention(30 * 24 * time.Hour)

		// Create logs spanning 40 days
		oldTime := time.Now().Add(-40 * 24 * time.Hour)
		newTime := time.Now()

		logger.Log(AuditLogEntry{
			Username:       "olduser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/old",
			IPAddress:      "192.168.1.100",
			Result:         "success",
			Timestamp:      oldTime,
		})

		logger.Log(AuditLogEntry{
			Username:       "newuser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/new",
			IPAddress:      "192.168.1.101",
			Result:         "success",
			Timestamp:      newTime,
		})

		// Old logs should be cleaned up
		logs := logger.GetAllLogs()
		assert.NotEqual(t, 2, len(logs), "Old logs should be purged")
	})

	t.Run("StructuredJSON", func(t *testing.T) {
		entry := AuditLogEntry{
			UserID:         "user-123",
			Username:       "testuser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/chat/completions",
			IPAddress:      "192.168.1.100",
			Result:         "success",
			Details:        "Model: gpt-4, Tokens: 100",
		}

		jsonOutput, err := logger.FormatAsJSON(entry)
		require.NoError(t, err)
		assert.NotEmpty(t, jsonOutput)
		assert.Contains(t, jsonOutput, "user_id")
		assert.Contains(t, jsonOutput, "operation_type")
		assert.Contains(t, jsonOutput, "result")
	})
}

// TestAuditLogQuery tests querying audit logs
func TestAuditLogQuery(t *testing.T) {
	logger := NewAuditLogger()

	t.Run("QueryByUser", func(t *testing.T) {
		logger.Log(AuditLogEntry{
			UserID:         "user-123",
			Username:       "testuser",
			OperationType:  "proxy_request",
			TargetResource: "/v1/models",
			IPAddress:      "192.168.1.100",
			Result:         "success",
		})

		logs, err := logger.QueryByUser("user-123")
		require.NoError(t, err)
		assert.True(t, len(logs) > 0)
	})

	t.Run("QueryByOperationType", func(t *testing.T) {
		logger.Log(AuditLogEntry{
			Username:       "user1",
			OperationType:  "login",
			TargetResource: "auth/session",
			IPAddress:      "192.168.1.100",
			Result:         "success",
		})

		logs, err := logger.QueryByOperation("login")
		require.NoError(t, err)
		assert.True(t, len(logs) > 0)
	})

	t.Run("QueryByTimeRange", func(t *testing.T) {
		start := time.Now().Add(-1 * time.Hour)
		end := time.Now().Add(1 * time.Hour)

		logger.Log(AuditLogEntry{
			Username:       "user1",
			OperationType:  "proxy_request",
			TargetResource: "/v1/chat",
			IPAddress:      "192.168.1.100",
			Result:         "success",
			Timestamp:      time.Now(),
		})

		logs, err := logger.QueryByTimeRange(start, end)
		require.NoError(t, err)
		assert.True(t, len(logs) > 0)
	})

	t.Run("QueryByResult", func(t *testing.T) {
		logger.Log(AuditLogEntry{
			Username:       "user1",
			OperationType:  "proxy_request",
			TargetResource: "/v1/models",
			IPAddress:      "192.168.1.100",
			Result:         "failure",
		})

		logs, err := logger.QueryByResult("failure")
		require.NoError(t, err)
		assert.True(t, len(logs) > 0)
	})
}

// AuditLogger is the audit logging interface
type AuditLogger interface {
	Log(entry AuditLogEntry) error
	LogAndRetrieve(entry AuditLogEntry) (*AuditLogEntry, error)
	GetRecentLogs(count int) []AuditLogEntry
	GetAllLogs() []AuditLogEntry
	FormatAsJSON(entry AuditLogEntry) (string, error)
	QueryByUser(userID string) ([]AuditLogEntry, error)
	QueryByOperation(operationType string) ([]AuditLogEntry, error)
	QueryByTimeRange(start, end time.Time) ([]AuditLogEntry, error)
	QueryByResult(result string) ([]AuditLogEntry, error)
}

type auditLogger struct {
	retention time.Duration
}

func NewAuditLogger() AuditLogger {
	return &auditLogger{
		retention: 30 * 24 * time.Hour, // 30 days default
	}
}

func NewAuditLoggerWithRetention(retention time.Duration) AuditLogger {
	return &auditLogger{
		retention: retention,
	}
}

func (al *auditLogger) Log(entry AuditLogEntry) error {
	// Implementation pending - test-first development
	return nil
}

func (al *auditLogger) LogAndRetrieve(entry AuditLogEntry) (*AuditLogEntry, error) {
	// Implementation pending - test-first development
	return nil, nil
}

func (al *auditLogger) GetRecentLogs(count int) []AuditLogEntry {
	// Implementation pending - test-first development
	return nil
}

func (al *auditLogger) GetAllLogs() []AuditLogEntry {
	// Implementation pending - test-first development
	return nil
}

func (al *auditLogger) FormatAsJSON(entry AuditLogEntry) (string, error) {
	// Implementation pending - test-first development
	return "", nil
}

func (al *auditLogger) QueryByUser(userID string) ([]AuditLogEntry, error) {
	// Implementation pending - test-first development
	return nil, nil
}

func (al *auditLogger) QueryByOperation(operationType string) ([]AuditLogEntry, error) {
	// Implementation pending - test-first development
	return nil, nil
}

func (al *auditLogger) QueryByTimeRange(start, end time.Time) ([]AuditLogEntry, error) {
	// Implementation pending - test-first development
	return nil, nil
}

func (al *auditLogger) QueryByResult(result string) ([]AuditLogEntry, error) {
	// Implementation pending - test-first development
	return nil, nil
}
