package middleware

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// CompressionConfig holds compression middleware configuration
type CompressionConfig struct {
	Level           int      // gzip compression level (1-9)
	MinSize         int      // Minimum response size to compress
	ExcludedPaths   []string // Paths to exclude from compression
	ExcludedTypes   []string // Content types to exclude from compression
}

// DefaultCompressionConfig returns default compression configuration
func DefaultCompressionConfig() *CompressionConfig {
	return &CompressionConfig{
		Level:         gzip.BestCompression,
		MinSize:       1024, // 1KB
		ExcludedPaths: []string{
			"/metrics",
			"/admin/api/v1/audit-logs/export",
		},
		ExcludedTypes: []string{
			"image/",
			"video/",
			"audio/",
			"application/zip",
			"application/x-gzip",
			"application/gzip",
		},
	}
}

// CompressionMiddleware provides response compression
func CompressionMiddleware(config *CompressionConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCompressionConfig()
	}

	// Pool of gzip writers for reuse
	gzipPool := &sync.Pool{
		New: func() interface{} {
			// Create a gzip writer with default settings
			// Will be configured per request
			return &gzip.Writer{}
		},
	}

	return func(c *gin.Context) {
		// Check if compression is supported
		if !shouldCompress(c.Request, config) {
			c.Next()
			return
		}

		// Check path exclusion
		if isExcludedPath(c.Request.URL.Path, config.ExcludedPaths) {
			c.Next()
			return
		}

		// Get the original response writer
		originalWriter := c.Writer

		// Create a compression writer
		compressedWriter := NewCompressionWriter(originalWriter, gzipPool, config.Level)
		defer compressedWriter.Close()

		// Replace the response writer
		c.Writer = compressedWriter

		// Set Vary header to indicate content is compressed
		c.Header("Vary", "Accept-Encoding")

		// Set Content-Encoding header
		c.Header("Content-Encoding", "gzip")

		// Process the request
		c.Next()

		// Write the compressed response
		compressedWriter.Flush()
	}
}

// shouldCompress checks if the request should be compressed
func shouldCompress(req *http.Request, config *CompressionConfig) bool {
	// Check if client accepts gzip
	acceptEncoding := req.Header.Get("Accept-Encoding")
	if acceptEncoding == "" {
		return false
	}

	// Check if client accepts gzip encoding
	if !strings.Contains(strings.ToLower(acceptEncoding), "gzip") {
		return false
	}

	// Don't compress if already compressed
	contentEncoding := req.Header.Get("Content-Encoding")
	if contentEncoding != "" {
		return false
	}

	// Check content type
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		for _, excludedType := range config.ExcludedTypes {
			if strings.HasPrefix(contentType, excludedType) {
				return false
			}
		}
	}

	return true
}

// isExcludedPath checks if a path should be excluded from compression
func isExcludedPath(path string, excludedPaths []string) bool {
	for _, excludedPath := range excludedPaths {
		if strings.HasPrefix(path, excludedPath) {
			return true
		}
	}
	return false
}

// CompressionWriter wraps gin.ResponseWriter to add compression
type CompressionWriter struct {
	gin.ResponseWriter
	gzipPool *sync.Pool
	gzip     *gzip.Writer
	level    int
	closed   bool
}

// NewCompressionWriter creates a new compression writer
func NewCompressionWriter(writer gin.ResponseWriter, gzipPool *sync.Pool, level int) *CompressionWriter {
	cw := &CompressionWriter{
		ResponseWriter: writer,
		gzipPool:       gzipPool,
		level:          level,
	}

	return cw
}

// Write writes compressed data
func (cw *CompressionWriter) Write(data []byte) (int, error) {
	if cw.closed {
		return 0, errors.New("body too large")
	}

	// Get gzip writer from pool
	gz := cw.gzipPool.Get().(*gzip.Writer)
	defer cw.gzipPool.Put(gz)

	// Initialize gzip writer if not already done
	if gz == nil {
		return 0, errors.New("body too large")
	}

	// Reset and configure the gzip writer
	gz.Reset(cw.ResponseWriter)

	// Write data through gzip
	n, err := gz.Write(data)
	if err != nil {
		return n, err
	}

	// Update status code if not set
	if cw.Status() == http.StatusOK {
		cw.ResponseWriter.WriteHeader(http.StatusOK)
	}

	return n, nil
}

// WriteHeader writes the response header
func (cw *CompressionWriter) WriteHeader(statusCode int) {
	// Set Content-Type if not already set
	if cw.Header().Get("Content-Type") == "" {
		cw.Header().Set("Content-Type", "text/html; charset=utf-8")
	}

	cw.ResponseWriter.WriteHeader(statusCode)
}

// Flush flushes the compressed data
func (cw *CompressionWriter) Flush() {
	if cw.closed {
		return
	}

	// Flush the gzip writer if available
	if cw.gzip != nil {
		cw.gzip.Flush()
	}
}

// Close closes the compression writer
func (cw *CompressionWriter) Close() error {
	if cw.closed {
		return nil
	}
	cw.closed = true

	// Close gzip writer if available
	if cw.gzip != nil {
		err := cw.gzip.Close()
		cw.gzip = nil
		return err
	}

	return nil
}

// DecompressionMiddleware provides request decompression
func DecompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request is compressed
		contentEncoding := c.Request.Header.Get("Content-Encoding")
		if !strings.Contains(strings.ToLower(contentEncoding), "gzip") {
			c.Next()
			return
		}

		// Read the compressed body
		compressedBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read compressed body",
			})
			c.Abort()
			return
		}

		// Decompress
		gz, err := gzip.NewReader(strings.NewReader(string(compressedBody)))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to decompress body",
			})
			c.Abort()
			return
		}
		defer gz.Close()

		// Read decompressed body
		decompressedBody, err := io.ReadAll(gz)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read decompressed body",
			})
			c.Abort()
			return
		}

		// Replace request body with decompressed data
		c.Request.Body = io.NopCloser(strings.NewReader(string(decompressedBody)))

		// Remove Content-Encoding header
		c.Request.Header.Del("Content-Encoding")

		c.Next()
	}
}

// NoCompressionMiddleware disables compression for specific endpoints
func NoCompressionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set header to indicate no compression
		c.Header("Cache-Control", "no-transform")
		c.Next()
	}
}
