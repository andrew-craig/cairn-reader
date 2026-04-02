package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/google/uuid"
)

// FeedItemRepository defines the interface for feed item operations
type FeedItemRepository interface {
	// Create creates a new feed item
	Create(ctx context.Context, item *models.FeedItem) error

	// GetByID retrieves a feed item by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.FeedItem, error)

	// GetByFeedAndGUID retrieves a feed item by feed ID and GUID
	GetByFeedAndGUID(ctx context.Context, feedID uuid.UUID, guid string) (*models.FeedItem, error)

	// Update updates an existing feed item
	Update(ctx context.Context, item *models.FeedItem) error

	// UpdateProcessingStatus updates the processing status of a feed item
	UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, contentHash *string, contentServiceID *uuid.UUID, processedAt *time.Time) error

	// UpdateContentUpdateInfo updates content update tracking information
	UpdateContentUpdateInfo(ctx context.Context, id uuid.UUID, httpLastModified, httpETag *string, lastCheckedAt *time.Time) error

	// IncrementRetryCount increments the retry count and updates error info
	IncrementRetryCount(ctx context.Context, id uuid.UUID, lastError string) error

	// GetPendingItems retrieves feed items with pending status
	GetPendingItems(ctx context.Context, limit int) ([]*models.FeedItem, error)

	// GetItemsForUpdateCheck retrieves completed items to check for content updates
	GetItemsForUpdateCheck(ctx context.Context, limit int) ([]*models.FeedItem, error)

	// GetByFeedID retrieves all items for a feed
	GetByFeedID(ctx context.Context, feedID uuid.UUID, limit int) ([]*models.FeedItem, error)

	// DeleteOldCompletedItems deletes completed items older than the given date
	DeleteOldCompletedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error)

	// DeleteOldFailedItems deletes failed items older than the given date
	DeleteOldFailedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error)

	// Delete deletes a feed item
	Delete(ctx context.Context, id uuid.UUID) error

	// GetMetrics retrieves metrics about feed items for monitoring
	GetMetrics(ctx context.Context) (*FeedItemMetrics, error)
}

// FeedItemMetrics contains metrics about feed items for monitoring
type FeedItemMetrics struct {
	PendingCount    int64
	ProcessingCount int64
	CompletedCount  int64
	FailedCount     int64
	TotalCount      int64
}

// feedItemRepository is the concrete implementation
type feedItemRepository struct {
	db *sql.DB
}

// NewFeedItemRepository creates a new FeedItemRepository
func NewFeedItemRepository(db *sql.DB) FeedItemRepository {
	return &feedItemRepository{db: db}
}

// Create creates a new feed item
func (r *feedItemRepository) Create(ctx context.Context, item *models.FeedItem) error {
	query := `
		INSERT INTO feed_items (
			id, feed_id, item_url, item_guid,
			processing_status, content_hash, content_service_id,
			title, author, published_at, description,
			http_last_modified, http_etag, last_checked_at,
			retry_count, last_error,
			discovered_at, processed_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, $16,
			$17, $18
		)
	`

	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.DiscoveredAt.IsZero() {
		item.DiscoveredAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query,
		item.ID, item.FeedID, item.ItemURL, item.ItemGUID,
		item.ProcessingStatus, item.ContentHash, item.ContentServiceID,
		item.Title, item.Author, item.PublishedAt, item.Description,
		item.HTTPLastModified, item.HTTPETag, item.LastCheckedAt,
		item.RetryCount, item.LastError,
		item.DiscoveredAt, item.ProcessedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create feed item: %w", err)
	}

	return nil
}

// GetByID retrieves a feed item by its ID
func (r *feedItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FeedItem, error) {
	query := `
		SELECT
			id, feed_id, item_url, item_guid,
			processing_status, content_hash, content_service_id,
			title, author, published_at, description,
			http_last_modified, http_etag, last_checked_at,
			retry_count, last_error,
			discovered_at, processed_at
		FROM feed_items
		WHERE id = $1
	`

	item := &models.FeedItem{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&item.ID, &item.FeedID, &item.ItemURL, &item.ItemGUID,
		&item.ProcessingStatus, &item.ContentHash, &item.ContentServiceID,
		&item.Title, &item.Author, &item.PublishedAt, &item.Description,
		&item.HTTPLastModified, &item.HTTPETag, &item.LastCheckedAt,
		&item.RetryCount, &item.LastError,
		&item.DiscoveredAt, &item.ProcessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed item by ID: %w", err)
	}

	return item, nil
}

