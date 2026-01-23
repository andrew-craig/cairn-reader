package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
)

var (
	// ErrTokenReused indicates that a refresh token was reused, which is a potential security compromise
	ErrTokenReused = errors.New("refresh token reused - possible token theft detected")

	// ErrRefreshTokenNotFound indicates that the refresh token was not found
	ErrRefreshTokenNotFound = errors.New("refresh token not found")
)

const (
	// TokenLength is the length of the random token in bytes (256 bits)
	// This produces a base64 string of 43 characters
	TokenLength = 32

	// DefaultRefreshTokenExpiry is the default lifetime for refresh tokens (30 days)
	DefaultRefreshTokenExpiry = 30 * 24 * time.Hour

	// TokenReuseGracePeriod is the time window where token reuse is considered acceptable.
	// If a token is used again within this window after first use, it's allowed to
	// accommodate network latency and retry logic without triggering false positives.
	TokenReuseGracePeriod = 15 * time.Second
)

// RefreshTokenService handles refresh token operations including generation,
// validation, rotation, and reuse detection
type RefreshTokenService struct {
	repo          database.RefreshTokenRepository
	tokenLifetime time.Duration
}

// NewRefreshTokenService creates a new refresh token service
func NewRefreshTokenService(repo database.RefreshTokenRepository, tokenLifetime time.Duration) *RefreshTokenService {
	if tokenLifetime == 0 {
		tokenLifetime = DefaultRefreshTokenExpiry
	}

	return &RefreshTokenService{
		repo:          repo,
		tokenLifetime: tokenLifetime,
	}
}

// GenerateToken creates a cryptographically secure random token
// Returns the raw token (to send to client) and its SHA-256 hash (to store in DB)
func (s *RefreshTokenService) GenerateToken() (token string, hash string, err error) {
	// Generate cryptographically secure random bytes
	randomBytes := make([]byte, TokenLength)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate random token: %w", err)
	}

	// Encode to base64 for safe transmission
	token = base64.RawURLEncoding.EncodeToString(randomBytes)

	// Hash the token with SHA-256 for storage
	hash = hashToken(token)

	return token, hash, nil
}

