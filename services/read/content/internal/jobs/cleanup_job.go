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

const (
	orphanedRetention = 90 * 24 * time.Hour
	feedRetention     = 30 * 24 * time.Hour
)

// CleanupJob expires old Feed items and cleans up orphaned content
type CleanupJob struct {
	contentRepo     repository.ContentRepository
	userContentRepo repository.UserContentRepository
	logger          *slog.Logger
	batchSize       int
}

// NewCleanupJob creates a new CleanupJob instance. batchSize caps the number
// of rows deleted per transaction; values <= 0 fall back to defaultCleanupBatchSize.
func NewCleanupJob(contentRepo repository.ContentRepository, userContentRepo repository.UserContentRepository, logger *slog.Logger, batchSize int) *CleanupJob {
	if batchSize <= 0 {
		batchSize = defaultCleanupBatchSize
	}
	return &CleanupJob{
		contentRepo:     contentRepo,
		userContentRepo: userContentRepo,
		logger:          logger,
		batchSize:       batchSize,
	}
}

// Run first deletes Feed items older than 30 days that aren't favorited, then
// deletes content orphaned for more than 90 days. Feed expiry runs first so the
// contents it orphans are picked up by later runs. Both delete in bounded
// batches so no single transaction locks the whole table.
func (j *CleanupJob) Run() {
	ctx := context.Background()

	j.logger.Info("starting cleanup job", slog.Int("batch_size", j.batchSize))

	j.drain("expired feed items", feedRetention, func() (int64, error) {
		return j.userContentRepo.DeleteExpiredFeed(ctx, feedRetention, j.batchSize)
	})
	j.drain("orphaned content", orphanedRetention, func() (int64, error) {
		return j.contentRepo.DeleteOrphaned(ctx, orphanedRetention, j.batchSize)
	})
}

// drain calls deleteBatch until it reports zero deletions or fails.
func (j *CleanupJob) drain(what string, olderThan time.Duration, deleteBatch func() (int64, error)) {
	totalDeleted := int64(0)

	for {
		deletedCount, err := deleteBatch()
		if err != nil {
			j.logger.Error("failed to delete "+what,
				slog.Any("error", err),
				slog.Int64("total_deleted_so_far", totalDeleted),
			)
			return
		}

		if deletedCount == 0 {
			break
		}

		totalDeleted += deletedCount

		j.logger.Debug("batch deletion completed",
			slog.String("target", what),
			slog.Int64("batch_deleted", deletedCount),
			slog.Int64("total_deleted", totalDeleted),
		)

		// Small delay between batches to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	if totalDeleted > 0 {
		j.logger.Info(what+" cleanup completed",
			slog.Int64("total_deleted", totalDeleted),
			slog.Int("older_than_days", int(olderThan.Hours()/24)),
		)
	} else {
		j.logger.Debug("nothing to cleanup", slog.String("target", what))
	}
}
