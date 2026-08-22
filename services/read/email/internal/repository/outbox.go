package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
)

// OutboxRepository defines the interface for content outbox data operations
type OutboxRepository interface {
	// Create creates a new outbox entry
	Create(ctx context.Context, outbox *models.ContentOutbox) error

	// GetByID retrieves an outbox entry by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error)

	// GetPendingEntries atomically claims and retrieves outbox entries ready
	// for delivery
	GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error)

	// UpdateDeliveryStatus updates the delivery status
	UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time) error

	// UpdateRetryInfo updates retry information after a failed delivery
	// attempt. On the non-terminal branch it also resets lease_expires_at so
	// the entry is immediately reselectable on the next poll.
	UpdateRetryInfo(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt time.Time, lastError string) error

	// ReleaseClaim resets lease_expires_at to now() without changing
	// delivery_status, so an entry that was atomically claimed but then
	// discarded before delivery started (e.g. the worker is shutting down) is
	// immediately reselectable on the next poll instead of sitting claimed
	// for the full lease duration.
	ReleaseClaim(ctx context.Context, id uuid.UUID) error

	// DeleteDelivered deletes up to batchSize delivered entries older than the given duration
	DeleteDelivered(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)
}

// outboxRepository is the concrete implementation of OutboxRepository
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
			id, raw_email_id,
			content_payload, user_id,
			delivery_status, retry_count, max_retries, next_retry_at, last_error,
			content_service_id,
			created_at, delivered_at
		) VALUES (
			$1, $2,
			$3, $4,
			$5, $6, $7, $8, $9,
			$10,
			$11, $12
		)
	`

	if outbox.ID == uuid.Nil {
		outbox.ID = uuid.New()
	}
	if outbox.CreatedAt.IsZero() {
		outbox.CreatedAt = time.Now()
	}

	// Marshal the content payload to JSONB
	payloadJSON, err := json.Marshal(outbox.ContentPayload)
	if err != nil {
		return fmt.Errorf("failed to marshal content payload: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		outbox.ID,
		outbox.RawEmailID,
		payloadJSON,
		outbox.UserID,
		outbox.DeliveryStatus,
		outbox.RetryCount,
		outbox.MaxRetries,
		outbox.NextRetryAt,
		outbox.LastError,
		outbox.ContentServiceID,
		outbox.CreatedAt,
		outbox.DeliveredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create outbox entry: %w", err)
	}

	return nil
}

// GetByID retrieves an outbox entry by ID
func (r *outboxRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error) {
	query := `
		SELECT id, raw_email_id,
		       content_payload, user_id,
		       delivery_status, retry_count, max_retries, next_retry_at, last_error,
		       content_service_id,
		       created_at, delivered_at
		FROM content_outbox
		WHERE id = $1
	`

	var outbox models.ContentOutbox
	var payloadJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&outbox.ID,
		&outbox.RawEmailID,
		&payloadJSON,
		&outbox.UserID,
		&outbox.DeliveryStatus,
		&outbox.RetryCount,
		&outbox.MaxRetries,
		&outbox.NextRetryAt,
		&outbox.LastError,
		&outbox.ContentServiceID,
		&outbox.CreatedAt,
		&outbox.DeliveredAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get outbox entry by ID: %w", err)
	}

	// Unmarshal the JSONB payload
	err = json.Unmarshal(payloadJSON, &outbox.ContentPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal content payload: %w", err)
	}

	return &outbox, nil
}

// GetPendingEntries atomically claims and retrieves outbox entries ready for
// delivery, plus entries stranded in 'sending' whose lease has expired (e.g.
// a worker that claimed the entry and then crashed mid-delivery). The claim
// is a single statement: SELECT ... FOR UPDATE SKIP LOCKED narrows the batch
// and holds row locks only for the instant of the following UPDATE, so two
// concurrent callers never claim the same entry, and the lease (not a held
// transaction) is what lets a crashed worker's entry be re-claimed later.
func (r *outboxRepository) GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	query := `
		WITH claimed AS (
			SELECT id FROM content_outbox
			WHERE (delivery_status = $1
			       OR (delivery_status = $2 AND (lease_expires_at IS NULL OR lease_expires_at < now())))
			  AND next_retry_at <= $3
			  AND retry_count < max_retries
			ORDER BY created_at ASC
			LIMIT $4
			FOR UPDATE SKIP LOCKED
		)
		UPDATE content_outbox
		SET delivery_status = $2,
		    lease_expires_at = now() + interval '10 minutes'
		FROM claimed
		WHERE content_outbox.id = claimed.id
		RETURNING
			content_outbox.id, content_outbox.raw_email_id,
			content_outbox.content_payload, content_outbox.user_id,
			content_outbox.delivery_status, content_outbox.retry_count, content_outbox.max_retries, content_outbox.next_retry_at, content_outbox.last_error,
			content_outbox.content_service_id,
			content_outbox.created_at, content_outbox.delivered_at
	`

	rows, err := r.db.QueryContext(ctx, query,
		models.DeliveryStatusPending,
		models.DeliveryStatusSending,
		time.Now(),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending outbox entries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []*models.ContentOutbox
	for rows.Next() {
		var outbox models.ContentOutbox
		var payloadJSON []byte

		err := rows.Scan(
			&outbox.ID,
			&outbox.RawEmailID,
			&payloadJSON,
			&outbox.UserID,
			&outbox.DeliveryStatus,
			&outbox.RetryCount,
			&outbox.MaxRetries,
			&outbox.NextRetryAt,
			&outbox.LastError,
			&outbox.ContentServiceID,
			&outbox.CreatedAt,
			&outbox.DeliveredAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan outbox entry: %w", err)
		}

		// Unmarshal the JSONB payload
		err = json.Unmarshal(payloadJSON, &outbox.ContentPayload)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal content payload: %w", err)
		}

		entries = append(entries, &outbox)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating outbox rows: %w", err)
	}

	return entries, nil
}

// UpdateDeliveryStatus updates the delivery status
func (r *outboxRepository) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time) error {
	query := `
		UPDATE content_outbox
		SET delivery_status = $1,
		    content_service_id = $2,
		    delivered_at = $3
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, status, contentServiceID, deliveredAt, id)
	if err != nil {
		return fmt.Errorf("failed to update outbox delivery status: %w", err)
	}

	return nil
}

