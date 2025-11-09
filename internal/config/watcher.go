package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// Watcher monitors config files for changes and reloads them
type Watcher struct {
	watcher    *fsnotify.Watcher
	configPath string
	mu         sync.RWMutex
	reloadFunc func() error
	logger     *logrus.Logger
	ctx        context.Context
	cancel     context.CancelFunc
	eventChan  chan fsnotify.Event
	quit       chan bool
	backoff    time.Duration
	maxBackoff time.Duration
}

// NewWatcher creates a new file watcher
func NewWatcher(configPath string, reloadFunc func() error, logger *logrus.Logger) *Watcher {
	ctx, cancel := context.WithCancel(context.Background())

	return &Watcher{
		configPath: configPath,
		reloadFunc: reloadFunc,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		eventChan:  make(chan fsnotify.Event, 100),
		quit:       make(chan bool),
		backoff:    time.Second,
		maxBackoff: time.Minute,
	}
}

// Start starts watching the config file
func (w *Watcher) Start() error {
	// Create watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file watcher: %w", err)
	}
	w.watcher = watcher

	// Watch the config directory
	configDir := filepath.Dir(w.configPath)
	if err := w.watcher.Add(configDir); err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", configDir, err)
	}

	w.logger.WithField("path", configDir).Info("Started watching config directory")

	// Start watching in a goroutine
	go w.watch()

	// Also watch the specific config file
	if err := w.watcher.Add(w.configPath); err != nil {
		w.logger.WithField("path", w.configPath).Warning("Could not watch config file directly")
	}

	return nil
}

// watch watches for file changes
func (w *Watcher) watch() {
	for {
		select {
		case <-w.ctx.Done():
			w.cleanup()
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Filter events
			if !w.shouldProcessEvent(event) {
				continue
			}

			w.logger.WithFields(logrus.Fields{
				"event": event.Op.String(),
				"path":  event.Name,
			}).Debug("Config file change detected")

			// Debounce rapid changes
			time.Sleep(100 * time.Millisecond)

			// Reload configuration
			w.reload()

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.WithField("error", err).Error("File watcher error")
		}
	}
}

// shouldProcessEvent determines if an event should be processed
func (w *Watcher) shouldProcessEvent(event fsnotify.Event) bool {
	// Check if it's a config file
	if !filepath.Base(event.Name) == filepath.Base(w.configPath) {
		return false
	}

	// Only process certain operations
	validOps := []fsnotify.Op{
		fsnotify.Write,
		fsnotify.Create,
		fsnotify.Remove,
		fsnotify.Rename,
	}

	for _, op := range validOps {
		if event.Op&op != 0 {
			return true
		}
	}

	return false
}

// reload reloads the configuration
func (w *Watcher) reload() {
	w.mu.RLock()
	reloadFunc := w.reloadFunc
	w.mu.RUnlock()

	// Add jitter to prevent thundering herd
	time.Sleep(time.Duration(float64(w.backoff) * (0.5 + 0.5*randFloat())))

	if err := reloadFunc(); err != nil {
		w.logger.WithError(err).Error("Failed to reload configuration")

		// Exponential backoff on error
		w.backoff = time.Min(float64(w.backoff*2), float64(w.maxBackoff))
	} else {
		w.logger.Info("Configuration reloaded successfully")
		w.backoff = time.Second
	}
}

// Stop stops the watcher
func (w *Watcher) Stop() error {
	w.logger.Info("Stopping file watcher")
	w.cancel()

	select {
	case <-w.quit:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for watcher to stop")
	}
}

// cleanup cleans up resources
func (w *Watcher) cleanup() {
	if w.watcher != nil {
		w.watcher.Close()
	}
	w.quit <- true
	close(w.quit)
	w.logger.Info("File watcher stopped")
}

// AddWatch adds a new path to watch
func (w *Watcher) AddWatch(path string) error {
	if err := w.watcher.Add(path); err != nil {
		return fmt.Errorf("failed to add watch for %s: %w", path, err)
	}
	w.logger.WithField("path", path).Info("Added watch")
	return nil
}

// RemoveWatch removes a watched path
func (w *Watcher) RemoveWatch(path string) error {
	if err := w.watcher.Remove(path); err != nil {
		return fmt.Errorf("failed to remove watch for %s: %w", path, err)
	}
	w.logger.WithField("path", path).Info("Removed watch")
	return nil
}

// GetWatchedPaths returns the list of watched paths
func (w *Watcher) GetWatchedPaths() ([]string, error) {
	events := w.watcher.Events()
	paths := make([]string, 0)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return paths, nil
			}
			if !containsPath(paths, event.Name) {
				paths = append(paths, event.Name)
			}
		default:
			return paths, nil
		}
	}
}

// WatchConfigFile watches a specific config file with a callback
func WatchConfigFile(configPath string, callback func() error, logger *logrus.Logger) error {
	watcher := NewWatcher(configPath, callback, logger)

	// Ensure config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist: %s", configPath)
	}

	// Start watching
	if err := watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	logger.WithField("path", configPath).Info("Config file watching started")

	// Keep running
	<-watcher.ctx.Done()

	return nil
}

// containsPath checks if a path exists in a slice
func containsPath(paths []string, path string) bool {
	for _, p := range paths {
		if p == path {
			return true
		}
	}
	return false
}

// randFloat returns a random float between 0 and 1
func randFloat() float64 {
	// Simple pseudo-random for jitter
	return float64(time.Now().UnixNano()%100) / 100.0
}
