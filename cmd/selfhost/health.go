package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

// pinger is satisfied by both *sql.DB (via sqlPinger below) and
// *pgxpool.Pool, which already implements Ping(ctx context.Context) error
// natively. This lets the same health checker cover services built on
// database/sql (content, email, rss) and services built on pgx (users,
// explore-recommender, explore-fetcher).
type pinger interface {
	Ping(ctx context.Context) error
}

// sqlPinger adapts *sql.DB's PingContext to the pinger interface.
type sqlPinger struct {
	db *sql.DB
}

func (s sqlPinger) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

type healthChecker struct {
	checks map[string]pinger
}

func newHealthChecker() *healthChecker {
	return &healthChecker{
		checks: make(map[string]pinger),
	}
}

func (h *healthChecker) addDB(name string, db *sql.DB) {
	h.checks[name] = sqlPinger{db}
}

// addPinger registers any connection pool that implements Ping(ctx) error
// (e.g. *pgxpool.Pool) for readiness checks.
func (h *healthChecker) addPinger(name string, p pinger) {
	h.checks[name] = p
}

func (h *healthChecker) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "cairn-selfhost",
		"version": version,
	})
}

func (h *healthChecker) readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allOK := true

	for name, p := range h.checks {
		if err := p.Ping(ctx); err != nil {
			checks[name] = "error"
			allOK = false
		} else {
			checks[name] = "ok"
		}
	}

	status := "healthy"
	code := http.StatusOK
	if !allOK {
		status = "unhealthy"
		code = http.StatusServiceUnavailable
	}

	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"checks": checks,
	})
}
