package database

import (
	"context"
	"testing"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cleanupRefreshToken removes a test refresh token from the database
func cleanupRefreshToken(t *testing.T, db *DB, tokenID uuid.UUID) {
	_, err := db.Pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE id = $1", tokenID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test refresh token: %v", err)
	}
}

// cleanupRefreshTokensByUser removes all refresh tokens for a user
func cleanupRefreshTokensByUser(t *testing.T, db *DB, userID uuid.UUID) {
	_, err := db.Pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test refresh tokens: %v", err)
	}
}

func TestNewRefreshTokenRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewRefreshTokenRepository(db)
	assert.NotNil(t, repo)
}

func TestRefreshTokenRepository_CreateRefreshToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "tokentest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success - basic token", func(t *testing.T) {
		tokenHash := "test-token-hash-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		assert.NotNil(t, token)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.NotEqual(t, uuid.Nil, token.ID)
		assert.Equal(t, user.ID, token.UserID)
		assert.Equal(t, tokenHash, token.TokenHash)
		assert.Equal(t, expiresAt.Unix(), token.ExpiresAt.Unix())
		assert.False(t, token.CreatedAt.IsZero())
		assert.False(t, token.LastUsedAt.IsZero())
		assert.Nil(t, token.DeviceInfo)
		assert.Nil(t, token.IPAddress)
		assert.Nil(t, token.TokenFamily)
	})

	t.Run("success - with all fields", func(t *testing.T) {
		tokenHash := "test-token-hash-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
		deviceInfo := "iPhone 14 Pro"
		ipAddress := "192.168.1.1"
		tokenFamily := uuid.New()

		token, err := tokenRepo.CreateRefreshToken(
			ctx,
			user.ID,
			tokenHash,
			expiresAt,
			&deviceInfo,
			&ipAddress,
			&tokenFamily,
		)
		require.NoError(t, err)
		assert.NotNil(t, token)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.NotEqual(t, uuid.Nil, token.ID)
		assert.Equal(t, user.ID, token.UserID)
		assert.Equal(t, tokenHash, token.TokenHash)
		assert.Equal(t, expiresAt.Unix(), token.ExpiresAt.Unix())
		assert.NotNil(t, token.DeviceInfo)
		assert.Equal(t, deviceInfo, *token.DeviceInfo)
		assert.NotNil(t, token.IPAddress)
		assert.Equal(t, ipAddress, *token.IPAddress)
		assert.NotNil(t, token.TokenFamily)
		assert.Equal(t, tokenFamily, *token.TokenFamily)
	})

	t.Run("empty user ID", func(t *testing.T) {
		tokenHash := "test-token-hash"
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, uuid.Nil, tokenHash, expiresAt, nil, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.ErrorIs(t, err, apperrors.ErrInvalidTokenData)
	})

	t.Run("empty token hash", func(t *testing.T) {
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, "", expiresAt, nil, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.ErrorIs(t, err, apperrors.ErrInvalidTokenData)
	})

	t.Run("non-existent user", func(t *testing.T) {
		tokenHash := "test-token-hash-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
		nonExistentUserID := uuid.New()

		token, err := tokenRepo.CreateRefreshToken(ctx, nonExistentUserID, tokenHash, expiresAt, nil, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("duplicate token hash", func(t *testing.T) {
		tokenHash := "duplicate-hash-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token1, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token1.ID)

		// Try to create another token with the same hash
		token2, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, token2)
	})
}

func TestRefreshTokenRepository_GetRefreshTokenByHash(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "gettokentest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success", func(t *testing.T) {
		tokenHash := "find-me-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		created, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, created.ID)

		retrieved, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, created.UserID, retrieved.UserID)
		assert.Equal(t, tokenHash, retrieved.TokenHash)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentHash := "non-existent-hash"
		token, err := tokenRepo.GetRefreshTokenByHash(ctx, nonExistentHash)
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.ErrorIs(t, err, apperrors.ErrTokenNotFound)
	})

	t.Run("empty hash", func(t *testing.T) {
		token, err := tokenRepo.GetRefreshTokenByHash(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, token)
		assert.ErrorIs(t, err, apperrors.ErrInvalidTokenData)
	})
}

