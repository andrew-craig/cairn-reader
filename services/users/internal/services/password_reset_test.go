package services

import (
	"context"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestAuthServiceWithReset creates an auth service that includes the password reset repo.
func setupTestAuthServiceWithReset(t *testing.T) (AuthService, *database.DB) {
	db := setupTestDB(t)

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
	passwordResetTokenRepo := database.NewPasswordResetTokenRepository(db)

	service := NewAuthService(AuthServiceConfig{
		UserRepo:               userRepo,
		RefreshTokenService:    refreshTokenService,
		PasswordResetTokenRepo: passwordResetTokenRepo,
		JWTManager:             jwtManager,
		PasswordHasher:         passwordHasher,
		PasswordMinLength:      8,
		RequireComplexity:      true,
	})

	return service, db
}

func TestAuthService_ForgotPassword(t *testing.T) {
	service, db := setupTestAuthServiceWithReset(t)
	defer db.Close()
	ctx := context.Background()

	t.Run("valid email creates token", func(t *testing.T) {
		email := "forgot@example.com"
		defer cleanupTestUserByEmail(t, db, email)

		// Register a user first
		_, err := service.Register(ctx, email, "ValidPass123!")
		require.NoError(t, err)

		// ForgotPassword should succeed
		err = service.ForgotPassword(ctx, email)
		assert.NoError(t, err)

		// Verify token was created in DB
		var count int
		err = db.Pool.QueryRow(ctx, `
			SELECT COUNT(*) FROM password_reset_tokens prt
			JOIN users u ON prt.user_id = u.id
			WHERE u.email = $1 AND prt.used_at IS NULL AND prt.expires_at > NOW()
		`, email).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("non-existent email returns success (no info leak)", func(t *testing.T) {
		err := service.ForgotPassword(ctx, "nonexistent@example.com")
		assert.NoError(t, err)
	})

	t.Run("empty email returns error", func(t *testing.T) {
		err := service.ForgotPassword(ctx, "")
		assert.ErrorIs(t, err, ErrInvalidInput)
	})
}

func TestAuthService_ResetPassword(t *testing.T) {
	service, db := setupTestAuthServiceWithReset(t)
	defer db.Close()
	ctx := context.Background()

	// Helper: register a user and generate a real reset token.
	setupUserAndToken := func(t *testing.T, email string) (userID interface{}, rawToken string) {
		t.Helper()

		_, err := service.Register(ctx, email, "OldPassword123!")
		require.NoError(t, err)

		err = service.ForgotPassword(ctx, email)
		require.NoError(t, err)

		// Pull the token hash from the DB, then derive the raw token by regenerating
		// via the repo directly to avoid white-box coupling. Instead, we use the
		// ForgotPassword log path which logs the raw token. Since we can't intercept
		// slog here, we use the database directly to craft a token we control.
		//
		// Insert a known token directly.
		tokenSvc := auth.NewRefreshTokenService(database.NewRefreshTokenRepository(db), time.Hour)
		rawTok, hash, err := tokenSvc.GenerateToken()
		require.NoError(t, err)

		var uid interface{}
		err = db.Pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&uid)
		require.NoError(t, err)

		_, err = db.Pool.Exec(ctx, `
			INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		`, uid, hash, time.Now().UTC().Add(time.Hour))
		require.NoError(t, err)

		return uid, rawTok
	}

	t.Run("valid token updates password", func(t *testing.T) {
		email := "reset_valid@example.com"
		defer cleanupTestUserByEmail(t, db, email)

		_, rawToken := setupUserAndToken(t, email)

		err := service.ResetPassword(ctx, rawToken, "NewPassword456!")
		assert.NoError(t, err)

		// Verify new password works by attempting to login.
		resp, err := service.Login(ctx, email, "NewPassword456!", "", "")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
	})

	t.Run("expired token fails", func(t *testing.T) {
		email := "reset_expired@example.com"
		defer cleanupTestUserByEmail(t, db, email)

		_, err := service.Register(ctx, email, "OldPassword123!")
		require.NoError(t, err)

		tokenSvc := auth.NewRefreshTokenService(database.NewRefreshTokenRepository(db), time.Hour)
		rawTok, hash, err := tokenSvc.GenerateToken()
		require.NoError(t, err)

		var uid interface{}
		err = db.Pool.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&uid)
		require.NoError(t, err)

		// Insert an already-expired token.
		_, err = db.Pool.Exec(ctx, `
			INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
			VALUES (gen_random_uuid(), $1, $2, $3, NOW())
		`, uid, hash, time.Now().UTC().Add(-time.Hour))
		require.NoError(t, err)

		err = service.ResetPassword(ctx, rawTok, "NewPassword456!")
		assert.ErrorIs(t, err, ErrInvalidOrExpiredToken)
	})

	t.Run("already-used token fails", func(t *testing.T) {
		email := "reset_used@example.com"
		defer cleanupTestUserByEmail(t, db, email)

		_, rawToken := setupUserAndToken(t, email)

		// Use the token once successfully.
		err := service.ResetPassword(ctx, rawToken, "NewPassword456!")
		require.NoError(t, err)

		// Using it again should fail.
		err = service.ResetPassword(ctx, rawToken, "AnotherPassword789!")
		assert.ErrorIs(t, err, ErrInvalidOrExpiredToken)
	})

	t.Run("invalid token fails", func(t *testing.T) {
		err := service.ResetPassword(ctx, "totally-invalid-token", "NewPassword456!")
		assert.ErrorIs(t, err, ErrInvalidOrExpiredToken)
	})

	t.Run("reset revokes refresh tokens", func(t *testing.T) {
		email := "reset_revoke@example.com"
		defer cleanupTestUserByEmail(t, db, email)

		loginResp, err := service.Register(ctx, email, "OldPassword123!")
		require.NoError(t, err)
		refreshToken := loginResp.RefreshToken

		_, rawToken := setupUserAndToken(t, email)

		err = service.ResetPassword(ctx, rawToken, "NewPassword456!")
		require.NoError(t, err)

		// The old refresh token should be revoked now.
		_, err = service.RefreshAccessToken(ctx, refreshToken, "", "")
		assert.Error(t, err)
	})

	t.Run("empty token returns error", func(t *testing.T) {
		err := service.ResetPassword(ctx, "", "NewPassword456!")
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("empty password returns error", func(t *testing.T) {
		err := service.ResetPassword(ctx, "sometoken", "")
		assert.ErrorIs(t, err, ErrInvalidInput)
	})

	t.Run("weak new password is rejected", func(t *testing.T) {
		email := "reset_weak@example.com"
		defer cleanupTestUserByEmail(t, db, email)

		_, rawToken := setupUserAndToken(t, email)

		err := service.ResetPassword(ctx, rawToken, "weak")
		assert.ErrorIs(t, err, ErrWeakPassword)
	})
}
