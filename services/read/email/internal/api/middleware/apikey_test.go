package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIKeyAuth(t *testing.T) {
	repo := &MockAPIKeyRepository{}
	m := NewAPIKeyAuth(repo)
	assert.NotNil(t, m)
	assert.Equal(t, repo, m.repo)
}

func TestRequireAPIKey_ValidKey(t *testing.T) {
	repo := &MockAPIKeyRepository{
		ValidateKeyFn: func(ctx context.Context, apiKey string) (bool, error) {
			assert.Equal(t, "valid-api-key-123", apiKey)
			return true, nil
		},
	}
	m := NewAPIKeyAuth(repo)

	called := false
	handler := m.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("POST", "/api/v1/source/email/ingest", nil)
	req.Header.Set(APIKeyHeader, "valid-api-key-123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.True(t, called, "handler should be called with valid key")
}

func TestRequireAPIKey_MissingHeader(t *testing.T) {
	repo := &MockAPIKeyRepository{
		ValidateKeyFn: func(ctx context.Context, apiKey string) (bool, error) {
			t.Error("ValidateKey should not be called when header is missing")
			return false, nil
		},
	}
	m := NewAPIKeyAuth(repo)

	called := false
	handler := m.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/api/v1/source/email/ingest", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "handler should not be called when key is missing")

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "unauthorized", resp.Error)
	assert.Equal(t, "missing API key", resp.Message)
}

func TestRequireAPIKey_EmptyHeader(t *testing.T) {
	repo := &MockAPIKeyRepository{
		ValidateKeyFn: func(ctx context.Context, apiKey string) (bool, error) {
			t.Error("ValidateKey should not be called when header is empty")
			return false, nil
		},
	}
	m := NewAPIKeyAuth(repo)

	handler := m.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called with empty key")
	}))

	req := httptest.NewRequest("POST", "/api/v1/source/email/ingest", nil)
	req.Header.Set(APIKeyHeader, "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAPIKey_InvalidKey(t *testing.T) {
	repo := &MockAPIKeyRepository{
		ValidateKeyFn: func(ctx context.Context, apiKey string) (bool, error) {
			return false, nil
		},
	}
	m := NewAPIKeyAuth(repo)

	called := false
	handler := m.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/api/v1/source/email/ingest", nil)
	req.Header.Set(APIKeyHeader, "wrong-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.False(t, called, "handler should not be called with invalid key")

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "unauthorized", resp.Error)
	assert.Equal(t, "invalid API key", resp.Message)
}

func TestRequireAPIKey_RepositoryError(t *testing.T) {
	repo := &MockAPIKeyRepository{
		ValidateKeyFn: func(ctx context.Context, apiKey string) (bool, error) {
			return false, errors.New("database connection failed")
		},
	}
	m := NewAPIKeyAuth(repo)

	called := false
	handler := m.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest("POST", "/api/v1/source/email/ingest", nil)
	req.Header.Set(APIKeyHeader, "some-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.False(t, called, "handler should not be called on repo error")

	var resp errorResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "internal_error", resp.Error)
	assert.Equal(t, "failed to validate API key", resp.Message)
}

func TestRequireAPIKey_ResponseContentType(t *testing.T) {
	repo := &MockAPIKeyRepository{
		ValidateKeyFn: func(ctx context.Context, apiKey string) (bool, error) {
			return false, nil
		},
	}
	m := NewAPIKeyAuth(repo)

	handler := m.RequireAPIKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	}))

	req := httptest.NewRequest("POST", "/api/v1/source/email/ingest", nil)
	req.Header.Set(APIKeyHeader, "invalid-key")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
}
