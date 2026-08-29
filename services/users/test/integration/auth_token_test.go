//go:build integration
// +build integration

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

func TestAuthService_TokenRefresh_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            env.UserRepo,
		RefreshTokenService: env.TokenService,
		JWTManager:          env.JWTManager,
		PasswordHasher:      env.PasswordHash,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	ctx := context.Background()

	t.Run("Refresh with valid token", func(t *testing.T) {
		// Create and login user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		oldAccessToken := loginResponse.AccessToken
		oldRefreshToken := loginResponse.RefreshToken

		// Wait a moment to ensure new token has different timestamp
		time.Sleep(100 * time.Millisecond)

		// Refresh tokens
		refreshResponse, err := authService.RefreshAccessToken(ctx, oldRefreshToken, "", "")
		require.NoError(t, err)
		assert.NotNil(t, refreshResponse)

		// Verify new tokens are different
		assert.NotEqual(t, oldAccessToken, refreshResponse.AccessToken)
		assert.NotEqual(t, oldRefreshToken, refreshResponse.RefreshToken)

		// Verify new access token is valid
		claims, err := env.JWTManager.ValidateToken(refreshResponse.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, userID.String(), claims.Subject)

		// Verify old refresh token cannot be reused (token rotation)
		_, err = authService.RefreshAccessToken(ctx, oldRefreshToken, "", "")
		assert.Error(t, err) // Should fail due to token reuse detection or not found
	})

	t.Run("Refresh with invalid token", func(t *testing.T) {
		invalidToken := "invalid_refresh_token_12345"

		response, err := authService.RefreshAccessToken(ctx, invalidToken, "", "")
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Refresh with empty token", func(t *testing.T) {
		response, err := authService.RefreshAccessToken(ctx, "", "", "")
		assert.Error(t, err)
		assert.Nil(t, response)
	})

	t.Run("Multiple refresh cycles", func(t *testing.T) {
		// Create and login user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		currentRefreshToken := loginResponse.RefreshToken

		// Perform 5 refresh cycles
		for i := 0; i < 5; i++ {
			time.Sleep(100 * time.Millisecond)

			response, err := authService.RefreshAccessToken(ctx, currentRefreshToken, "", "")
			require.NoError(t, err, "Refresh cycle %d failed", i+1)

			// Verify new tokens
			assert.NotEmpty(t, response.AccessToken)
			assert.NotEmpty(t, response.RefreshToken)
			assert.NotEqual(t, currentRefreshToken, response.RefreshToken)

			// Update for next cycle
			currentRefreshToken = response.RefreshToken
		}
	})

	t.Run("Refresh with device tracking", func(t *testing.T) {
		// Create and login user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		deviceInfo := "iPhone 15 Pro"
		ipAddress := "203.0.113.100"

		// Refresh with device info
		response, err := authService.RefreshAccessToken(ctx, loginResponse.RefreshToken, deviceInfo, ipAddress)
		require.NoError(t, err)
		assert.NotNil(t, response)
	})

	t.Run("Token reuse detection", func(t *testing.T) {
		// Create and login user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		refreshToken := loginResponse.RefreshToken

		// First refresh - should succeed
		response1, err := authService.RefreshAccessToken(ctx, refreshToken, "", "")
		require.NoError(t, err)

		// Wait to ensure we're outside grace period
		time.Sleep(6 * time.Second)

		// Second refresh with same old token - should detect reuse
		response2, err := authService.RefreshAccessToken(ctx, refreshToken, "", "")
		assert.Error(t, err)
		assert.Nil(t, response2)

		// Verify token family was revoked
		// New refresh token should also not work
		_, err = authService.RefreshAccessToken(ctx, response1.RefreshToken, "", "")
		assert.Error(t, err) // Should be revoked due to reuse detection
	})
}

func TestAuthService_Logout_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            env.UserRepo,
		RefreshTokenService: env.TokenService,
		JWTManager:          env.JWTManager,
		PasswordHasher:      env.PasswordHash,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	ctx := context.Background()

	t.Run("Logout single session", func(t *testing.T) {
		// Create and login user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		refreshToken := loginResponse.RefreshToken

		// Logout
		err = authService.Logout(ctx, refreshToken)
		require.NoError(t, err)

		// Try to use refresh token after logout
		_, err = authService.RefreshAccessToken(ctx, refreshToken, "", "")
		assert.Error(t, err) // Should fail - token revoked
	})

	t.Run("Logout with invalid token", func(t *testing.T) {
		// Logout with non-existent token should not error
		err := authService.Logout(ctx, "invalid_token")
		assert.NoError(t, err) // Should succeed silently
	})

	t.Run("Logout with empty token", func(t *testing.T) {
		err := authService.Logout(ctx, "")
		assert.NoError(t, err) // Should succeed silently
	})

	t.Run("Logout already logged out token", func(t *testing.T) {
		// Create and login user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		refreshToken := loginResponse.RefreshToken

		// Logout once
		err = authService.Logout(ctx, refreshToken)
		require.NoError(t, err)

		// Logout again with same token
		err = authService.Logout(ctx, refreshToken)
		assert.NoError(t, err) // Should succeed silently
	})
}

