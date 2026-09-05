//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"

	"github.com/andrew-craig/cairn-reader/pkg/auth"
	contentMigrations "github.com/andrew-craig/cairn-reader/services/read/content/migrations"
	contentSelfhost "github.com/andrew-craig/cairn-reader/services/read/content/selfhost"
	emailMigrations "github.com/andrew-craig/cairn-reader/services/read/email/migrations"
	emailSelfhost "github.com/andrew-craig/cairn-reader/services/read/email/selfhost"
)

type testDBConn struct {
	host     string
	port     int
	user     string
	password string
	sslMode  string
}

func testDBConnInfo(t *testing.T) testDBConn {
	t.Helper()

	port, err := strconv.Atoi(envOrDefault("TEST_DB_PORT", "5432"))
	if err != nil {
		t.Fatalf("invalid TEST_DB_PORT: %v", err)
	}

	return testDBConn{
		host:     envOrDefault("TEST_DB_HOST", "localhost"),
		port:     port,
		user:     envOrDefault("TEST_DB_USER", "postgres"),
		password: envOrDefault("TEST_DB_PASSWORD", "postgres"),
		sslMode:  envOrDefault("TEST_DB_SSL_MODE", "disable"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// createTestDatabase creates a uniquely-named database and registers its removal.
func createTestDatabase(t *testing.T, conn testDBConn, prefix string) string {
	t.Helper()

	admin, err := sql.Open("postgres", fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		conn.host, conn.port, conn.user, conn.password, conn.sslMode,
	))
	if err != nil {
		t.Skipf("database not available: %v", err)
	}
	defer func() { _ = admin.Close() }()

	if err := admin.Ping(); err != nil {
		t.Skipf("database not available: %v", err)
	}

	name := fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		t.Fatalf("create database %s: %v", name, err)
	}

	t.Cleanup(func() {
		dropAdmin, err := sql.Open("postgres", fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
			conn.host, conn.port, conn.user, conn.password, conn.sslMode,
		))
		if err != nil {
			return
		}
		defer func() { _ = dropAdmin.Close() }()
		_, _ = dropAdmin.Exec("DROP DATABASE IF EXISTS " + name + " WITH (FORCE)")
	})

	return name
}

// TestMasterRouter_EmailInternalSendersRoute exercises the composed selfhost router —
// the master router's middleware plus the content and email services mounted on it in
// the same order main() mounts them.
//
// The content service mounts a catch-all at /api/v1/internal and /api/v1/internal/*,
// so unless the email service mounts its own internal prefix explicitly, requests to
// GET /api/v1/internal/source/email/user/{user_id}/senders reach the content router
// and 404. That endpoint is what the content service's subscription aggregator calls
// over loopback HTTP to list a user's newsletter senders.
//
// This asserts the whole path end to end: correct route resolution (not 404), the
// master router's X-Forwarded-Proto injection satisfying the email router's
// RequireHTTPS (not 403), and internal API key auth (not 401).
func TestMasterRouter_EmailInternalSendersRoute(t *testing.T) {
	conn := testDBConnInfo(t)
	contentDBName := createTestDatabase(t, conn, "selfhost_content_test")
	emailDBName := createTestDatabase(t, conn, "selfhost_email_test")

	contentCfg := contentSelfhost.ContentConfig{
		DBHost:         conn.host,
		DBPort:         conn.port,
		DBUser:         conn.user,
		DBPassword:     conn.password,
		DBName:         contentDBName,
		DBSSLMode:      conn.sslMode,
		InternalAPIKey: testInternalAPIKey,
	}
	if err := contentSelfhost.RunMigrations(contentCfg, contentMigrations.FS); err != nil {
		t.Fatalf("content migrations: %v", err)
	}

	emailCfg := emailSelfhost.EmailConfig{
		DBHost:         conn.host,
		DBPort:         conn.port,
		DBUser:         conn.user,
		DBPassword:     conn.password,
		DBName:         emailDBName,
		DBSSLMode:      conn.sslMode,
		EmailDomain:    "read.example.com",
		InternalAPIKey: testInternalAPIKey,
	}
	if err := emailSelfhost.RunEmailMigrations(emailCfg, emailMigrations.FS); err != nil {
		t.Fatalf("email migrations: %v", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newMasterRouter(logger)

	authMiddleware := auth.NewMiddleware(auth.NewValidator(&privateKey.PublicKey))
	internalAuthMiddleware := auth.NewInternalAuthMiddleware(testInternalAPIKey)

	// Mount order mirrors main(): content first, then email.
	_, contentCleanup, err := contentSelfhost.Mount(contentCfg, router, authMiddleware, internalAuthMiddleware, logger)
	if err != nil {
		t.Fatalf("mount content: %v", err)
	}
	t.Cleanup(contentCleanup)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	_, emailCleanup, err := emailSelfhost.MountEmail(ctx, emailCfg, router, &privateKey.PublicKey, logger)
	if err != nil {
		t.Fatalf("mount email: %v", err)
	}
	t.Cleanup(emailCleanup)

	const userID = "11111111-1111-1111-1111-111111111111"
	path := "/api/v1/internal/source/email/user/" + userID + "/senders"

	t.Run("valid internal API key returns the email service response", func(t *testing.T) {
		// No X-Forwarded-Proto and no TLS, exactly as the content service's loopback
		// client sends it (services/read/content/internal/service/email_ingest_client.go).
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set(auth.InternalAPIKeyHeader, testInternalAPIKey)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200 (404 = content catch-all swallowed the route, "+
				"403 = RequireHTTPS rejected the loopback call); body: %s", rr.Code, rr.Body.String())
		}

		var body struct {
			Data []map[string]any `json:"data"`
		}
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode response: %v; body: %s", err, rr.Body.String())
		}
		if body.Data == nil {
			t.Errorf("response has no \"data\" array, so it did not come from the email service; body: %s", rr.Body.String())
		}
	})

	t.Run("missing internal API key is rejected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want 401; body: %s", rr.Code, rr.Body.String())
		}
	})
}

const testInternalAPIKey = "test-internal-api-key"
