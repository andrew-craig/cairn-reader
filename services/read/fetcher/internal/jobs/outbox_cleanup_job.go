package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
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

	log.Println("Starting outbox cleanup job")

	// Get and log metrics before cleanup
	j.logMetrics("before_cleanup")

	// Clean up delivered entries older than retention period
	j.cleanupDeliveredEntries(ctx)

	// Log failed entries for investigation
	j.logFailedEntries(ctx)

	// Get and log metrics after cleanup
	j.logMetrics("after_cleanup")

	log.Println("Outbox cleanup job completed")
}

// cleanupDeliveredEntries removes old delivered outbox entries in batches
func (j *OutboxCleanupJob) cleanupDeliveredEntries(ctx context.Context) {
	olderThan := time.Now().Add(-time.Duration(j.config.RetentionDays) * 24 * time.Hour)
	totalDeleted := 0

	log.Printf("Cleaning up delivered outbox entries older than %d days (before %s)",
		j.config.RetentionDays, olderThan.Format(time.RFC3339))

	// Continue deleting in batches until no more rows are affected
	for {
		deletedCount, err := j.outboxRepo.DeleteOldDeliveredEntries(ctx, olderThan, j.config.BatchSize)
		if err != nil {
			log.Printf("Error deleting old delivered entries: %v (total_deleted_so_far=%d)",
				err, totalDeleted)
			return
		}

		totalDeleted += deletedCount

		// If we deleted less than the batch size, we're done
		if deletedCount == 0 {
			break
		}

		log.Printf("Deleted batch of %d delivered entries (total_deleted=%d)",
			deletedCount, totalDeleted)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		log.Printf("Successfully deleted %d old delivered outbox entries", totalDeleted)
	} else {
		log.Println("No old delivered entries to cleanup")
	}
}

// logFailedEntries retrieves and logs failed outbox entries for investigation
func (j *OutboxCleanupJob) logFailedEntries(ctx context.Context) {
	failedEntries, err := j.outboxRepo.GetFailedEntries(ctx, j.config.FailedEntryLogLimit)
	if err != nil {
		log.Printf("Error retrieving failed outbox entries: %v", err)
		return
	}

	if len(failedEntries) == 0 {
		log.Println("No failed outbox entries found")
		return
	}

	log.Printf("Found %d failed outbox entries for investigation:", len(failedEntries))

	for i, entry := range failedEntries {
		lastError := "no error message"
		if entry.LastError != nil {
			lastError = *entry.LastError
		}

		log.Printf("  [%d] ID: %s, FeedItemID: %s, RetryCount: %d, UserCount: %d, Created: %s, LastError: %s",
			i+1,
			entry.ID,
			entry.FeedItemID,
			entry.RetryCount,
			len(entry.UserIDs),
			entry.CreatedAt.Format(time.RFC3339),
			lastError,
		)
	}

	log.Printf("Failed entries investigation complete. Consider manual intervention for persistent failures.")
}

// logMetrics retrieves and logs current outbox metrics
func (j *OutboxCleanupJob) logMetrics(stage string) {
	ctx := context.Background()

	metrics, err := j.outboxRepo.GetMetrics(ctx)
	if err != nil {
		log.Printf("Error retrieving outbox metrics (%s): %v", stage, err)
		return
	}

	log.Printf("Outbox metrics (%s):", stage)
	log.Printf("  - Pending: %d", metrics.PendingCount)
	log.Printf("  - Sending: %d", metrics.SendingCount)
	log.Printf("  - Delivered: %d", metrics.DeliveredCount)
	log.Printf("  - Failed: %d", metrics.FailedCount)
	log.Printf("  - Total: %d", metrics.TotalCount)
	log.Printf("  - Queue depth (pending + sending): %d", metrics.PendingCount+metrics.SendingCount)

	// Log warnings for concerning metrics
	if metrics.FailedCount > 0 {
		log.Printf("  ⚠ WARNING: %d failed deliveries detected - investigation recommended", metrics.FailedCount)
	}

	queueDepth := metrics.PendingCount + metrics.SendingCount
	if queueDepth > 1000 {
		log.Printf("  ⚠ WARNING: High queue depth (%d) - outbox may be backing up", queueDepth)
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

	log.Printf("Running outbox cleanup with custom retention: %d days", retentionDays)
	j.Run()

	// Restore original config
	j.config.RetentionDays = originalRetention
}

// String returns a string representation of the job for logging
func (j *OutboxCleanupJob) String() string {
	return fmt.Sprintf("OutboxCleanupJob(retention=%dd, batch=%d)",
		j.config.RetentionDays, j.config.BatchSize)
}
