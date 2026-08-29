package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pkgauth "github.com/andrew-craig/cairn-reader/pkg/auth"
	internalAuth "github.com/andrew-craig/cairn-reader/services/users/internal/auth"
	"github.com/andrew-craig/cairn-reader/services/users/internal/services"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// stubAuthService is a no-op AuthService used to build a router without a live DB.
type stubAuthService struct{}

func (s *stubAuthService) Register(context.Context, string, string) (*services.AuthResponse, error) {
	return nil, nil
}
func (s *stubAuthService) RegisterMobile(context.Context, string, string, string) (*services.AuthResponse, error) {
	return nil, nil
}
func (s *stubAuthService) Login(context.Context, string, string, string, string) (*services.AuthResponse, error) {
	return nil, nil
}
func (s *stubAuthService) LoginMobile(context.Context, string, string, string) (*services.AuthResponse, error) {
	return nil, nil
}
func (s *stubAuthService) RefreshAccessToken(context.Context, string, string, string) (*services.AuthResponse, error) {
	return nil, nil
}
func (s *stubAuthService) Logout(context.Context, string) error { return nil }
func (s *stubAuthService) LogoutAll(context.Context, uuid.UUID) error {
	return nil
}

// TestForgotPasswordAndResetPasswordRoutesRemoved proves the password-reset routes
// (which panicked in the self-host build per H4 and were never delivered per H3) are
// no longer registered on the production router at all, rather than left half-working.
func TestForgotPasswordAndResetPasswordRoutesRemoved(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	router := Router(RouterConfig{
		AuthService:              &stubAuthService{},
		EmailVerificationService: &mockEmailVerificationService{},
		Validator:                pkgauth.NewValidator(&privateKey.PublicKey),
		Logger:                   slog.Default(),
	})

	for _, path := range []string{"/api/v1/auth/forgot-password", "/api/v1/auth/reset-password"} {
		req := httptest.NewRequest("POST", path, nil)
		req.Header.Set("X-Forwarded-Proto", "https")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, 404, w.Code, "expected %s to be unregistered (404), got %d", path, w.Code)
	}
}

// TestKeyRotationReachesRouterValidator is the regression test for task_41e2:
// the users service signs tokens with a rotated key
// (KeyRotationManager.OnRotation -> jwtManager.UpdateKeys, see
// cmd/user-service/main.go) but, before this fix, its own router built the
// auth middleware's *pkgauth.Validator from a one-time public key snapshot
// taken at startup (router.go used to call
// auth.NewValidator(config.JWTManager.GetPublicKey()) inline), so the
// service could never verify the tokens it had just issued. This test
// proves the router now shares a live *pkgauth.Validator with the caller,
// so pushing a rotated key via UpdatePublicKey (exactly what the OnRotation
// callback does) is enough to recover verification with no restart and no
// new router.
func TestKeyRotationReachesRouterValidator(t *testing.T) {
	privA, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	jwtManager := internalAuth.NewJWTManagerWithConfig(internalAuth.JWTManagerConfig{
		PrivateKey: privA,
		PublicKey:  &privA.PublicKey,
		Expiry:     time.Hour,
		Issuer:     "cairn-user-service",
		Audience:   "cairn-api",
	})
	validator := pkgauth.NewValidator(&privA.PublicKey)

	router := Router(RouterConfig{
		AuthService:              &stubAuthService{},
		EmailVerificationService: &mockEmailVerificationService{},
		Validator:                validator,
		Logger:                   slog.Default(),
	})

	userID := uuid.New()
	tokenA, err := jwtManager.GenerateToken(userID)
	require.NoError(t, err)
	requireLogoutAllStatus(t, router, tokenA, http.StatusOK)

	// Rotate the signing key exactly as KeyRotationManager.rotateKeys does:
	// the manager fetches a new pair from Vault and the OnRotation callback
	// hands it to the JWT manager. A token signed with the new key is now
	// rejected by the router's still-stale validator - this is the bug.
	privB, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	jwtManager.UpdateKeys(privB, &privB.PublicKey)
	tokenB, err := jwtManager.GenerateToken(userID)
	require.NoError(t, err)
	requireLogoutAllStatus(t, router, tokenB, http.StatusUnauthorized)

	// The rest of the OnRotation callback pushes the new public key into the
	// same validator instance the router is already using - no restart, no
	// new router or middleware.
	validator.UpdatePublicKey(&privB.PublicKey)
	requireLogoutAllStatus(t, router, tokenB, http.StatusOK)
}

func requireLogoutAllStatus(t *testing.T, router http.Handler, token string, want int) {
	t.Helper()
	req := httptest.NewRequest("POST", "/api/v1/auth/logout-all", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	require.Equal(t, want, w.Code, "logout-all response body: %s", w.Body.String())
}