func TestRefreshTokenRepository_UpdateLastUsedAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "updatetest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success", func(t *testing.T) {
		tokenHash := "update-me-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		created, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, created.ID)

		originalLastUsed := created.LastUsedAt

		// Wait a bit to ensure time difference
		time.Sleep(10 * time.Millisecond)

		err = tokenRepo.UpdateLastUsedAt(ctx, created.ID)
		require.NoError(t, err)

		// Retrieve the token to verify update
		retrieved, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.True(t, retrieved.LastUsedAt.After(originalLastUsed))
	})

	t.Run("token not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := tokenRepo.UpdateLastUsedAt(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrTokenNotFound)
	})
}

func TestRefreshTokenRepository_RevokeToken(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "revoketest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success", func(t *testing.T) {
		tokenHash := "revoke-me-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		created, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)

		err = tokenRepo.RevokeToken(ctx, created.ID)
		require.NoError(t, err)

		// Verify token is deleted
		retrieved, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.ErrorIs(t, err, apperrors.ErrTokenNotFound)
	})

	t.Run("token not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := tokenRepo.RevokeToken(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrTokenNotFound)
	})
}

func TestRefreshTokenRepository_RevokeAllUserTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "revokealltest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success - multiple tokens", func(t *testing.T) {
		// Create multiple tokens for the user
		tokenHash1 := "revoke-all-1-" + uuid.New().String()
		tokenHash2 := "revoke-all-2-" + uuid.New().String()
		tokenHash3 := "revoke-all-3-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token1, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash1, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token1.ID)

		token2, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash2, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token2.ID)

		token3, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash3, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token3.ID)

		// Revoke all tokens
		err = tokenRepo.RevokeAllUserTokens(ctx, user.ID)
		require.NoError(t, err)

		// Verify all tokens are deleted
		retrieved1, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash1)
		assert.Error(t, err)
		assert.Nil(t, retrieved1)

		retrieved2, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash2)
		assert.Error(t, err)
		assert.Nil(t, retrieved2)

		retrieved3, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash3)
		assert.Error(t, err)
		assert.Nil(t, retrieved3)
	})

	t.Run("success - no tokens", func(t *testing.T) {
		// Create a new user with no tokens
		user2, err := userRepo.CreateUser(ctx, "notokens@example.com", "hash")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user2.ID)

		// Should succeed even with no tokens
		err = tokenRepo.RevokeAllUserTokens(ctx, user2.ID)
		require.NoError(t, err)
	})
}

func TestRefreshTokenRepository_RevokeTokenFamily(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "familytest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success - revoke family", func(t *testing.T) {
		family1 := uuid.New()
		family2 := uuid.New()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		// Create tokens in family 1
		tokenHash1 := "family1-token1-" + uuid.New().String()
		tokenHash2 := "family1-token2-" + uuid.New().String()

		token1, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash1, expiresAt, nil, nil, &family1)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token1.ID)

		token2, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash2, expiresAt, nil, nil, &family1)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token2.ID)

		// Create token in family 2 (should not be affected)
		tokenHash3 := "family2-token1-" + uuid.New().String()
		token3, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash3, expiresAt, nil, nil, &family2)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token3.ID)

		// Revoke family 1
		err = tokenRepo.RevokeTokenFamily(ctx, family1)
		require.NoError(t, err)

		// Verify family 1 tokens are deleted
		retrieved1, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash1)
		assert.Error(t, err)
		assert.Nil(t, retrieved1)

		retrieved2, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash2)
		assert.Error(t, err)
		assert.Nil(t, retrieved2)

		// Verify family 2 token still exists
		retrieved3, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash3)
		require.NoError(t, err)
		assert.NotNil(t, retrieved3)
	})

	t.Run("empty family ID", func(t *testing.T) {
		err := tokenRepo.RevokeTokenFamily(ctx, uuid.Nil)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrInvalidTokenData)
	})

	t.Run("success - non-existent family", func(t *testing.T) {
		nonExistentFamily := uuid.New()
		// Should succeed even if no tokens with this family exist
		err := tokenRepo.RevokeTokenFamily(ctx, nonExistentFamily)
		require.NoError(t, err)
	})
}

