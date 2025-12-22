package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/andrew-craig/cairn-explore/pkg/models"
)

// FeedRepository handles database operations for feeds
type FeedRepository struct {
	db *sql.DB
}

// NewFeedRepository creates a new FeedRepository
func NewFeedRepository(db *sql.DB) *FeedRepository {
	return &FeedRepository{db: db}
}

// GetNextFeed returns the next feed to fetch
// Prioritizes: 1) Never fetched (last_fetched_at IS NULL)
//              2) Oldest fetched
// Only returns enabled feeds
func (r *FeedRepository) GetNextFeed(ctx context.Context) (*models.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched_at, consecutive_failures, enabled, created_at, updated_at
		FROM feeds
		WHERE enabled = true
		ORDER BY last_fetched_at NULLS FIRST, last_fetched_at ASC
		LIMIT 1
	`

	var feed models.Feed
	var title, description sql.NullString
	err := r.db.QueryRowContext(ctx, query).Scan(
		&feed.ID,
		&feed.URL,
		&title,
		&description,
		&feed.LastFetchedAt,
		&feed.ConsecutiveFailures,
		&feed.Enabled,
		&feed.CreatedAt,
		&feed.UpdatedAt,
	)

	if title.Valid {
		feed.Title = title.String
	}
	if description.Valid {
		feed.Description = description.String
	}

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query next feed: %w", err)
	}

	return &feed, nil
}

// UpdateFetchResult records fetch success or failure
func (r *FeedRepository) UpdateFetchResult(ctx context.Context, feedID int, success bool) error {
	if success {
		// Reset consecutive_failures to 0, update last_fetched_at
		query := `
			UPDATE feeds
			SET last_fetched_at = NOW(),
				consecutive_failures = 0,
				updated_at = NOW()
			WHERE id = $1
		`
		_, err := r.db.ExecContext(ctx, query, feedID)
		if err != nil {
			return fmt.Errorf("update feed success: %w", err)
		}
		log.Printf("Feed %d: fetch successful, reset failures to 0", feedID)
	} else {
		// Increment consecutive_failures, check if we should disable
		query := `
			UPDATE feeds
			SET consecutive_failures = consecutive_failures + 1,
				enabled = CASE WHEN consecutive_failures + 1 >= 10 THEN false ELSE enabled END,
				updated_at = NOW()
			WHERE id = $1
			RETURNING consecutive_failures, enabled
		`
		var failures int
		var enabled bool
		err := r.db.QueryRowContext(ctx, query, feedID).Scan(&failures, &enabled)
		if err != nil {
			return fmt.Errorf("update feed failure: %w", err)
		}

		if !enabled {
			log.Printf("Feed %d: disabled after %d consecutive failures", feedID, failures)
		} else {
			log.Printf("Feed %d: failure recorded (%d consecutive)", feedID, failures)
		}
	}

	return nil
}

// ImportFeeds upserts feeds from Kagi list
// Only adds new feeds, preserves existing feed metadata
func (r *FeedRepository) ImportFeeds(ctx context.Context, feedURLs []string) error {
	if len(feedURLs) == 0 {
		return nil
	}

	query := `
		INSERT INTO feeds (url, enabled)
		VALUES ($1, true)
		ON CONFLICT (url) DO NOTHING
	`

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	newCount := 0
	for _, url := range feedURLs {
		result, err := stmt.ExecContext(ctx, url)
		if err != nil {
			return fmt.Errorf("insert feed %s: %w", url, err)
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			newCount++
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	log.Printf("Imported %d new feeds (total in list: %d)", newCount, len(feedURLs))
	return nil
}

// ListFeeds returns all feeds with optional filtering
func (r *FeedRepository) ListFeeds(ctx context.Context, enabledOnly bool) ([]models.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched_at, consecutive_failures, enabled, created_at, updated_at
		FROM feeds
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY id ASC"

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query feeds: %w", err)
	}
	defer rows.Close()

	var feeds []models.Feed
	for rows.Next() {
		var feed models.Feed
		err := rows.Scan(
			&feed.ID,
			&feed.URL,
			&feed.Title,
			&feed.Description,
			&feed.LastFetchedAt,
			&feed.ConsecutiveFailures,
			&feed.Enabled,
			&feed.CreatedAt,
			&feed.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate feeds: %w", err)
	}

	return feeds, nil
}

// RecordFetchHistory logs fetch attempt details
func (r *FeedRepository) RecordFetchHistory(ctx context.Context, feedID int, success bool, articlesFound, articlesSent int, errorMsg string) error {
	query := `
		INSERT INTO fetch_history (feed_id, fetch_started_at, fetch_completed_at, success, articles_found, articles_sent, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, feedID, now, now, success, articlesFound, articlesSent, errorMsg)
	if err != nil {
		return fmt.Errorf("insert fetch history: %w", err)
	}

	return nil
}

// GetFeedByID returns a feed by its ID
func (r *FeedRepository) GetFeedByID(ctx context.Context, feedID int) (*models.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched_at, consecutive_failures, enabled, created_at, updated_at
		FROM feeds
		WHERE id = $1
	`

	var feed models.Feed
	err := r.db.QueryRowContext(ctx, query, feedID).Scan(
		&feed.ID,
		&feed.URL,
		&feed.Title,
		&feed.Description,
		&feed.LastFetchedAt,
		&feed.ConsecutiveFailures,
		&feed.Enabled,
		&feed.CreatedAt,
		&feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("feed %d not found", feedID)
	}
	if err != nil {
		return nil, fmt.Errorf("query feed: %w", err)
	}

	return &feed, nil
}

// GetFeedStats returns statistics about feeds
func (r *FeedRepository) GetFeedStats(ctx context.Context) (total, enabled, disabled, neverFetched int, err error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE enabled = true) as enabled,
			COUNT(*) FILTER (WHERE enabled = false) as disabled,
			COUNT(*) FILTER (WHERE last_fetched_at IS NULL) as never_fetched
		FROM feeds
	`

	err = r.db.QueryRowContext(ctx, query).Scan(&total, &enabled, &disabled, &neverFetched)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("query feed stats: %w", err)
	}

	return total, enabled, disabled, neverFetched, nil
}
