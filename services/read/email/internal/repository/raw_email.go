package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
)

// RawEmailRepository defines the interface for raw email data operations
type RawEmailRepository interface {
	// Create creates a new raw email record
	Create(ctx context.Context, email *models.RawEmail) error

	// GetByID retrieves a raw email by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.RawEmail, error)

	// GetPendingEmails retrieves raw emails with pending status for processing
	GetPendingEmails(ctx context.Context, limit int) ([]*models.RawEmail, error)

	// UpdateStatus updates the processing status of a raw email
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, processedAt *time.Time) error

	// UpdateError updates the error information for a failed email
	UpdateError(ctx context.Context, id uuid.UUID, retryCount int, errorMsg string) error

	// ReleaseClaim resets lease_expires_at to now() without changing
	// processing_status, so an email that was atomically claimed but then
	// discarded before processing started (e.g. the worker is shutting down)
	// is immediately reselectable on the next poll instead of sitting claimed
	// for the full lease duration.
	ReleaseClaim(ctx context.Context, id uuid.UUID) error

	// DeleteProcessed deletes up to batchSize processed emails older than the given duration
	DeleteProcessed(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)
}

// rawEmailRepository is the concrete implementation of RawEmailRepository
type rawEmailRepository struct {
	db *sql.DB
}

// NewRawEmailRepository creates a new RawEmailRepository
func NewRawEmailRepository(db *sql.DB) RawEmailRepository {
	return &rawEmailRepository{db: db}
}

// Create creates a new raw email record
func (r *rawEmailRepository) Create(ctx context.Context, email *models.RawEmail) error {
	query := `
		INSERT INTO raw_emails (
			id, user_id, sender_id,
			recipient, sender_email, sender_name, subject, html_body, text_body, received_at,
			processing_status, content_hash, retry_count, last_error,
			created_at, processed_at
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14,
			$15, $16
		)
	`

	if email.ID == uuid.Nil {
		email.ID = uuid.New()
	}
	if email.CreatedAt.IsZero() {
		email.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query,
		email.ID,
		email.UserID,
		email.SenderID,
		email.Recipient,
		email.SenderEmail,
		email.SenderName,
		email.Subject,
		email.HTMLBody,
		email.TextBody,
		email.ReceivedAt,
		email.ProcessingStatus,
		email.ContentHash,
		email.RetryCount,
		email.LastError,
		email.CreatedAt,
		email.ProcessedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create raw email: %w", err)
	}

	return nil
}

// GetByID retrieves a raw email by ID
func (r *rawEmailRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.RawEmail, error) {
	query := `
		SELECT id, user_id, sender_id,
		       recipient, sender_email, sender_name, subject, html_body, text_body, received_at,
		       processing_status, content_hash, retry_count, last_error,
		       created_at, processed_at
		FROM raw_emails
		WHERE id = $1
	`

	var email models.RawEmail
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&email.ID,
		&email.UserID,
		&email.SenderID,
		&email.Recipient,
		&email.SenderEmail,
		&email.SenderName,
		&email.Subject,
		&email.HTMLBody,
		&email.TextBody,
		&email.ReceivedAt,
		&email.ProcessingStatus,
		&email.ContentHash,
		&email.RetryCount,
		&email.LastError,
		&email.CreatedAt,
		&email.ProcessedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get raw email by ID: %w", err)
	}

	return &email, nil
}

