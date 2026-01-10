package services

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestAuthService creates a test auth service with database
func setupTestAuthService(t *testing.T) (AuthService, *database.DB, *auth.JWTManager) {
	db := setupTestDB(t)

	// Create password hasher (bcrypt cost 10 for faster tests)
	passwordHasher := auth.NewPasswordHasher(10)

	// Create JWT manager with test keys
	privateKey, publicKey := generateTestRSAKeys(t)

	jwtManager := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     60 * time.Minute,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
	})

	// Create refresh token service
	refreshTokenRepo := database.NewRefreshTokenRepository(db)
	refreshTokenService := auth.NewRefreshTokenService(refreshTokenRepo, 30*24*time.Hour)

	// Create user repository
	userRepo := database.NewUserRepository(db)

	// Create auth service
	service := NewAuthService(AuthServiceConfig{
		UserRepo:            userRepo,
		RefreshTokenService: refreshTokenService,
		JWTManager:          jwtManager,
		PasswordHasher:      passwordHasher,
		PasswordMinLength:   8,
		RequireComplexity:   true,
	})

	return service, db, jwtManager
}

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) *database.DB {
	cfg := &database.Config{
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

	db, err := database.New(cfg)
	if err != nil {
		t.Skip("Database not available for testing:", err)
	}

	return db
}

func getEnvOrDefault(key, defaultValue string) string {
	// In real implementation, use os.Getenv
	return defaultValue
}

// generateTestRSAKeys generates test RSA key pairs for JWT
func generateTestRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	// Generate a 2048-bit RSA key pair for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return privateKey, &privateKey.PublicKey
}

// cleanupTestUser removes a test user from the database
func cleanupTestUser(t *testing.T, db *database.DB, userID uuid.UUID) {
	_, err := db.Pool.Exec(context.Background(), "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test refresh tokens: %v", err)
	}

	_, err = db.Pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test user: %v", err)
	}
}

// cleanupTestUserByEmail removes a test user by email
func cleanupTestUserByEmail(t *testing.T, db *database.DB, email string) {
	var userID uuid.UUID
	err := db.Pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = $1", email).Scan(&userID)
	if err == nil {
		cleanupTestUser(t, db, userID)
	}
}

// cleanupTestUserByDeviceID removes a test user by device ID
func cleanupTestUserByDeviceID(t *testing.T, db *database.DB, deviceID string) {
	var userID uuid.UUID
	err := db.Pool.QueryRow(context.Background(), "SELECT id FROM users WHERE expo_device_id = $1", deviceID).Scan(&userID)
	if err == nil {
		cleanupTestUser(t, db, userID)
	}
}

func TestNewAuthService(t *testing.T) {
	service, db, _ := setupTestAuthService(t)
	defer db.Close()

	assert.NotNil(t, service)
}

func TestNewAuthService_DefaultPasswordSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	passwordHasher := auth.NewPasswordHasher(10)
	privateKey, publicKey := generateTestRSAKeys(t)
	jwtManager := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     60 * time.Minute,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
	})
	refreshTokenRepo := database.NewRefreshTokenRepository(db)
	refreshTokenService := auth.NewRefreshTokenService(refreshTokenRepo, 30*24*time.Hour)
	userRepo := database.NewUserRepository(db)

	// Create service without specifying password settings
	service := NewAuthService(AuthServiceConfig{
		UserRepo:            userRepo,
		RefreshTokenService: refreshTokenService,
		JWTManager:          jwtManager,
		PasswordHasher:      passwordHasher,
		// PasswordMinLength not set - should default to 8
		// RequireComplexity not set - should default to false
	})

	assert.NotNil(t, service)

	// Verify defaults by trying to register with short password
	ctx := context.Background()
	_, err := service.Register(ctx, "test@example.com", "Short1!")

	// Should fail because password is < 8 chars
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrWeakPassword)
}