// hashToken creates a SHA-256 hash of the token for secure storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// CreateRefreshToken generates and stores a new refresh token
func (s *RefreshTokenService) CreateRefreshToken(
	ctx context.Context,
	userID uuid.UUID,
	deviceInfo, ipAddress *string,
	tokenFamily *uuid.UUID,
) (token string, tokenModel *models.RefreshToken, err error) {
	// Generate new token
	token, hash, err := s.GenerateToken()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// If no token family provided, create a new one
	if tokenFamily == nil {
		family := uuid.New()
		tokenFamily = &family
	}

	// Calculate expiration time
	expiresAt := time.Now().UTC().Add(s.tokenLifetime)

	// Store token in database
	tokenModel, err = s.repo.CreateRefreshToken(
		ctx,
		userID,
		hash,
		expiresAt,
		deviceInfo,
		ipAddress,
		tokenFamily,
	)
	if err != nil {
		return "", nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return token, tokenModel, nil
}

// ValidateAndRotateToken validates a refresh token and performs rotation
// This implements the complete refresh flow including:
// - Token validation
// - Reuse detection
// - Token rotation
// - Automatic revocation on compromise
func (s *RefreshTokenService) ValidateAndRotateToken(
	ctx context.Context,
	token string,
	deviceInfo, ipAddress *string,
) (newToken string, userID uuid.UUID, err error) {
	// Hash the incoming token
	hash := hashToken(token)

	// Retrieve token from database
	tokenModel, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, apperrors.ErrTokenNotFound) {
			return "", uuid.Nil, ErrRefreshTokenNotFound
		}
		return "", uuid.Nil, fmt.Errorf("failed to retrieve token: %w", err)
	}

	// Check if token is expired
	if tokenModel.IsExpired() {
		// Clean up expired token
		_ = s.repo.RevokeToken(ctx, tokenModel.ID)
		return "", uuid.Nil, ErrTokenExpired
	}

	// Check for token reuse (compromise detection)
	if s.isTokenReused(tokenModel) {
		// SECURITY: Token reuse detected - possible token theft
		// Revoke all tokens in this family immediately
		if tokenModel.TokenFamily != nil {
			err := s.repo.RevokeTokenFamily(ctx, *tokenModel.TokenFamily)
			if err != nil {
				slog.Error("failed to revoke token family on reuse",
					slog.String("user_id", tokenModel.UserID.String()),
					slog.String("token_family", tokenModel.TokenFamily.String()),
					slog.Any("error", err),
				)
			}
		} else {
			// Fallback: revoke all user tokens if no family tracking
			err := s.repo.RevokeAllUserTokens(ctx, tokenModel.UserID)
			if err != nil {
				slog.Error("failed to revoke user tokens on reuse",
					slog.String("user_id", tokenModel.UserID.String()),
					slog.Any("error", err),
				)
			}
		}
		return "", uuid.Nil, ErrTokenReused
	}

	// Update last used timestamp to track usage
	err = s.repo.UpdateLastUsedAt(ctx, tokenModel.ID)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to update token usage: %w", err)
	}

	// Rotate: Create new token in the same family
	newToken, _, err = s.CreateRefreshToken(
		ctx,
		tokenModel.UserID,
		deviceInfo,
		ipAddress,
		tokenModel.TokenFamily,
	)
	if err != nil {
		return "", uuid.Nil, fmt.Errorf("failed to create new token: %w", err)
	}

	// Revoke the old token (completing the rotation)
	err = s.repo.RevokeToken(ctx, tokenModel.ID)
	if err != nil {
		// If we fail to revoke the old token, we should still succeed but log the error
		// The old token will be cleaned up eventually
		slog.Warn("failed to revoke old token during rotation",
			slog.String("user_id", tokenModel.UserID.String()),
			slog.String("token_id", tokenModel.ID.String()),
			slog.Any("error", err),
		)
	}

	return newToken, tokenModel.UserID, nil
}

// isTokenReused detects if a token has been used multiple times
// This is a key security feature to detect token theft
func (s *RefreshTokenService) isTokenReused(token *models.RefreshToken) bool {
	// If the token was last used very recently (within grace period),
	// it's likely being reused (either by an attacker or a race condition)
	timeSinceLastUse := token.TimeSinceLastUsed()

	// If this is the first use (LastUsedAt == CreatedAt), it's not reuse
	if token.LastUsedAt.Equal(token.CreatedAt) {
		return false
	}

	// If used again within the grace period, consider it reuse
	if timeSinceLastUse < TokenReuseGracePeriod {
		return true
	}

	return false
}

// RevokeToken revokes a single refresh token
func (s *RefreshTokenService) RevokeToken(ctx context.Context, token string) error {
	hash := hashToken(token)

	tokenModel, err := s.repo.GetRefreshTokenByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, apperrors.ErrTokenNotFound) {
			return ErrRefreshTokenNotFound
		}
		return fmt.Errorf("failed to retrieve token: %w", err)
	}

	err = s.repo.RevokeToken(ctx, tokenModel.ID)
	if err != nil {
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
// This is used for logout-all functionality
func (s *RefreshTokenService) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	err := s.repo.RevokeAllUserTokens(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke all user tokens: %w", err)
	}
	return nil
}

// RevokeTokenFamily revokes all tokens in a token family
// This is used when a compromise is detected
func (s *RefreshTokenService) RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID) error {
	err := s.repo.RevokeTokenFamily(ctx, tokenFamily)
	if err != nil {
		return fmt.Errorf("failed to revoke token family: %w", err)
	}
	return nil
}

// CleanupExpiredTokens removes expired tokens from the database
// This should be called periodically (e.g., via a cron job)
func (s *RefreshTokenService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	count, err := s.repo.CleanupExpiredTokens(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return count, nil
}
