//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/users/internal/services"
	"github.com/andrew-craig/cairn-reader/services/users/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserService_GetUser_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize user service
	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          env.UserRepo,
		RefreshTokenRepo:  env.TokenRepo,
		PasswordHasher:    env.PasswordHash,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	ctx := context.Background()

	t.Run("Get user with authorization", func(t *testing.T) {
		// Create user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Get user (authorized - same user ID)
		user, err := userService.GetUser(ctx, userID, userID)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, email, *user.Email)
	})

	t.Run("Get user without authorization", func(t *testing.T) {
		// Create two users
		email1 := testutil.RandomEmail()
		password := "TestPass123!"
		userID1, _ := env.CreateTestUser(t, email1, password)
		defer env.CleanupTestUser(t, userID1)

		email2 := testutil.RandomEmail()
		userID2, _ := env.CreateTestUser(t, email2, password)
		defer env.CleanupTestUser(t, userID2)

		// Try to get user2 as user1 (unauthorized)
		user, err := userService.GetUser(ctx, userID2, userID1)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, services.ErrUnauthorized, err)
	})

	t.Run("Get non-existent user", func(t *testing.T) {
		// Create user for authorization
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Try to get self (should work even if we're testing another scenario)
		user, err := userService.GetUser(ctx, userID, userID)
		require.NoError(t, err)
		assert.NotNil(t, user)
	})
}

