package auth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"testing"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRefreshTokenRepository is a mock implementation of RefreshTokenRepository
type MockRefreshTokenRepository struct {
	mock.Mock
}

func (m *MockRefreshTokenRepository) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time, deviceInfo, ipAddress *string, tokenFamily *uuid.UUID) (*models.RefreshToken, error) {
	args := m.Called(ctx, userID, tokenHash, expiresAt, deviceInfo, ipAddress, tokenFamily)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	args := m.Called(ctx, tokenHash)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockRefreshTokenRepository) UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) RevokeToken(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) RevokeTokenFamily(ctx context.Context, tokenFamily uuid.UUID) error {
	args := m.Called(ctx, tokenFamily)
	return args.Error(0)
}

func (m *MockRefreshTokenRepository) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func TestRefreshTokenGeneration(t *testing.T) {
	repo := new(MockRefreshTokenRepository)
	service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

	t.Run("generates unique tokens", func(t *testing.T) {
		token1, hash1, err1 := service.GenerateToken()
		token2, hash2, err2 := service.GenerateToken()

		assert.NoError(t, err1)
		assert.NoError(t, err2)
		assert.NotEmpty(t, token1)
		assert.NotEmpty(t, token2)
		assert.NotEmpty(t, hash1)
		assert.NotEmpty(t, hash2)

		// Tokens should be unique
		assert.NotEqual(t, token1, token2)
		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("token is proper length", func(t *testing.T) {
		token, hash, err := service.GenerateToken()

		assert.NoError(t, err)
		// Base64 encoding of 32 bytes produces 43 characters
		assert.Equal(t, 43, len(token))
		assert.Equal(t, 43, len(hash))
	})

	t.Run("hash matches token", func(t *testing.T) {
		token, hash, err := service.GenerateToken()

		assert.NoError(t, err)

		// Verify hash
		expectedHash := sha256.Sum256([]byte(token))
		expectedHashStr := base64.RawURLEncoding.EncodeToString(expectedHash[:])
		assert.Equal(t, expectedHashStr, hash)
	})
}

func TestHashToken(t *testing.T) {
	t.Run("produces consistent hash", func(t *testing.T) {
		token := "test-token-123"
		hash1 := hashToken(token)
		hash2 := hashToken(token)

		assert.Equal(t, hash1, hash2)
	})

	t.Run("different tokens produce different hashes", func(t *testing.T) {
		hash1 := hashToken("token1")
		hash2 := hashToken("token2")

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("produces SHA-256 hash", func(t *testing.T) {
		token := "test-token"
		hash := hashToken(token)

		// SHA-256 produces 32 bytes, which encodes to 43 characters in base64
		assert.Equal(t, 43, len(hash))
	})
}

func TestCreateRefreshToken(t *testing.T) {
	repo := new(MockRefreshTokenRepository)
	service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)
	ctx := context.Background()

	userID := uuid.New()
	deviceInfo := "iOS 17.0"
	ipAddress := "192.168.1.1"

	t.Run("creates token successfully", func(t *testing.T) {
		tokenFamily := uuid.New()
		expectedToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   "some-hash",
			ExpiresAt:   time.Now().Add(DefaultRefreshTokenExpiry),
			CreatedAt:   time.Now(),
			LastUsedAt:  time.Now(),
			DeviceInfo:  &deviceInfo,
			IPAddress:   &ipAddress,
			TokenFamily: &tokenFamily,
		}

		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, mock.AnythingOfType("*uuid.UUID")).
			Return(expectedToken, nil)

		token, tokenModel, err := service.CreateRefreshToken(ctx, userID, &deviceInfo, &ipAddress, &tokenFamily)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, tokenModel)
		assert.Equal(t, userID, tokenModel.UserID)
		repo.AssertExpectations(t)
	})

	t.Run("creates new token family if not provided", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, mock.AnythingOfType("*uuid.UUID")).
			Return(&models.RefreshToken{
				ID:     uuid.New(),
				UserID: userID,
			}, nil)

		_, _, err := service.CreateRefreshToken(ctx, userID, &deviceInfo, &ipAddress, nil)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestValidateAndRotateToken(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	tokenFamily := uuid.New()
	deviceInfo := "iOS 17.0"
	ipAddress := "192.168.1.1"

	t.Run("successfully validates and rotates token", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		// Generate a token to work with
		token, hash, _ := service.GenerateToken()

		// Mock the existing token in database
		existingToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour), // Last used 1 hour ago
			TokenFamily: &tokenFamily,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)
		repo.On("UpdateLastUsedAt", ctx, existingToken.ID).Return(nil)
		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, &tokenFamily).
			Return(&models.RefreshToken{ID: uuid.New(), UserID: userID}, nil)
		repo.On("RevokeToken", ctx, existingToken.ID).Return(nil)

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, token, newToken)
		assert.Equal(t, userID, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("rejects expired token", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		expiredToken := &models.RefreshToken{
			ID:         uuid.New(),
			UserID:     userID,
			TokenHash:  hash,
			ExpiresAt:  time.Now().UTC().Add(-1 * time.Hour), // Expired 1 hour ago
			CreatedAt:  time.Now().UTC().Add(-25 * time.Hour),
			LastUsedAt: time.Now().UTC().Add(-25 * time.Hour),
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(expiredToken, nil)
		repo.On("RevokeToken", ctx, expiredToken.ID).Return(nil)

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenExpired, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("detects token reuse and revokes family", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		// Token already revoked (rotated away) well before this replay - the
		// real threat this guards against, not a same-request race.
		revokedAt := time.Now().UTC().Add(-1 * time.Hour)
		reusedToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour),
			TokenFamily: &tokenFamily,
			RevokedAt:   &revokedAt,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(reusedToken, nil)
		repo.On("RevokeTokenFamily", ctx, tokenFamily).Return(nil)

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenReused, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("revokes all user tokens when no family tracking", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		// Token without family tracking, already revoked well before this replay
		revokedAt := time.Now().UTC().Add(-1 * time.Hour)
		reusedToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour),
			TokenFamily: nil, // No family tracking
			RevokedAt:   &revokedAt,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(reusedToken, nil)
		repo.On("RevokeAllUserTokens", ctx, userID).Return(nil)

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenReused, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("returns error for non-existent token", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(nil, apperrors.ErrTokenNotFound)

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Equal(t, ErrRefreshTokenNotFound, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})
}

func TestIsTokenReused(t *testing.T) {
	repo := new(MockRefreshTokenRepository)
	service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

	t.Run("active token is not reuse", func(t *testing.T) {
		token := &models.RefreshToken{RevokedAt: nil}

		assert.False(t, service.isTokenReused(token))
	})

	t.Run("revoked token is reuse", func(t *testing.T) {
		revokedAt := time.Now().UTC().Add(-2 * time.Second)
		token := &models.RefreshToken{RevokedAt: &revokedAt}

		assert.True(t, service.isTokenReused(token))
	})
}

func TestRevokeToken(t *testing.T) {
	ctx := context.Background()

	t.Run("revokes token successfully", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()
		tokenID := uuid.New()

		existingToken := &models.RefreshToken{
			ID:        tokenID,
			TokenHash: hash,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)
		repo.On("RevokeToken", ctx, tokenID).Return(nil)

		err := service.RevokeToken(ctx, token)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("returns error for non-existent token", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(nil, apperrors.ErrTokenNotFound)

		err := service.RevokeToken(ctx, token)

		assert.Error(t, err)
		assert.Equal(t, ErrRefreshTokenNotFound, err)
		repo.AssertExpectations(t)
	})
}

func TestRevokeAllUserTokens(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("revokes all user tokens successfully", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("RevokeAllUserTokens", ctx, userID).Return(nil)

		err := service.RevokeAllUserTokens(ctx, userID)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRevokeTokenFamily(t *testing.T) {
	ctx := context.Background()
	tokenFamily := uuid.New()

	t.Run("revokes token family successfully", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("RevokeTokenFamily", ctx, tokenFamily).Return(nil)

		err := service.RevokeTokenFamily(ctx, tokenFamily)

		assert.NoError(t, err)
		repo.AssertExpectations(t)
	})
}

func TestCleanupExpiredTokens(t *testing.T) {
	ctx := context.Background()

	t.Run("cleans up expired tokens successfully", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("CleanupExpiredTokens", ctx).Return(int64(5), nil)

		count, err := service.CleanupExpiredTokens(ctx)

		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
		repo.AssertExpectations(t)
	})
}

func TestNewRefreshTokenService(t *testing.T) {
	t.Run("uses default lifetime when zero provided", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, 0)

		assert.Equal(t, DefaultRefreshTokenExpiry, service.tokenLifetime)
	})

	t.Run("uses provided lifetime", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		customLifetime := 7 * 24 * time.Hour
		service := NewRefreshTokenService(repo, customLifetime)

		assert.Equal(t, customLifetime, service.tokenLifetime)
	})
}

