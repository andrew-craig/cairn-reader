package testutil

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// TestEnv holds the test environment dependencies
type TestEnv struct {
	DB           *database.DB
	VaultClient  *auth.VaultClient
	JWTManager   *auth.JWTManager
	UserRepo     database.UserRepository
	TokenRepo    database.RefreshTokenRepository
	PasswordHash *auth.PasswordHasher
	TokenService *auth.RefreshTokenService
}

// SetupTestEnvironment initializes all dependencies for integration tests
func SetupTestEnvironment(t *testing.T) *TestEnv {
	// Load test database configuration
	dbConfig := &database.Config{
		Host:            getEnv("DB_HOST", "localhost"),
		Port:            getEnv("DB_PORT", "5433"),
		User:            getEnv("DB_USER", "cairn_test"),
		Password:        getEnv("DB_PASSWORD", "test_password"),
		Database:        getEnv("DB_NAME", "cairn_users_test"),
		SSLMode:         getEnv("DB_SSLMODE", "disable"),
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	}

	// Connect to database
	db, err := database.New(dbConfig)
	require.NoError(t, err, "Failed to connect to test database")

	// Initialize Vault client (optional for unit tests)
	var vaultClient *auth.VaultClient
	vaultConfig := &auth.VaultConfig{
		Address: getEnv("VAULT_ADDR", "http://localhost:8201"),
		Token:   getEnv("VAULT_TOKEN", "test-root-token"),
	}

	vaultClient, err = auth.NewVaultClient(vaultConfig)
	if err != nil {
		t.Logf("Warning: Vault not available, using generated keys: %v", err)
	}

	// Load JWT keys from Vault or generate test keys
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey

	if vaultClient != nil {
		privateKey, publicKey, err = vaultClient.GetJWTKeys(
			getEnv("JWT_PRIVATE_KEY_PATH", "secret/data/cairn/jwt/private"),
			getEnv("JWT_PUBLIC_KEY_PATH", "secret/data/cairn/jwt/public"),
		)
		if err != nil {
			t.Logf("Warning: Could not load keys from Vault, generating test keys: %v", err)
			privateKey, publicKey = generateTestRSAKeys(t)
		}
	} else {
		privateKey, publicKey = generateTestRSAKeys(t)
	}

	// Initialize JWT manager
	jwtManager := auth.NewJWTManagerWithConfig(auth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     15 * time.Minute,
		Issuer:     "cairn-test",
		Audience:   "cairn-api-test",
	})

	// Initialize repositories
	userRepo := database.NewUserRepository(db)
	tokenRepo := database.NewRefreshTokenRepository(db)

	// Initialize services
	passwordHash := auth.NewPasswordHasher(10) // Lower cost for faster tests
	tokenService := auth.NewRefreshTokenService(tokenRepo, 7*24*time.Hour)

	return &TestEnv{
		DB:           db,
		VaultClient:  vaultClient,
		JWTManager:   jwtManager,
		UserRepo:     userRepo,
		TokenRepo:    tokenRepo,
		PasswordHash: passwordHash,
		TokenService: tokenService,
	}
}

// generateTestRSAKeys generates RSA key pairs for testing
func generateTestRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err, "Failed to generate RSA key pair")
	return privateKey, &privateKey.PublicKey
}

// Cleanup cleans up test environment
func (env *TestEnv) Cleanup(t *testing.T) {
	if env.DB != nil {
		env.DB.Close()
	}
}

// CleanupTestUser removes a test user and all associated data
func (env *TestEnv) CleanupTestUser(t *testing.T, userID uuid.UUID) {
	ctx := context.Background()

	// Delete refresh tokens first (foreign key)
	_, err := env.DB.Pool.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup refresh tokens: %v", err)
	}

	// Delete user
	_, err = env.DB.Pool.Exec(ctx, "DELETE FROM users WHERE id = $1", userID)
	if err != nil {
		t.Logf("Warning: failed to cleanup test user: %v", err)
	}
}

