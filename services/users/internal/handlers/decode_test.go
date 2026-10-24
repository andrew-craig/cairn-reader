package handlers

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/logging"
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

// containsNUL must cover every container a future request struct might use,
// not just today's flat string fields.
func TestContainsNUL_Containers(t *testing.T) {
	type nested struct {
		Aliases []string
		Tags    [1]string
		Meta    map[string]string
		Any     any
	}
	tests := []struct {
		name string
		v    any
		want bool
	}{
		{"clean", nested{Aliases: []string{"a"}, Meta: map[string]string{"k": "v"}, Any: "x"}, false},
		{"slice element", nested{Aliases: []string{"a", "b\x00"}}, true},
		{"array element", nested{Tags: [1]string{"\x00"}}, true},
		{"map value", nested{Meta: map[string]string{"k": "\x00"}}, true},
		{"map key", nested{Meta: map[string]string{"\x00": "v"}}, true},
		{"interface", nested{Any: "\x00"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containsNUL(reflect.ValueOf(&tt.v)))
		})
	}
}

// A NUL rejection is a routine client error: Refresh must log it at Warn,
// not on the Error-level "failed to parse JSON" path.
func TestRefresh_NULLogsAtWarn(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	handler := NewAuthHandler(&stubAuthService{}, &mockEmailVerificationService{})

	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", strings.NewReader(`{"refresh_token":"abc\u0000"}`))
	req = req.WithContext(logging.WithLogger(req.Context(), logger))
	rec := httptest.NewRecorder()

	handler.Refresh(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	level, found := findLogLevel(t, logBuf.Bytes(), "refresh request: NUL in string field")
	require.True(t, found, "expected a NUL log line, got: %s", logBuf.String())
	assert.Equal(t, slog.LevelWarn.String(), level)
	assert.NotContains(t, logBuf.String(), "failed to parse JSON")
}
