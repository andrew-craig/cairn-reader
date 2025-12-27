package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB creates a test database connection
// This assumes you have a test database configured
func setupTestDB(t *testing.T) *DB {
	// Note: In a real test environment, you would:
	// 1. Use a test database
	// 2. Run migrations
	// 3. Clean up after tests
	// For now, this is a placeholder that requires proper setup

	cfg := &Config{
		Host:            getEnvOrDefault("TEST_DB_HOST", "localhost"),
		Port:            getEnvOrDefault("TEST_DB_PORT", "5432"),
		User:            getEnvOrDefault("TEST_DB_USER", "postgres"),
		Password:        getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		Database:        getEnvOrDefault("TEST_DB_NAME", "cairn_test"),
		SSLMode:         getEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	db, err := New(cfg)
	if err != nil {
		t.Skip("Database not available for testing:", err)
	}

	return db
}

func getEnvOrDefault(key, defaultValue string) string {
	// In real implementation, use os.Getenv
	return defaultValue
}

// cleanupTestUser removes a test user from the database
func cleanupTestUser(t *testing.T, db *DB, userID uuid.UUID) {
	_, err := db.Pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test user: %v", err)
	}
}

func TestNewUserRepository(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	assert.NotNil(t, repo)
}

func TestUserRepository_CreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "test@example.com"
		passwordHash := "hashed_password_123"

		user, err := repo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		assert.NotNil(t, user)
		defer cleanupTestUser(t, db, user.ID)

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.NotNil(t, user.Email)
		assert.Equal(t, email, *user.Email)
		assert.NotNil(t, user.PasswordHash)
		assert.Equal(t, passwordHash, *user.PasswordHash)
		assert.Nil(t, user.ExpoDeviceID)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
		assert.Nil(t, user.LastLoginAt)

		// Verify account type
		assert.True(t, user.IsEmailOnly())
		assert.False(t, user.IsMobileOnly())
		assert.False(t, user.IsHybrid())
	})

	t.Run("duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"
		passwordHash := "hashed_password_123"

		user1, err := repo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user1.ID)

		user2, err := repo.CreateUser(ctx, email, "different_hash")
		assert.Error(t, err)
		assert.Nil(t, user2)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})

	t.Run("empty email", func(t *testing.T) {
		user, err := repo.CreateUser(ctx, "", "hashed_password")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})

	t.Run("empty password hash", func(t *testing.T) {
		user, err := repo.CreateUser(ctx, "test@example.com", "")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})
}

func TestUserRepository_CreateMobileUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()

		user, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		assert.NotNil(t, user)
		defer cleanupTestUser(t, db, user.ID)

		assert.NotEqual(t, uuid.Nil, user.ID)
		assert.Nil(t, user.Email)
		assert.Nil(t, user.PasswordHash)
		assert.NotNil(t, user.ExpoDeviceID)
		assert.Equal(t, deviceID, *user.ExpoDeviceID)
		assert.False(t, user.CreatedAt.IsZero())
		assert.False(t, user.UpdatedAt.IsZero())
		assert.Nil(t, user.LastLoginAt)

		// Verify account type
		assert.True(t, user.IsMobileOnly())
		assert.False(t, user.IsEmailOnly())
		assert.False(t, user.IsHybrid())
	})

	t.Run("duplicate device ID", func(t *testing.T) {
		deviceID := "duplicate-device-" + uuid.New().String()

		user1, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user1.ID)

		user2, err := repo.CreateMobileUser(ctx, deviceID)
		assert.Error(t, err)
		assert.Nil(t, user2)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})

	t.Run("empty device ID", func(t *testing.T) {
		user, err := repo.CreateMobileUser(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})
}

func TestUserRepository_GetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success - email user", func(t *testing.T) {
		email := "getuserbyid@example.com"
		created, err := repo.CreateUser(ctx, email, "hashed_password")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		retrieved, err := repo.GetUserByID(ctx, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, email, *retrieved.Email)
	})

	t.Run("success - mobile user", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		created, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		retrieved, err := repo.GetUserByID(ctx, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, deviceID, *retrieved.ExpoDeviceID)
	})

	t.Run("not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		user, err := repo.GetUserByID(ctx, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUserRepository_GetUserByEmail(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "getbyemail@example.com"
		created, err := repo.CreateUser(ctx, email, "hashed_password")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		retrieved, err := repo.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, email, *retrieved.Email)
	})

	t.Run("not found", func(t *testing.T) {
		user, err := repo.GetUserByEmail(ctx, "nonexistent@example.com")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("empty email", func(t *testing.T) {
		user, err := repo.GetUserByEmail(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})
}

func TestUserRepository_GetUserByExpoDeviceID(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		created, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		retrieved, err := repo.GetUserByExpoDeviceID(ctx, deviceID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, created.ID, retrieved.ID)
		assert.Equal(t, deviceID, *retrieved.ExpoDeviceID)
	})

	t.Run("not found", func(t *testing.T) {
		user, err := repo.GetUserByExpoDeviceID(ctx, "nonexistent-device")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("empty device ID", func(t *testing.T) {
		user, err := repo.GetUserByExpoDeviceID(ctx, "")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})
}

func TestUserRepository_UpdateUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success - update email", func(t *testing.T) {
		originalEmail := "original@example.com"
		created, err := repo.CreateUser(ctx, originalEmail, "hashed_password")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		newEmail := "updated@example.com"
		updated, err := repo.UpdateUser(ctx, created.ID, &newEmail)
		require.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newEmail, *updated.Email)
		assert.True(t, updated.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("user not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		newEmail := "test@example.com"
		updated, err := repo.UpdateUser(ctx, nonExistentID, &newEmail)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("duplicate email", func(t *testing.T) {
		email1 := "user1@example.com"
		email2 := "user2@example.com"

		user1, err := repo.CreateUser(ctx, email1, "hash1")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user1.ID)

		user2, err := repo.CreateUser(ctx, email2, "hash2")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user2.ID)

		// Try to update user2 to use user1's email
		updated, err := repo.UpdateUser(ctx, user2.ID, &email1)
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})
}

