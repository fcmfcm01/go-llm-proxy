package unit

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/internal/repository"
)

func TestAuditLogCompleteness(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create temporary directory
	tempDir := t.TempDir()

	// Create repository
	repo := repository.NewAuditLogRepository(logger, tempDir)

	t.Run("Required fields are present in audit log", func(t *testing.T) {
		// Create an audit log with all required fields
		log := &repository.AuditLog{
			ID:            "test-123",
			Timestamp:     time.Now(),
			Level:         "info",
			Action:        "create",
			Entity:        "provider",
			EntityID:      "prov-456",
			Actor:         "admin",
			IPAddress:     "192.168.1.1",
			UserAgent:     "TestAgent/1.0",
			Status:        "success",
			Message:       "Test message",
			Duration:      100,
			CorrelationID: "corr-789",
		}

		// Create the log
		err := repo.Create(log)
		require.NoError(t, err)

		// Retrieve the log
		retrievedLog, err := repo.GetByID("test-123")
		require.NoError(t, err)

		// Verify all required fields are present
		assert.Equal(t, "test-123", retrievedLog.ID)
		assert.False(t, retrievedLog.Timestamp.IsZero(), "Timestamp should be set")
		assert.Equal(t, "info", retrievedLog.Level)
		assert.Equal(t, "create", retrievedLog.Action)
		assert.Equal(t, "provider", retrievedLog.Entity)
		assert.Equal(t, "prov-456", retrievedLog.EntityID)
		assert.Equal(t, "admin", retrievedLog.Actor)
		assert.Equal(t, "192.168.1.1", retrievedLog.IPAddress)
		assert.Equal(t, "TestAgent/1.0", retrievedLog.UserAgent)
		assert.Equal(t, "success", retrievedLog.Status)
		assert.Equal(t, "Test message", retrievedLog.Message)
		assert.Equal(t, int64(100), retrievedLog.Duration)
		assert.Equal(t, "corr-789", retrievedLog.CorrelationID)
	})

	t.Run("All required fields are present for different actions", func(t *testing.T) {
		actions := []string{"create", "update", "delete", "login", "logout", "view", "export", "import"}
		entities := []string{"provider", "model_mapping", "user", "config", "session"}
		statuses := []string{"success", "failure", "error"}

		for _, action := range actions {
			for _, entity := range entities {
				for _, status := range statuses {
					logID := "test-" + action + "-" + entity + "-" + status

					log := &repository.AuditLog{
						ID:        logID,
						Timestamp: time.Now(),
						Level:     "info",
						Action:    action,
						Entity:    entity,
						EntityID:  "test-id-123",
						Actor:     "test-user",
						IPAddress: "10.0.0.1",
						UserAgent: "TestAgent/1.0",
						Status:    status,
						Message:   "Test " + action + " on " + entity,
						Duration:  50,
					}

					err := repo.Create(log)
					require.NoError(t, err, "Failed to create log for action=%s, entity=%s, status=%s", action, entity, status)

					// Retrieve and verify
					retrievedLog, err := repo.GetByID(logID)
					require.NoError(t, err)

					// Verify all required fields
					assert.NotEmpty(t, retrievedLog.ID)
					assert.False(t, retrievedLog.Timestamp.IsZero())
					assert.NotEmpty(t, retrievedLog.Action)
					assert.NotEmpty(t, retrievedLog.Entity)
					assert.NotEmpty(t, retrievedLog.Actor)
					assert.NotEmpty(t, retrievedLog.Status)
					assert.NotEmpty(t, retrievedLog.Message)
				}
			}
		}
	})

	t.Run("Metadata is captured correctly", func(t *testing.T) {
		metadata := map[string]interface{}{
			"field1": "value1",
			"field2": 123,
			"field3": true,
			"nested": map[string]interface{}{
				"key1": "val1",
				"key2": 456,
			},
		}

		log := &repository.AuditLog{
			ID:        "test-metadata",
			Timestamp: time.Now(),
			Action:    "create",
			Entity:    "provider",
			Actor:     "admin",
			Status:    "success",
			Message:   "Test with metadata",
			Metadata:  metadata,
		}

		err := repo.Create(log)
		require.NoError(t, err)

		retrievedLog, err := repo.GetByID("test-metadata")
		require.NoError(t, err)

		// Verify metadata is preserved
		assert.Equal(t, metadata["field1"], retrievedLog.Metadata["field1"])
		assert.Equal(t, metadata["field2"], retrievedLog.Metadata["field2"])
		assert.Equal(t, metadata["field3"], retrievedLog.Metadata["field3"])
		assert.Equal(t, metadata["nested"], retrievedLog.Metadata["nested"])
	})

	t.Run("Query returns complete audit logs", func(t *testing.T) {
		// Create multiple logs
		for i := 0; i < 10; i++ {
			log := &repository.AuditLog{
				ID:        "query-test-" + string(rune(i)),
				Timestamp: time.Now(),
				Action:    "view",
				Entity:    "provider",
				Actor:     "user" + string(rune(i)),
				Status:    "success",
				Message:   "View operation " + string(rune(i)),
				Duration:  int64(i * 10),
			}
			err := repo.Create(log)
			require.NoError(t, err)
		}

		// Query all logs
		params := repository.AuditLogQueryParams{
			Limit:     50,
			Offset:    0,
			SortBy:    "timestamp",
			SortOrder: "desc",
		}

		logs, total, err := repo.Query(params)
		require.NoError(t, err)
		assert.True(t, total >= 10, "Should have at least 10 logs")

		// Verify all returned logs have all required fields
		for _, log := range logs {
			assert.NotEmpty(t, log.ID, "ID should not be empty")
			assert.False(t, log.Timestamp.IsZero(), "Timestamp should be set")
			assert.NotEmpty(t, log.Action, "Action should not be empty")
			assert.NotEmpty(t, log.Entity, "Entity should not be empty")
			assert.NotEmpty(t, log.Actor, "Actor should not be empty")
			assert.NotEmpty(t, log.Status, "Status should not be empty")
			assert.NotEmpty(t, log.Message, "Message should not be empty")
		}
	})

	t.Run("Logs are ordered by timestamp", func(t *testing.T) {
		// Create logs with different timestamps
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
		log1 := &repository.AuditLog{
			ID:        "order-test-1",
			Timestamp: time.Now(),
			Action:    "create",
			Entity:    "provider",
			Actor:     "admin",
			Status:    "success",
			Message:   "First log",
		}
		time.Sleep(10 * time.Millisecond)
		log2 := &repository.AuditLog{
			ID:        "order-test-2",
			Timestamp: time.Now(),
			Action:    "update",
			Entity:    "provider",
			Actor:     "admin",
			Status:    "success",
			Message:   "Second log",
		}

		err := repo.Create(log1)
		require.NoError(t, err)
		err = repo.Create(log2)
		require.NoError(t, err)

		// Query with ascending order
		params := repository.AuditLogQueryParams{
			Limit:     50,
			Offset:    0,
			SortBy:    "timestamp",
			SortOrder: "asc",
		}

		logs, _, err := repo.Query(params)
		require.NoError(t, err)

		// Find our test logs
		var test1, test2 *repository.AuditLog
		for i := range logs {
			if logs[i].ID == "order-test-1" {
				test1 = &logs[i]
			}
			if logs[i].ID == "order-test-2" {
				test2 = &logs[i]
			}
		}

		require.NotNil(t, test1, "First log should be found")
		require.NotNil(t, test2, "Second log should be found")
		assert.True(t, test1.Timestamp.Before(test2.Timestamp), "First log should be before second log")
	})

	t.Run("Required fields from specifications are present", func(t *testing.T) {
		// According to FR-032, audit logs should include: action, entity, actor, status, message, timestamp
		// Plus additional fields for forensic analysis

		log := &repository.AuditLog{
			ID:            "spec-test",
			Timestamp:     time.Now(),
			Action:        "create",
			Entity:        "provider",
			Actor:         "admin",
			Status:        "success",
			Message:       "Created provider",
			IPAddress:     "192.168.1.100",
			UserAgent:     "Mozilla/5.0",
			Duration:      250,
			CorrelationID: "req-123",
		}

		err := repo.Create(log)
		require.NoError(t, err)

		retrievedLog, err := repo.GetByID("spec-test")
		require.NoError(t, err)

		// Verify specification-required fields
		assert.Equal(t, "create", retrievedLog.Action, "Action is required by FR-032")
		assert.Equal(t, "provider", retrievedLog.Entity, "Entity is required by FR-032")
		assert.Equal(t, "admin", retrievedLog.Actor, "Actor is required by FR-032")
		assert.Equal(t, "success", retrievedLog.Status, "Status is required by FR-032")
		assert.Equal(t, "Created provider", retrievedLog.Message, "Message is required by FR-032")
		assert.False(t, retrievedLog.Timestamp.IsZero(), "Timestamp is required by FR-032")

		// Verify additional forensic fields
		assert.NotEmpty(t, retrievedLog.IPAddress, "IP address required for forensic analysis")
		assert.NotEmpty(t, retrievedLog.UserAgent, "User agent required for forensic analysis")
		assert.True(t, retrievedLog.Duration > 0, "Duration should be measured")
		assert.NotEmpty(t, retrievedLog.ID, "ID is required for tracking")
		assert.NotEmpty(t, retrievedLog.CorrelationID, "Correlation ID is required for request tracing")
	})
}
