package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/logging"
	"github.com/andrew-craig/cairn-reader/services/users/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRefresh_LogLevelFollowsErrorMapping pins the log level of the /auth/refresh
// failure path to whether the error is mapped by serviceErrorTable. Routine
// client-side causes (expired/replayed refresh tokens) are 4xx and must log at
// Warn so they don't inflate the error rate; an unmapped error is the 500 path
// (a genuine internal failure) and must stay at Error, where error-rate
// monitoring can see it.
func TestRefresh_LogLevelFollowsErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		refreshErr error
		wantStatus int
		wantLevel  slog.Level
	}{
		{
			name:       "expired refresh token is a routine 401",
			refreshErr: services.ErrRefreshTokenExpired,
			wantStatus: http.StatusUnauthorized,
			wantLevel:  slog.LevelWarn,
		},
		{
			name:       "token reuse is a routine 401",
			refreshErr: services.ErrTokenReused,
			wantStatus: http.StatusUnauthorized,
			wantLevel:  slog.LevelWarn,
		},
		{
			name:       "wrapped sentinel still maps to Warn",
			refreshErr: fmt.Errorf("refresh failed: %w", services.ErrInvalidCredentials),
			wantStatus: http.StatusUnauthorized,
			wantLevel:  slog.LevelWarn,
		},
		{
			name:       "unmapped error is an internal failure",
			refreshErr: errors.New("dial tcp: connection refused"),
			wantStatus: http.StatusInternalServerError,
			wantLevel:  slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			handler := NewAuthHandler(&stubAuthService{refreshErr: tt.refreshErr}, &mockEmailVerificationService{})

			body, err := json.Marshal(RefreshRequest{RefreshToken: "some-refresh-token"})
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(logging.WithLogger(req.Context(), logger))
			w := httptest.NewRecorder()

			handler.Refresh(w, req)

			require.Equal(t, tt.wantStatus, w.Code)

			level, found := findLogLevel(t, logBuf.Bytes(), "refresh request failed")
			require.True(t, found, "expected a 'refresh request failed' log line, got: %s", logBuf.String())
			assert.Equal(t, tt.wantLevel.String(), level)
		})
	}
}

// findLogLevel returns the level of the first JSON log line whose message is msg.
func findLogLevel(t *testing.T, logs []byte, msg string) (string, bool) {
	t.Helper()
	for _, line := range bytes.Split(bytes.TrimSpace(logs), []byte("\n")) {
		if len(line) == 0 {
			continue
		}
		var entry struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
		}
		require.NoError(t, json.Unmarshal(line, &entry))
		if entry.Msg == msg {
			return entry.Level, true
		}
	}
	return "", false
}
