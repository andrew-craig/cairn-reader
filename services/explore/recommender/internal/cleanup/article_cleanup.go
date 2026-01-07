// Package cleanup provides background jobs for article retention management.
// It handles soft-deletion (marking articles as deleted) and hard-deletion
// (permanent removal) of old articles based on configurable retention periods.
package cleanup

import (
	"context"
	"log/slog"
	"time"

	"github.com/andrew-craig/cairn/services/explore/recommender/internal/db"
)

// ArticleCleanup handles periodic cleanup of old articles
type ArticleCleanup struct {
	articleRepo     db.ArticleRepositoryInterface
	retentionDays   int
	cleanupInterval time.Duration
	stopChan        chan struct{}
	logger          *slog.Logger
}

// NewArticleCleanup creates a new article cleanup job
func NewArticleCleanup(articleRepo db.ArticleRepositoryInterface, retentionDays int, cleanupInterval time.Duration, logger *slog.Logger) *ArticleCleanup {
	if logger == nil {
		logger = slog.Default()
	}
	return &ArticleCleanup{
		articleRepo:     articleRepo,
		retentionDays:   retentionDays,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
		logger:          logger,
	}
}

// Start begins the periodic cleanup job
// Runs cleanup immediately on start, then periodically based on cleanupInterval
func (c *ArticleCleanup) Start() {
	c.logger.Info("starting article cleanup job",
		slog.Int("retention_days", c.retentionDays),
		slog.Duration("interval", c.cleanupInterval),
	)

	// Run cleanup immediately on start
	c.runCleanup()

	// Start periodic cleanup
	ticker := time.NewTicker(c.cleanupInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.runCleanup()
			case <-c.stopChan:
				ticker.Stop()
				c.logger.Info("article cleanup job stopped")
				return
			}
		}
	}()
}

// Stop stops the cleanup job
func (c *ArticleCleanup) Stop() {
	close(c.stopChan)
}

// runCleanup performs the actual cleanup operation
func (c *ArticleCleanup) runCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	c.logger.Info("running article cleanup",
		slog.Int("retention_days", c.retentionDays),
	)

	// Mark old articles as deleted (soft delete)
	count, err := c.articleRepo.MarkOldArticlesAsDeleted(ctx, c.retentionDays)
	if err != nil {
		c.logger.Error("failed to mark old articles as deleted",
			slog.Any("error", err),
		)
		return
	}

	if count > 0 {
		c.logger.Info("marked articles as deleted", slog.Int("count", count))
	} else {
		c.logger.Info("no articles to mark as deleted")
	}

	// Optional: Hard delete articles that have been soft-deleted for a while
	// This maintains referential integrity by only deleting articles already marked as deleted
	hardDeleteDays := c.retentionDays + 30 // Delete articles that have been soft-deleted for 30+ days
	hardDeleteCount, err := c.articleRepo.HardDeleteOldArticles(ctx, hardDeleteDays)
	if err != nil {
		c.logger.Error("failed to hard delete old articles",
			slog.Any("error", err),
		)
		return
	}

	if hardDeleteCount > 0 {
		c.logger.Info("hard deleted articles",
			slog.Int("count", hardDeleteCount),
			slog.Int("days", hardDeleteDays),
		)
	}

	c.logger.Info("article cleanup completed")
}

// RunOnce runs the cleanup job once and returns
// Useful for manual cleanup or testing
func (c *ArticleCleanup) RunOnce() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Mark old articles as deleted
	count, err := c.articleRepo.MarkOldArticlesAsDeleted(ctx, c.retentionDays)
	if err != nil {
		return err
	}
	c.logger.Info("marked articles as deleted", slog.Int("count", count))

	// Hard delete old soft-deleted articles
	hardDeleteDays := c.retentionDays + 30
	hardDeleteCount, err := c.articleRepo.HardDeleteOldArticles(ctx, hardDeleteDays)
	if err != nil {
		return err
	}
	c.logger.Info("hard deleted articles", slog.Int("count", hardDeleteCount))

	return nil
}
