package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// OutboxRepository defines the interface for content outbox operations
type OutboxRepository interface {
	// Create creates a new outbox entry
	Create(ctx context.Context, outbox *models.ContentOutbox) error

	// GetByID retrieves an outbox entry by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error)

	// GetPendingEntries retrieves outbox entries ready for delivery
	GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error)

	// UpdateDeliveryStatus updates the delivery status of an outbox entry
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time, lastError *string) error

	// IncrementRetryCount increments the retry count and sets next retry time
	IncrementRetryCount(ctx context.Context, id uuid.UUID, nextRetryAt time.Time, lastError string) error

	// DeleteOldDeliveredEntries deletes delivered entries older than the given date
	DeleteOldDeliveredEntries(ctx context.Context, olderThan time.Time, batchSize int) (int, error)

	// GetFailedEntries retrieves failed outbox entries for investigation
	GetFailedEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error)

	// Delete deletes an outbox entry
	Delete(ctx context.Context, id uuid.UUID) error

	// GetMetrics retrieves metrics about the outbox for monitoring
	GetMetrics(ctx context.Context) (*OutboxMetrics, error)
}

// OutboxMetrics contains metrics about the outbox for monitoring
type OutboxMetrics struct {
	PendingCount   int64
	SendingCount   int64
	DeliveredCount int64
	FailedCount    int64
	TotalCount     int64
}

// outboxRepository is the concrete implementation
type outboxRepository struct {
	db *sql.DB
}

// NewOutboxRepository creates a new OutboxRepository
func NewOutboxRepository(db *sql.DB) OutboxRepository {
	return &outboxRepository{db: db}
}

// Create creates a new outbox entry
func (r *outboxRepository) Create(ctx context.Context, outbox *models.ContentOutbox) error {
	query := `
		INSERT INTO content_outbox (
			id, feed_item_id, content_payload, user_ids,
			delivery_status, retry_count, max_retries, next_retry_at, last_error,
			content_service_id, created_at, delivered_at
		) VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8, $9,
			$10, $11, $12
		)
	`

	if outbox.ID == uuid.Nil {
		outbox.ID = uuid.New()
	}
	if outbox.CreatedAt.IsZero() {
		outbox.CreatedAt = time.Now()
	}

	// Convert content_payload to JSON
	contentPayloadJSON, err := json.Marshal(outbox.ContentPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal content_payload: %w", err)
	}

	// Convert user_ids to PostgreSQL array
	userIDStrings := make([]string, len(outbox.UserIDs))
	for i, id := range outbox.UserIDs {
		userIDStrings[i] = id.String()
	}

	_, err = r.db.ExecContext(ctx, query,
		outbox.ID, outbox.FeedItemID, contentPayloadJSON, pq.Array(userIDStrings),
		outbox.DeliveryStatus, outbox.RetryCount, outbox.MaxRetries, outbox.NextRetryAt, outbox.LastError,
		outbox.ContentServiceID, outbox.CreatedAt, outbox.DeliveredAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create outbox entry: %w", err)
	}

	return nil
}

// GetByID retrieves an outbox entry by its ID
func (r *outboxRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error) {
	query := `
		SELECT
			id, feed_item_id, content_payload, user_ids,
			delivery_status, retry_count, max_retries, next_retry_at, last_error,
			content_service_id, created_at, delivered_at
		FROM content_outbox
		WHERE id = $1
	`

	outbox := &models.ContentOutbox{}
	var contentPayloadJSON []byte
	var userIDStrings []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&outbox.ID, &outbox.FeedItemID, &contentPayloadJSON, pq.Array(&userIDStrings),
		&outbox.DeliveryStatus, &outbox.RetryCount, &outbox.MaxRetries, &outbox.NextRetryAt, &outbox.LastError,
		&outbox.ContentServiceID, &outbox.CreatedAt, &outbox.DeliveredAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox entry by ID: %w", err)
	}

	// Unmarshal content_payload
	if err := json.Unmarshal(contentPayloadJSON, &outbox.ContentPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal content_payload: %w", err)
	}

	// Convert user_ids from strings to UUIDs
	outbox.UserIDs = make([]uuid.UUID, len(userIDStrings))
	for i, idStr := range userIDStrings {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse user_id: %w", err)
		}
		outbox.UserIDs[i] = id
	}

	return outbox, nil
}

// GetPendingEntries retrieves outbox entries ready for delivery
func (r *outboxRepository) GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	query := `
		SELECT
			id, feed_item_id, content_payload, user_ids,
			delivery_status, retry_count, max_retries, next_retry_at, last_error,
			content_service_id, created_at, delivered_at
		FROM content_outbox
		WHERE delivery_status = 'pending' AND next_retry_at <= $1
		ORDER BY next_retry_at ASC
		LIMIT $2
	`

	return r.queryOutboxEntries(ctx, query, time.Now(), limit)
}

