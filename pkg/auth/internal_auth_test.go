package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testAPIKey = "test-internal-api-key-12345"

func TestNewInternalAuthMiddleware(t *testing.T) {
	m := NewInternalAuthMiddleware(testAPIKey)
	if m == nil {
		t.Fatal("expected middleware to be created, got nil")
	}
	if m.apiKey != testAPIKey {
		t.Errorf("expected apiKey %q, got %q", testAPIKey, m.apiKey)
	}
}

func TestRequireInternalAPIKey_ValidKey(t *testing.T) {
	m := NewInternalAuthMiddleware(testAPIKey)

	called := false
	handler := m.RequireInternalAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/internal/content/user/bulk", nil)
	req.Header.Set(InternalAPIKeyHeader, testAPIKey)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestRequireInternalAPIKey_MissingKey(t *testing.T) {
	m := NewInternalAuthMiddleware(testAPIKey)

	called := false
	handler := m.RequireInternalAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/api/v1/internal/content/user/bulk", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if called {
		t.Error("handler should not be called when key is missing")
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", resp.Error)
	}
	if resp.Message != "missing internal API key" {
		t.Errorf("expected message 'missing internal API key', got %q", resp.Message)
	}
}

func TestRequireInternalAPIKey_InvalidKey(t *testing.T) {
	m := NewInternalAuthMiddleware(testAPIKey)

	called := false
	handler := m.RequireInternalAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/api/v1/internal/content/user/bulk", nil)
	req.Header.Set(InternalAPIKeyHeader, "wrong-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
	if called {
		t.Error("handler should not be called with invalid key")
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Error != "unauthorized" {
		t.Errorf("expected error 'unauthorized', got %q", resp.Error)
	}
	if resp.Message != "invalid internal API key" {
		t.Errorf("expected message 'invalid internal API key', got %q", resp.Message)
	}
}

func TestRequireInternalAPIKey_EmptyKey(t *testing.T) {
	m := NewInternalAuthMiddleware(testAPIKey)

	handler := m.RequireInternalAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with empty key")
	}))

	req := httptest.NewRequest("POST", "/api/v1/internal/content/user/bulk", nil)
	req.Header.Set(InternalAPIKeyHeader, "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", rec.Code)
	}
}