func TestRefreshTokenRepository_CleanupExpiredTokens(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "cleanuptest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("success - cleanup expired", func(t *testing.T) {
		// Create expired tokens
		expiredTime := time.Now().UTC().Add(-1 * time.Hour)
		tokenHash1 := "expired-1-" + uuid.New().String()
		tokenHash2 := "expired-2-" + uuid.New().String()

		token1, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash1, expiredTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token1.ID)

		token2, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash2, expiredTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token2.ID)

		// Create a valid token (should not be deleted)
		validTime := time.Now().UTC().Add(30 * 24 * time.Hour)
		tokenHash3 := "valid-token-" + uuid.New().String()
		token3, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash3, validTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token3.ID)

		// Cleanup expired tokens
		count, err := tokenRepo.CleanupExpiredTokens(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, int64(2)) // At least our 2 expired tokens

		// Verify expired tokens are deleted
		retrieved1, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash1)
		assert.Error(t, err)
		assert.Nil(t, retrieved1)

		retrieved2, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash2)
		assert.Error(t, err)
		assert.Nil(t, retrieved2)

		// Verify valid token still exists
		retrieved3, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash3)
		require.NoError(t, err)
		assert.NotNil(t, retrieved3)
	})

	t.Run("success - no expired tokens", func(t *testing.T) {
		// Clean up any existing tokens first
		cleanupRefreshTokensByUser(t, db, user.ID)

		// Create only valid tokens
		validTime := time.Now().UTC().Add(30 * 24 * time.Hour)
		tokenHash := "valid-only-" + uuid.New().String()
		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, validTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		// Cleanup should find 0 expired tokens
		count, err := tokenRepo.CleanupExpiredTokens(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)

		// Verify valid token still exists
		retrieved, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
	})
}

// Test token expiration detection using model methods
func TestRefreshTokenRepository_TokenExpiration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "expirationtest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("expired token", func(t *testing.T) {
		expiredTime := time.Now().UTC().Add(-1 * time.Hour)
		tokenHash := "expired-check-" + uuid.New().String()

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiredTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.True(t, token.IsExpired())
		assert.False(t, token.IsValid())
		assert.Equal(t, time.Duration(0), token.TimeUntilExpiry())
	})

	t.Run("valid token", func(t *testing.T) {
		validTime := time.Now().UTC().Add(30 * 24 * time.Hour)
		tokenHash := "valid-check-" + uuid.New().String()

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, validTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.False(t, token.IsExpired())
		assert.True(t, token.IsValid())
		assert.Greater(t, token.TimeUntilExpiry(), time.Duration(0))
	})

	t.Run("should rotate", func(t *testing.T) {
		validTime := time.Now().UTC().Add(30 * 24 * time.Hour)
		tokenHash := "rotate-check-" + uuid.New().String()

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, validTime, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		// Per requirements, tokens should always rotate on use
		assert.True(t, token.ShouldRotate())
	})
}

// Test concurrent token operations
func TestRefreshTokenRepository_ConcurrentOperations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "concurrenttest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("create multiple tokens concurrently", func(t *testing.T) {
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
		done := make(chan bool, 5)

		for i := 0; i < 5; i++ {
			go func(index int) {
				tokenHash := uuid.New().String()
				token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
				assert.NoError(t, err)
				if token != nil {
					defer cleanupRefreshToken(t, db, token.ID)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < 5; i++ {
			<-done
		}
	})
}

