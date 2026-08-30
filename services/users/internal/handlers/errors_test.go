package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/andrew-craig/cairn-reader/pkg/errors"
	"github.com/andrew-craig/cairn-reader/services/users/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeErrorBody(t *testing.T, body []byte) (code, message string) {
	t.Helper()
	var envelope struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(body, &envelope))
	return envelope.Error, envelope.Message
}

func TestWriteServiceError_Mapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   string
		wantMsg    string
	}{
		{"unauthorized", services.ErrUnauthorized, http.StatusForbidden, "forbidden", "you can only access your own account"},
		{"hybrid device login", services.ErrHybridAccountDeviceLogin, http.StatusForbidden, "forbidden", "device login not allowed for accounts with email/password"},
		{"token reuse", services.ErrTokenReused, http.StatusUnauthorized, "unauthorized", "token reuse detected - all tokens have been revoked"},
		{"refresh token expired", services.ErrRefreshTokenExpired, http.StatusUnauthorized, "unauthorized", "refresh token has expired"},
		{"incorrect password", services.ErrIncorrectPassword, http.StatusUnauthorized, "unauthorized", "current password is incorrect"},
		{"invalid credentials", services.ErrInvalidCredentials, http.StatusUnauthorized, "unauthorized", "invalid credentials"},
		{"account locked", services.ErrAccountLocked, http.StatusTooManyRequests, "too_many_requests", services.ErrAccountLocked.Error()},
		{"account exists", services.ErrAccountExists, http.StatusConflict, "conflict", "an account with these credentials already exists"},
		{"email already verified", services.ErrEmailAlreadyVerified, http.StatusConflict, "conflict", "email is already verified"},
		{"user not found", apperrors.ErrUserNotFound, http.StatusNotFound, "not_found", "user not found"},
		{"weak password", services.ErrWeakPassword, http.StatusBadRequest, "validation_error", services.ErrWeakPassword.Error()},
		{"invalid verification token", services.ErrInvalidVerificationToken, http.StatusBadRequest, "bad_request", "invalid or expired verification token"},
		{"not mobile account", services.ErrNotMobileAccount, http.StatusBadRequest, "bad_request", "account is not a mobile-only account"},
		{"no password", services.ErrNoPassword, http.StatusBadRequest, "bad_request", "account does not have a password"},
		{"invalid input", services.ErrInvalidInput, http.StatusBadRequest, "bad_request", "invalid input provided"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Bare sentinel.
			rec := httptest.NewRecorder()
			writeServiceError(rec, tt.err, "fallback")
			assert.Equal(t, tt.wantStatus, rec.Code)
			code, msg := decodeErrorBody(t, rec.Body.Bytes())
			assert.Equal(t, tt.wantCode, code)
			assert.Equal(t, tt.wantMsg, msg)

			// Wrapped sentinel — errors.Is must still resolve it.
			recWrapped := httptest.NewRecorder()
			writeServiceError(recWrapped, fmt.Errorf("service call failed: %w", tt.err), "fallback")
			assert.Equal(t, tt.wantStatus, recWrapped.Code, "wrapped sentinel should map the same")
		})
	}
}

func TestWriteServiceError_UnmappedIs500(t *testing.T) {
	rec := httptest.NewRecorder()
	writeServiceError(rec, fmt.Errorf("some unexpected failure"), "failed to do the thing")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	code, msg := decodeErrorBody(t, rec.Body.Bytes())
	assert.Equal(t, "internal_error", code)
	assert.Equal(t, "failed to do the thing", msg)
}

// TestServiceErrorTable_CoversAllSentinels fails loudly if a service-layer
// sentinel that the handlers can surface is missing from serviceErrorTable —
// the whole point of the table is that a new sentinel cannot silently fall
// through to a 500.
func TestServiceErrorTable_CoversAllSentinels(t *testing.T) {
	expected := []error{
		services.ErrInvalidCredentials,
		services.ErrRefreshTokenExpired,
		services.ErrTokenReused,
		services.ErrAccountExists,
		services.ErrHybridAccountDeviceLogin,
		services.ErrWeakPassword,
		services.ErrInvalidInput,
		services.ErrAccountLocked,
		services.ErrInvalidVerificationToken,
		services.ErrEmailAlreadyVerified,
		services.ErrUnauthorized,
		services.ErrNotMobileAccount,
		services.ErrIncorrectPassword,
		services.ErrNoPassword,
		apperrors.ErrUserNotFound,
	}

	inTable := make(map[error]bool, len(serviceErrorTable))
	for _, m := range serviceErrorTable {
		inTable[m.sentinel] = true
	}

	for _, want := range expected {
		assert.Truef(t, inTable[want], "sentinel %v is not mapped in serviceErrorTable", want)
	}
	assert.Len(t, serviceErrorTable, len(expected), "serviceErrorTable has rows not covered by this test")
}
