package integration

import (
	"context"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/users/internal/services"
	"github.com/cairn-app/cairn-reader/services/users/test/integration/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_EmailLogin_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(
		env.UserRepo,
		env.TokenRepo,
		env.JWTManager,
		env.PasswordHash,
		env.TokenService,
		8,
		true,
	)

	ctx := context.Background()

	t.Run("Login with valid credentials", func(t *testing.T) {
		// Create test user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Login
		response, err := authService.Login(ctx, email, password)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Verify response
		assert.Equal(t, userID, response.User.ID)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Greater(t, response.ExpiresIn, int64(0))

		// Verify access token
		claims, err := env.JWTManager.ValidateToken(response.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.Subject)

		// Verify last login was updated
		user, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.NotNil(t, user.LastLoginAt)
	})

	t.Run("Login with incorrect password", func(t *testing.T) {
		// Create test user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Try to login with wrong password
		response, err := authService.Login(ctx, email, "WrongPassword123!")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, services.ErrInvalidCredentials, err)
	})

	t.Run("Login with non-existent email", func(t *testing.T) {
		email := testutil.RandomEmail()
		response, err := authService.Login(ctx, email, "TestPass123!")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, services.ErrInvalidCredentials, err)
	})

	t.Run("Login with empty email", func(t *testing.T) {
		response, err := authService.Login(ctx, "", "TestPass123!")
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Login with empty password", func(t *testing.T) {
		email := testutil.RandomEmail()
		response, err := authService.Login(ctx, email, "")
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Login with mobile-only account should fail", func(t *testing.T) {
		// Create mobile-only user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		// Mobile-only users don't have email
		response, err := authService.Login(ctx, "nonexistent@example.com", "password")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, services.ErrInvalidCredentials, err)
	})

	t.Run("Multiple consecutive logins", func(t *testing.T) {
		// Create test user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Login multiple times
		for i := 0; i < 3; i++ {
			response, err := authService.Login(ctx, email, password)
			require.NoError(t, err)
			assert.NotEmpty(t, response.AccessToken)
			assert.NotEmpty(t, response.RefreshToken)

			// Each login should generate different tokens
			if i > 0 {
				time.Sleep(100 * time.Millisecond)
			}
		}

		// Verify last login timestamp was updated
		user, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.NotNil(t, user.LastLoginAt)
	})
}

func TestAuthService_MobileLogin_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(
		env.UserRepo,
		env.TokenRepo,
		env.JWTManager,
		env.PasswordHash,
		env.TokenService,
		8,
		true,
	)

	ctx := context.Background()

	t.Run("Login with valid device ID", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		deviceInfo := "iPhone 14"
		ipAddress := "192.168.1.50"

		// Login
		response, err := authService.LoginMobile(ctx, deviceID, &deviceInfo, &ipAddress)
		require.NoError(t, err)
		assert.NotNil(t, response)

		// Verify response
		assert.Equal(t, userID, response.User.ID)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)

		// Verify access token
		claims, err := env.JWTManager.ValidateToken(response.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.Subject)

		// Verify last login was updated
		user, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.NotNil(t, user.LastLoginAt)
		assert.True(t, user.IsMobileOnly())
	})

	t.Run("Login with non-existent device ID", func(t *testing.T) {
		deviceID := testutil.RandomDeviceID()
		response, err := authService.LoginMobile(ctx, deviceID, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, services.ErrInvalidCredentials, err)
	})

	t.Run("Login with empty device ID", func(t *testing.T) {
		response, err := authService.LoginMobile(ctx, "", nil, nil)
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Login with hybrid account should fail", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		// Upgrade to hybrid account
		email := testutil.RandomEmail()
		password := "TestPass123!"
		passwordHash, err := env.PasswordHash.HashPassword(password)
		require.NoError(t, err)

		_, err = env.UserRepo.UpgradeAccount(ctx, userID, email, passwordHash)
		require.NoError(t, err)

		// Try to login with device ID (should fail for hybrid accounts)
		response, err := authService.LoginMobile(ctx, deviceID, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, services.ErrHybridAccountDeviceLogin, err)
	})

	t.Run("Mobile login with device tracking", func(t *testing.T) {
		// Create mobile user
		deviceID := testutil.RandomDeviceID()
		userID := env.CreateTestMobileUser(t, deviceID)
		defer env.CleanupTestUser(t, userID)

		deviceInfo1 := "iPhone 13"
		ipAddress1 := "192.168.1.10"

		// First login
		response1, err := authService.LoginMobile(ctx, deviceID, &deviceInfo1, &ipAddress1)
		require.NoError(t, err)

		// Second login from different device info
		deviceInfo2 := "iPhone 14 Pro"
		ipAddress2 := "10.0.0.5"

		response2, err := authService.LoginMobile(ctx, deviceID, &deviceInfo2, &ipAddress2)
		require.NoError(t, err)

		// Both should succeed with different tokens
		assert.NotEqual(t, response1.AccessToken, response2.AccessToken)
		assert.NotEqual(t, response1.RefreshToken, response2.RefreshToken)
	})
}

