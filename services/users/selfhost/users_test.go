package selfhost

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	userDB "github.com/cairn-app/cairn-reader/services/users/internal/database"
	userMigrations "github.com/cairn-app/cairn-reader/services/users/migrations"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
)

func testUsersConfig(t *testing.T) UsersConfig {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	return UsersConfig{
		DBHost:           getEnvOrDefault("TEST_DB_HOST", "localhost"),
		DBPort:           getEnvOrDefault("TEST_DB_PORT", "5432"),
		DBUser:           getEnvOrDefault("TEST_DB_USER", "postgres"),
		DBPassword:       getEnvOrDefault("TEST_DB_PASSWORD", "postgres"),
		DBName:           getEnvOrDefault("TEST_DB_NAME", "cairn_test"),
		DBSSLMode:        getEnvOrDefault("TEST_DB_SSLMODE", "disable"),
		PrivateKey:       privateKey,
		PublicKey:        &privateKey.PublicKey,
		JWTAccessExpiry:  time.Hour,
		JWTRefreshExpiry: 24 * time.Hour,
		BcryptCost:       10,
	}
}

func getEnvOrDefault(_, defaultValue string) string {
	return defaultValue
}

// TestMountUsers_VerifyEmailDoesNotPanic reproduces H4: the self-host build never
// wired EmailVerificationService/PasswordResetTokenRepo into AuthServiceConfig /
// RouterConfig, so hitting /verify-email or /forgot-password through the self-host
// wiring nil-pointer panicked. Password-reset was removed outright (see
// router_test.go in internal/handlers), so this asserts /verify-email now returns a
// coherent, non-panicking response through the exact wiring self-host uses.
func TestMountUsers_VerifyEmailDoesNotPanic(t *testing.T) {
	cfg := testUsersConfig(t)

	if err := RunUsersMigrations(cfg, userMigrations.FS); err != nil {
		t.Skip("database not available for testing:", err)
	}

	// Confirm the DB is actually reachable (RunUsersMigrations can no-op against an
	// already-migrated DB even when the connection would otherwise fail later).
	dbCfg := &userDB.Config{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
		Database: cfg.DBName,
		SSLMode:  cfg.DBSSLMode,
	}
	probe, err := userDB.New(dbCfg)
	if err != nil {
		t.Skip("database not available for testing:", err)
	}
	probe.Close()

	router := chi.NewRouter()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cleanup, err := MountUsers(ctx, cfg, router, slog.Default())
	require.NoError(t, err)
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/auth/verify-email?token=bogus-token", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	w := httptest.NewRecorder()

	require.NotPanics(t, func() {
		router.ServeHTTP(w, req)
	})

	require.Equal(t, 400, w.Code, "expected a coherent 400 for an invalid token, not a recovered panic")

	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, "bad_request", body["error"],
		"an \"internal_error\" code would mean the request panicked and was recovered, not handled; response body: %s", w.Body.String())
}
