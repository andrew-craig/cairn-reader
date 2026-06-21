package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	internalAuth "github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/database"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestPasswordResetHandler creates an auth handler wired with a password reset repo.
func setupTestPasswordResetHandler(t *testing.T) (*AuthHandler, *database.DB, func()) {
	db := setupTestDB(t)

	passwordHasher := internalAuth.NewPasswordHasher(10)
	privateKey, publicKey := generateTestRSAKeys(t)
	jwtManager := internalAuth.NewJWTManagerWithConfig(internalAuth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Expiry:     60 * time.Minute,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
	})

	refreshTokenRepo := database.NewRefreshTokenRepository(db)
	refreshTokenService := internalAuth.NewRefreshTokenService(refreshTokenRepo, 30*24*time.Hour)
	userRepo := database.NewUserRepository(db)
	passwordResetTokenRepo := database.NewPasswordResetTokenRepository(db)

	authService := services.NewAuthService(services.AuthServiceConfig{
		UserRepo:               userRepo,
		RefreshTokenService:    refreshTokenService,
		PasswordResetTokenRepo: passwordResetTokenRepo,
		JWTManager:             jwtManager,
		PasswordHasher:         passwordHasher,
		PasswordMinLength:      8,
		RequireComplexity:      true,
	})

	handler := NewAuthHandler(authService, &mockEmailVerificationService{})
	cleanup := func() { db.Close() }
	return handler, db, cleanup
}

func TestForgotPassword(t *testing.T) {
	handler, db, cleanup := setupTestPasswordResetHandler(t)
	defer cleanup()

	t.Run("returns 200 for registered email", func(t *testing.T) {
		email := "forgot_handler@example.com"
		defer cleanupTestUser(t, db, registerTestUser(t, handler, email, "ValidPass123!"))

		body, _ := json.Marshal(ForgotPasswordRequest{Email: email})
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ForgotPassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "message")
	})

	t.Run("returns 200 for unknown email (no info leak)", func(t *testing.T) {
		body, _ := json.Marshal(ForgotPasswordRequest{Email: "nobody@example.com"})
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ForgotPassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("returns 400 for missing email", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{})
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ForgotPassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("returns 400 for invalid JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", bytes.NewBufferString("not json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ForgotPassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestResetPassword(t *testing.T) {
	handler, db, cleanup := setupTestPasswordResetHandler(t)
	defer cleanup()

	// Helper: insert a known token directly into the database for a given user.
	insertResetToken := func(t *testing.T, email string, expiry time.Duration) string {
		t.Helper()
		ctx := t.Context()

		tokenSvc := internalAuth.NewRefreshTokenService(database.NewRefreshTokenRepository(db), time.Hour)
		rawTok, hash, err := tokenSvc.GenerateToken()
		require.NoError(t, err)

		_, err = db.Pool.Exec(ctx, `
			INSERT INTO password_reset_tokens (id, user_id, token_hash, expires_at, created_at)
			SELECT gen_random_uuid(), id, $2, $3, NOW()
			FROM users WHERE email = $1
		`, email, hash, time.Now().UTC().Add(expiry))
		require.NoError(t, err)
		return rawTok
	}

	t.Run("valid token returns 200 and updates password", func(t *testing.T) {
		email := "reset_handler_valid@example.com"
		userID := registerTestUser(t, handler, email, "OldPass123!")
		defer cleanupTestUser(t, db, userID)

		rawToken := insertResetToken(t, email, time.Hour)

		body, _ := json.Marshal(ResetPasswordRequest{Token: rawToken, NewPassword: "NewPass456!"})
		req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "message")
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		body, _ := json.Marshal(ResetPasswordRequest{Token: "invalid-token", NewPassword: "NewPass456!"})
		req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("expired token returns 401", func(t *testing.T) {
		email := "reset_handler_expired@example.com"
		userID := registerTestUser(t, handler, email, "OldPass123!")
		defer cleanupTestUser(t, db, userID)

		rawToken := insertResetToken(t, email, -time.Hour) // already expired

		body, _ := json.Marshal(ResetPasswordRequest{Token: rawToken, NewPassword: "NewPass456!"})
		req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("missing fields returns 400", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{"token": "tok"})
		req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("weak password returns 400", func(t *testing.T) {
		email := "reset_handler_weak@example.com"
		userID := registerTestUser(t, handler, email, "OldPass123!")
		defer cleanupTestUser(t, db, userID)

		rawToken := insertResetToken(t, email, time.Hour)

		body, _ := json.Marshal(ResetPasswordRequest{Token: rawToken, NewPassword: "weak"})
		req := httptest.NewRequest(http.MethodPost, "/auth/reset-password", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ResetPassword(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

// registerTestUser registers a user via the handler and returns the user ID.
func registerTestUser(t *testing.T, handler *AuthHandler, email, password string) uuid.UUID {
	t.Helper()

	body, _ := json.Marshal(RegisterRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp services.AuthResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)

	return resp.User.ID
}