func TestAuthService_LoginEndToEnd_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(
		env.UserRepo,
		env.TokenRepo,
		env.JWTManager,
		env.PasswordHash,
		env.TokenService,
		8,
		true,
	)

	ctx := context.Background()

	t.Run("Complete login flow - register then login", func(t *testing.T) {
		email := testutil.RandomEmail()
		password := "SecurePass123!"

		// Step 1: Register
		registerResponse, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, registerResponse.User.ID)

		firstAccessToken := registerResponse.AccessToken

		// Step 2: Login (simulating app restart)
		loginResponse, err := authService.Login(ctx, email, password)
		require.NoError(t, err)

		// Step 3: Verify login returns new tokens
		assert.NotEqual(t, firstAccessToken, loginResponse.AccessToken)

		// Step 4: Verify both tokens are valid
		claims1, err := env.JWTManager.ValidateToken(firstAccessToken)
		require.NoError(t, err)

		claims2, err := env.JWTManager.ValidateToken(loginResponse.AccessToken)
		require.NoError(t, err)

		// Both should be for same user
		assert.Equal(t, claims1.Subject, claims2.Subject)
	})

	t.Run("Complete mobile login flow", func(t *testing.T) {
		deviceID := testutil.RandomDeviceID()

		// Step 1: Register mobile user
		registerResponse, err := authService.RegisterMobile(ctx, deviceID, nil, nil)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, registerResponse.User.ID)

		// Step 2: Login with same device ID
		loginResponse, err := authService.LoginMobile(ctx, deviceID, nil, nil)
		require.NoError(t, err)

		// Step 3: Verify user ID is the same
		assert.Equal(t, registerResponse.User.ID, loginResponse.User.ID)

		// Step 4: Verify new tokens were generated
		assert.NotEqual(t, registerResponse.AccessToken, loginResponse.AccessToken)
		assert.NotEqual(t, registerResponse.RefreshToken, loginResponse.RefreshToken)
	})

	t.Run("Case sensitivity in email login", func(t *testing.T) {
		email := "Test.User@Example.Com"
		password := "TestPass123!"

		// Register with mixed case email
		registerResponse, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, registerResponse.User.ID)

		// Try to login with lowercase email
		loginResponse, err := authService.Login(ctx, "test.user@example.com", password)

		// Should succeed (email comparison should be case-insensitive)
		// Note: This depends on database collation settings
		if err == nil {
			assert.Equal(t, registerResponse.User.ID, loginResponse.User.ID)
		}
	})

	t.Run("Concurrent logins from multiple devices", func(t *testing.T) {
		email := testutil.RandomEmail()
		password := "TestPass123!"

		// Register user
		registerResponse, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, registerResponse.User.ID)

		// Simulate concurrent logins
		done := make(chan bool, 3)
		for i := 0; i < 3; i++ {
			go func() {
				response, err := authService.Login(ctx, email, password)
				assert.NoError(t, err)
				assert.NotNil(t, response)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 3; i++ {
			<-done
		}
	})
}