func TestAuthService_LogoutAll_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            env.UserRepo,
		RefreshTokenService: env.TokenService,
		JWTManager:          env.JWTManager,
		PasswordHasher:      env.PasswordHash,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	ctx := context.Background()

	t.Run("Logout all sessions", func(t *testing.T) {
		// Create user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Login from 3 different devices
		var refreshTokens []string
		for i := 0; i < 3; i++ {
			response, err := authService.Login(ctx, email, password, "", "")
			require.NoError(t, err)
			refreshTokens = append(refreshTokens, response.RefreshToken)
		}

		// Logout all sessions
		err := authService.LogoutAll(ctx, userID)
		require.NoError(t, err)

		// Verify all refresh tokens are invalid
		for i, token := range refreshTokens {
			_, err := authService.RefreshAccessToken(ctx, token, "", "")
			assert.Error(t, err, "Token %d should be revoked", i)
		}
	})

	t.Run("Logout all with no active sessions", func(t *testing.T) {
		// Create user without logging in
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Logout all (should succeed even with no tokens)
		err := authService.LogoutAll(ctx, userID)
		assert.NoError(t, err)
	})

	t.Run("Logout all after individual logout", func(t *testing.T) {
		// Create user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Login twice
		response1, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		response2, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)

		// Logout first session
		err = authService.Logout(ctx, response1.RefreshToken)
		require.NoError(t, err)

		// Logout all sessions
		err = authService.LogoutAll(ctx, userID)
		require.NoError(t, err)

		// Both tokens should be invalid
		_, err = authService.RefreshAccessToken(ctx, response1.RefreshToken, "", "")
		assert.Error(t, err)

		_, err = authService.RefreshAccessToken(ctx, response2.RefreshToken, "", "")
		assert.Error(t, err)
	})
}

func TestAuthService_TokenLifecycle_Integration(t *testing.T) {
	// Wait for dependencies
	testutil.WaitForDatabase(t, 30*time.Second)
	testutil.WaitForVault(t, 30*time.Second)

	// Setup test environment
	env := testutil.SetupTestEnvironment(t)
	defer env.Cleanup(t)
	defer env.CleanupAllTestData(t)

	// Initialize auth service
	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:            env.UserRepo,
		RefreshTokenService: env.TokenService,
		JWTManager:          env.JWTManager,
		PasswordHasher:      env.PasswordHash,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	ctx := context.Background()

	t.Run("Complete token lifecycle", func(t *testing.T) {
		// Step 1: Register
		email := testutil.RandomEmail()
		password := "TestPass123!"

		registerResponse, err := authService.Register(ctx, email, password)
		require.NoError(t, err)
		defer env.CleanupTestUser(t, registerResponse.User.ID)

		token1 := registerResponse.RefreshToken

		// Step 2: Refresh token
		time.Sleep(100 * time.Millisecond)
		refreshResponse1, err := authService.RefreshAccessToken(ctx, token1, "", "")
		require.NoError(t, err)
		token2 := refreshResponse1.RefreshToken

		// Step 3: Refresh again
		time.Sleep(100 * time.Millisecond)
		refreshResponse2, err := authService.RefreshAccessToken(ctx, token2, "", "")
		require.NoError(t, err)
		token3 := refreshResponse2.RefreshToken

		// Step 4: Login again (new session)
		loginResponse, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)
		token4 := loginResponse.RefreshToken

		// Now we have 2 active tokens: token3 and token4
		// Step 5: Logout one session
		err = authService.Logout(ctx, token3)
		require.NoError(t, err)

		// token3 should be invalid
		_, err = authService.RefreshAccessToken(ctx, token3, "", "")
		assert.Error(t, err)

		// token4 should still work
		_, err = authService.RefreshAccessToken(ctx, token4, "", "")
		assert.NoError(t, err)
	})

	t.Run("Multi-device scenario", func(t *testing.T) {
		// Create user
		email := testutil.RandomEmail()
		password := "TestPass123!"
		userID, _ := env.CreateTestUser(t, email, password)
		defer env.CleanupTestUser(t, userID)

		// Device 1: Login
		device1Response, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)
		device1Token := device1Response.RefreshToken

		// Device 2: Login
		device2Response, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)
		device2Token := device2Response.RefreshToken

		// Device 3: Login
		device3Response, err := authService.Login(ctx, email, password, "", "")
		require.NoError(t, err)
		device3Token := device3Response.RefreshToken

		// All devices should have valid tokens
		_, err = authService.RefreshAccessToken(ctx, device1Token, "", "")
		assert.NoError(t, err)

		_, err = authService.RefreshAccessToken(ctx, device2Token, "", "")
		assert.NoError(t, err)

		_, err = authService.RefreshAccessToken(ctx, device3Token, "", "")
		assert.NoError(t, err)

		// Logout all devices
		err = authService.LogoutAll(ctx, userID)
		require.NoError(t, err)

		// All tokens should be invalid
		_, err = authService.RefreshAccessToken(ctx, device1Token, "", "")
		assert.Error(t, err)

		_, err = authService.RefreshAccessToken(ctx, device2Token, "", "")
		assert.Error(t, err)

		_, err = authService.RefreshAccessToken(ctx, device3Token, "", "")
		assert.Error(t, err)
	})
}