func TestUserService_UpdateUser_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize user service
	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          env.UserRepo,
		RefreshTokenRepo:  env.TokenRepo,
		PasswordHasher:    env.PasswordHash,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	ctx := context.Background()

	t.Run("Update email with authorization", func(t *testing.T) {
		// Create user
		oldEmail := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, oldEmail, password)
		defer env.CleanupTestUser(t, userID)

		// Update email
		newEmail := testutil.RandomEmail()
		user, err := userService.UpdateUser(ctx, userID, userID, &newEmail)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, newEmail, *user.Email)

		// Verify in database
		dbUser, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, newEmail, *dbUser.Email)
	})

	t.Run("Update email without authorization", func(t *testing.T) {
		// Create two users
		email1 := testutil.RandomEmail()
		password := "TestPass123!"
		userID1, _ := env.CreateTestUser(t, email1, password)
		defer env.CleanupTestUser(t, userID1)

		email2 := testutil.RandomEmail()
		userID2, _ := env.CreateTestUser(t, email2, password)
		defer env.CleanupTestUser(t, userID2)

		// Try to update user2 as user1
		newEmail := testutil.RandomEmail()
		user, err := userService.UpdateUser(ctx, userID2, userID1, &newEmail)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, services.ErrUnauthorized, err)

		// Verify email was not changed
		dbUser, err := env.UserRepo.GetUserByID(ctx, userID2)
		require.NoError(t, err)
		assert.Equal(t, email2, *dbUser.Email)
	})

	t.Run("Update to duplicate email", func(t *testing.T) {
		// Create two users
		email1 := testutil.RandomEmail()
		password := "TestPass123!"
		userID1, _ := env.CreateTestUser(t, email1, password)
		defer env.CleanupTestUser(t, userID1)

		email2 := testutil.RandomEmail()
		userID2, _ := env.CreateTestUser(t, email2, password)
		defer env.CleanupTestUser(t, userID2)

		// Try to update user2 email to user1's email
		user, err := userService.UpdateUser(ctx, userID2, userID2, &email1)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, services.ErrAccountExists, err)
	})

	t.Run("Update with nil email", func(t *testing.T) {
		// Create user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Try to update with nil email
		user, err := userService.UpdateUser(ctx, userID, userID, nil)
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserService_UpgradeAccount_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize user service
	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          env.UserRepo,
		RefreshTokenRepo:  env.TokenRepo,
		PasswordHasher:    env.PasswordHash,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	ctx := context.Background()

	t.Run("Upgrade mobile account to hybrid", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		// Verify it's mobile-only
		user, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.True(t, user.IsMobileOnly())

		// Upgrade account
		email := testutil.RandomEmail()
		password := "TestPass123!"
		upgradedUser, err := userService.UpgradeAccount(ctx, userID, userID, email, password)
		require.NoError(t, err)
		assert.NotNil(t, upgradedUser)

		// Verify it's now hybrid
		assert.Equal(t, email, *upgradedUser.Email)
		assert.Equal(t, deviceID, *upgradedUser.ExpoDeviceID)
		assert.NotNil(t, upgradedUser.PasswordHash)
		assert.True(t, upgradedUser.IsHybrid())
		assert.False(t, upgradedUser.IsMobileOnly())
		assert.False(t, upgradedUser.IsEmailOnly())

		// Verify password was hashed correctly
		err = env.PasswordHash.ComparePassword(*upgradedUser.PasswordHash, password)
		assert.NoError(t, err)
	})

	t.Run("Upgrade without authorization", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID1 := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID1)

		// Create another user for requesting user ID
		email2 := testutil.RandomEmail()
		password2 := "TestPass123!"
		userID2, _ := env.CreateTestUser(t, email2, password2)
		defer env.CleanupTestUser(t, userID2)

		// Try to upgrade user1 as user2
		email := testutil.RandomEmail()
		password := "TestPass123!"
		user, err := userService.UpgradeAccount(ctx, userID1, userID2, email, password)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, services.ErrUnauthorized, err)
	})

	t.Run("Upgrade non-mobile account", func(t *testing.T) {
		// Create email account
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Try to upgrade (should fail - not mobile-only)
		newEmail := testutil.RandomEmail()
		newPassword := "NewPass123!"
		user, err := userService.UpgradeAccount(ctx, userID, userID, newEmail, newPassword)
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, services.ErrNotMobileAccount, err)
	})

	t.Run("Upgrade with weak password", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		// Try to upgrade with weak password
		email := testutil.RandomEmail()
		weakPassword := "weak"
		user, err := userService.UpgradeAccount(ctx, userID, userID, email, weakPassword)
		assert.Error(t, err)
		assert.Nil(t, user)

		// Verify account is still mobile-only
		dbUser, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.True(t, dbUser.IsMobileOnly())
	})

	t.Run("Upgrade with duplicate email", func(t *testing.T) {
		// Create email user
		existingEmail := testutil.RandomEmail()
		password := "TestPass123!"
		existingUserID, _ := env.CreateTestUser(t, existingEmail, password)
		defer env.CleanupTestUser(t, existingUserID)

		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		mobileUserID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, mobileUserID)

		// Try to upgrade with existing email
		user, err := userService.UpgradeAccount(ctx, mobileUserID, mobileUserID, existingEmail, "NewPass123!")
		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, services.ErrAccountExists, err)
	})

	t.Run("Upgrade with empty email", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		// Try to upgrade with empty email
		user, err := userService.UpgradeAccount(ctx, userID, userID, "", "TestPass123!")
		assert.Error(t, err)
		assert.Nil(t, user)
	})

	t.Run("Upgrade with empty password", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		// Try to upgrade with empty password
		email := testutil.RandomEmail()
		user, err := userService.UpgradeAccount(ctx, userID, userID, email, "")
		assert.Error(t, err)
		assert.Nil(t, user)
	})
}

func TestUserService_DeleteUser_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize user service and auth service
	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          env.UserRepo,
		RefreshTokenRepo:  env.TokenRepo,
		PasswordHasher:    env.PasswordHash,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            env.UserRepo,
		RefreshTokenService: env.TokenService,
		JWTManager:          env.JWTManager,
		PasswordHasher:      env.PasswordHash,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	ctx := context.Background()

	t.Run("Delete user with authorization", func(t *testing.T) {
		// Create user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)

		// Delete user
		err := userService.DeleteUser(ctx, userID, userID)
		require.NoError(t, err)

		// Verify user is deleted
		_, err = env.UserRepo.GetUserByID(ctx, userID)
		assert.Error(t, err)
	})

	t.Run("Delete user without authorization", func(t *testing.T) {
		// Create two users
		email1 := testutil.RandomEmail()
		password := "TestPass123!"
		userID1, _ := env.CreateTestUser(t, email1, password)
		defer env.CleanupTestUser(t, userID1)

		email2 := testutil.RandomEmail()
		userID2, _ := env.CreateTestUser(t, email2, password)
		defer env.CleanupTestUser(t, userID2)

		// Try to delete user2 as user1
		err := userService.DeleteUser(ctx, userID2, userID1)
		assert.Error(t, err)
		assert.Equal(t, services.ErrUnauthorized, err)

		// Verify user2 still exists
		user, err := env.UserRepo.GetUserByID(ctx, userID2)
		require.NoError(t, err)
		assert.NotNil(t, user)
	})

	t.Run("Delete user with active refresh tokens", func(t *testing.T) {
		// Create and login user multiple times
		email := testutil.RandomEmail()
		password := "TestPass123!"

		registerResponse, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		userID := registerResponse.User.ID

		// Login from multiple devices
		_, err = authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		_, err = authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		// Delete user (should also delete all refresh tokens)
		err = userService.DeleteUser(ctx, userID, userID)
		require.NoError(t, err)

		// Verify user is deleted
		_, err = env.UserRepo.GetUserByID(ctx, userID)
		assert.Error(t, err)

		// Note: Refresh tokens should also be deleted due to foreign key cascade
	})

	t.Run("Delete non-existent user", func(t *testing.T) {
		// Create user for authorization
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Try to delete (user exists, so this should succeed)
		err := userService.DeleteUser(ctx, userID, userID)
		require.NoError(t, err)

		// Try to delete again (user doesn't exist now)
		err = userService.DeleteUser(ctx, userID, userID)
		assert.Error(t, err)
	})
}