func TestCreateRefreshToken_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	deviceInfo := "iOS 17.0"
	ipAddress := "192.168.1.1"

	t.Run("handles database error", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, mock.AnythingOfType("*uuid.UUID")).
			Return(nil, apperrors.ErrUserNotFound)

		token, tokenModel, err := service.CreateRefreshToken(ctx, userID, &deviceInfo, &ipAddress, nil)

		assert.Error(t, err)
		assert.Empty(t, token)
		assert.Nil(t, tokenModel)
		repo.AssertExpectations(t)
	})

	t.Run("creates token with nil device info and ip", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), (*string)(nil), (*string)(nil), mock.AnythingOfType("*uuid.UUID")).
			Return(&models.RefreshToken{
				ID:     uuid.New(),
				UserID: userID,
			}, nil)

		token, tokenModel, err := service.CreateRefreshToken(ctx, userID, nil, nil, nil)

		assert.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.NotNil(t, tokenModel)
		repo.AssertExpectations(t)
	})
}

func TestValidateAndRotateToken_EdgeCases(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()
	tokenFamily := uuid.New()
	deviceInfo := "iOS 17.0"
	ipAddress := "192.168.1.1"

	t.Run("handles database error during token retrieval", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(nil, fmt.Errorf("database connection error"))

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("handles error during UpdateLastUsedAt", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		existingToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour),
			TokenFamily: &tokenFamily,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)
		repo.On("UpdateLastUsedAt", ctx, existingToken.ID).Return(fmt.Errorf("update failed"))

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("handles error during new token creation", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		existingToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour),
			TokenFamily: &tokenFamily,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)
		repo.On("UpdateLastUsedAt", ctx, existingToken.ID).Return(nil)
		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, &tokenFamily).
			Return(nil, fmt.Errorf("creation failed"))

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		assert.Error(t, err)
		assert.Empty(t, newToken)
		assert.Equal(t, uuid.Nil, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("succeeds even if old token revocation fails", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		existingToken := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour),
			TokenFamily: &tokenFamily,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)
		repo.On("UpdateLastUsedAt", ctx, existingToken.ID).Return(nil)
		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, &tokenFamily).
			Return(&models.RefreshToken{ID: uuid.New(), UserID: userID}, nil)
		repo.On("RevokeToken", ctx, existingToken.ID).Return(fmt.Errorf("revocation failed"))

		newToken, returnedUserID, err := service.ValidateAndRotateToken(ctx, token, &deviceInfo, &ipAddress)

		// Should still succeed even though revocation failed
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.Equal(t, userID, returnedUserID)
		repo.AssertExpectations(t)
	})

	t.Run("multiple rotation cycles work correctly", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		// First rotation
		token1, hash1, _ := service.GenerateToken()
		existingToken1 := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash1,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			LastUsedAt:  time.Now().UTC().Add(-1 * time.Hour),
			TokenFamily: &tokenFamily,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash1).Return(existingToken1, nil)
		repo.On("UpdateLastUsedAt", ctx, existingToken1.ID).Return(nil)
		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, &tokenFamily).
			Return(&models.RefreshToken{ID: uuid.New(), UserID: userID}, nil).Once()
		repo.On("RevokeToken", ctx, existingToken1.ID).Return(nil)

		newToken1, _, err := service.ValidateAndRotateToken(ctx, token1, &deviceInfo, &ipAddress)
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken1)
		assert.NotEqual(t, token1, newToken1)

		// Second rotation
		hash2 := hashToken(newToken1)
		existingToken2 := &models.RefreshToken{
			ID:          uuid.New(),
			UserID:      userID,
			TokenHash:   hash2,
			ExpiresAt:   time.Now().UTC().Add(24 * time.Hour),
			CreatedAt:   time.Now().UTC().Add(-30 * time.Minute),
			LastUsedAt:  time.Now().UTC().Add(-30 * time.Minute),
			TokenFamily: &tokenFamily,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash2).Return(existingToken2, nil)
		repo.On("UpdateLastUsedAt", ctx, existingToken2.ID).Return(nil)
		repo.On("CreateRefreshToken", ctx, userID, mock.AnythingOfType("string"), mock.AnythingOfType("time.Time"), &deviceInfo, &ipAddress, &tokenFamily).
			Return(&models.RefreshToken{ID: uuid.New(), UserID: userID}, nil).Once()
		repo.On("RevokeToken", ctx, existingToken2.ID).Return(nil)

		newToken2, _, err := service.ValidateAndRotateToken(ctx, newToken1, &deviceInfo, &ipAddress)
		assert.NoError(t, err)
		assert.NotEmpty(t, newToken2)
		assert.NotEqual(t, newToken1, newToken2)

		repo.AssertExpectations(t)
	})
}

