package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func generateTestKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

func generateTestToken(t *testing.T, privateKey *rsa.PrivateKey, userID uuid.UUID, issuer, audience string, expiry time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := &auth.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tokenString, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatalf("Failed to sign token: %v", err)
	}
	return tokenString
}

func TestNewJWTAuth(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	m := NewJWTAuth(publicKey)
	assert.NotNil(t, m)
	assert.NotNil(t, m.Middleware)
}

func TestNewJWTAuthFromValidator(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	validator := auth.NewValidator(publicKey)
	m := NewJWTAuthFromValidator(validator)
	assert.NotNil(t, m)
	assert.NotNil(t, m.Middleware)
}

func TestJWTAuth_RequireAuth_ValidToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	m := NewJWTAuth(publicKey)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	called := false
	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		contextUserID, ok := auth.GetUserIDFromContext(r.Context())
		assert.True(t, ok, "expected user ID in context")
		assert.Equal(t, userID, contextUserID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/api/v1/source/email/user/123/address", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should be called with valid token")
}

func TestJWTAuth_RequireAuth_MissingToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	m := NewJWTAuth(publicKey)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with missing token")
	}))

	req := httptest.NewRequest("GET", "/api/v1/source/email/user/123/address", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}

func TestJWTAuth_RequireAuth_InvalidToken(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	m := NewJWTAuth(publicKey)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid token")
	}))

	req := httptest.NewRequest("GET", "/api/v1/source/email/user/123/address", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_RequireAuth_ExpiredToken(t *testing.T) {
	privateKey, publicKey := generateTestKeys(t)
	m := NewJWTAuth(publicKey)
	userID := uuid.New()

	token := generateTestToken(t, privateKey, userID, "cairn-user-service", "cairn-api", -1*time.Hour)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with expired token")
	}))

	req := httptest.NewRequest("GET", "/api/v1/source/email/user/123/address", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_RequireAuth_WrongSigningKey(t *testing.T) {
	wrongKey, _ := generateTestKeys(t)
	_, publicKey := generateTestKeys(t) // different key pair
	m := NewJWTAuth(publicKey)
	userID := uuid.New()

	// Sign with wrong key
	token := generateTestToken(t, wrongKey, userID, "cairn-user-service", "cairn-api", 1*time.Hour)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with wrong signing key")
	}))

	req := httptest.NewRequest("GET", "/api/v1/source/email/user/123/address", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestJWTAuth_RequireAuth_InvalidAuthHeader(t *testing.T) {
	_, publicKey := generateTestKeys(t)
	m := NewJWTAuth(publicKey)

	handler := m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with invalid auth header")
	}))

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "token123"},
		{"wrong scheme", "Basic token123"},
		{"empty bearer", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}
