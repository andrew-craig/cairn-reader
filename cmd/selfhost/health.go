package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type healthChecker struct {
	checks map[string]*sql.DB
}

func newHealthChecker() *healthChecker {
	return &healthChecker{
		checks: make(map[string]*sql.DB),
	}
}

func (h *healthChecker) addDB(name string, db *sql.DB) {
	h.checks[name] = db
}

func (h *healthChecker) livenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"service": "cairn-selfhost",
	})
}

func (h *healthChecker) readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	checks := make(map[string]string)
	allOK := true

	for name, db := range h.checks {
		if err := db.PingContext(ctx); err != nil {
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