// GetPendingEmails atomically claims and retrieves raw emails with pending
// status, plus emails stranded in 'processing' whose lease has expired (e.g.
// a worker that claimed the email and then crashed before finishing). The
// claim is a single statement: SELECT ... FOR UPDATE SKIP LOCKED narrows the
// batch and holds row locks only for the instant of the following UPDATE, so
// two concurrent callers never claim the same email, and the lease (not a
// held transaction) is what lets a crashed worker's email be re-claimed later.
func (r *rawEmailRepository) GetPendingEmails(ctx context.Context, limit int) ([]*models.RawEmail, error) {
	query := `
		WITH claimed AS (
			SELECT id FROM raw_emails
			WHERE (processing_status = 'pending'
			       OR (processing_status = 'processing' AND (lease_expires_at IS NULL OR lease_expires_at < now())))
			  AND retry_count < 5
			ORDER BY created_at ASC
			LIMIT $1
			FOR UPDATE SKIP LOCKED
		)
		UPDATE raw_emails
		SET processing_status = 'processing',
		    lease_expires_at = now() + interval '10 minutes'
		FROM claimed
		WHERE raw_emails.id = claimed.id
		RETURNING
			raw_emails.id, raw_emails.user_id, raw_emails.sender_id,
			raw_emails.recipient, raw_emails.sender_email, raw_emails.sender_name, raw_emails.subject, raw_emails.html_body, raw_emails.text_body, raw_emails.received_at,
			raw_emails.processing_status, raw_emails.content_hash, raw_emails.retry_count, raw_emails.last_error,
			raw_emails.created_at, raw_emails.processed_at
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending emails: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var emails []*models.RawEmail
	for rows.Next() {
		var email models.RawEmail
		err := rows.Scan(
			&email.ID,
			&email.UserID,
			&email.SenderID,
			&email.Recipient,
			&email.SenderEmail,
			&email.SenderName,
			&email.Subject,
			&email.HTMLBody,
			&email.TextBody,
			&email.ReceivedAt,
			&email.ProcessingStatus,
			&email.ContentHash,
			&email.RetryCount,
			&email.LastError,
			&email.CreatedAt,
			&email.ProcessedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan raw email: %w", err)
		}
		emails = append(emails, &email)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating raw email rows: %w", err)
	}

	return emails, nil
}

// UpdateStatus updates the processing status of a raw email
func (r *rawEmailRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, processedAt *time.Time) error {
	query := `
		UPDATE raw_emails
		SET processing_status = $1,
		    processed_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, status, processedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update raw email status: %w", err)
	}

	return nil
}

// ReleaseClaim resets lease_expires_at to now() without changing
// processing_status, making a claimed-but-not-yet-processed email immediately
// reselectable on the next poll. Deliberately doesn't check RowsAffected,
// matching this file's other update methods (UpdateStatus, UpdateError) --
// a release racing a row that's already gone (e.g. deleted by cleanup) is a
// harmless no-op on the shutdown path, not an error worth surfacing.
func (r *rawEmailRepository) ReleaseClaim(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE raw_emails SET lease_expires_at = now() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to release claim: %w", err)
	}
	return nil
}

// UpdateError updates the error information for a failed email. On the
// non-terminal branch it also resets lease_expires_at so the email is
// immediately reselectable by GetPendingEmails on the next poll instead of
// waiting out the claim lease.
func (r *rawEmailRepository) UpdateError(ctx context.Context, id uuid.UUID, retryCount int, errorMsg string) error {
	query := `
		UPDATE raw_emails
		SET retry_count = $1,
		    last_error = $2,
		    processing_status = CASE
		        WHEN $1 >= 5 THEN $3::VARCHAR(20)
		        ELSE processing_status
		    END,
		    lease_expires_at = CASE
		        WHEN $1 >= 5 THEN lease_expires_at
		        ELSE now()
		    END
		WHERE id = $4
	`

	_, err := r.db.ExecContext(ctx, query, retryCount, errorMsg, models.ProcessingStatusFailed, id)
	if err != nil {
		return fmt.Errorf("failed to update raw email error: %w", err)
	}

	return nil
}

// DeleteProcessed deletes up to batchSize processed emails older than the given duration
func (r *rawEmailRepository) DeleteProcessed(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	query := `
		DELETE FROM raw_emails
		WHERE id IN (
			SELECT id FROM raw_emails
			WHERE processing_status = $1 AND processed_at < $2
			LIMIT $3
		)
	`

	cutoff := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, query, models.ProcessingStatusCompleted, cutoff, batchSize)
	if err != nil {
		return 0, fmt.Errorf("failed to delete processed emails: %w", err)
	}

	count, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return count, nil
}
