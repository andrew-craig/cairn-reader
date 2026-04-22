//go:build integration
// +build integration

package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/database"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/testutil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"crypto/rand"
	"crypto/rsa"
)

// jwtTestEnv holds shared state for JWT auth integration tests.
type jwtTestEnv struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
}

// setupJWTTestEnv creates a test HTTP server with a generated RSA key pair —
// no Vault dependency. Cleanup is registered via t.Cleanup.
func setupJWTTestEnv(t *testing.T) *jwtTestEnv {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	testDB := testutil.SetupTestDatabase(t)
	t.Cleanup(testDB.Cleanup)

	validator := auth.NewValidator(&privateKey.PublicKey)
	authMiddleware := auth.NewMiddleware(validator)
	internalAuthMiddleware := auth.NewInternalAuthMiddleware("test-internal-key")

	dbWrapper := &database.DB{DB: testDB.DB}
	router := api.NewRouter(dbWrapper, "", authMiddleware, internalAuthMiddleware)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	return &jwtTestEnv{server: server, privateKey: privateKey}
}

// makeToken signs a JWT for userID with the given expiry (negative = already expired).
func (e *jwtTestEnv) makeToken(t *testing.T, userID uuid.UUID, expiry time.Duration) string {
	t.Helper()
	now := time.Now()
	claims := &auth.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "cairn-user-service",
			Audience:  jwt.ClaimStrings{"cairn-api"},
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(e.privateKey)
	require.NoError(t, err)
	return signed
}

// get issues a GET to path, setting X-Forwarded-Proto: https (required by RequireHTTPS middleware)
// and optionally an Authorization header.
func (e *jwtTestEnv) get(t *testing.T, path, authHeader string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, e.server.URL+path, nil)
	require.NoError(t, err)
	req.Header.Set("X-Forwarded-Proto", "https")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// TestJWTAuthIntegration exercises the full RequireAuth middleware → handler pipeline
// without Vault. The four sub-tests cover the contract for protected endpoints.
func TestJWTAuthIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	env := setupJWTTestEnv(t)
	userID := uuid.New()
	otherUserID := uuid.New()

	t.Run("ValidJWT_ReturnsOK", func(t *testing.T) {
		token := env.makeToken(t, userID, time.Hour)
		path := fmt.Sprintf("/api/v1/content/user/%s", userID)
		resp := env.get(t, path, "Bearer "+token)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("MissingAuthHeader_ReturnsUnauthorized", func(t *testing.T) {
		path := fmt.Sprintf("/api/v1/content/user/%s", userID)
		resp := env.get(t, path, "")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("ExpiredJWT_ReturnsUnauthorized", func(t *testing.T) {
		token := env.makeToken(t, userID, -time.Hour)
		path := fmt.Sprintf("/api/v1/content/user/%s", userID)
		resp := env.get(t, path, "Bearer "+token)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("WrongUserID_ReturnsForbidden", func(t *testing.T) {
		// Valid token for userID but accessing otherUserID's content → 403
		token := env.makeToken(t, userID, time.Hour)
		path := fmt.Sprintf("/api/v1/content/user/%s", otherUserID)
		resp := env.get(t, path, "Bearer "+token)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})
}

// TestVaultJWTIntegration verifies that the Vault client can load the JWT public key
// and construct a working validator. Skipped automatically when VAULT_ADDR is unset
// so it is safe to run in CI without Vault.
func TestVaultJWTIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		t.Skip("VAULT_ADDR not set — skipping Vault integration test")
	}

	publicKeyPath := os.Getenv("JWT_PUBLIC_KEY_PATH")
	if publicKeyPath == "" {
		publicKeyPath = "secret/data/jwt/public-key"
	}

	vaultClient, err := auth.NewVaultClient(&auth.VaultConfig{
		Address: vaultAddr,
		Token:   os.Getenv("VAULT_TOKEN"),
	})
	require.NoError(t, err, "failed to create Vault client")

	publicKey, err := vaultClient.GetPublicKey(publicKeyPath)
	require.NoError(t, err, "failed to fetch JWT public key from Vault")
	require.NotNil(t, publicKey)

	validator := auth.NewValidator(publicKey)
	require.NotNil(t, validator)
	t.Logf("Vault JWT public key loaded from %s (path: %s)", vaultAddr, publicKeyPath)
}
