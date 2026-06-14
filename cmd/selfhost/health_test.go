package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthChecker_Liveness(t *testing.T) {
	h := newHealthChecker()

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	rr := httptest.NewRecorder()

	h.livenessHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %q, want ok", body["status"])
	}
	if body["service"] != "cairn-selfhost" {
		t.Errorf("service field = %q, want cairn-selfhost", body["service"])
	}
	if body["version"] != version {
		t.Errorf("version field = %q, want %q", body["version"], version)
	}
}

func TestHealthChecker_Readiness_NoChecks(t *testing.T) {
	h := newHealthChecker()

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	rr := httptest.NewRecorder()

	h.readinessHandler(rr, req)

	// With no registered DB checks, readiness should pass.
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("could not decode body: %v", err)
	}
	if body["status"] != "healthy" {
		t.Errorf("status field = %v, want healthy", body["status"])
	}
}

func TestNewHealthChecker(t *testing.T) {
	h := newHealthChecker()
	if h == nil {
		t.Fatal("newHealthChecker() returned nil")
	}
	if h.checks == nil {
		t.Error("checks map not initialized")
	}
	if len(h.checks) != 0 {
		t.Errorf("expected empty checks map, got %d entries", len(h.checks))
	}
}