func TestIsTokenReused_EdgeCases(t *testing.T) {
	repo := new(MockRefreshTokenRepository)
	service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

	t.Run("revoked moments ago is still reuse", func(t *testing.T) {
		revokedAt := time.Now().UTC().Add(-1 * time.Millisecond)
		token := &models.RefreshToken{RevokedAt: &revokedAt}

		assert.True(t, service.isTokenReused(token))
	})

	// This is the C3 scenario: an attacker replays a stolen token well after
	// the victim legitimately rotated it. There is no grace-period cutoff -
	// a revoked token is reuse for as long as its row is retained (up to its
	// original expiry).
	t.Run("revoked long ago is still reuse", func(t *testing.T) {
		revokedAt := time.Now().UTC().Add(-10 * 24 * time.Hour)
		token := &models.RefreshToken{RevokedAt: &revokedAt}

		assert.True(t, service.isTokenReused(token))
	})
}

func TestRevokeToken_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("already-revoked token is treated as not found", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()
		revokedAt := time.Now().UTC().Add(-1 * time.Minute)
		existingToken := &models.RefreshToken{
			ID:        uuid.New(),
			TokenHash: hash,
			RevokedAt: &revokedAt,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)

		err := service.RevokeToken(ctx, token)

		assert.Equal(t, ErrRefreshTokenNotFound, err)
		repo.AssertExpectations(t)
	})

	t.Run("handles database error during retrieval", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(nil, fmt.Errorf("database error"))

		err := service.RevokeToken(ctx, token)

		assert.Error(t, err)
		assert.NotEqual(t, ErrRefreshTokenNotFound, err)
		repo.AssertExpectations(t)
	})

	t.Run("handles database error during revocation", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		token, hash, _ := service.GenerateToken()
		tokenID := uuid.New()

		existingToken := &models.RefreshToken{
			ID:        tokenID,
			TokenHash: hash,
		}

		repo.On("GetRefreshTokenByHash", ctx, hash).Return(existingToken, nil)
		repo.On("RevokeToken", ctx, tokenID).Return(fmt.Errorf("revocation failed"))

		err := service.RevokeToken(ctx, token)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRevokeAllUserTokens_EdgeCases(t *testing.T) {
	ctx := context.Background()
	userID := uuid.New()

	t.Run("handles database error", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("RevokeAllUserTokens", ctx, userID).Return(fmt.Errorf("database error"))

		err := service.RevokeAllUserTokens(ctx, userID)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestRevokeTokenFamily_EdgeCases(t *testing.T) {
	ctx := context.Background()
	tokenFamily := uuid.New()

	t.Run("handles database error", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("RevokeTokenFamily", ctx, tokenFamily).Return(fmt.Errorf("database error"))

		err := service.RevokeTokenFamily(ctx, tokenFamily)

		assert.Error(t, err)
		repo.AssertExpectations(t)
	})
}

func TestCleanupExpiredTokens_EdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("handles database error", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("CleanupExpiredTokens", ctx).Return(int64(0), fmt.Errorf("cleanup failed"))

		count, err := service.CleanupExpiredTokens(ctx)

		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		repo.AssertExpectations(t)
	})

	t.Run("returns zero count when no expired tokens", func(t *testing.T) {
		repo := new(MockRefreshTokenRepository)
		service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

		repo.On("CleanupExpiredTokens", ctx).Return(int64(0), nil)

		count, err := service.CleanupExpiredTokens(ctx)

		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		repo.AssertExpectations(t)
	})
}

// Benchmark tests for performance verification
func BenchmarkRefreshTokenGenerate(b *testing.B) {
	repo := new(MockRefreshTokenRepository)
	service := NewRefreshTokenService(repo, DefaultRefreshTokenExpiry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := service.GenerateToken()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRefreshTokenHash(b *testing.B) {
	token := "test-token-for-benchmarking-purposes-123456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hashToken(token)
	}
}
