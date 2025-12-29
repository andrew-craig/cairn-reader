package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
)

// CleanupJob handles the cleanup of orphaned content
type CleanupJob struct {
	contentRepo repository.ContentRepository
	logger      *slog.Logger
}

// NewCleanupJob creates a new CleanupJob instance
func NewCleanupJob(contentRepo repository.ContentRepository, logger *slog.Logger) *CleanupJob {
	return &CleanupJob{
		contentRepo: contentRepo,
		logger:      logger,
	}
}

// Run executes the cleanup job
// It deletes orphaned content that has been orphaned for more than 90 days
func (j *CleanupJob) Run() {
	ctx := context.Background()

	j.logger.Info("starting orphaned content cleanup job")

	// Delete content orphaned for more than 90 days
	olderThan := 90 * 24 * time.Hour

	deletedCount, err := j.contentRepo.DeleteOrphaned(ctx, olderThan)
	if err != nil {
		j.logger.Error("failed to delete orphaned content",
			slog.Any("error", err),
		)
		return
	}

	if deletedCount > 0 {
		j.logger.Info("orphaned content cleanup completed",
			slog.Int64("deleted_count", deletedCount),
			slog.Int("older_than_days", 90),
		)
	} else {
		j.logger.Debug("no orphaned content to cleanup")
	}
}

// RunWithBatching executes the cleanup job with batching to avoid long locks
// This is useful for very large datasets
func (j *CleanupJob) RunWithBatching(batchSize int) {
	ctx := context.Background()

	j.logger.Info("starting orphaned content cleanup job with batching",
		slog.Int("batch_size", batchSize),
	)

	totalDeleted := int64(0)

	// Continue deleting in batches until no more rows are affected
	for {
		// Delete content orphaned for more than 90 days
		olderThan := 90 * 24 * time.Hour

		deletedCount, err := j.contentRepo.DeleteOrphaned(ctx, olderThan)
		if err != nil {
			j.logger.Error("failed to delete orphaned content",
				slog.Any("error", err),
				slog.Int64("total_deleted_so_far", totalDeleted),
			)
			return
		}

		totalDeleted += deletedCount

		// If we deleted less than the batch size, we're done
		if deletedCount == 0 {
			break
		}

		j.logger.Debug("batch deletion completed",
			slog.Int64("batch_deleted", deletedCount),
			slog.Int64("total_deleted", totalDeleted),
		)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		j.logger.Info("orphaned content cleanup completed",
			slog.Int64("total_deleted", totalDeleted),
			slog.Int("older_than_days", 90),
		)
	} else {
		j.logger.Debug("no orphaned content to cleanup")
	}
}
