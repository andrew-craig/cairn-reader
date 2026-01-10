package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
)

// OutboxCleanupJobConfig holds configuration for the outbox cleanup job
type OutboxCleanupJobConfig struct {
	// RetentionDays is the number of days to retain delivered outbox entries
	RetentionDays int
	// BatchSize is the number of entries to delete per batch
	BatchSize int
	// FailedEntryLogLimit is the maximum number of failed entries to log
	FailedEntryLogLimit int
}

// DefaultOutboxCleanupJobConfig returns default configuration
func DefaultOutboxCleanupJobConfig() *OutboxCleanupJobConfig {
	return &OutboxCleanupJobConfig{
		RetentionDays:       7,
		BatchSize:           1000,
		FailedEntryLogLimit: 100,
	}
}

// OutboxCleanupJob handles cleanup of old outbox entries
type OutboxCleanupJob struct {
	config     *OutboxCleanupJobConfig
	outboxRepo repository.OutboxRepository
}

// NewOutboxCleanupJob creates a new OutboxCleanupJob instance
func NewOutboxCleanupJob(config *OutboxCleanupJobConfig, outboxRepo repository.OutboxRepository) *OutboxCleanupJob {
	if config == nil {
		config = DefaultOutboxCleanupJobConfig()
	}

	return &OutboxCleanupJob{
		config:     config,
		outboxRepo: outboxRepo,
	}
}

// Run executes the outbox cleanup job
// It deletes old delivered entries and logs failed entries for investigation
func (j *OutboxCleanupJob) Run() {
	ctx := context.Background()

	slog.Info("Starting outbox cleanup job")

	// Get and log metrics before cleanup
	j.logMetrics("before_cleanup")

	// Clean up delivered entries older than retention period
	j.cleanupDeliveredEntries(ctx)

	// Log failed entries for investigation
	j.logFailedEntries(ctx)

	// Get and log metrics after cleanup
	j.logMetrics("after_cleanup")

	slog.Info("Outbox cleanup job completed")
}

// cleanupDeliveredEntries removes old delivered outbox entries in batches
func (j *OutboxCleanupJob) cleanupDeliveredEntries(ctx context.Context) {
	olderThan := time.Now().Add(-time.Duration(j.config.RetentionDays) * 24 * time.Hour)
	totalDeleted := 0

	slog.Info("Cleaning up delivered outbox entries", "retention_days", j.config.RetentionDays, "older_than", olderThan.Format(time.RFC3339))

	// Continue deleting in batches until no more rows are affected
	for {
		deletedCount, err := j.outboxRepo.DeleteOldDeliveredEntries(ctx, olderThan, j.config.BatchSize)
		if err != nil {
			slog.Error("Error deleting old delivered entries", "error", err, "total_deleted_so_far", totalDeleted)
			return
		}

		totalDeleted += deletedCount

		// If we deleted less than the batch size, we're done
		if deletedCount == 0 {
			break
		}

		slog.Info("Deleted batch of delivered entries", "deleted_count", deletedCount, "total_deleted", totalDeleted)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		slog.Info("Successfully deleted old delivered outbox entries", "count", totalDeleted)
	} else {
		slog.Info("No old delivered entries to cleanup")
	}
}

// logFailedEntries retrieves and logs failed outbox entries for investigation
func (j *OutboxCleanupJob) logFailedEntries(ctx context.Context) {
	failedEntries, err := j.outboxRepo.GetFailedEntries(ctx, j.config.FailedEntryLogLimit)
	if err != nil {
		slog.Error("Error retrieving failed outbox entries", "error", err)
		return
	}

	if len(failedEntries) == 0 {
		slog.Info("No failed outbox entries found")
		return
	}

	slog.Info("Found failed outbox entries for investigation", "count", len(failedEntries))

	for i, entry := range failedEntries {
		lastError := "no error message"
		if entry.LastError != nil {
			lastError = *entry.LastError
		}

		slog.Info("Failed outbox entry",
			"index", i+1,
			"id", entry.ID,
			"feed_item_id", entry.FeedItemID,
			"retry_count", entry.RetryCount,
			"user_count", len(entry.UserIDs),
			"created_at", entry.CreatedAt.Format(time.RFC3339),
			"last_error", lastError)
	}

	slog.Info("Failed entries investigation complete - consider manual intervention for persistent failures")
}

// logMetrics retrieves and logs current outbox metrics
func (j *OutboxCleanupJob) logMetrics(stage string) {
	ctx := context.Background()

	metrics, err := j.outboxRepo.GetMetrics(ctx)
	if err != nil {
		slog.Error("Error retrieving outbox metrics", "stage", stage, "error", err)
		return
	}

	slog.Info("Outbox metrics", "stage", stage,
		"pending", metrics.PendingCount,
		"sending", metrics.SendingCount,
		"delivered", metrics.DeliveredCount,
		"failed", metrics.FailedCount,
		"total", metrics.TotalCount,
		"queue_depth", metrics.PendingCount+metrics.SendingCount)

	// Log warnings for concerning metrics
	if metrics.FailedCount > 0 {
		slog.Warn("Failed deliveries detected - investigation recommended", "failed_count", metrics.FailedCount)
	}

	queueDepth := metrics.PendingCount + metrics.SendingCount
	if queueDepth > 1000 {
		slog.Warn("High queue depth - outbox may be backing up", "queue_depth", queueDepth)
	}
}

// GetMetrics retrieves current outbox metrics for external monitoring
// This can be used by monitoring systems or health check endpoints
func (j *OutboxCleanupJob) GetMetrics() (*repository.OutboxMetrics, error) {
	ctx := context.Background()
	return j.outboxRepo.GetMetrics(ctx)
}

// RunWithCustomRetention runs the cleanup job with a custom retention period
// This is useful for one-off cleanup operations
func (j *OutboxCleanupJob) RunWithCustomRetention(retentionDays int) {
	originalRetention := j.config.RetentionDays
	j.config.RetentionDays = retentionDays

	slog.Info("Running outbox cleanup with custom retention", "retention_days", retentionDays)
	j.Run()

	// Restore original config
	j.config.RetentionDays = originalRetention
}

// String returns a string representation of the job for logging
func (j *OutboxCleanupJob) String() string {
	return fmt.Sprintf("OutboxCleanupJob(retention=%dd, batch=%d)",
		j.config.RetentionDays, j.config.BatchSize)
}
