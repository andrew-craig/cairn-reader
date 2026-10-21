package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeJSONBody(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		dst        any
		wantErr    bool
		wantStatus int
	}{
		{"valid body", `{"email":"a@example.com","password":"x"}`, &LoginRequest{}, false, http.StatusOK},
		{"malformed JSON", `{"email":`, &LoginRequest{}, true, http.StatusBadRequest},
		{"too large", `{"email":"` + strings.Repeat("a", maxRequestBodySize) + `"}`, &LoginRequest{}, true, http.StatusRequestEntityTooLarge},
		// PostgreSQL text cannot hold U+0000; it must be a 400, not a driver error.
		{"NUL in string field", `{"email":"a@example.com\u0000","password":"x"}`, &LoginRequest{}, true, http.StatusBadRequest},
		{"NUL in pointer field", `{"email":"\u0000"}`, &UpdateUserRequest{}, true, http.StatusBadRequest},
		{"escaped backslash is not NUL", `{"email":"a\\u0000@example.com","password":"x"}`, &LoginRequest{}, false, http.StatusOK},
		{"NUL in unknown field is ignored", `{"email":"a@example.com","other":"\u0000"}`, &LoginRequest{}, false, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			err := decodeJSONBody(rec, req, tt.dst)

			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tt.wantStatus, rec.Code)
			code, _ := decodeErrorBody(t, rec.Body.Bytes())
			assert.Equal(t, "bad_request", code)
		})
	}
}

func TestLogin_NULInEmailIs400(t *testing.T) {
	handler := NewAuthHandler(&stubAuthService{}, &mockEmailVerificationService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/login",
		strings.NewReader(`{"email":"admin@example.com\u0000","password":"password"}`))
	rec := httptest.NewRecorder()

	handler.Login(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
