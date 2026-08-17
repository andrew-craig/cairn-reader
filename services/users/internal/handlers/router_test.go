package handlers

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	internalAuth "github.com/cairn-app/cairn-reader/services/users/internal/auth"
	"github.com/cairn-app/cairn-reader/services/users/internal/services"
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

	jwtManager := internalAuth.NewJWTManagerWithConfig(internalAuth.JWTManagerConfig{
		PrivateKey: privateKey,
		PublicKey:  &privateKey.PublicKey,
		Expiry:     time.Hour,
		Issuer:     "test-issuer",
		Audience:   "test-audience",
	})

	router := Router(RouterConfig{
		AuthService:              &stubAuthService{},
		EmailVerificationService: &mockEmailVerificationService{},
		JWTManager:               jwtManager,
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
