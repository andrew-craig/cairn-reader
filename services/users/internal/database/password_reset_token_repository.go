package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PasswordResetTokenRepository defines the interface for password reset token data operations.
type PasswordResetTokenRepository interface {
	// CreatePasswordResetToken stores a new password reset token.
	CreatePasswordResetToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) (*models.PasswordResetToken, error)

	// GetPasswordResetToken retrieves a token by hash if it is valid (not expired, not used).
	GetPasswordResetToken(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error)

	// MarkPasswordResetTokenUsed marks the token as consumed.
	MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error

	// DeleteExpiredPasswordResetTokens removes tokens that have passed their expiry.
	DeleteExpiredPasswordResetTokens(ctx context.Context) (int64, error)
}

type passwordResetTokenRepository struct {
	db *DB
}

// NewPasswordResetTokenRepository creates a new PasswordResetTokenRepository.
func NewPasswordResetTokenRepository(db *DB) PasswordResetTokenRepository {
	return &passwordResetTokenRepository{db: db}
}

// CreatePasswordResetToken stores a new password reset token.
func (r *passwordResetTokenRepository) CreatePasswordResetToken(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
) (*models.PasswordResetToken, error) {
	if userID == uuid.Nil || tokenHash == "" {
		return nil, apperrors.ErrInvalidTokenData
	}

	token := &models.PasswordResetToken{}

	query := `
		INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, user_id, token_hash, expires_at, used_at, created_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		uuid.New(),
		userID,
		tokenHash,
		expiresAt,
		time.Now().UTC(),
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create password reset token: %w", err)
	}

	return token, nil
}

// GetPasswordResetToken retrieves a token by hash.
// Returns ErrTokenNotFound if no matching token exists.
// The caller should check IsExpired() and IsUsed() to determine validity.
func (r *passwordResetTokenRepository) GetPasswordResetToken(ctx context.Context, tokenHash string) (*models.PasswordResetToken, error) {
	if tokenHash == "" {
		return nil, apperrors.ErrInvalidTokenData
	}

	token := &models.PasswordResetToken{}

	query := `
		SELECT id, user_id, token_hash, expires_at, used_at, created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}

	return token, nil
}

// MarkPasswordResetTokenUsed sets used_at on the token to prevent reuse.
func (r *passwordResetTokenRepository) MarkPasswordResetTokenUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `UPDATE password_reset_tokens SET used_at = $1 WHERE id = $2`

	result, err := r.db.Pool.Exec(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark password reset token used: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrTokenNotFound
	}

	return nil
}

// DeleteExpiredPasswordResetTokens removes all expired tokens and returns the count deleted.
func (r *passwordResetTokenRepository) DeleteExpiredPasswordResetTokens(ctx context.Context) (int64, error) {
	query := `DELETE FROM password_reset_tokens WHERE expires_at < $1`

	result, err := r.db.Pool.Exec(ctx, query, time.Now().UTC())
	if err != nil {
		return 0, fmt.Errorf("failed to delete expired password reset tokens: %w", err)
	}

	return result.RowsAffected(), nil
}
