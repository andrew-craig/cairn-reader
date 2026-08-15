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
	"github.com/jackc/pgx/v5/pgconn"
)

// RefreshTokenRepository defines the interface for refresh token data operations
type RefreshTokenRepository interface {
	// CreateRefreshToken creates a new refresh token
	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, deviceInfo, ipAddress *string, tokenFamily *uuid.UUID) (*models.RefreshToken, error)

	// GetRefreshTokenByHash retrieves a refresh token by its hash
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error)

	// UpdateLastUsedAt updates the last used timestamp for a refresh token
	UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error

	// RevokeToken marks a specific refresh token as revoked (retained until expiry)
	RevokeToken(ctx context.Context, id uuid.UUID) error

	// RevokeAllUserTokens marks all refresh tokens for a user as revoked (retained until expiry)
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error

	// RevokeTokenFamily marks all tokens in a token family as revoked (retained until expiry, for reuse detection)
	RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID) error

	// CleanupExpiredTokens removes expired tokens from the database
	CleanupExpiredTokens(ctx context.Context) (int64, error)
}

// refreshTokenRepository is the concrete implementation of RefreshTokenRepository
type refreshTokenRepository struct {
	db *DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *DB) RefreshTokenRepository {
	return &refreshTokenRepository{db: db}
}

// CreateRefreshToken creates a new refresh token
func (r *refreshTokenRepository) CreateRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	tokenHash string,
	expiresAt time.Time,
	deviceInfo, ipAddress *string,
	tokenFamily *uuid.UUID,
) (*models.RefreshToken, error) {
	if userID == uuid.Nil || tokenHash == "" {
		return nil, apperrors.ErrInvalidTokenData
	}

	token := &models.RefreshToken{
		ID:          uuid.New(),
		UserID:      userID,
		TokenHash:   tokenHash,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().UTC(),
		LastUsedAt:  time.Now().UTC(),
		DeviceInfo:  deviceInfo,
		IPAddress:   ipAddress,
		TokenFamily: tokenFamily,
	}

	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, created_at, last_used_at, device_info, ip_address, token_family)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, user_id, token_hash, expires_at, created_at, last_used_at, device_info, ip_address, token_family, revoked_at
	`

	err := r.db.Pool.QueryRow(
		ctx,
		query,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
		token.CreatedAt,
		token.LastUsedAt,
		token.DeviceInfo,
		token.IPAddress,
		token.TokenFamily,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.LastUsedAt,
		&token.DeviceInfo,
		&token.IPAddress,
		&token.TokenFamily,
		&token.RevokedAt,
	)

	if err != nil {
		// Check for unique constraint violation
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return nil, fmt.Errorf("duplicate token hash: %w", err)
		}
		// Check for foreign key violation
		if errors.As(err, &pgErr) && pgErr.Code == "23503" { // foreign_key_violation
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to create refresh token: %w", err)
	}

	return token, nil
}

// GetRefreshTokenByHash retrieves a refresh token by its hash
func (r *refreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	if tokenHash == "" {
		return nil, apperrors.ErrInvalidTokenData
	}

	token := &models.RefreshToken{}

	// Revoked tokens are intentionally included (not filtered out): a rotated
	// or logged-out token must still be found by hash so ValidateAndRotateToken
	// can recognize its replay as reuse instead of a generic "not found".
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at, last_used_at, device_info, ip_address, token_family, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1
	`

	err := r.db.Pool.QueryRow(ctx, query, tokenHash).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.CreatedAt,
		&token.LastUsedAt,
		&token.DeviceInfo,
		&token.IPAddress,
		&token.TokenFamily,
		&token.RevokedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.ErrTokenNotFound
		}
		return nil, fmt.Errorf("failed to get refresh token by hash: %w", err)
	}

	return token, nil
}

// UpdateLastUsedAt updates the last used timestamp for a refresh token
func (r *refreshTokenRepository) UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()

	query := `
		UPDATE refresh_tokens
		SET last_used_at = $1
		WHERE id = $2
	`

	result, err := r.db.Pool.Exec(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last used at: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrTokenNotFound
	}

	return nil
}

// RevokeToken marks a specific refresh token as revoked. The row is retained
// (not deleted) until it naturally expires, so a later replay of it can be
// recognized as reuse instead of returning a generic "not found".
func (r *refreshTokenRepository) RevokeToken(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`

	result, err := r.db.Pool.Exec(ctx, query, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrTokenNotFound
	}

	return nil
}

// RevokeAllUserTokens marks all of a user's refresh tokens as revoked. Rows
// are retained (not deleted), matching RevokeToken, so a replay of any of
// them is still recognized as reuse instead of a generic "not found".
func (r *refreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL`

	result, err := r.db.Pool.Exec(ctx, query, time.Now().UTC(), userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}

	// It's okay if no tokens were found (returns 0 rows affected)
	// User might not have any active tokens
	_ = result.RowsAffected()

	return nil
}

// RevokeTokenFamily marks all tokens in a token family as revoked (rows
// retained, not deleted, for the same reason as RevokeToken/RevokeAllUserTokens).
// This is used for refresh token reuse detection - if a token is reused,
// all tokens in its family should be revoked as a security measure
func (r *refreshTokenRepository) RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID) error {
	if tokenFamily == uuid.Nil {
		return apperrors.ErrInvalidTokenData
	}

	query := `UPDATE refresh_tokens SET revoked_at = $1 WHERE token_family = $2 AND revoked_at IS NULL`

	result, err := r.db.Pool.Exec(ctx, query, time.Now().UTC(), tokenFamily)
	if err != nil {
		return fmt.Errorf("failed to revoke token family: %w", err)
	}

	// It's okay if no tokens were found
	_ = result.RowsAffected()

	return nil
}

// CleanupExpiredTokens removes expired tokens from the database
// Returns the number of tokens deleted
func (r *refreshTokenRepository) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	now := time.Now().UTC()

	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`

	result, err := r.db.Pool.Exec(ctx, query, now)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}

	return result.RowsAffected(), nil
}
