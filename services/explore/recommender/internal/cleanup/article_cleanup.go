// Package cleanup provides background jobs for article retention management.
// It handles soft-deletion (marking articles as deleted) and hard-deletion
// (permanent removal) of old articles based on configurable retention periods.
package cleanup

import (
	"context"
	"log"
	"time"

	"github.com/andrew-craig/cairn-explore/recommender/internal/db"
)

// ArticleCleanup handles periodic cleanup of old articles
type ArticleCleanup struct {
	articleRepo     *db.ArticleRepository
	retentionDays   int
	cleanupInterval time.Duration
	stopChan        chan struct{}
}

// NewArticleCleanup creates a new article cleanup job
func NewArticleCleanup(articleRepo *db.ArticleRepository, retentionDays int, cleanupInterval time.Duration) *ArticleCleanup {
	return &ArticleCleanup{
		articleRepo:     articleRepo,
		retentionDays:   retentionDays,
		cleanupInterval: cleanupInterval,
		stopChan:        make(chan struct{}),
	}
}

// Start begins the periodic cleanup job
// Runs cleanup immediately on start, then periodically based on cleanupInterval
func (c *ArticleCleanup) Start() {
	log.Printf("Starting article cleanup job (retention: %d days, interval: %s)", c.retentionDays, c.cleanupInterval)

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
				log.Println("Article cleanup job stopped")
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

	log.Printf("Running article cleanup (marking articles older than %d days as deleted)...", c.retentionDays)

	// Mark old articles as deleted (soft delete)
	count, err := c.articleRepo.MarkOldArticlesAsDeleted(ctx, c.retentionDays)
	if err != nil {
		log.Printf("Error marking old articles as deleted: %v", err)
		return
	}

	if count > 0 {
		log.Printf("Marked %d articles as deleted", count)
	} else {
		log.Println("No articles to mark as deleted")
	}

	// Optional: Hard delete articles that have been soft-deleted for a while
	// This maintains referential integrity by only deleting articles already marked as deleted
	hardDeleteDays := c.retentionDays + 30 // Delete articles that have been soft-deleted for 30+ days
	hardDeleteCount, err := c.articleRepo.HardDeleteOldArticles(ctx, hardDeleteDays)
	if err != nil {
		log.Printf("Error hard deleting old articles: %v", err)
		return
	}

	if hardDeleteCount > 0 {
		log.Printf("Hard deleted %d articles (older than %d days and already marked as deleted)", hardDeleteCount, hardDeleteDays)
	}

	log.Println("Article cleanup completed")
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
	log.Printf("Marked %d articles as deleted", count)

	// Hard delete old soft-deleted articles
	hardDeleteDays := c.retentionDays + 30
	hardDeleteCount, err := c.articleRepo.HardDeleteOldArticles(ctx, hardDeleteDays)
	if err != nil {
		return err
	}
	log.Printf("Hard deleted %d articles", hardDeleteCount)

	return nil
}
