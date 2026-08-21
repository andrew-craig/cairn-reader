package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/google/uuid"
)

// FeedRepository defines the interface for feed data operations
type FeedRepository interface {
	// Create creates a new feed
	Create(ctx context.Context, feed *models.Feed) error

	// GetByID retrieves a feed by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Feed, error)

	// GetByURL retrieves a feed by its URL
	GetByURL(ctx context.Context, url string) (*models.Feed, error)

	// Update updates an existing feed
	Update(ctx context.Context, feed *models.Feed) error

	// UpdatePollingInfo updates polling-related fields
	UpdatePollingInfo(ctx context.Context, id uuid.UUID, lastFetchedAt, lastPublishedAt *time.Time, nextPollAt time.Time, tier models.PollingTier) error

	// UpdateErrorInfo updates error tracking fields
	UpdateErrorInfo(ctx context.Context, id uuid.UUID, consecutiveErrorDays int, lastErrorAt *time.Time, lastErrorMessage *string) error

	// UpdateStatus updates the feed status
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.FeedStatus) error

	// GetFeedsDueForPolling retrieves feeds that are ready to be polled
	GetFeedsDueForPolling(ctx context.Context, limit int) ([]*models.Feed, error)

	// GetFeedsForTierUpdate retrieves all active feeds for tier management
	GetFeedsForTierUpdate(ctx context.Context) ([]*models.Feed, error)

	// Delete deletes a feed (and cascades to subscriptions and items)
	Delete(ctx context.Context, id uuid.UUID) error
}

// feedRepository is the concrete implementation of FeedRepository
type feedRepository struct {
	db *sql.DB
}

// NewFeedRepository creates a new FeedRepository
func NewFeedRepository(db *sql.DB) FeedRepository {
	return &feedRepository{db: db}
}

// Create creates a new feed
func (r *feedRepository) Create(ctx context.Context, feed *models.Feed) error {
	query := `
		INSERT INTO feeds (
			id, feed_url, title, description, site_url,
			polling_tier, last_fetched_at, last_published_at, next_poll_at,
			status, consecutive_error_days, last_error_at, last_error_message,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15
		)
	`

	now := time.Now()
	if feed.ID == uuid.Nil {
		feed.ID = uuid.New()
	}
	if feed.CreatedAt.IsZero() {
		feed.CreatedAt = now
	}
	if feed.UpdatedAt.IsZero() {
		feed.UpdatedAt = now
	}
	if feed.NextPollAt.IsZero() {
		feed.NextPollAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		feed.ID, feed.FeedURL, feed.Title, feed.Description, feed.SiteURL,
		feed.PollingTier, feed.LastFetchedAt, feed.LastPublishedAt, feed.NextPollAt,
		feed.Status, feed.ConsecutiveErrorDays, feed.LastErrorAt, feed.LastErrorMessage,
		feed.CreatedAt, feed.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	return nil
}

// GetByID retrieves a feed by its ID
func (r *feedRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Feed, error) {
	query := `
		SELECT
			id, feed_url, title, description, site_url,
			polling_tier, last_fetched_at, last_published_at, next_poll_at,
			status, consecutive_error_days, last_error_at, last_error_message,
			created_at, updated_at
		FROM feeds
		WHERE id = $1
	`

	feed := &models.Feed{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&feed.ID, &feed.FeedURL, &feed.Title, &feed.Description, &feed.SiteURL,
		&feed.PollingTier, &feed.LastFetchedAt, &feed.LastPublishedAt, &feed.NextPollAt,
		&feed.Status, &feed.ConsecutiveErrorDays, &feed.LastErrorAt, &feed.LastErrorMessage,
		&feed.CreatedAt, &feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by ID: %w", err)
	}

	return feed, nil
}

// GetByURL retrieves a feed by its URL
func (r *feedRepository) GetByURL(ctx context.Context, url string) (*models.Feed, error) {
	query := `
		SELECT
			id, feed_url, title, description, site_url,
			polling_tier, last_fetched_at, last_published_at, next_poll_at,
			status, consecutive_error_days, last_error_at, last_error_message,
			created_at, updated_at
		FROM feeds
		WHERE feed_url = $1
	`

	feed := &models.Feed{}
	err := r.db.QueryRowContext(ctx, query, url).Scan(
		&feed.ID, &feed.FeedURL, &feed.Title, &feed.Description, &feed.SiteURL,
		&feed.PollingTier, &feed.LastFetchedAt, &feed.LastPublishedAt, &feed.NextPollAt,
		&feed.Status, &feed.ConsecutiveErrorDays, &feed.LastErrorAt, &feed.LastErrorMessage,
		&feed.CreatedAt, &feed.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed by URL: %w", err)
	}

	return feed, nil
}

// Update updates an existing feed
func (r *feedRepository) Update(ctx context.Context, feed *models.Feed) error {
	query := `
		UPDATE feeds
		SET
			feed_url = $2,
			title = $3,
			description = $4,
			site_url = $5,
			polling_tier = $6,
			last_fetched_at = $7,
			last_published_at = $8,
			next_poll_at = $9,
			status = $10,
			consecutive_error_days = $11,
			last_error_at = $12,
			last_error_message = $13,
			updated_at = $14
		WHERE id = $1
	`

	feed.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		feed.ID, feed.FeedURL, feed.Title, feed.Description, feed.SiteURL,
		feed.PollingTier, feed.LastFetchedAt, feed.LastPublishedAt, feed.NextPollAt,
		feed.Status, feed.ConsecutiveErrorDays, feed.LastErrorAt, feed.LastErrorMessage,
		feed.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update feed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed not found")
	}

	return nil
}

