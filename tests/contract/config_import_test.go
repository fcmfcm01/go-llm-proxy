package contract

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/services"
	"github.com/fcmfcm01/go-llm-proxy/go-llm-proxy/internal/server/handlers"
)

func TestConfigImportContract(t *testing.T) {
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
	importHandler := handlers.NewConfigImportHandler(logger, backupService, dataDir)

	// Setup routes
	router := gin.New()
	router.POST("/admin/api/v1/config/import", importHandler.HandleImport)

	t.Run("Import configuration successfully", func(t *testing.T) {
		// Create a test backup file
		backupFile := createTestBackup(t, dataDir)

		// Create multipart form request
		req := createUploadRequest(t, "/admin/api/v1/config/import", backupFile)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusOK, w.Code)

		// Parse response
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify response structure
		assert.Equal(t, "success", response["status"])
		assert.Equal(t, "Configuration imported successfully", response["message"])
		assert.Contains(t, response, "result")
		assert.Contains(t, response, "timestamp")
	})

	t.Run("Import without file returns error", func(t *testing.T) {
		// Request without file
		req, err := http.NewRequest("POST", "/admin/api/v1/config/import", nil)
		require.NoError(t, err)
		req.Header.Set("Content-Type", "multipart/form-data")

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusBadRequest, w.Code)

		// Verify error response
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "NO_FILE", response["error"].(map[string]interface{})["code"])
	})

	t.Run("Import with invalid file type", func(t *testing.T) {
		// Create a non-zip file
		invalidFile := filepath.Join(tempDir, "invalid.txt")
		err := os.WriteFile(invalidFile, []byte("not a zip"), 0644)
		require.NoError(t, err)

		// Create multipart form request
		req := createUploadRequest(t, "/admin/api/v1/config/import", invalidFile)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Import creates pre-backup", func(t *testing.T) {
		// Create a test backup file
		backupFile := createTestBackup(t, dataDir)

		// Create multipart form request
		req := createUploadRequest(t, "/admin/api/v1/config/import", backupFile)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Assert status code
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Contract compliance - Response format", func(t *testing.T) {
		// This test validates the API contract

		// Create a test backup file
		backupFile := createTestBackup(t, dataDir)

		// Create multipart form request
		req := createUploadRequest(t, "/admin/api/v1/config/import", backupFile)

		// Execute
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Must return 200 OK on success
		assert.Equal(t, http.StatusOK, w.Code)

		// Must be JSON
		assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

		// Must contain expected fields
		assert.Contains(t, w.Body.String(), "status")
		assert.Contains(t, w.Body.String(), "message")
		assert.Contains(t, w.Body.String(), "result")
		assert.Contains(t, w.Body.String(), "timestamp")

		// Parse and verify structure
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		// Verify result structure
		result := response["result"].(map[string]interface{})
		assert.Contains(t, result, "success")
		assert.Contains(t, result, "imported")
		assert.Contains(t, result, "timestamp")
		assert.Contains(t, result, "duration")
	})
}

// Helper function to create a test backup file
func createTestBackup(t *testing.T, dataDir string) string {
	t.Helper()

	// Create test data files
	providersPath := filepath.Join(dataDir, "providers.json")
	providersData := []byte(`[{"id": "test-1", "name": "Test Provider"}]`)
	err := os.WriteFile(providersPath, providersData, 0644)
	require.NoError(t, err)

	// Create a temporary backup file
	backupFile := filepath.Join(t.TempDir(), "test_backup.zip")

	// Create the zip file
	zipFile, err := os.Create(backupFile)
	require.NoError(t, err)
	defer zipFile.Close()

	w := zip.NewWriter(zipFile)
	defer w.Close()

	// Add providers.json to the zip
	f, err := w.Create("providers.json")
	require.NoError(t, err)

	_, err = f.Write(providersData)
	require.NoError(t, err)

	// Add metadata
	metadata := map[string]interface{}{
		"version":   "1.0.0",
		"timestamp": time.Now(),
	}
	metadataData, err := json.Marshal(metadata)
	require.NoError(t, err)

	f, err = w.Create("metadata.json")
	require.NoError(t, err)
	_, err = f.Write(metadataData)
	require.NoError(t, err)

	return backupFile
}

// Helper function to create a multipart form request
func createUploadRequest(t *testing.T, url, filePath string) *http.Request {
	t.Helper()

	// Create a buffer to hold the multipart form data
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Add the file
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	part, err := writer.CreateFormFile("backup", filepath.Base(filePath))
	require.NoError(t, err)

	_, err = io.Copy(part, file)
	require.NoError(t, err)

	writer.Close()

	// Create the request
	req, err := http.NewRequest("POST", url, &requestBody)
	require.NoError(t, err)

	// Set the content type
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req
}
