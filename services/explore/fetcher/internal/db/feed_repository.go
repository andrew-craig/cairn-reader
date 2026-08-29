package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	apperrors "github.com/andrew-craig/cairn-reader/pkg/errors"
	"github.com/andrew-craig/cairn-reader/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// feedRepository handles database operations for feeds
type feedRepository struct {
	db *pgxpool.Pool
}

// NewFeedRepository creates a new FeedRepositoryInterface
func NewFeedRepository(db *pgxpool.Pool) FeedRepositoryInterface {
	return &feedRepository{db: db}
}

// GetNextFeed atomically claims and returns the next feed to fetch.
// Prioritizes: 1) Never fetched (last_fetched_at IS NULL)
//
//  2. Oldest fetched
//
// Only returns enabled feeds not currently claimed by another fetcher. The
// claim is a single statement: SELECT ... FOR UPDATE SKIP LOCKED holds the
// row lock only for the instant of the following UPDATE (not for the
// duration of the fetch itself), so two concurrent fetchers never claim the
// same feed. feeds has no in-flight status, so a crashed fetch needs no
// recovery path here -- last_fetched_at stays untouched and the feed is
// picked first again once fetch_lease_expires_at itself expires.
func (r *feedRepository) GetNextFeed(ctx context.Context) (*models.Feed, error) {
	query := `
		WITH claimed AS (
			SELECT id FROM feeds
			WHERE enabled = true
			  AND (fetch_lease_expires_at IS NULL OR fetch_lease_expires_at < now())
			ORDER BY last_fetched_at NULLS FIRST, last_fetched_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE feeds
		SET fetch_lease_expires_at = now() + interval '5 minutes'
		FROM claimed
		WHERE feeds.id = claimed.id
		RETURNING feeds.id, feeds.url, feeds.title, feeds.description, feeds.last_fetched_at,
			feeds.consecutive_failures, feeds.enabled, feeds.etag, feeds.last_modified,
			feeds.created_at, feeds.updated_at
	`

	var feed models.Feed
	var title, description, etag, lastModified sql.NullString
	err := r.db.QueryRow(ctx, query).Scan(
		&feed.ID,
		&feed.URL,
		&title,
		&description,
		&feed.LastFetchedAt,
		&feed.ConsecutiveFailures,
		&feed.Enabled,
		&etag,
		&lastModified,
		&feed.CreatedAt,
		&feed.UpdatedAt,
	)

	if title.Valid {
		feed.Title = title.String
	}
	if description.Valid {
		feed.Description = description.String
	}
	if etag.Valid {
		feed.ETag = etag.String
	}
	if lastModified.Valid {
		feed.LastModified = lastModified.String
	}

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query next feed: %w", err)
	}

	return &feed, nil
}

// UpdateFetchResult records fetch success or failure.
// On success, etag and lastModified are persisted (empty string clears the value).
// On failure, etag and lastModified are ignored.
func (r *feedRepository) UpdateFetchResult(ctx context.Context, feedID int, success bool, etag, lastModified string) error {
	if success {
		query := `
			UPDATE feeds
			SET last_fetched_at = NOW(),
				consecutive_failures = 0,
				etag = NULLIF($2, ''),
				last_modified = NULLIF($3, ''),
				fetch_lease_expires_at = NULL,
				updated_at = NOW()
			WHERE id = $1
		`
		_, err := r.db.Exec(ctx, query, feedID, etag, lastModified)
		if err != nil {
			return fmt.Errorf("update feed success: %w", err)
		}
		slog.Info("feed fetch successful",
			slog.Int("feed_id", feedID),
			slog.Int("consecutive_failures", 0),
		)
	} else {
		// Increment consecutive_failures, check if we should disable
		query := `
			UPDATE feeds
			SET consecutive_failures = consecutive_failures + 1,
				enabled = CASE WHEN consecutive_failures + 1 >= 10 THEN false ELSE enabled END,
				fetch_lease_expires_at = NULL,
				updated_at = NOW()
			WHERE id = $1
			RETURNING consecutive_failures, enabled
		`
		var failures int
		var enabled bool
		err := r.db.QueryRow(ctx, query, feedID).Scan(&failures, &enabled)
		if err != nil {
			return fmt.Errorf("update feed failure: %w", err)
		}

		if !enabled {
			slog.Warn("feed disabled after consecutive failures",
				slog.Int("feed_id", feedID),
				slog.Int("consecutive_failures", failures),
			)
		} else {
			slog.Info("feed failure recorded",
				slog.Int("feed_id", feedID),
				slog.Int("consecutive_failures", failures),
			)
		}
	}

	return nil
}

// ImportFeeds upserts feeds from Kagi list
// Only adds new feeds, preserves existing feed metadata
func (r *feedRepository) ImportFeeds(ctx context.Context, feedURLs []string) error {
	if len(feedURLs) == 0 {
		return nil
	}

	query := `
		INSERT INTO feeds (url, enabled)
		VALUES ($1, true)
		ON CONFLICT (url) DO NOTHING
	`

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("error rolling back transaction", "error", err)
		}
	}()

	newCount := 0
	for _, url := range feedURLs {
		result, err := tx.Exec(ctx, query, url)
		if err != nil {
			return fmt.Errorf("insert feed %s: %w", url, err)
		}

		if result.RowsAffected() > 0 {
			newCount++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	slog.Info("imported feeds",
		slog.Int("new_feeds", newCount),
		slog.Int("total_in_list", len(feedURLs)),
	)
	return nil
}

// ListFeeds returns all feeds with optional filtering
func (r *feedRepository) ListFeeds(ctx context.Context, enabledOnly bool) ([]models.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched_at, consecutive_failures, enabled, created_at, updated_at
		FROM feeds
	`

	if enabledOnly {
		query += " WHERE enabled = true"
	}

	query += " ORDER BY id ASC"

	rows, err := r.db.Query(ctx, query)
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
func (r *feedRepository) RecordFetchHistory(ctx context.Context, feedID int, success bool, articlesFound, articlesSent int, errorMsg string) error {
	query := `
		INSERT INTO fetch_history (feed_id, fetch_started_at, fetch_completed_at, success, articles_found, articles_sent, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''))
	`

	now := time.Now()
	_, err := r.db.Exec(ctx, query, feedID, now, now, success, articlesFound, articlesSent, errorMsg)
	if err != nil {
		return fmt.Errorf("insert fetch history: %w", err)
	}

	return nil
}

// GetFeedByID returns a feed by its ID
func (r *feedRepository) GetFeedByID(ctx context.Context, feedID int) (*models.Feed, error) {
	query := `
		SELECT id, url, title, description, last_fetched_at, consecutive_failures, enabled, created_at, updated_at
		FROM feeds
		WHERE id = $1
	`

	var feed models.Feed
	err := r.db.QueryRow(ctx, query, feedID).Scan(
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

	if err == pgx.ErrNoRows {
		return nil, apperrors.ErrFeedNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query feed: %w", err)
	}

	return &feed, nil
}

// GetFeedStats returns statistics about feeds
func (r *feedRepository) GetFeedStats(ctx context.Context) (total, enabled, disabled, neverFetched int, err error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE enabled = true) as enabled,
			COUNT(*) FILTER (WHERE enabled = false) as disabled,
			COUNT(*) FILTER (WHERE last_fetched_at IS NULL) as never_fetched
		FROM feeds
	`

	err = r.db.QueryRow(ctx, query).Scan(&total, &enabled, &disabled, &neverFetched)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("query feed stats: %w", err)
	}

	return total, enabled, disabled, neverFetched, nil
}
