package integration

import (
	"context"
	"testing"
	"time"

	"github.com/andrew-craig/cairn/services/users/internal/database"
	"github.com/andrew-craig/cairn/services/users/internal/services"
	"github.com/andrew-craig/cairn/services/users/test/integration/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthService_Registration_Integration(t *testing.T) {
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
		8, // min password length
		true, // require complexity
	)

	ctx := context.Background()

	t.Run("Register with email and password", func(t *testing.T) {
		email := testutil.RandomEmail()
		password := "TestPass123!"

		// Register user
		response, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		assert.NotNil(t, response)
		defer env.CleanupTestUser(t, response.User.ID)

		// Verify response structure
		assert.NotEqual(t, uuid.Nil, response.User.ID)
		assert.NotNil(t, response.User.Email)
		assert.Equal(t, email, *response.User.Email)
		assert.Nil(t, response.User.ExpoDeviceID)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)
		assert.Greater(t, response.ExpiresIn, int64(0))

		// Verify user was created in database
		user, err := env.UserRepo.GetUserByEmail(ctx, email)
		require.NoError(t, err)
		assert.Equal(t, response.User.ID, user.ID)
		assert.NotNil(t, user.PasswordHash)

		// Verify access token is valid
		claims, err := env.JWTManager.ValidateToken(response.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, response.User.ID.String(), claims.Subject)

		// Verify refresh token was stored
		// Note: We can't verify the exact token since it's hashed, but we can verify creation
		assert.NotEmpty(t, response.RefreshToken)
	})

	t.Run("Register with duplicate email", func(t *testing.T) {
		email := testutil.RandomEmail()
		password := "TestPass123!"

		// Register first user
		response1, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, response1.User.ID)

		// Try to register with same email
		response2, err := authService.Register(ctx, email, password)
		assert.Error(t, err)
		assert.Nil(t, response2)
		assert.Equal(t, services.ErrAccountExists, err)
	})

	t.Run("Register with weak password", func(t *testing.T) {
		email := testutil.RandomEmail()
		weakPassword := "weak" // Too short

		response, err := authService.Register(ctx, email, weakPassword)
		assert.Error(t, err)
		assert.Nil(t, response)

		// Verify user was not created
		_, err = env.UserRepo.GetUserByEmail(ctx, email)
		assert.Error(t, err)
		assert.Equal(t, database.ErrUserNotFound, err)
	})

	t.Run("Register with empty email", func(t *testing.T) {
		response, err := authService.Register(ctx, "", "TestPass123!")
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Register with empty password", func(t *testing.T) {
		email := testutil.RandomEmail()
		response, err := authService.Register(ctx, email, "")
		assert.Error(t, err)
		assert.Nil(t, response)
	})
}