// CleanupAllTestData removes all data from test database
func (env *TestEnv) CleanupAllTestData(t *testing.T) {
	ctx := context.Background()

	// Delete all refresh tokens
	_, err := env.DB.Pool.Exec(ctx, "DELETE FROM refresh_tokens")
	if err != nil {
		t.Logf("Warning: failed to cleanup refresh tokens: %v", err)
	}

	// Delete all users
	_, err = env.DB.Pool.Exec(ctx, "DELETE FROM users")
	if err != nil {
		t.Logf("Warning: failed to cleanup users: %v", err)
	}
}

// CreateTestUser creates a test user with email/password
func (env *TestEnv) CreateTestUser(t *testing.T, email, password string) (uuid.UUID, string) {
	ctx := context.Background()

	// Hash password
	passwordHash, err := env.PasswordHash.HashPassword(password)
	require.NoError(t, err, "Failed to hash password")

	// Create user
	user, err := env.UserRepo.CreateUser(ctx, email, passwordHash)
	require.NoError(t, err, "Failed to create test user")

	return user.ID, passwordHash
}

// CreateTestMobileUser creates a test user with device ID
func (env *TestEnv) CreateTestMobileUser(t *testing.T, deviceID string) uuid.UUID {
	ctx := context.Background()

	user, err := env.UserRepo.CreateMobileUser(ctx, deviceID)
	require.NoError(t, err, "Failed to create mobile test user")

	return user.ID
}

// GenerateTestTokens generates access and refresh tokens for a user
func (env *TestEnv) GenerateTestTokens(t *testing.T, userID uuid.UUID, deviceInfo, ipAddress *string) (string, string) {
	ctx := context.Background()

	// Generate access token
	accessToken, err := env.JWTManager.GenerateToken(userID)
	require.NoError(t, err, "Failed to generate access token")

	// Generate refresh token
	refreshToken, tokenHash, err := env.TokenService.GenerateToken()
	require.NoError(t, err, "Failed to generate refresh token")

	// Store refresh token
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	_, err = env.TokenRepo.CreateRefreshToken(ctx, userID, tokenHash, expiresAt, deviceInfo, ipAddress, nil)
	require.NoError(t, err, "Failed to store refresh token")

	return accessToken, refreshToken
}

// WaitForDatabase waits for database to be ready
func WaitForDatabase(t *testing.T, maxWait time.Duration) {
	start := time.Now()
	for {
		dbConfig := &database.Config{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5433"),
			User:     getEnv("DB_USER", "cairn_test"),
			Password: getEnv("DB_PASSWORD", "test_password"),
			Database: getEnv("DB_NAME", "cairn_users_test"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		}

		db, err := database.New(dbConfig)
		if err == nil {
			db.Close()
			return
		}

		if time.Since(start) > maxWait {
			t.Fatalf("Database not ready after %v: %v", maxWait, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// WaitForVault waits for Vault to be ready
func WaitForVault(t *testing.T, maxWait time.Duration) {
	start := time.Now()
	for {
		vaultConfig := &auth.VaultConfig{
			Address: getEnv("VAULT_ADDR", "http://localhost:8201"),
			Token:   getEnv("VAULT_TOKEN", "test-root-token"),
		}

		client, err := auth.NewVaultClient(vaultConfig)
		if err == nil {
			// Try to read a key to verify it's fully initialized
			_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, _, err = client.GetJWTKeys(
				getEnv("JWT_PRIVATE_KEY_PATH", "secret/data/cairn/jwt/private"),
				getEnv("JWT_PUBLIC_KEY_PATH", "secret/data/cairn/jwt/public"),
			)
			cancel()

			if err == nil {
				return
			}
		}

		if time.Since(start) > maxWait {
			t.Fatalf("Vault not ready after %v: %v", maxWait, err)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// getEnv retrieves environment variable or returns default
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RandomEmail generates a random email address for testing
func RandomEmail() string {
	return fmt.Sprintf("test-%s@example.com", uuid.New().String()[:8])
}

// RandomDeviceID generates a random device ID for testing
func RandomDeviceID() string {
	return fmt.Sprintf("ExponentPushToken[%s]", uuid.New().String())
}