// UpdateDeliveryStatus updates the delivery status of an outbox entry
func (r *outboxRepository) UpdateDeliveryStatus(
	ctx context.Context,
	id uuid.UUID,
	status models.DeliveryStatus,
	contentServiceID *uuid.UUID,
	deliveredAt *time.Time,
	lastError *string,
) error {
	query := `
		UPDATE content_outbox
		SET
			delivery_status = $2,
			content_service_id = $3,
			delivered_at = $4,
			last_error = $5
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, status, contentServiceID, deliveredAt, lastError)
	if err != nil {
		return fmt.Errorf("failed to update delivery status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("outbox entry not found")
	}

	return nil
}

// IncrementRetryCount increments the retry count and sets next retry time
func (r *outboxRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID, nextRetryAt time.Time, lastError string) error {
	query := `
		UPDATE content_outbox
		SET
			retry_count = retry_count + 1,
			next_retry_at = $2,
			last_error = $3,
			delivery_status = CASE
				WHEN retry_count + 1 >= max_retries THEN 'failed'::delivery_status
				ELSE 'pending'::delivery_status
			END
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, nextRetryAt, lastError)
	if err != nil {
		return fmt.Errorf("failed to increment retry count: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("outbox entry not found")
	}

	return nil
}

// DeleteOldDeliveredEntries deletes delivered entries older than the given date
func (r *outboxRepository) DeleteOldDeliveredEntries(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	query := `
		DELETE FROM content_outbox
		WHERE id IN (
			SELECT id FROM content_outbox
			WHERE delivery_status = 'delivered' AND delivered_at < $1
			LIMIT $2
		)
	`

	result, err := r.db.ExecContext(ctx, query, olderThan, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old delivered entries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetFailedEntries retrieves failed outbox entries for investigation
func (r *outboxRepository) GetFailedEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	query := `
		SELECT
			id, feed_item_id, content_payload, user_ids,
			delivery_status, retry_count, max_retries, next_retry_at, last_error,
			content_service_id, created_at, delivered_at
		FROM content_outbox
		WHERE delivery_status = 'failed'
		ORDER BY created_at DESC
		LIMIT $1
	`

	return r.queryOutboxEntries(ctx, query, limit)
}

// Delete deletes an outbox entry
func (r *outboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM content_outbox WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete outbox entry: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("outbox entry not found")
	}

	return nil
}

// queryOutboxEntries is a helper function to query and scan outbox entries
func (r *outboxRepository) queryOutboxEntries(ctx context.Context, query string, args ...interface{}) ([]*models.ContentOutbox, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query outbox entries: %w", err)
	}
	defer rows.Close()

	entries := make([]*models.ContentOutbox, 0)
	for rows.Next() {
		outbox := &models.ContentOutbox{}
		var contentPayloadJSON []byte
		var userIDStrings []string

		err := rows.Scan(
			&outbox.ID, &outbox.FeedItemID, &contentPayloadJSON, pq.Array(&userIDStrings),
			&outbox.DeliveryStatus, &outbox.RetryCount, &outbox.MaxRetries, &outbox.NextRetryAt, &outbox.LastError,
			&outbox.ContentServiceID, &outbox.CreatedAt, &outbox.DeliveredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
		}

		// Unmarshal content_payload
		if err := json.Unmarshal(contentPayloadJSON, &outbox.ContentPayload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal content_payload: %w", err)
		}

		// Convert user_ids from strings to UUIDs
		outbox.UserIDs = make([]uuid.UUID, len(userIDStrings))
		for i, idStr := range userIDStrings {
			id, err := uuid.Parse(idStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse user_id: %w", err)
			}
			outbox.UserIDs[i] = id
		}

		entries = append(entries, outbox)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outbox entries: %w", err)
	}

	return entries, nil
}

// GetMetrics retrieves metrics about the outbox for monitoring
func (r *outboxRepository) GetMetrics(ctx context.Context) (*OutboxMetrics, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE delivery_status = 'pending') AS pending_count,
			COUNT(*) FILTER (WHERE delivery_status = 'sending') AS sending_count,
			COUNT(*) FILTER (WHERE delivery_status = 'delivered') AS delivered_count,
			COUNT(*) FILTER (WHERE delivery_status = 'failed') AS failed_count,
			COUNT(*) AS total_count
		FROM content_outbox
	`

	metrics := &OutboxMetrics{}
	err := r.db.QueryRowContext(ctx, query).Scan(
		&metrics.PendingCount,
		&metrics.SendingCount,
		&metrics.DeliveredCount,
		&metrics.FailedCount,
		&metrics.TotalCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get outbox metrics: %w", err)
	}

	return metrics, nil
}