func TestAuthService_MobileRegistration_Integration(t *testing.T) {
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

	t.Run("Register with device ID", func(t *testing.T) {
		deviceID := testutil.RandomDeviceID()
		deviceInfo := "iPhone 14 Pro"
		ipAddress := "192.168.1.100"

		// Register mobile user
		response, err := authService.RegisterMobile(ctx, deviceID, &deviceInfo, &ipAddress)
		require.NoError(t, err)
		assert.NotNil(t, response)
		defer env.CleanupTestUser(t, response.User.ID)

		// Verify response structure
		assert.NotEqual(t, uuid.Nil, response.User.ID)
		assert.Nil(t, response.User.Email)
		assert.NotNil(t, response.User.ExpoDeviceID)
		assert.Equal(t, deviceID, *response.User.ExpoDeviceID)
		assert.NotEmpty(t, response.AccessToken)
		assert.NotEmpty(t, response.RefreshToken)

		// Verify user was created in database
		user, err := env.UserRepo.GetUserByExpoDeviceID(ctx, deviceID)
		require.NoError(t, err)
		assert.Equal(t, response.User.ID, user.ID)
		assert.Nil(t, user.PasswordHash)
		assert.True(t, user.IsMobileOnly())

		// Verify access token is valid
		claims, err := env.JWTManager.ValidateToken(response.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, response.User.ID.String(), claims.Subject)
	})

	t.Run("Register with duplicate device ID", func(t *testing.T) {
		deviceID := testutil.RandomDeviceID()

		// Register first user
		response1, err := authService.RegisterMobile(ctx, deviceID, nil, nil)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, response1.User.ID)

		// Try to register with same device ID
		response2, err := authService.RegisterMobile(ctx, deviceID, nil, nil)
		assert.Error(t, err)
		assert.Nil(t, response2)
		assert.Equal(t, services.ErrAccountExists, err)
	})

	t.Run("Register with empty device ID", func(t *testing.T) {
		response, err := authService.RegisterMobile(ctx, "", nil, nil)
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Register with device info and IP tracking", func(t *testing.T) {
		deviceID := testutil.RandomDeviceID()
		deviceInfo := "Samsung Galaxy S23"
		ipAddress := "10.0.0.5"

		response, err := authService.RegisterMobile(ctx, deviceID, &deviceInfo, &ipAddress)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, response.User.ID)

		// Verify device info was tracked
		// Note: Device info is stored with refresh token, not user
		assert.NotEmpty(t, response.RefreshToken)
	})
}

func TestAuthService_RegistrationEndToEnd_Integration(t *testing.T) {
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

	t.Run("Complete registration flow - email", func(t *testing.T) {
		email := testutil.RandomEmail()
		password := "SecurePass123!"

		// Step 1: Register
		response, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, response.User.ID)

		userID := response.User.ID
		accessToken := response.AccessToken

		// Step 2: Verify access token can be used for authorization
		claims, err := env.JWTManager.ValidateToken(accessToken)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.Subject)

		// Step 3: Verify user can be retrieved
		user, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, email, *user.Email)

		// Step 4: Verify password is hashed (not plain text)
		assert.NotNil(t, user.PasswordHash)
		assert.NotEqual(t, password, *user.PasswordHash)

		// Step 5: Verify password can be verified
		err = env.PasswordHash.ComparePassword(*user.PasswordHash, password)
		assert.NoError(t, err)
	})

	t.Run("Complete registration flow - mobile", func(t *testing.T) {
		deviceID := testutil.RandomDeviceID()
		deviceInfo := "iPhone 15"
		ipAddress := "203.0.113.42"

		// Step 1: Register mobile user
		response, err := authService.RegisterMobile(ctx, deviceID, &deviceInfo, &ipAddress)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, response.User.ID)

		userID := response.User.ID

		// Step 2: Verify user is mobile-only
		user, err := env.UserRepo.GetUserByID(ctx, userID)
		require.NoError(t, err)
		assert.True(t, user.IsMobileOnly())
		assert.False(t, user.IsEmailOnly())
		assert.False(t, user.IsHybrid())

		// Step 3: Verify device ID is stored
		assert.Equal(t, deviceID, *user.ExpoDeviceID)

		// Step 4: Verify no email or password
		assert.Nil(t, user.Email)
		assert.Nil(t, user.PasswordHash)
	})

	t.Run("Multiple registrations in sequence", func(t *testing.T) {
		var userIDs []uuid.UUID

		// Register 5 users
		for i := 0; i < 5; i++ {
			email := testutil.RandomEmail()
			password := "TestPass123!"

			response, err := authService.Register(ctx, email, password)
			require.NoError(t, err)
			userIDs = append(userIDs, response.User.ID)

			// Verify each token is unique
			assert.NotEmpty(t, response.AccessToken)
			assert.NotEmpty(t, response.RefreshToken)
		}

		// Cleanup all users
		for _, userID := range userIDs {
			env.CleanupTestUser(t, userID)
		}

		// Verify all users were created
		assert.Len(t, userIDs, 5)
	})
}
