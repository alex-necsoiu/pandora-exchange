package service

import (
	"context"
	"fmt"
	"time"

	"github.com/alex-necsoiu/pandora-exchange/internal/domain"
	"github.com/alex-necsoiu/pandora-exchange/internal/observability"
)

// AuditCleanupJob handles periodic cleanup of expired audit logs
type AuditCleanupJob struct {
	auditRepo       domain.AuditRepository
	logger          *observability.Logger
	cleanupInterval time.Duration
	stopChan        chan struct{}
	doneChan        chan struct{}
}

// NewAuditCleanupJob creates a new audit cleanup job
func NewAuditCleanupJob(
	auditRepo domain.AuditRepository,
	logger *observability.Logger,
	cleanupInterval time.Duration,
) *AuditCleanupJob {
	return &AuditCleanupJob{
		auditRepo:       auditRepo,
		logger:          logger,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
		doneChan:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup job
// Runs in a goroutine and can be stopped with Stop()
func (j *AuditCleanupJob) Start(ctx context.Context) {
	j.logger.WithField("interval", j.cleanupInterval.String()).Info("Starting audit log cleanup job")

	// Run cleanup immediately on start
	if err := j.runCleanup(ctx); err != nil {
		j.logger.WithError(err).Error("Initial audit log cleanup failed")
	}

	// Start ticker for periodic cleanup
	ticker := time.NewTicker(j.cleanupInterval)
	defer ticker.Stop()

	go func() {
		defer close(j.doneChan)

		for {
			select {
			case <-ticker.C:
				if err := j.runCleanup(ctx); err != nil {
					j.logger.WithError(err).Error("Scheduled audit log cleanup failed")
				}
			case <-j.stopChan:
				j.logger.Info("Audit cleanup job stopped")
				return
			case <-ctx.Done():
				j.logger.Info("Audit cleanup job context cancelled")
				return
			}
		}
	}()
}

// Stop gracefully stops the cleanup job
func (j *AuditCleanupJob) Stop() {
	j.logger.Info("Stopping audit cleanup job")
	close(j.stopChan)
	<-j.doneChan
	j.logger.Info("Audit cleanup job stopped successfully")
}

// runCleanup executes the cleanup operation
func (j *AuditCleanupJob) runCleanup(ctx context.Context) error {
	startTime := time.Now()
	j.logger.Debug("Running audit log cleanup")

	// Create a timeout context for the cleanup operation
	cleanupCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Execute cleanup
	err := j.auditRepo.DeleteExpired(cleanupCtx)
	if err != nil {
		return fmt.Errorf("failed to delete expired audit logs: %w", err)
	}

	duration := time.Since(startTime)
	j.logger.WithFields(map[string]interface{}{
		"duration_ms": duration.Milliseconds(),
	}).Info("Audit log cleanup completed successfully")

	return nil
}

// RunOnce executes a single cleanup operation (useful for testing)
func (j *AuditCleanupJob) RunOnce(ctx context.Context) error {
	return j.runCleanup(ctx)
}
