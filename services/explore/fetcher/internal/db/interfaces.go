package db

import (
	"context"

	"github.com/andrew-craig/cairn/pkg/models"
)

// FeedRepositoryInterface defines the contract for feed database operations
type FeedRepositoryInterface interface {
	// GetNextFeed returns the next feed to fetch
	// Prioritizes: 1) Never fetched (last_fetched_at IS NULL)
	//              2) Oldest fetched
	// Only returns enabled feeds
	GetNextFeed(ctx context.Context) (*models.Feed, error)

	// UpdateFetchResult records fetch success or failure
	UpdateFetchResult(ctx context.Context, feedID int, success bool) error

	// ImportFeeds upserts feeds from Kagi list
	// Only adds new feeds, preserves existing feed metadata
	ImportFeeds(ctx context.Context, feedURLs []string) error

	// ListFeeds returns all feeds with optional filtering
	ListFeeds(ctx context.Context, enabledOnly bool) ([]models.Feed, error)

	// RecordFetchHistory logs fetch attempt details
	RecordFetchHistory(ctx context.Context, feedID int, success bool, articlesFound, articlesSent int, errorMsg string) error

	// GetFeedByID returns a feed by its ID
	GetFeedByID(ctx context.Context, feedID int) (*models.Feed, error)

	// GetFeedStats returns statistics about feeds
	GetFeedStats(ctx context.Context) (total, enabled, disabled, neverFetched int, err error)
}
