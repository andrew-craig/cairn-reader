package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
)

// SenderRepository defines the interface for email sender data operations
type SenderRepository interface {
	// Create creates a new sender record
	Create(ctx context.Context, sender *models.EmailSender) error

	// GetByID retrieves a sender by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.EmailSender, error)

	// GetByUserAndEmail retrieves a sender by user ID and sender email
	GetByUserAndEmail(ctx context.Context, userID uuid.UUID, senderEmail string) (*models.EmailSender, error)

	// Upsert creates or updates a sender, incrementing email count
	Upsert(ctx context.Context, sender *models.EmailSender) error

	// ListByUser retrieves all senders for a user, ordered by last received
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error)

	// IncrementEmailCount increments the email count and updates last received time
	IncrementEmailCount(ctx context.Context, id uuid.UUID, receivedAt time.Time) error
}

// senderRepository is the concrete implementation of SenderRepository
type senderRepository struct {
	db *sql.DB
}

// NewSenderRepository creates a new SenderRepository
func NewSenderRepository(db *sql.DB) SenderRepository {
	return &senderRepository{db: db}
}

// Create creates a new sender record
func (r *senderRepository) Create(ctx context.Context, sender *models.EmailSender) error {
	query := `
		INSERT INTO email_senders (
			id, user_id, sender_email, sender_name,
			email_count, last_received_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
	`

	now := time.Now()
	if sender.ID == uuid.Nil {
		sender.ID = uuid.New()
	}
	if sender.CreatedAt.IsZero() {
		sender.CreatedAt = now
	}
	if sender.UpdatedAt.IsZero() {
		sender.UpdatedAt = now
	}

	_, err := r.db.ExecContext(ctx, query,
		sender.ID,
		sender.UserID,
		sender.SenderEmail,
		sender.SenderName,
		sender.EmailCount,
		sender.LastReceivedAt,
		sender.CreatedAt,
		sender.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create sender: %w", err)
	}

	return nil
}

// GetByID retrieves a sender by ID
func (r *senderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailSender, error) {
	query := `
		SELECT id, user_id, sender_email, sender_name,
		       email_count, last_received_at, created_at, updated_at
		FROM email_senders
		WHERE id = $1
	`

	var sender models.EmailSender
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&sender.ID,
		&sender.UserID,
		&sender.SenderEmail,
		&sender.SenderName,
		&sender.EmailCount,
		&sender.LastReceivedAt,
		&sender.CreatedAt,
		&sender.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sender by ID: %w", err)
	}

	return &sender, nil
}

// GetByUserAndEmail retrieves a sender by user ID and sender email
func (r *senderRepository) GetByUserAndEmail(ctx context.Context, userID uuid.UUID, senderEmail string) (*models.EmailSender, error) {
	query := `
		SELECT id, user_id, sender_email, sender_name,
		       email_count, last_received_at, created_at, updated_at
		FROM email_senders
		WHERE user_id = $1 AND sender_email = $2
	`

	var sender models.EmailSender
	err := r.db.QueryRowContext(ctx, query, userID, senderEmail).Scan(
		&sender.ID,
		&sender.UserID,
		&sender.SenderEmail,
		&sender.SenderName,
		&sender.EmailCount,
		&sender.LastReceivedAt,
		&sender.CreatedAt,
		&sender.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sender by user and email: %w", err)
	}

	return &sender, nil
}

// Upsert creates or updates a sender, incrementing email count
func (r *senderRepository) Upsert(ctx context.Context, sender *models.EmailSender) error {
	query := `
		INSERT INTO email_senders (
			id, user_id, sender_email, sender_name,
			email_count, last_received_at,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (user_id, sender_email)
		DO UPDATE SET
			sender_name = COALESCE(EXCLUDED.sender_name, email_senders.sender_name),
			email_count = email_senders.email_count + 1,
			last_received_at = EXCLUDED.last_received_at,
			updated_at = EXCLUDED.updated_at
		RETURNING id, email_count, created_at, updated_at
	`

	now := time.Now()
	if sender.ID == uuid.Nil {
		sender.ID = uuid.New()
	}
	if sender.CreatedAt.IsZero() {
		sender.CreatedAt = now
	}
	if sender.UpdatedAt.IsZero() {
		sender.UpdatedAt = now
	}

	err := r.db.QueryRowContext(ctx, query,
		sender.ID,
		sender.UserID,
		sender.SenderEmail,
		sender.SenderName,
		sender.EmailCount,
		sender.LastReceivedAt,
		sender.CreatedAt,
		sender.UpdatedAt,
	).Scan(&sender.ID, &sender.EmailCount, &sender.CreatedAt, &sender.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert sender: %w", err)
	}

	return nil
}

// ListByUser retrieves all senders for a user, ordered by last received
func (r *senderRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
	query := `
		SELECT id, user_id, sender_email, sender_name,
		       email_count, last_received_at, created_at, updated_at
		FROM email_senders
		WHERE user_id = $1
		ORDER BY last_received_at DESC NULLS LAST
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list senders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var senders []*models.EmailSender
	for rows.Next() {
		var sender models.EmailSender
		err := rows.Scan(
			&sender.ID,
			&sender.UserID,
			&sender.SenderEmail,
			&sender.SenderName,
			&sender.EmailCount,
			&sender.LastReceivedAt,
			&sender.CreatedAt,
			&sender.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan sender: %w", err)
		}
		senders = append(senders, &sender)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating sender rows: %w", err)
	}

	return senders, nil
}

// IncrementEmailCount increments the email count and updates last received time
func (r *senderRepository) IncrementEmailCount(ctx context.Context, id uuid.UUID, receivedAt time.Time) error {
	query := `
		UPDATE email_senders
		SET email_count = email_count + 1,
		    last_received_at = $1,
		    updated_at = $2
		WHERE id = $3
	`

	_, err := r.db.ExecContext(ctx, query, receivedAt, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to increment email count: %w", err)
	}

	return nil
}
