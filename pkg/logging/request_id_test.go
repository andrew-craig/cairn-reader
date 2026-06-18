package logging

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"log/slog"
)

// TestChiRequestLoggerRequestID verifies that ChiRequestLogger reads an incoming
// X-Request-ID, stores it in the context, and echoes it on the response.
func TestChiRequestLoggerRequestID(t *testing.T) {
	logger := slog.Default()

	t.Run("propagates provided X-Request-ID to context and response", func(t *testing.T) {
		const wantID = "test-request-id-12345"

		var gotContextID string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotContextID = GetRequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := ChiRequestLogger(logger)(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderXRequestID, wantID)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if gotContextID != wantID {
			t.Errorf("context request ID = %q, want %q", gotContextID, wantID)
		}
		if got := w.Header().Get(HeaderXRequestID); got != wantID {
			t.Errorf("response X-Request-ID = %q, want %q", got, wantID)
		}
	})

	t.Run("generates a request ID when none is provided", func(t *testing.T) {
		var gotContextID string
		next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotContextID = GetRequestIDFromContext(r.Context())
			w.WriteHeader(http.StatusOK)
		})

		handler := ChiRequestLogger(logger)(next)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if gotContextID == "" {
			t.Error("expected a generated request ID in context, got empty string")
		}
		if got := w.Header().Get(HeaderXRequestID); got == "" {
			t.Error("expected a generated X-Request-ID on response, got empty string")
		}
		if gotContextID != w.Header().Get(HeaderXRequestID) {
			t.Errorf("context ID %q != response header %q", gotContextID, w.Header().Get(HeaderXRequestID))
		}
	})
}

// TestSetRequestIDHeader verifies that SetRequestIDHeader copies the context ID
// onto the outbound request, and generates one when the context is empty.
func TestSetRequestIDHeader(t *testing.T) {
	t.Run("sets header from context", func(t *testing.T) {
		const wantID = "upstream-id-abc"
		ctx := WithRequestID(context.Background(), wantID)

		req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
		SetRequestIDHeader(ctx, req)

		if got := req.Header.Get(HeaderXRequestID); got != wantID {
			t.Errorf("outbound X-Request-ID = %q, want %q", got, wantID)
		}
	})

	t.Run("generates header when context has no request ID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
		SetRequestIDHeader(context.Background(), req)

		got := req.Header.Get(HeaderXRequestID)
		if got == "" {
			t.Error("expected a generated X-Request-ID on outbound request, got empty string")
		}
	})

	t.Run("two calls with empty context produce different IDs", func(t *testing.T) {
		req1, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
		req2, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
		SetRequestIDHeader(context.Background(), req1)
		SetRequestIDHeader(context.Background(), req2)

		id1 := req1.Header.Get(HeaderXRequestID)
		id2 := req2.Header.Get(HeaderXRequestID)
		if id1 == id2 {
			t.Errorf("expected unique IDs, got the same ID %q twice", id1)
		}
	})
}
