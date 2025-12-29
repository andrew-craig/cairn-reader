package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-read/services/content-service/internal/repository"
	"go.uber.org/zap"
)

// CleanupJob handles the cleanup of orphaned content
type CleanupJob struct {
	contentRepo repository.ContentRepository
	logger      *zap.Logger
}

// NewCleanupJob creates a new CleanupJob instance
func NewCleanupJob(contentRepo repository.ContentRepository, logger *zap.Logger) *CleanupJob {
	return &CleanupJob{
		contentRepo: contentRepo,
		logger:      logger,
	}
}

// Run executes the cleanup job
// It deletes orphaned content that has been orphaned for more than 90 days
func (j *CleanupJob) Run() {
	ctx := context.Background()

	j.logger.Info("Starting orphaned content cleanup job")

	// Delete content orphaned for more than 90 days
	olderThan := 90 * 24 * time.Hour

	deletedCount, err := j.contentRepo.DeleteOrphaned(ctx, olderThan)
	if err != nil {
		j.logger.Error("Failed to delete orphaned content",
			zap.Error(err),
		)
		return
	}

	if deletedCount > 0 {
		j.logger.Info("Orphaned content cleanup completed",
			zap.Int64("deleted_count", deletedCount),
			zap.String("older_than", fmt.Sprintf("%d days", 90)),
		)
	} else {
		j.logger.Debug("No orphaned content to cleanup")
	}
}

// RunWithBatching executes the cleanup job with batching to avoid long locks
// This is useful for very large datasets
func (j *CleanupJob) RunWithBatching(batchSize int) {
	ctx := context.Background()

	j.logger.Info("Starting orphaned content cleanup job with batching",
		zap.Int("batch_size", batchSize),
	)

	totalDeleted := int64(0)

	// Continue deleting in batches until no more rows are affected
	for {
		// Delete content orphaned for more than 90 days
		olderThan := 90 * 24 * time.Hour

		deletedCount, err := j.contentRepo.DeleteOrphaned(ctx, olderThan)
		if err != nil {
			j.logger.Error("Failed to delete orphaned content",
				zap.Error(err),
				zap.Int64("total_deleted_so_far", totalDeleted),
			)
			return
		}

		totalDeleted += deletedCount

		// If we deleted less than the batch size, we're done
		if deletedCount == 0 {
			break
		}

		j.logger.Debug("Batch deletion completed",
			zap.Int64("batch_deleted", deletedCount),
			zap.Int64("total_deleted", totalDeleted),
		)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		j.logger.Info("Orphaned content cleanup completed",
			zap.Int64("total_deleted", totalDeleted),
			zap.String("older_than", fmt.Sprintf("%d days", 90)),
		)
	} else {
		j.logger.Debug("No orphaned content to cleanup")
	}
}