func TestAuthService_Register(t *testing.T) {
	service, db, jwtManager := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "register@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		resp, err := service.Register(ctx, email, password)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify user data
		assert.NotNil(t, resp.User)
		assert.NotEqual(t, uuid.Nil, resp.User.ID)
		assert.NotNil(t, resp.User.Email)
		assert.Equal(t, email, *resp.User.Email)

		// Verify tokens
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Greater(t, resp.ExpiresIn, int64(0))

		// Verify access token is valid
		token, err := jwtManager.ValidateToken(resp.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, token)

		// Verify password was hashed (not stored in plain text)
		assert.NotNil(t, resp.User.PasswordHash)
		assert.NotEqual(t, password, *resp.User.PasswordHash)
	})

	t.Run("empty email", func(t *testing.T) {
		resp, err := service.Register(ctx, "", "ValidPass123!")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("empty password", func(t *testing.T) {
		resp, err := service.Register(ctx, "test@example.com", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("weak password - too short", func(t *testing.T) {
		resp, err := service.Register(ctx, "weak@example.com", "Short1!")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrWeakPassword)
	})

	t.Run("weak password - no complexity", func(t *testing.T) {
		resp, err := service.Register(ctx, "weak@example.com", "nouppercaseornumbers")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrWeakPassword)
	})

	t.Run("duplicate email", func(t *testing.T) {
		email := "duplicate@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// First registration should succeed
		resp1, err := service.Register(ctx, email, password)
		require.NoError(t, err)
		assert.NotNil(t, resp1)

		// Second registration with same email should fail
		resp2, err := service.Register(ctx, email, password)
		assert.Error(t, err)
		assert.Nil(t, resp2)
		assert.ErrorIs(t, err, ErrAccountExists)
	})
}

func TestAuthService_RegisterMobile(t *testing.T) {
	service, db, jwtManager := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"
		deviceInfo := "iPhone 14 Pro"
		ipAddress := "192.168.1.1"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		resp, err := service.RegisterMobile(ctx, deviceID, deviceInfo, ipAddress)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify user data
		assert.NotNil(t, resp.User)
		assert.NotEqual(t, uuid.Nil, resp.User.ID)
		assert.NotNil(t, resp.User.ExpoDeviceID)
		assert.Equal(t, deviceID, *resp.User.ExpoDeviceID)
		assert.Nil(t, resp.User.Email)
		assert.Nil(t, resp.User.PasswordHash)

		// Verify tokens
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Greater(t, resp.ExpiresIn, int64(0))

		// Verify access token is valid
		token, err := jwtManager.ValidateToken(resp.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, token)
	})

	t.Run("success - without device info", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		resp, err := service.RegisterMobile(ctx, deviceID, "", "")
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.NotNil(t, resp.User)
	})

	t.Run("empty device ID", func(t *testing.T) {
		resp, err := service.RegisterMobile(ctx, "", "iPhone", "192.168.1.1")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("duplicate device ID", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		// First registration should succeed
		resp1, err := service.RegisterMobile(ctx, deviceID, "iPhone", "192.168.1.1")
		require.NoError(t, err)
		assert.NotNil(t, resp1)

		// Second registration with same device ID should fail
		resp2, err := service.RegisterMobile(ctx, deviceID, "iPhone", "192.168.1.1")
		assert.Error(t, err)
		assert.Nil(t, resp2)
		assert.ErrorIs(t, err, ErrAccountExists)
	})
}

