//go:build integration

package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

// TestReadiness_DetectsDownedUsersDB reproduces the "readiness probe lies"
// finding (task_fd42): before the fix, the users database was never
// registered with the selfhost health checker's addDB/addPinger, so
// /health/ready kept reporting "healthy" even after the users DB became
// unreachable — the same lie the finding calls out for explore-recommender
// and explore-fetcher.
//
// The users DB must be reachable at Mount time (userDB.New pings eagerly on
// connect, so pointing at a dead address up front just fails Mount itself,
// not the readiness probe). To reproduce an outage that happens *after* a
// clean start — the actual scenario a readiness probe exists to catch — this
// mounts against a real database and then closes that connection out from
// under the service to simulate the DB going away.
func TestReadiness_DetectsDownedUsersDB(t *testing.T) {
	conn := testDBConnInfo(t)
	dbName := createTestDatabase(t, conn, "selfhost_users_health_test")

	cfg := &Config{
		DB: DBConfig{
			Host:     conn.host,
			Port:     conn.port,
			User:     conn.user,
			Password: conn.password,
			SSLMode:  conn.sslMode,
		},
		DBNameUsers:      dbName,
		JWTAccessExpiry:  15 * time.Minute,
		JWTRefreshExpiry: 7 * 24 * time.Hour,
		BcryptCost:       4,
	}

	if err := runUsersMigrations(cfg); err != nil {
		t.Fatalf("users migrations: %v", err)
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	health := newHealthChecker()
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := chi.NewRouter()

	closer, err := mountUserService(context.Background(), cfg, router, privateKey, &privateKey.PublicKey, health, logger)
	if err != nil {
		t.Fatalf("mountUserService: %v", err)
	}

	// Sanity check: readiness is healthy while the DB is actually up.
	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()
	health.readinessHandler(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("readiness while DB is up: status = %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	// Simulate the users DB going down after a clean start.
	closer()

	req2 := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr2 := httptest.NewRecorder()
	health.readinessHandler(rr2, req2)

	if rr2.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503 once the users DB connection is closed; body: %s", rr2.Code, rr2.Body.String())
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr2.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "unhealthy" {
		t.Errorf("status field = %v, want unhealthy; body: %s", body["status"], rr2.Body.String())
	}
}