// UpdateRetryInfo updates retry information after a failed delivery attempt
func (r *outboxRepository) UpdateRetryInfo(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt time.Time, lastError string) error {
	query := `
		UPDATE content_outbox
		SET retry_count = $1,
		    next_retry_at = $2,
		    last_error = $3,
		    delivery_status = CASE
		        WHEN $1 >= max_retries THEN $4::VARCHAR(20)
		        ELSE delivery_status
		    END,
		    lease_expires_at = CASE
		        WHEN $1 >= max_retries THEN lease_expires_at
		        ELSE now()
		    END
		WHERE id = $5
	`

	_, err := r.db.ExecContext(ctx, query, retryCount, nextRetryAt, lastError, models.DeliveryStatusFailed, id)
	if err != nil {
		return fmt.Errorf("failed to update outbox retry info: %w", err)
	}

	return nil
}

// ReleaseClaim resets lease_expires_at to now() without changing
// delivery_status, making a claimed-but-not-yet-delivered entry immediately
// reselectable on the next poll. Deliberately doesn't check RowsAffected,
// matching this file's other update methods (UpdateDeliveryStatus,
// UpdateRetryInfo) -- a release racing a row that's already gone (e.g.
// deleted by cleanup) is a harmless no-op on the shutdown path, not an error
// worth surfacing.
func (r *outboxRepository) ReleaseClaim(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE content_outbox SET lease_expires_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to release claim: %w", err)
	}
	return nil
}

// DeleteDelivered deletes up to batchSize delivered entries older than the given duration
func (r *outboxRepository) DeleteDelivered(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	query := `
		DELETE FROM content_outbox
		WHERE id IN (
			SELECT id FROM content_outbox
			WHERE delivery_status = $1 AND delivered_at < $2
			LIMIT $3
		)
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, query, models.DeliveryStatusDelivered, cutoff, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to delete delivered outbox entries: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}