// UpdatePollingInfo updates polling-related fields
func (r *feedRepository) UpdatePollingInfo(ctx context.Context, id uuid.UUID, lastFetchedAt, lastPublishedAt *time.Time, nextPollAt time.Time, tier models.PollingTier) error {
	query := `
		UPDATE feeds
		SET
			last_fetched_at = $2,
			last_published_at = $3,
			next_poll_at = $4,
			polling_tier = $5,
			updated_at = $6
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, lastFetchedAt, lastPublishedAt, nextPollAt, tier, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update polling info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed not found")
	}

	return nil
}

// UpdateErrorInfo updates error tracking fields
func (r *feedRepository) UpdateErrorInfo(ctx context.Context, id uuid.UUID, consecutiveErrorDays int, lastErrorAt *time.Time, lastErrorMessage *string) error {
	query := `
		UPDATE feeds
		SET
			consecutive_error_days = $2,
			last_error_at = $3,
			last_error_message = $4,
			updated_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, consecutiveErrorDays, lastErrorAt, lastErrorMessage, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update error info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed not found")
	}

	return nil
}

// UpdateStatus updates the feed status
func (r *feedRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.FeedStatus) error {
	query := `
		UPDATE feeds
		SET
			status = $2,
			updated_at = $3
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed not found")
	}

	return nil
}

// GetFeedsDueForPolling atomically claims and retrieves feeds that are ready
// to be polled. Unlike feed_items/content_outbox there is no in-flight status
// on feeds — a crashed poll just leaves next_poll_at in the past and gets
// picked up next cycle, so lease_expires_at here exists solely to stop two
// concurrent pollers (or a retry racing a still-in-flight poll) from fetching
// the same feed at once; it does not participate in crash recovery. The claim
// is a single statement: SELECT ... FOR UPDATE SKIP LOCKED narrows the batch
// and holds row locks only for the instant of the following UPDATE, so two
// concurrent callers never claim the same feed. next_poll_at is left
// untouched here — that's owned by UpdateFetchResult/tier logic downstream.
func (r *feedRepository) GetFeedsDueForPolling(ctx context.Context, limit int) ([]*models.Feed, error) {
	query := `
		WITH claimed AS (
			SELECT id FROM feeds
			WHERE status = 'active'
			  AND next_poll_at <= $1
			  AND (lease_expires_at IS NULL OR lease_expires_at < $1)
			ORDER BY next_poll_at ASC
			LIMIT $2
			FOR UPDATE SKIP LOCKED
		)
		UPDATE feeds
		SET lease_expires_at = $1 + interval '10 minutes'
		FROM claimed
		WHERE feeds.id = claimed.id
		RETURNING
			feeds.id, feeds.feed_url, feeds.title, feeds.description, feeds.site_url,
			feeds.polling_tier, feeds.last_fetched_at, feeds.last_published_at, feeds.next_poll_at,
			feeds.status, feeds.consecutive_error_days, feeds.last_error_at, feeds.last_error_message,
			feeds.created_at, feeds.updated_at
	`

	rows, err := r.db.QueryContext(ctx, query, time.Now(), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query feeds due for polling: %w", err)
	}
	defer func() { _ = rows.Close() }()

	feeds := make([]*models.Feed, 0)
	for rows.Next() {
		feed := &models.Feed{}
		err := rows.Scan(
			&feed.ID, &feed.FeedURL, &feed.Title, &feed.Description, &feed.SiteURL,
			&feed.PollingTier, &feed.LastFetchedAt, &feed.LastPublishedAt, &feed.NextPollAt,
			&feed.Status, &feed.ConsecutiveErrorDays, &feed.LastErrorAt, &feed.LastErrorMessage,
			&feed.CreatedAt, &feed.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	return feeds, nil
}

// GetFeedsForTierUpdate retrieves all active feeds for tier management
func (r *feedRepository) GetFeedsForTierUpdate(ctx context.Context) ([]*models.Feed, error) {
	query := `
		SELECT
			id, feed_url, title, description, site_url,
			polling_tier, last_fetched_at, last_published_at, next_poll_at,
			status, consecutive_error_days, last_error_at, last_error_message,
			created_at, updated_at
		FROM feeds
		WHERE status = 'active'
		ORDER BY last_published_at DESC NULLS LAST
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query feeds for tier update: %w", err)
	}
	defer func() { _ = rows.Close() }()

	feeds := make([]*models.Feed, 0)
	for rows.Next() {
		feed := &models.Feed{}
		err := rows.Scan(
			&feed.ID, &feed.FeedURL, &feed.Title, &feed.Description, &feed.SiteURL,
			&feed.PollingTier, &feed.LastFetchedAt, &feed.LastPublishedAt, &feed.NextPollAt,
			&feed.Status, &feed.ConsecutiveErrorDays, &feed.LastErrorAt, &feed.LastErrorMessage,
			&feed.CreatedAt, &feed.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed: %w", err)
		}
		feeds = append(feeds, feed)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feeds: %w", err)
	}

	return feeds, nil
}

// Delete deletes a feed (and cascades to subscriptions and items)
func (r *feedRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM feeds WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feed: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed not found")
	}

	return nil
}
