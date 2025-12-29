package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
)

// FeedItemsCleanupJobConfig holds configuration for the feed items cleanup job
type FeedItemsCleanupJobConfig struct {
	// CompletedRetentionDays is the number of days to retain completed feed items
	CompletedRetentionDays int
	// FailedRetentionDays is the number of days to retain failed feed items
	FailedRetentionDays int
	// BatchSize is the number of items to delete per batch
	BatchSize int
}

// DefaultFeedItemsCleanupJobConfig returns default configuration
func DefaultFeedItemsCleanupJobConfig() *FeedItemsCleanupJobConfig {
	return &FeedItemsCleanupJobConfig{
		CompletedRetentionDays: 7,
		FailedRetentionDays:    30,
		BatchSize:              1000,
	}
}

// FeedItemsCleanupJob handles cleanup of old feed items
type FeedItemsCleanupJob struct {
	config       *FeedItemsCleanupJobConfig
	feedItemRepo repository.FeedItemRepository
}

// NewFeedItemsCleanupJob creates a new FeedItemsCleanupJob instance
func NewFeedItemsCleanupJob(config *FeedItemsCleanupJobConfig, feedItemRepo repository.FeedItemRepository) *FeedItemsCleanupJob {
	if config == nil {
		config = DefaultFeedItemsCleanupJobConfig()
	}

	return &FeedItemsCleanupJob{
		config:       config,
		feedItemRepo: feedItemRepo,
	}
}

// Run executes the feed items cleanup job
// It deletes old completed and failed feed items
func (j *FeedItemsCleanupJob) Run() {
	ctx := context.Background()

	log.Println("Starting feed items cleanup job")

	// Get and log metrics before cleanup
	j.logMetrics("before_cleanup")

	// Clean up old completed items
	j.cleanupCompletedItems(ctx)

	// Clean up old failed items
	j.cleanupFailedItems(ctx)

	// Get and log metrics after cleanup
	j.logMetrics("after_cleanup")

	log.Println("Feed items cleanup job completed")
}

// cleanupCompletedItems removes old completed feed items in batches
func (j *FeedItemsCleanupJob) cleanupCompletedItems(ctx context.Context) {
	olderThan := time.Now().Add(-time.Duration(j.config.CompletedRetentionDays) * 24 * time.Hour)
	totalDeleted := 0

	log.Printf("Cleaning up completed feed items older than %d days (before %s)",
		j.config.CompletedRetentionDays, olderThan.Format(time.RFC3339))

	// Continue deleting in batches until no more rows are affected
	for {
		deletedCount, err := j.feedItemRepo.DeleteOldCompletedItems(ctx, olderThan, j.config.BatchSize)
		if err != nil {
			log.Printf("Error deleting old completed items: %v (total_deleted_so_far=%d)",
				err, totalDeleted)
			return
		}

		totalDeleted += deletedCount

		// If we deleted less than the batch size, we're done
		if deletedCount == 0 {
			break
		}

		log.Printf("Deleted batch of %d completed items (total_deleted=%d)",
			deletedCount, totalDeleted)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		log.Printf("Successfully deleted %d old completed feed items", totalDeleted)
	} else {
		log.Println("No old completed items to cleanup")
	}
}

// cleanupFailedItems removes old failed feed items in batches
func (j *FeedItemsCleanupJob) cleanupFailedItems(ctx context.Context) {
	olderThan := time.Now().Add(-time.Duration(j.config.FailedRetentionDays) * 24 * time.Hour)
	totalDeleted := 0

	log.Printf("Cleaning up failed feed items older than %d days (before %s)",
		j.config.FailedRetentionDays, olderThan.Format(time.RFC3339))

	// Continue deleting in batches until no more rows are affected
	for {
		deletedCount, err := j.feedItemRepo.DeleteOldFailedItems(ctx, olderThan, j.config.BatchSize)
		if err != nil {
			log.Printf("Error deleting old failed items: %v (total_deleted_so_far=%d)",
				err, totalDeleted)
			return
		}

		totalDeleted += deletedCount

		// If we deleted less than the batch size, we're done
		if deletedCount == 0 {
			break
		}

		log.Printf("Deleted batch of %d failed items (total_deleted=%d)",
			deletedCount, totalDeleted)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		log.Printf("Successfully deleted %d old failed feed items", totalDeleted)
	} else {
		log.Println("No old failed items to cleanup")
	}
}

// logMetrics retrieves and logs current feed items metrics
func (j *FeedItemsCleanupJob) logMetrics(stage string) {
	ctx := context.Background()

	metrics, err := j.feedItemRepo.GetMetrics(ctx)
	if err != nil {
		log.Printf("Error retrieving feed items metrics (%s): %v", stage, err)
		return
	}

	log.Printf("Feed items metrics (%s):", stage)
	log.Printf("  - Pending: %d", metrics.PendingCount)
	log.Printf("  - Processing: %d", metrics.ProcessingCount)
	log.Printf("  - Completed: %d", metrics.CompletedCount)
	log.Printf("  - Failed: %d", metrics.FailedCount)
	log.Printf("  - Total: %d", metrics.TotalCount)

	// Log warnings for concerning metrics
	if metrics.FailedCount > 100 {
		log.Printf("  ⚠ WARNING: High number of failed items (%d) - investigation recommended", metrics.FailedCount)
	}

	if metrics.PendingCount > 1000 {
		log.Printf("  ⚠ WARNING: High number of pending items (%d) - processing may be backing up", metrics.PendingCount)
	}

	// Calculate table size estimate (for monitoring)
	log.Printf("  - Table health: pending=%d, processing=%d (items in flight)",
		metrics.PendingCount, metrics.ProcessingCount)
}

// GetMetrics retrieves current feed items metrics for external monitoring
// This can be used by monitoring systems or health check endpoints
func (j *FeedItemsCleanupJob) GetMetrics() (*repository.FeedItemMetrics, error) {
	ctx := context.Background()
	return j.feedItemRepo.GetMetrics(ctx)
}

// RunWithCustomRetention runs the cleanup job with custom retention periods
// This is useful for one-off cleanup operations
func (j *FeedItemsCleanupJob) RunWithCustomRetention(completedDays, failedDays int) {
	originalCompletedRetention := j.config.CompletedRetentionDays
	originalFailedRetention := j.config.FailedRetentionDays

	j.config.CompletedRetentionDays = completedDays
	j.config.FailedRetentionDays = failedDays

	log.Printf("Running feed items cleanup with custom retention: completed=%d days, failed=%d days",
		completedDays, failedDays)
	j.Run()

	// Restore original config
	j.config.CompletedRetentionDays = originalCompletedRetention
	j.config.FailedRetentionDays = originalFailedRetention
}

// String returns a string representation of the job for logging
func (j *FeedItemsCleanupJob) String() string {
	return fmt.Sprintf("FeedItemsCleanupJob(completed_retention=%dd, failed_retention=%dd, batch=%d)",
		j.config.CompletedRetentionDays, j.config.FailedRetentionDays, j.config.BatchSize)
}