// GetByFeedAndGUID retrieves a feed item by feed ID and GUID
func (r *feedItemRepository) GetByFeedAndGUID(ctx context.Context, feedID uuid.UUID, guid string) (*models.FeedItem, error) {
	query := `
		SELECT
			id, feed_id, item_url, item_guid,
			processing_status, content_hash, content_service_id,
			title, author, published_at, description,
			http_last_modified, http_etag, last_checked_at,
			retry_count, last_error,
			discovered_at, processed_at
		FROM feed_items
		WHERE feed_id = $1 AND item_guid = $2
	`

	item := &models.FeedItem{}
	err := r.db.QueryRowContext(ctx, query, feedID, guid).Scan(
		&item.ID, &item.FeedID, &item.ItemURL, &item.ItemGUID,
		&item.ProcessingStatus, &item.ContentHash, &item.ContentServiceID,
		&item.Title, &item.Author, &item.PublishedAt, &item.Description,
		&item.HTTPLastModified, &item.HTTPETag, &item.LastCheckedAt,
		&item.RetryCount, &item.LastError,
		&item.DiscoveredAt, &item.ProcessedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get feed item by feed and GUID: %w", err)
	}

	return item, nil
}

// Update updates an existing feed item
func (r *feedItemRepository) Update(ctx context.Context, item *models.FeedItem) error {
	query := `
		UPDATE feed_items
		SET
			feed_id = $2,
			item_url = $3,
			item_guid = $4,
			processing_status = $5,
			content_hash = $6,
			content_service_id = $7,
			title = $8,
			author = $9,
			published_at = $10,
			description = $11,
			http_last_modified = $12,
			http_etag = $13,
			last_checked_at = $14,
			retry_count = $15,
			last_error = $16,
			processed_at = $17
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query,
		item.ID, item.FeedID, item.ItemURL, item.ItemGUID,
		item.ProcessingStatus, item.ContentHash, item.ContentServiceID,
		item.Title, item.Author, item.PublishedAt, item.Description,
		item.HTTPLastModified, item.HTTPETag, item.LastCheckedAt,
		item.RetryCount, item.LastError,
		item.ProcessedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update feed item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed item not found")
	}

	return nil
}

// UpdateProcessingStatus updates the processing status of a feed item
func (r *feedItemRepository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, contentHash *string, contentServiceID *uuid.UUID, processedAt *time.Time) error {
	query := `
		UPDATE feed_items
		SET
			processing_status = $2,
			content_hash = $3,
			content_service_id = $4,
			processed_at = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, contentHash, contentServiceID, processedAt)
	if err != nil {
		return fmt.Errorf("failed to update processing status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed item not found")
	}

	return nil
}

// UpdateContentUpdateInfo updates content update tracking information
func (r *feedItemRepository) UpdateContentUpdateInfo(ctx context.Context, id uuid.UUID, httpLastModified, httpETag *string, lastCheckedAt *time.Time) error {
	query := `
		UPDATE feed_items
		SET
			http_last_modified = $2,
			http_etag = $3,
			last_checked_at = $4
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, httpLastModified, httpETag, lastCheckedAt)
	if err != nil {
		return fmt.Errorf("failed to update content update info: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed item not found")
	}

	return nil
}

// IncrementRetryCount increments the retry count and updates error info
func (r *feedItemRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID, lastError string) error {
	query := `
		UPDATE feed_items
		SET
			retry_count = retry_count + 1,
			last_error = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, lastError)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed item not found")
	}

	return nil
}

// GetPendingItems retrieves feed items with pending status
func (r *feedItemRepository) GetPendingItems(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	query := `
		SELECT
			id, feed_id, item_url, item_guid,
			processing_status, content_hash, content_service_id,
			title, author, published_at, description,
			http_last_modified, http_etag, last_checked_at,
			retry_count, last_error,
			discovered_at, processed_at
		FROM feed_items
		WHERE processing_status = 'pending'
		ORDER BY discovered_at ASC
		LIMIT $1
	`

	return r.queryFeedItems(ctx, query, limit)
}

