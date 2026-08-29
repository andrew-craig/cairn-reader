package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
)

// defaultCleanupBatchSize bounds each DELETE transaction so cleanup never
// holds a long, WAL-heavy lock on the contents table.
const defaultCleanupBatchSize = 1000

// CleanupJob handles the cleanup of orphaned content
type CleanupJob struct {
	contentRepo repository.ContentRepository
	logger      *slog.Logger
	batchSize   int
}

// NewCleanupJob creates a new CleanupJob instance. batchSize caps the number
// of rows deleted per transaction; values <= 0 fall back to defaultCleanupBatchSize.
func NewCleanupJob(contentRepo repository.ContentRepository, logger *slog.Logger, batchSize int) *CleanupJob {
	if batchSize <= 0 {
		batchSize = defaultCleanupBatchSize
	}
	return &CleanupJob{
		contentRepo: contentRepo,
		logger:      logger,
		batchSize:   batchSize,
	}
}

// Run executes the cleanup job, deleting content orphaned for more than 90
// days in bounded batches so no single transaction locks the whole table.
func (j *CleanupJob) Run() {
	ctx := context.Background()

	j.logger.Info("starting orphaned content cleanup job",
		slog.Int("batch_size", j.batchSize),
	)

	// Delete content orphaned for more than 90 days
	olderThan := 90 * 24 * time.Hour

	totalDeleted := int64(0)

	// Continue deleting in batches until no more rows are affected
	for {
		deletedCount, err := j.contentRepo.DeleteOrphaned(ctx, olderThan, j.batchSize)
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
