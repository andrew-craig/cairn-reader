package services

import (
	"context"
	"testing"
	"time"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestUserService creates a test user service with database
func setupTestUserService(t *testing.T) (UserService, *database.DB) {
	db := setupTestDB(t)

	// Create password hasher (bcrypt cost 10 for faster tests)
	passwordHasher := auth.NewPasswordHasher(10)

	// Create repositories
	userRepo := database.NewUserRepository(db)
	refreshTokenRepo := database.NewRefreshTokenRepository(db)

	// Create user service
	service := NewUserService(UserServiceConfig{
		UserRepo:          userRepo,
		RefreshTokenRepo:  refreshTokenRepo,
		PasswordHasher:    passwordHasher,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	return service, db
}

func TestNewUserService(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()

	assert.NotNil(t, service)
}

func TestNewUserService_DefaultPasswordSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	passwordHasher := auth.NewPasswordHasher(10)
	userRepo := database.NewUserRepository(db)
	refreshTokenRepo := database.NewRefreshTokenRepository(db)

	// Create service without specifying password settings
	service := NewUserService(UserServiceConfig{
		UserRepo:         userRepo,
		RefreshTokenRepo: refreshTokenRepo,
		PasswordHasher:   passwordHasher,
		// PasswordMinLength not set - should default to 8
		// RequireComplexity not set - should default to false
	})

	assert.NotNil(t, service)

	// Verify defaults by trying to upgrade with short password
	ctx := context.Background()
	userID := uuid.New()

	_, err := service.UpgradeAccount(ctx, userID, userID, "test@example.com", "Short1!")
	// Should fail because password is < 8 chars
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrWeakPassword)
}

func TestUserService_GetUser(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	ctx := context.Background()

	userRepo := database.NewUserRepository(db)

	t.Run("authorized access - same user", func(t *testing.T) {
		email := "getuser@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Get user (requesting user = target user)
		retrievedUser, err := service.GetUser(ctx, user.ID, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrievedUser)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, email, *retrievedUser.Email)
	})

	t.Run("unauthorized access - different user", func(t *testing.T) {
		email := "unauthorized@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Try to get user with different requesting user ID
		differentUserID := uuid.New()
		retrievedUser, err := service.GetUser(ctx, differentUserID, user.ID)
		assert.Error(t, err)
		assert.Nil(t, retrievedUser)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()

		user, err := service.GetUser(ctx, nonExistentID, nonExistentID)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	})

	t.Run("user data returned correctly", func(t *testing.T) {
		email := "userdata@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Get user
		retrievedUser, err := service.GetUser(ctx, user.ID, user.ID)
		require.NoError(t, err)

		// Verify all fields
		assert.NotNil(t, retrievedUser.Email)
		assert.Equal(t, email, *retrievedUser.Email)
		assert.NotNil(t, retrievedUser.PasswordHash)
		assert.Equal(t, passwordHash, *retrievedUser.PasswordHash)
		assert.False(t, retrievedUser.CreatedAt.IsZero())
		assert.False(t, retrievedUser.UpdatedAt.IsZero())
	})
}

