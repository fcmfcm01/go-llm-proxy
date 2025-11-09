package auth

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
)

// SessionCleanupJob handles periodic cleanup of expired sessions
type SessionCleanupJob struct {
	authService *AuthService
	logger      *logrus.Logger
	interval    time.Duration
	stopChan    chan struct{}
}

// NewSessionCleanupJob creates a new session cleanup job
func NewSessionCleanupJob(authService *AuthService, logger *logrus.Logger, interval time.Duration) *SessionCleanupJob {
	if interval == 0 {
		interval = 1 * time.Hour // Default cleanup every hour
	}

	return &SessionCleanupJob{
		authService: authService,
		logger:      logger,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

// Start starts the cleanup job
func (j *SessionCleanupJob) Start() {
	go j.run()
}

// Stop stops the cleanup job
func (j *SessionCleanupJob) Stop() {
	close(j.stopChan)
}

// run runs the cleanup loop
func (j *SessionCleanupJob) run() {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	if j.logger != nil {
		j.logger.Info("Session cleanup job started")
	}

	// Run cleanup immediately on start
	j.cleanup()

	for {
		select {
		case <-ticker.C:
			j.cleanup()
		case <-j.stopChan:
			if j.logger != nil {
				j.logger.Info("Session cleanup job stopped")
			}
			return
		}
	}
}

// cleanup performs the actual cleanup
func (j *SessionCleanupJob) cleanup() {
	ctx := context.Background()

	count, err := j.authService.CleanupExpiredSessions(ctx)
	if err != nil {
		if j.logger != nil {
			j.logger.WithError(err).Error("Failed to cleanup expired sessions")
		}
		return
	}

	if count > 0 && j.logger != nil {
		j.logger.WithField("count", count).Info("Cleaned up expired sessions")
	}
}

// RunOnce runs cleanup once (useful for testing)
func (j *SessionCleanupJob) RunOnce() error {
	ctx := context.Background()
	_, err := j.authService.CleanupExpiredSessions(ctx)
	return err
}