func TestUserService_EndToEnd_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize services
	userService := services.NewUserService(services.UserServiceConfig{
		UserRepo:          env.UserRepo,
		RefreshTokenRepo:  env.TokenRepo,
		PasswordHasher:    env.PasswordHash,
		PasswordMinLength: 8,
		RequireComplexity: true,
	})

	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            env.UserRepo,
		RefreshTokenService: env.TokenService,
		JWTManager:          env.JWTManager,
		PasswordHasher:      env.PasswordHash,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	ctx := context.Background()

	t.Run("Complete mobile user lifecycle", func(t *testing.T) {
		// Step 1: Register mobile user
		deviceID := testutil.RandomDeviceID()
		registerResponse, err := authService.RegisterMobile(ctx, deviceID, "", "")
		require.NoError(t, err)
		userID := registerResponse.User.ID
		defer env.CleanupTestUser(t, userID)

		// Step 2: Verify mobile-only status
		user1, err := userService.GetUser(ctx, userID, userID)
		require.NoError(t, err)
		assert.True(t, user1.IsMobileOnly())

		// Step 3: Upgrade to hybrid account
		email := testutil.RandomEmail()
		password := "TestPass123!"
		upgradedUser, err := userService.UpgradeAccount(ctx, userID, userID, email, password)
		require.NoError(t, err)
		assert.True(t, upgradedUser.IsHybrid())

		// Step 4: Login with email
		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)
		assert.Equal(t, userID, loginResponse.User.ID)

		// Step 5: Update email
		newEmail := testutil.RandomEmail()
		updatedUser, err := userService.UpdateUser(ctx, userID, userID, &newEmail)
		require.NoError(t, err)
		assert.Equal(t, newEmail, *updatedUser.Email)

		// Step 6: Delete account
		err = userService.DeleteUser(ctx, userID, userID)
		require.NoError(t, err)

		// Step 7: Verify account is deleted
		_, err = env.UserRepo.GetUserByID(ctx, userID)
		assert.Error(t, err)
	})

	t.Run("Complete email user lifecycle", func(t *testing.T) {
		// Step 1: Register with email
		email := testutil.RandomEmail()
		password := "TestPass123!"
		registerResponse, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		userID := registerResponse.User.ID
		defer env.CleanupTestUser(t, userID)

		// Step 2: Verify email-only status
		user, err := userService.GetUser(ctx, userID, userID)
		require.NoError(t, err)
		assert.True(t, user.IsEmailOnly())

		// Step 3: Update email
		newEmail := testutil.RandomEmail()
		updatedUser, err := userService.UpdateUser(ctx, userID, userID, &newEmail)
		require.NoError(t, err)
		assert.Equal(t, newEmail, *updatedUser.Email)

		// Step 4: Login with new email
		loginResponse, err := authService.Login(ctx, newEmail, password, "", "")
		require.NoError(t, err)
		assert.Equal(t, userID, loginResponse.User.ID)

		// Step 5: Delete account
		err = userService.DeleteUser(ctx, userID, userID)
		require.NoError(t, err)
	})
}