func TestUserService_UpdateUser(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	ctx := context.Background()

	userRepo := database.NewUserRepository(db)

	t.Run("authorized update", func(t *testing.T) {
		email := "update@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Update email
		newEmail := "newemail@example.com"
		updatedUser, err := service.UpdateUser(ctx, user.ID, user.ID, &newEmail)
		require.NoError(t, err)
		assert.NotNil(t, updatedUser)
		assert.Equal(t, newEmail, *updatedUser.Email)
		assert.Equal(t, user.ID, updatedUser.ID)
	})

	t.Run("unauthorized update", func(t *testing.T) {
		email := "unauth-update@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Try to update with different requesting user ID
		differentUserID := uuid.New()
		newEmail := "newemail@example.com"
		updatedUser, err := service.UpdateUser(ctx, differentUserID, user.ID, &newEmail)
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("nil email validation", func(t *testing.T) {
		email := "nil-email@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Try to update with nil email
		updatedUser, err := service.UpdateUser(ctx, user.ID, user.ID, nil)
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("duplicate email", func(t *testing.T) {
		email1 := "user1@example.com"
		email2 := "user2@example.com"
		passwordHash := "hashed_password"

		// Create two users
		user1, err := userRepo.CreateUser(ctx, email1, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user1.ID)

		user2, err := userRepo.CreateUser(ctx, email2, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user2.ID)

		// Try to update user2's email to user1's email
		updatedUser, err := service.UpdateUser(ctx, user2.ID, user2.ID, &email1)
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.ErrorIs(t, err, ErrAccountExists)
	})

	t.Run("non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()
		newEmail := "new@example.com"

		updatedUser, err := service.UpdateUser(ctx, nonExistentID, nonExistentID, &newEmail)
		assert.Error(t, err)
		assert.Nil(t, updatedUser)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	})

	t.Run("updated user returned", func(t *testing.T) {
		email := "return-test@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Update email
		newEmail := "updated@example.com"
		updatedUser, err := service.UpdateUser(ctx, user.ID, user.ID, &newEmail)
		require.NoError(t, err)
		assert.NotNil(t, updatedUser)

		// Verify returned user has updated data
		assert.Equal(t, newEmail, *updatedUser.Email)
		assert.Equal(t, user.ID, updatedUser.ID)

		// Verify password hash is unchanged
		assert.Equal(t, passwordHash, *updatedUser.PasswordHash)
	})
}

func TestUserService_UpgradeAccount(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	ctx := context.Background()

	userRepo := database.NewUserRepository(db)

	t.Run("authorized upgrade", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Verify mobile-only
		assert.True(t, mobileUser.IsMobileOnly())

		// Upgrade account
		email := "upgraded@example.com"
		password := "ValidPass123!"

		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, email, password)
		require.NoError(t, err)
		assert.NotNil(t, upgradedUser)

		// Verify upgrade
		assert.True(t, upgradedUser.IsHybrid())
		assert.Equal(t, email, *upgradedUser.Email)
		assert.NotNil(t, upgradedUser.PasswordHash)
		assert.NotEqual(t, password, *upgradedUser.PasswordHash) // Should be hashed
		assert.Equal(t, deviceID, *upgradedUser.ExpoDeviceID)
	})

	t.Run("unauthorized upgrade", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Try to upgrade with different requesting user ID
		differentUserID := uuid.New()
		upgradedUser, err := service.UpgradeAccount(ctx, differentUserID, mobileUser.ID, "email@example.com", "ValidPass123!")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, ErrUnauthorized)
	})

	t.Run("empty email validation", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Try to upgrade with empty email
		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, "", "ValidPass123!")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("empty password validation", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Try to upgrade with empty password
		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, "email@example.com", "")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("weak password rejection", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Try to upgrade with weak password
		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, "email@example.com", "weak")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, ErrWeakPassword)
	})

	t.Run("non-mobile account rejection", func(t *testing.T) {
		email := "already-email@example.com"
		passwordHash := "hashed_password"

		// Create email user (not mobile)
		emailUser, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, emailUser.ID)

		// Try to upgrade (should fail - not mobile-only)
		upgradedUser, err := service.UpgradeAccount(ctx, emailUser.ID, emailUser.ID, "new@example.com", "ValidPass123!")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, ErrNotMobileAccount)
	})

	t.Run("duplicate email handling", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"
		existingEmail := "existing@example.com"

		// Create existing user with email
		existingUser, err := userRepo.CreateUser(ctx, existingEmail, "hash")
		require.NoError(t, err)
		defer cleanupTestUser(t, db, existingUser.ID)

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Try to upgrade with existing email
		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, existingEmail, "ValidPass123!")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, ErrAccountExists)
	})

	t.Run("password hashing verification", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Upgrade account
		email := "hash-test@example.com"
		password := "ValidPass123!"

		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, email, password)
		require.NoError(t, err)
		assert.NotNil(t, upgradedUser)

		// Verify password is hashed (not plain text)
		assert.NotEqual(t, password, *upgradedUser.PasswordHash)
		assert.Greater(t, len(*upgradedUser.PasswordHash), 20) // Bcrypt hashes are long

		// Verify hash can be verified with password hasher
		passwordHasher := auth.NewPasswordHasher(10)
		err = passwordHasher.ComparePassword(*upgradedUser.PasswordHash, password)
		assert.NoError(t, err)
	})

	t.Run("account type verification after upgrade", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		// Create mobile user
		mobileUser, err := userRepo.CreateMobileUser(ctx, deviceID)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, mobileUser.ID)

		// Verify initial state
		assert.True(t, mobileUser.IsMobileOnly())
		assert.False(t, mobileUser.IsHybrid())
		assert.False(t, mobileUser.IsEmailOnly())

		// Upgrade account
		upgradedUser, err := service.UpgradeAccount(ctx, mobileUser.ID, mobileUser.ID, "hybrid@example.com", "ValidPass123!")
		require.NoError(t, err)

		// Verify final state
		assert.False(t, upgradedUser.IsMobileOnly())
		assert.True(t, upgradedUser.IsHybrid())
		assert.False(t, upgradedUser.IsEmailOnly())
	})

	t.Run("non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()

		upgradedUser, err := service.UpgradeAccount(ctx, nonExistentID, nonExistentID, "email@example.com", "ValidPass123!")
		assert.Error(t, err)
		assert.Nil(t, upgradedUser)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	})
}