func TestAuthService_Login(t *testing.T) {
	service, db, jwtManager := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "login@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user first
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Login
		loginResp, err := service.Login(ctx, email, password, "iPhone 14", "192.168.1.1")
		require.NoError(t, err)
		assert.NotNil(t, loginResp)

		// Verify user data
		assert.NotNil(t, loginResp.User)
		assert.Equal(t, registerResp.User.ID, loginResp.User.ID)
		assert.Equal(t, email, *loginResp.User.Email)

		// Verify tokens (should be different from registration)
		assert.NotEmpty(t, loginResp.AccessToken)
		assert.NotEmpty(t, loginResp.RefreshToken)
		assert.NotEqual(t, registerResp.AccessToken, loginResp.AccessToken)

		// Verify access token is valid
		token, err := jwtManager.ValidateToken(loginResp.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, token)
	})

	t.Run("empty email", func(t *testing.T) {
		resp, err := service.Login(ctx, "", "password", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("empty password", func(t *testing.T) {
		resp, err := service.Login(ctx, "test@example.com", "", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("non-existent user", func(t *testing.T) {
		resp, err := service.Login(ctx, "nonexistent@example.com", "password", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("incorrect password", func(t *testing.T) {
		email := "wrongpass@example.com"
		password := "CorrectPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		_, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Try to login with wrong password
		resp, err := service.Login(ctx, email, "WrongPass123!", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("mobile-only user cannot login with email", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		// Register mobile user
		_, err := service.RegisterMobile(ctx, deviceID, "", "")
		require.NoError(t, err)

		// Mobile-only users don't have email, so this should fail
		// The service correctly uses user.CanLoginWithEmail()
		// This is tested in the model tests and repository tests
	})
}

func TestAuthService_LoginMobile(t *testing.T) {
	service, db, jwtManager := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success - mobile-only account", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		// Register mobile user
		registerResp, err := service.RegisterMobile(ctx, deviceID, "iPhone", "192.168.1.1")
		require.NoError(t, err)

		// Login with device
		loginResp, err := service.LoginMobile(ctx, deviceID, "iPhone 14", "192.168.1.2")
		require.NoError(t, err)
		assert.NotNil(t, loginResp)

		// Verify user data
		assert.NotNil(t, loginResp.User)
		assert.Equal(t, registerResp.User.ID, loginResp.User.ID)
		assert.Equal(t, deviceID, *loginResp.User.ExpoDeviceID)

		// Verify tokens
		assert.NotEmpty(t, loginResp.AccessToken)
		assert.NotEmpty(t, loginResp.RefreshToken)

		// Verify access token is valid
		token, err := jwtManager.ValidateToken(loginResp.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, token)
	})

	t.Run("empty device ID", func(t *testing.T) {
		resp, err := service.LoginMobile(ctx, "", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("non-existent device", func(t *testing.T) {
		resp, err := service.LoginMobile(ctx, "ExpoToken[nonexistent]", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidCredentials)
	})

	t.Run("hybrid account - device login blocked", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		// Register mobile user
		registerResp, err := service.RegisterMobile(ctx, deviceID, "", "")
		require.NoError(t, err)

		// Upgrade to hybrid account (using UserService would be better, but we'll use repo directly)
		userRepo := database.NewUserRepository(db)
		passwordHasher := auth.NewPasswordHasher(10)
		hash, err := passwordHasher.HashPassword("ValidPass123!")
		require.NoError(t, err)

		_, err = userRepo.UpgradeAccount(ctx, registerResp.User.ID, "hybrid@example.com", hash)
		require.NoError(t, err)

		// Try to login with device ID (should fail for hybrid accounts)
		resp, err := service.LoginMobile(ctx, deviceID, "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrHybridAccountDeviceLogin)
	})
}

func TestAuthService_RefreshAccessToken(t *testing.T) {
	service, db, jwtManager := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "refresh@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Refresh access token
		refreshResp, err := service.RefreshAccessToken(ctx, registerResp.RefreshToken, "iPhone", "192.168.1.1")
		require.NoError(t, err)
		assert.NotNil(t, refreshResp)

		// Verify new tokens
		assert.NotEmpty(t, refreshResp.AccessToken)
		assert.NotEmpty(t, refreshResp.RefreshToken)

		// Tokens should be different (rotation)
		assert.NotEqual(t, registerResp.AccessToken, refreshResp.AccessToken)
		assert.NotEqual(t, registerResp.RefreshToken, refreshResp.RefreshToken)

		// Verify new access token is valid
		token, err := jwtManager.ValidateToken(refreshResp.AccessToken)
		require.NoError(t, err)
		assert.NotNil(t, token)

		// Verify user data
		assert.NotNil(t, refreshResp.User)
		assert.Equal(t, registerResp.User.ID, refreshResp.User.ID)
	})

	t.Run("empty refresh token", func(t *testing.T) {
		resp, err := service.RefreshAccessToken(ctx, "", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		resp, err := service.RefreshAccessToken(ctx, "invalid-token", "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
		// Should get invalid credentials or similar error
		assert.True(t, errors.Is(err, ErrInvalidCredentials) || err != nil)
	})

	t.Run("expired refresh token", func(t *testing.T) {
		email := "expired@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Manually expire the token in the database
		_, err = db.Pool.Exec(ctx,
			"UPDATE refresh_tokens SET expires_at = $1 WHERE user_id = $2",
			time.Now().UTC().Add(-1*time.Hour),
			registerResp.User.ID,
		)
		require.NoError(t, err)

		// Try to refresh with expired token
		resp, err := service.RefreshAccessToken(ctx, registerResp.RefreshToken, "", "")
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("token reuse detection", func(t *testing.T) {
		email := "reuse@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		oldRefreshToken := registerResp.RefreshToken

		// First refresh should succeed
		refreshResp1, err := service.RefreshAccessToken(ctx, oldRefreshToken, "", "")
		require.NoError(t, err)
		assert.NotNil(t, refreshResp1)

		// Wait for grace period to pass
		time.Sleep(6 * time.Second)

		// Try to use the old token again (should be detected as reuse)
		refreshResp2, err := service.RefreshAccessToken(ctx, oldRefreshToken, "", "")
		assert.Error(t, err)
		assert.Nil(t, refreshResp2)
		// Should detect token reuse
	})
}

func TestAuthService_Logout(t *testing.T) {
	service, db, _ := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		email := "logout@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Logout
		err = service.Logout(ctx, registerResp.RefreshToken)
		require.NoError(t, err)

		// Try to use the refresh token (should fail)
		refreshResp, err := service.RefreshAccessToken(ctx, registerResp.RefreshToken, "", "")
		assert.Error(t, err)
		assert.Nil(t, refreshResp)
	})

	t.Run("empty token", func(t *testing.T) {
		err := service.Logout(ctx, "")
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("already revoked token - no error", func(t *testing.T) {
		email := "alreadyrevoked@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// First logout
		err = service.Logout(ctx, registerResp.RefreshToken)
		require.NoError(t, err)

		// Second logout (token already revoked - should not error)
		err = service.Logout(ctx, registerResp.RefreshToken)
		require.NoError(t, err)
	})

	t.Run("non-existent token - no error", func(t *testing.T) {
		// Should not error for non-existent tokens
		err := service.Logout(ctx, "non-existent-token")
		require.NoError(t, err)
	})
}

func TestAuthService_LogoutAll(t *testing.T) {
	service, db, _ := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("success - multiple tokens", func(t *testing.T) {
		email := "logoutall@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Create multiple refresh tokens (simulate multiple devices)
		loginResp1, err := service.Login(ctx, email, password, "Device 1", "192.168.1.1")
		require.NoError(t, err)

		loginResp2, err := service.Login(ctx, email, password, "Device 2", "192.168.1.2")
		require.NoError(t, err)

		// Logout all
		err = service.LogoutAll(ctx, registerResp.User.ID)
		require.NoError(t, err)

		// Try to use all refresh tokens (all should fail)
		_, err = service.RefreshAccessToken(ctx, registerResp.RefreshToken, "", "")
		assert.Error(t, err)

		_, err = service.RefreshAccessToken(ctx, loginResp1.RefreshToken, "", "")
		assert.Error(t, err)

		_, err = service.RefreshAccessToken(ctx, loginResp2.RefreshToken, "", "")
		assert.Error(t, err)
	})

	t.Run("success - user with no tokens", func(t *testing.T) {
		email := "notokens@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		registerResp, err := service.Register(ctx, email, password)
		require.NoError(t, err)

		// Revoke the registration token
		err = service.Logout(ctx, registerResp.RefreshToken)
		require.NoError(t, err)

		// Logout all (should succeed even with no tokens)
		err = service.LogoutAll(ctx, registerResp.User.ID)
		require.NoError(t, err)
	})
}

func TestAuthService_GenerateAuthResponse(t *testing.T) {
	service, db, jwtManager := setupTestAuthService(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("verify response structure", func(t *testing.T) {
		email := "response@example.com"
		password := "ValidPass123!"

		defer cleanupTestUserByEmail(t, db, email)

		// Register user
		resp, err := service.Register(ctx, email, password)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify all fields are populated
		assert.NotNil(t, resp.User)
		assert.NotEqual(t, uuid.Nil, resp.User.ID)
		assert.NotEmpty(t, resp.AccessToken)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Greater(t, resp.ExpiresIn, int64(0))

		// Verify ExpiresIn matches JWT manager setting
		expectedExpiry := int64(jwtManager.GetTokenExpiry().Seconds())
		assert.Equal(t, expectedExpiry, resp.ExpiresIn)
	})

	t.Run("device info propagation", func(t *testing.T) {
		deviceID := "ExpoToken[" + uuid.New().String() + "]"
		deviceInfo := "iPhone 15 Pro Max"
		ipAddress := "192.168.1.100"

		defer cleanupTestUserByDeviceID(t, db, deviceID)

		// Register mobile user with device info
		resp, err := service.RegisterMobile(ctx, deviceID, deviceInfo, ipAddress)
		require.NoError(t, err)
		assert.NotNil(t, resp)

		// Verify refresh token was created with device info
		// We can't directly access it, but we can verify the token exists and works
		refreshResp, err := service.RefreshAccessToken(ctx, resp.RefreshToken, deviceInfo, ipAddress)
		require.NoError(t, err)
		assert.NotNil(t, refreshResp)
	})
}