// GetItemsForUpdateCheck retrieves completed items to check for content updates
func (r *feedItemRepository) GetItemsForUpdateCheck(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	query := `
		SELECT
			id, feed_id, item_url, item_guid,
			processing_status, content_hash, content_service_id,
			title, author, published_at, description,
			http_last_modified, http_etag, last_checked_at,
			retry_count, last_error,
			discovered_at, processed_at
		FROM feed_items
		WHERE processing_status = 'completed'
			AND (last_checked_at IS NULL OR last_checked_at < NOW() - INTERVAL '24 hours')
		ORDER BY last_checked_at ASC NULLS FIRST
		LIMIT $1
	`

	return r.queryFeedItems(ctx, query, limit)
}

// GetByFeedID retrieves all items for a feed
func (r *feedItemRepository) GetByFeedID(ctx context.Context, feedID uuid.UUID, limit int) ([]*models.FeedItem, error) {
	query := `
		SELECT
			id, feed_id, item_url, item_guid,
			processing_status, content_hash, content_service_id,
			title, author, published_at, description,
			http_last_modified, http_etag, last_checked_at,
			retry_count, last_error,
			discovered_at, processed_at
		FROM feed_items
		WHERE feed_id = $1
		ORDER BY discovered_at DESC
		LIMIT $2
	`

	return r.queryFeedItems(ctx, query, feedID, limit)
}

// DeleteOldCompletedItems deletes completed items older than the given date
func (r *feedItemRepository) DeleteOldCompletedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	query := `
		DELETE FROM feed_items
		WHERE id IN (
			SELECT id FROM feed_items
			WHERE processing_status = 'completed' AND processed_at < $1
			LIMIT $2
		)
	`

	result, err := r.db.ExecContext(ctx, query, olderThan, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old completed items: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// DeleteOldFailedItems deletes failed items older than the given date
func (r *feedItemRepository) DeleteOldFailedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	query := `
		DELETE FROM feed_items
		WHERE id IN (
			SELECT id FROM feed_items
			WHERE processing_status = 'failed' AND discovered_at < $1
			LIMIT $2
		)
	`

	result, err := r.db.ExecContext(ctx, query, olderThan, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old failed items: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// Delete deletes a feed item
func (r *feedItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM feed_items WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete feed item: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("feed item not found")
	}

	return nil
}

// queryFeedItems is a helper function to query and scan feed items
func (r *feedItemRepository) queryFeedItems(ctx context.Context, query string, args ...interface{}) ([]*models.FeedItem, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query feed items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	items := make([]*models.FeedItem, 0)
	for rows.Next() {
		item := &models.FeedItem{}
		err := rows.Scan(
			&item.ID, &item.FeedID, &item.ItemURL, &item.ItemGUID,
			&item.ProcessingStatus, &item.ContentHash, &item.ContentServiceID,
			&item.Title, &item.Author, &item.PublishedAt, &item.Description,
			&item.HTTPLastModified, &item.HTTPETag, &item.LastCheckedAt,
			&item.RetryCount, &item.LastError,
			&item.DiscoveredAt, &item.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feed item: %w", err)
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating feed items: %w", err)
	}

	return items, nil
}

// GetMetrics retrieves metrics about feed items for monitoring
func (r *feedItemRepository) GetMetrics(ctx context.Context) (*FeedItemMetrics, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE processing_status = 'pending') AS pending_count,
			COUNT(*) FILTER (WHERE processing_status = 'processing') AS processing_count,
			COUNT(*) FILTER (WHERE processing_status = 'completed') AS completed_count,
			COUNT(*) FILTER (WHERE processing_status = 'failed') AS failed_count,
			COUNT(*) AS total_count
		FROM feed_items
	`

	metrics := &FeedItemMetrics{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&metrics.PendingCount,
		&metrics.ProcessingCount,
		&metrics.CompletedCount,
		&metrics.FailedCount,
		&metrics.TotalCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get feed items metrics: %w", err)
	}

	return metrics, nil
}
