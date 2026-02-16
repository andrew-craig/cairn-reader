package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
)

// AddressRepository defines the interface for email address data operations
type AddressRepository interface {
	// Create creates a new email address for a user
	Create(ctx context.Context, address *models.EmailAddress) error

	// GetByUserID retrieves the email address for a user
	GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error)

	// GetByLocalPart retrieves an email address by its local part
	GetByLocalPart(ctx context.Context, localPart string) (*models.EmailAddress, error)

	// Delete deletes a user's email address
	Delete(ctx context.Context, userID uuid.UUID) error
}

// addressRepository is the concrete implementation of AddressRepository
type addressRepository struct {
	db *sql.DB
}

// NewAddressRepository creates a new AddressRepository
func NewAddressRepository(db *sql.DB) AddressRepository {
	return &addressRepository{db: db}
}

// Create creates a new email address for a user
func (r *addressRepository) Create(ctx context.Context, address *models.EmailAddress) error {
	query := `
		INSERT INTO email_addresses (id, user_id, local_part, created_at)
		VALUES ($1, $2, $3, $4)
	`

	if address.ID == uuid.Nil {
		address.ID = uuid.New()
	}
	if address.CreatedAt.IsZero() {
		address.CreatedAt = time.Now()
	}

	_, err := r.db.ExecContext(ctx, query, address.ID, address.UserID, address.LocalPart, address.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create email address: %w", err)
	}

	return nil
}

// GetByUserID retrieves the email address for a user
func (r *addressRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*models.EmailAddress, error) {
	query := `
		SELECT id, user_id, local_part, created_at
		FROM email_addresses
		WHERE user_id = $1
	`

	var address models.EmailAddress
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&address.ID,
		&address.UserID,
		&address.LocalPart,
		&address.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email address by user ID: %w", err)
	}

	return &address, nil
}

// GetByLocalPart retrieves an email address by its local part
func (r *addressRepository) GetByLocalPart(ctx context.Context, localPart string) (*models.EmailAddress, error) {
	query := `
		SELECT id, user_id, local_part, created_at
		FROM email_addresses
		WHERE local_part = $1
	`

	var address models.EmailAddress
	err := r.db.QueryRowContext(ctx, query, localPart).Scan(
		&address.ID,
		&address.UserID,
		&address.LocalPart,
		&address.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email address by local part: %w", err)
	}

	return &address, nil
}

// Delete deletes a user's email address
func (r *addressRepository) Delete(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM email_addresses WHERE user_id = $1`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete email address: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return nil // Address didn't exist, not an error
	}

	return nil
}