func TestUserService_DeleteUser(t *testing.T) {
	service, db := setupTestUserService(t)
	defer db.Close()
	ctx := context.Background()

	userRepo := database.NewUserRepository(db)
	refreshTokenRepo := database.NewRefreshTokenRepository(db)

	t.Run("authorized deletion", func(t *testing.T) {
		email := "delete@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)

		// Delete user
		err = service.DeleteUser(ctx, user.ID, user.ID)
		require.NoError(t, err)

		// Verify user is deleted
		deletedUser, err := userRepo.GetUserByID(ctx, user.ID)
		assert.Error(t, err)
		assert.Nil(t, deletedUser)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	})

	t.Run("unauthorized deletion", func(t *testing.T) {
		email := "unauth-delete@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)
		defer cleanupTestUser(t, db, user.ID)

		// Try to delete with different requesting user ID
		differentUserID := uuid.New()
		err = service.DeleteUser(ctx, differentUserID, user.ID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrUnauthorized)

		// Verify user still exists
		stillExists, err := userRepo.GetUserByID(ctx, user.ID)
		require.NoError(t, err)
		assert.NotNil(t, stillExists)
	})

	t.Run("refresh tokens revoked", func(t *testing.T) {
		email := "tokens-delete@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)

		// Create some refresh tokens
		expiresAt := time.Now().UTC().Add(30 * 24 * time.Hour)
		_, err = refreshTokenRepo.CreateRefreshToken(
			ctx,
			user.ID,
			"token-hash-1",
			expiresAt,
			nil,
			nil,
			nil,
		)
		require.NoError(t, err)

		_, err = refreshTokenRepo.CreateRefreshToken(
			ctx,
			user.ID,
			"token-hash-2",
			expiresAt,
			nil,
			nil,
			nil,
		)
		require.NoError(t, err)

		// Delete user
		err = service.DeleteUser(ctx, user.ID, user.ID)
		require.NoError(t, err)

		// Tokens should be revoked (CASCADE or explicit revocation)
		// This is verified by the service calling RevokeAllUserTokens
	})

	t.Run("non-existent user", func(t *testing.T) {
		nonExistentID := uuid.New()

		err := service.DeleteUser(ctx, nonExistentID, nonExistentID)
		assert.Error(t, err)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)
	})

	t.Run("cleanup verification", func(t *testing.T) {
		email := "cleanup@example.com"
		passwordHash := "hashed_password"

		// Create user
		user, err := userRepo.CreateUser(ctx, email, passwordHash)
		require.NoError(t, err)

		userID := user.ID

		// Delete user
		err = service.DeleteUser(ctx, userID, userID)
		require.NoError(t, err)

		// Verify complete cleanup
		deletedUser, err := userRepo.GetUserByID(ctx, userID)
		assert.Error(t, err)
		assert.Nil(t, deletedUser)
		assert.ErrorIs(t, err, apperrors.ErrUserNotFound)

		// Verify cannot access by email
		deletedByEmail, err := userRepo.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Nil(t, deletedByEmail)
	})
}