func TestUserRepository_UpgradeAccount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		created, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		assert.True(t, created.IsMobileOnly())

		email := "upgraded@example.com"
		passwordHash := "hashed_password_123"
		upgraded, err := repo.UpgradeAccount(ctx, created.ID, email, passwordHash)
		require.NoError(t, err)
		assert.NotNil(t, upgraded)

		assert.Equal(t, created.ID, upgraded.ID)
		assert.Equal(t, email, *upgraded.Email)
		assert.Equal(t, passwordHash, *upgraded.PasswordHash)
		assert.Equal(t, deviceID, *upgraded.ExpoDeviceID)
		assert.True(t, upgraded.IsHybrid())
		assert.True(t, upgraded.UpdatedAt.After(created.UpdatedAt))
	})

	t.Run("not mobile-only account", func(t *testing.T) {
		email := "emailuser@example.com"
		created, err := repo.CreateUser(ctx, email, "hash")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		upgraded, err := repo.UpgradeAccount(ctx, created.ID, "new@example.com", "newhash")
		assert.Error(t, err)
		assert.Nil(t, upgraded)
	})

	t.Run("user not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		upgraded, err := repo.UpgradeAccount(ctx, nonExistentID, "test@example.com", "hash")
		assert.Error(t, err)
		assert.Nil(t, upgraded)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("empty email", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		created, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		upgraded, err := repo.UpgradeAccount(ctx, created.ID, "", "hash")
		assert.Error(t, err)
		assert.Nil(t, upgraded)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})

	t.Run("empty password hash", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		created, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		upgraded, err := repo.UpgradeAccount(ctx, created.ID, "test@example.com", "")
		assert.Error(t, err)
		assert.Nil(t, upgraded)
		assert.ErrorIs(t, err, ErrInvalidUserData)
	})

	t.Run("duplicate email on upgrade", func(t *testing.T) {
		existingEmail := "existing@example.com"
		existingUser, err := repo.CreateUser(ctx, existingEmail, "hash1")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, existingUser.ID)

		deviceID := "test-device-" + uuid.New().String()
		mobileUser, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		upgraded, err := repo.UpgradeAccount(ctx, mobileUser.ID, existingEmail, "hash2")
		assert.Error(t, err)
		assert.Nil(t, upgraded)
		assert.ErrorIs(t, err, ErrUserAlreadyExists)
	})
}

func TestUserRepository_DeleteUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "deleteme@example.com"
		created, err := repo.CreateUser(ctx, email, "hash")
		require.NoError(t, err)

		err = repo.DeleteUser(ctx, created.ID)
		require.NoError(t, err)

		// Verify user is deleted
		retrieved, err := repo.GetUserByID(ctx, created.ID)
		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})

	t.Run("user not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.DeleteUser(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

func TestUserRepository_UpdateLastLoginAt(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "logintest@example.com"
		created, err := repo.CreateUser(ctx, email, "hash")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, created.ID)

		assert.Nil(t, created.LastLoginAt)

		err = repo.UpdateLastLoginAt(ctx, created.ID)
		require.NoError(t, err)

		// Verify last login was updated
		retrieved, err := repo.GetUserByID(ctx, created.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved.LastLoginAt)
		assert.False(t, retrieved.LastLoginAt.IsZero())
	})

	t.Run("user not found", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.UpdateLastLoginAt(ctx, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUserNotFound)
	})
}

// Test account type detection after various operations
func TestUserRepository_AccountTypeDetection(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	t.Run("mobile-only account type", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		user, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		assert.True(t, user.IsMobileOnly())
		assert.False(t, user.IsEmailOnly())
		assert.False(t, user.IsHybrid())
		assert.Equal(t, "mobile-only", user.AccountType())
		assert.True(t, user.CanLoginWithDevice())
		assert.False(t, user.CanLoginWithEmail())
	})

	t.Run("email-only account type", func(t *testing.T) {
		email := "emailonly@example.com"
		user, err := repo.CreateUser(ctx, email, "hash")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		assert.False(t, user.IsMobileOnly())
		assert.True(t, user.IsEmailOnly())
		assert.False(t, user.IsHybrid())
		assert.Equal(t, "email-only", user.AccountType())
		assert.False(t, user.CanLoginWithDevice())
		assert.True(t, user.CanLoginWithEmail())
	})

	t.Run("hybrid account type after upgrade", func(t *testing.T) {
		deviceID := "test-device-" + uuid.New().String()
		mobileUser, err := repo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		email := "hybrid@example.com"
		hybridUser, err := repo.UpgradeAccount(ctx, mobileUser.ID, email, "hash")
		require.NoError(t, err)

		assert.False(t, hybridUser.IsMobileOnly())
		assert.False(t, hybridUser.IsEmailOnly())
		assert.True(t, hybridUser.IsHybrid())
		assert.Equal(t, "hybrid", hybridUser.AccountType())
		// Per requirements: hybrid accounts cannot login with device ID
		assert.False(t, hybridUser.CanLoginWithDevice())
		assert.True(t, hybridUser.CanLoginWithEmail())
	})
}
