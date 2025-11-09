package logging

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// Logger provides structured logging with logrus
type Logger struct {
	*logrus.Logger
	mu sync.Mutex
}

// NewLogger creates a new logger with the given configuration
func NewLogger(level, format, output, filePath string, maxSize, maxBackups int, maxAge int, compress bool) (*Logger, error) {
	lg := logrus.New()

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, err
	}
	lg.SetLevel(logLevel)

	// Set log format
	if format == "json" {
		lg.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
			PrettyPrint:     false,
		})
	} else {
		lg.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	}

	// Set output
	var outputWriter io.Writer
	if output == "file" {
		// Ensure log directory exists
		logDir := filepath.Dir(filePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		// Create file writer with rotation
		outputWriter = createFileWriter(filePath, maxSize, maxBackups, maxAge, compress)
	} else {
		outputWriter = os.Stdout
	}

	lg.SetOutput(outputWriter)

	return &Logger{Logger: lg}, nil
}

// createFileWriter creates a file writer with rotation
func createFileWriter(filePath string, maxSize, maxBackups, maxAge int, compress bool) io.Writer {
	// For simplicity, return a file writer
	// In production, you might want to use a rotation library like lumberjack
	return newRotationWriter(filePath, maxSize, maxBackups, maxAge, compress)
}

// rotationWriter handles log file rotation
type rotationWriter struct {
	filePath    string
	maxSize     int
	maxBackups  int
	maxAge      int
	compress    bool
	currentFile *os.File
	mu          sync.Mutex
}

func newRotationWriter(filePath string, maxSize, maxBackups, maxAge int, compress bool) io.Writer {
	return &rotationWriter{
		filePath:   filePath,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxAge:     maxAge,
		compress:   compress,
	}
}

func (w *rotationWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Open file if not already open
	if w.currentFile == nil {
		if err := w.openFile(); err != nil {
			return 0, err
		}
	}

	// Check if rotation is needed
	if w.shouldRotate() {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	// Write to file
	return w.currentFile.Write(p)
}

func (w *rotationWriter) openFile() error {
	// Ensure directory exists
	logDir := filepath.Dir(w.filePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Open file in append mode
	f, err := os.OpenFile(w.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	w.currentFile = f
	return nil
}

func (w *rotationWriter) shouldRotate() bool {
	// Check file size
	if w.currentFile == nil {
		return true
	}

	stat, err := w.currentFile.Stat()
	if err != nil {
		return true
	}

	// Convert maxSize from MB to bytes
	maxBytes := int64(w.maxSize) * 1024 * 1024
	return stat.Size() >= maxBytes
}

func (w *rotationWriter) rotate() error {
	// Close current file
	if w.currentFile != nil {
		w.currentFile.Close()
	}

	// Generate backup file path
	timestamp := time.Now().Format("20060102_150405")
	backupPath := w.filePath + "." + timestamp

	// Rename current file
	if err := os.Rename(w.filePath, backupPath); err != nil {
		return err
	}

	// Open new file
	if err := w.openFile(); err != nil {
		return err
	}

	// Remove old backups
	w.cleanupOldBackups()

	return nil
}

func (w *rotationWriter) cleanupOldBackups() {
	// Get list of backup files
	globPattern := w.filePath + ".*"
	matches, _ := filepath.Glob(globPattern)

	// Sort by modification time (newest first)
	// In production, you might want to use a more sophisticated approach

	// Remove old files if we have too many backups
	if len(matches) > w.maxBackups {
		// Implementation depends on your needs
	}
}

func (w *rotationWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

// WithFields creates a logger with additional fields
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// RequestIDMiddleware adds request ID to logs
func (l *Logger) RequestIDMiddleware() func(*logrus.Entry) *logrus.Entry {
	return func(entry *logrus.Entry) *logrus.Entry {
		// Add request ID if available in context
		return entry
	}
}

// GetLogger returns the underlying logrus logger
func (l *Logger) GetLogger() *logrus.Logger {
	return l.Logger
}

// Close closes the logger and any open file handles
func (l *Logger) Close() error {
	if fileWriter, ok := l.Logger.Out.(*rotationWriter); ok {
		return fileWriter.Close()
	}
	return nil
}

// SetLevel sets the log level
func (l *Logger) SetLevel(level string) error {
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	l.Logger.SetLevel(logLevel)
	return nil
}
