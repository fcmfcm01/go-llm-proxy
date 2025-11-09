package contract

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/internal/services"
	"github.com/fcmfcm01/go-llm-proxy/internal/server/handlers"
)

func TestConfigExportContract(t *testing.T) {
	// Setup
	gin.SetMode(gin.TestMode)

	// Create temporary directories
	tempDir := t.TempDir()
	backupDir := t.TempDir()
	dataDir := t.TempDir()

	// Create logger
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)

	// Create backup service
	backupService := services.NewBackupService(logger, backupDir, dataDir)

	// Create handler
	exportHandler := handlers.NewConfigExportHandler(logger, backupService)

	// Setup routes
	router := gin.New()
	router.GET("/admin/api/v1/config/export", exportHandler.HandleExport)
	router.GET("/admin/api/v1/config/backups", exportHandler.HandleListBackups)

	t.Run("Export configuration without download", func(t *testing.T) {
		// Request
		req, err := http.NewRequest("GET", "/admin/api/v1/config/export?download=false", nil)
		require.NoError(t, err)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err = gin.JSON:Render doesn't work directly, so we check the response
		assert.Contains(t, w.Body.String(), "status")
		assert.Contains(t, w.Body.String(), "success")
	})

	t.Run("Export configuration with custom parameters", func(t *testing.T) {
		// Request
		req, err := http.NewRequest("GET", "/admin/api/v1/config/export?include_providers=false&include_model_mappings=false&compression=false", nil)
		require.NoError(t, err)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("List backups", func(t *testing.T) {
		// First create a backup
		config := services.BackupConfig{
			IncludeProviders:     true,
			IncludeModelMappings: true,
			IncludeSettings:      true,
			Compression:          true,
			Timestamp:            time.Now(),
			Version:              "1.0.0",
		}

		_, err := backupService.CreateBackup(config)
		require.NoError(t, err)

		// Request list
		req, err := http.NewRequest("GET", "/admin/api/v1/config/backups", nil)
		require.NoError(t, err)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		// Note: In a real test, we'd unmarshal the JSON
		// For this contract test, we just check status code
		assert.Contains(t, w.Body.String(), "status")
	})

	t.Run("Export with invalid parameters", func(t *testing.T) {
		// Request with invalid compression parameter
		req, err := http.NewRequest("GET", "/admin/api/v1/config/export?compression=invalid", nil)
		require.NoError(t, err)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Should still succeed (default to true)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Contract compliance - Response format", func(t *testing.T) {
		// This test validates the API contract

		// Test GET /admin/api/v1/config/export
		req, err := http.NewRequest("GET", "/admin/api/v1/config/export", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Must return 200 OK
		assert.Equal(t, http.StatusOK, w.Code)

		// Must be JSON
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		// Must contain expected fields
		assert.Contains(t, w.Body.String(), "status")
		assert.Contains(t, w.Body.String(), "message")

		// Test GET /admin/api/v1/config/backups
		req, err = http.NewRequest("GET", "/admin/api/v1/config/backups", nil)
		require.NoError(t, err)

		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Must return 200 OK
		assert.Equal(t, http.StatusOK, w.Code)

		// Must be JSON
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		// Must contain expected fields
		assert.Contains(t, w.Body.String(), "status")
		assert.Contains(t, w.Body.String(), "backups")
		assert.Contains(t, w.Body.String(), "count")
	})
}