// Test additional edge cases for comprehensive coverage
func TestRefreshTokenRepository_EdgeCases(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	userRepo := NewUserRepository(db)
	tokenRepo := NewRefreshTokenRepository(db)
	ctx := context.Background()

	// Create a test user
	user, err := userRepo.CreateUser(ctx, "edgetest@example.com", "hash")
	require.NoError(t, err)
	defer cleanupTestUser(t, db, user.ID)

	t.Run("token with far future expiration", func(t *testing.T) {
		// Test with a very far future date (e.g., 10 years)
		farFuture := time.Now().UTC().Add(10 * 365 * 24 * time.Hour)
		tokenHash := "far-future-" + uuid.New().String()

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, farFuture, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.False(t, token.IsExpired())
		assert.True(t, token.IsValid())
		assert.Greater(t, token.TimeUntilExpiry(), 9*365*24*time.Hour)
	})

	t.Run("token with past expiration", func(t *testing.T) {
		// Test with a date in the past
		pastDate := time.Now().UTC().Add(-365 * 24 * time.Hour)
		tokenHash := "past-date-" + uuid.New().String()

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, pastDate, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.True(t, token.IsExpired())
		assert.False(t, token.IsValid())
	})

	t.Run("token with very long device info", func(t *testing.T) {
		// Test with long device info string
		longDeviceInfo := "iPhone 15 Pro Max (Model A2849) running iOS 17.5.1 with 256GB storage and specific device identifier: " + uuid.New().String()
		ipAddress := "192.168.100.255"
		tokenHash := "long-device-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, &longDeviceInfo, &ipAddress, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.NotNil(t, token.DeviceInfo)
		assert.Equal(t, longDeviceInfo, *token.DeviceInfo)
	})

	t.Run("token with IPv6 address", func(t *testing.T) {
		// Test with IPv6 address
		deviceInfo := "Android Device"
		ipv6Address := "2001:0db8:85a3:0000:0000:8a2e:0370:7334"
		tokenHash := "ipv6-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, &deviceInfo, &ipv6Address, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		assert.NotNil(t, token.IPAddress)
		assert.Equal(t, ipv6Address, *token.IPAddress)
	})

	t.Run("multiple cleanup operations", func(t *testing.T) {
		// Create some expired tokens
		expiredTime := time.Now().UTC().Add(-1 * time.Hour)
		for i := 0; i < 3; i++ {
			tokenHash := uuid.New().String()
			token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiredTime, nil, nil, nil)
			require.NoError(t, err)
			defer cleanupRefreshToken(t, db, token.ID)
		}

		// First cleanup
		count1, err := tokenRepo.CleanupExpiredTokens(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count1, int64(3))

		// Second cleanup (should find 0 or fewer tokens)
		count2, err := tokenRepo.CleanupExpiredTokens(ctx)
		require.NoError(t, err)
		assert.LessOrEqual(t, count2, count1)
	})

	t.Run("revoke already revoked token", func(t *testing.T) {
		tokenHash := "revoke-twice-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)

		// First revoke
		err = tokenRepo.RevokeToken(ctx, token.ID)
		require.NoError(t, err)

		// Second revoke should fail with apperrors.ErrTokenNotFound
		err = tokenRepo.RevokeToken(ctx, token.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrTokenNotFound)
	})

	t.Run("update last used on already used token", func(t *testing.T) {
		tokenHash := "update-multiple-" + uuid.New().String()
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)

		token, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash, expiresAt, nil, nil, nil)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token.ID)

		originalTime := token.LastUsedAt
		time.Sleep(10 * time.Millisecond)

		// First update
		err = tokenRepo.UpdateLastUsedAt(ctx, token.ID)
		require.NoError(t, err)

		token1, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.True(t, token1.LastUsedAt.After(originalTime))

		time.Sleep(10 * time.Millisecond)

		// Second update
		err = tokenRepo.UpdateLastUsedAt(ctx, token.ID)
		require.NoError(t, err)

		token2, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash)
		require.NoError(t, err)
		assert.True(t, token2.LastUsedAt.After(token1.LastUsedAt))
	})

	t.Run("token family with mixed expiration states", func(t *testing.T) {
		family := uuid.New()
		expiresAtValid := time.Now().UTC().Add(30 * 24 * time.Hour)
		expiresAtExpired := time.Now().UTC().Add(-1 * time.Hour)

		// Create one valid token in family
		tokenHash1 := "family-valid-" + uuid.New().String()
		token1, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash1, expiresAtValid, nil, nil, &family)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token1.ID)

		// Create one expired token in same family
		tokenHash2 := "family-expired-" + uuid.New().String()
		token2, err := tokenRepo.CreateRefreshToken(ctx, user.ID, tokenHash2, expiresAtExpired, nil, nil, &family)
		require.NoError(t, err)
		defer cleanupRefreshToken(t, db, token2.ID)

		// Revoke entire family (should remove both valid and expired)
		err = tokenRepo.RevokeTokenFamily(ctx, family)
		require.NoError(t, err)

		// Verify both are gone
		retrieved1, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash1)
		assert.Error(t, err)
		assert.Nil(t, retrieved1)

		retrieved2, err := tokenRepo.GetRefreshTokenByHash(ctx, tokenHash2)
		assert.Error(t, err)
		assert.Nil(t, retrieved2)
	})
}
