package services

import (
	"time"

	"github.com/sirupsen/logrus"
)

// BackupJob handles scheduled backup operations
type BackupJob struct {
	logger         *logrus.Logger
	backupService  *BackupService
	interval       time.Duration
	stopCh         chan bool
	isRunning      bool
}

// NewBackupJob creates a new backup job
func NewBackupJob(logger *logrus.Logger, backupService *BackupService, interval time.Duration) *BackupJob {
	return &BackupJob{
		logger:        logger,
		backupService: backupService,
		interval:      interval,
		stopCh:        make(chan bool, 1),
		isRunning:     false,
	}
}

// Start starts the backup job
func (j *BackupJob) Start() {
	if j.isRunning {
		j.logger.Warn("Backup job is already running")
		return
	}

	j.isRunning = true
	j.logger.Infof("Starting backup job with interval: %v", j.interval)

	go j.run()
}

// Stop stops the backup job
func (j *BackupJob) Stop() {
	if !j.isRunning {
		j.logger.Warn("Backup job is not running")
		return
	}

	j.logger.Info("Stopping backup job...")
	j.stopCh <- true
	j.isRunning = false
}

// run runs the backup job loop
func (j *BackupJob) run() {
	// Run immediately on start
	j.runOnce()

	// Set up ticker
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			j.runOnce()
		case <-j.stopCh:
			j.logger.Info("Backup job stopped")
			return
		}
	}
}

// runOnce performs a single backup operation
func (j *BackupJob) runOnce() {
	j.logger.Info("Starting scheduled backup...")

	config := BackupConfig{
		IncludeProviders:     true,
		IncludeModelMappings: true,
		IncludeSettings:      true,
		Compression:          true,
		Timestamp:            time.Now(),
		Version:              "1.0.0",
	}

	result, err := j.backupService.CreateBackup(config)
	if err != nil {
		j.logger.Errorf("Scheduled backup failed: %v", err)
	} else {
		j.logger.Infof("Scheduled backup completed successfully: %s", result.BackupPath)
	}
}

// IsRunning returns whether the job is running
func (j *BackupJob) IsRunning() bool {
	return j.isRunning
}
