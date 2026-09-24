// Package middlewaretest holds test helpers for asserting how a service's real
// router composes the shared middleware.
package middlewaretest

import (
	"bufio"
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/andrew-craig/cairn-reader/pkg/logging"
)

// AssertPanicLogCarriesRequestID mounts a panicking route on mux, sends a
// request with a known X-Request-ID, and fails t unless the "panic" log line
// written by middleware.Recovery carries that ID. It fails when Recovery sits
// outside the request-ID middleware, because the ID is not yet in the context
// when the panic is recovered.
//
// It swaps slog's default logger, so callers must not run in parallel.
func AssertPanicLogCarriesRequestID(t *testing.T, mux *chi.Mux) {
	t.Helper()

	var buf bytes.Buffer
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	mux.Get("/__middlewaretest/panic", func(http.ResponseWriter, *http.Request) {
		panic("middlewaretest panic")
	})

	const requestID = "middlewaretest-request-id"
	req := httptest.NewRequest(http.MethodGet, "/__middlewaretest/panic", nil)
	req.Header.Set(logging.HeaderXRequestID, requestID)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}

	scanner := bufio.NewScanner(&buf)
	scanner.Buffer(nil, 1<<20)
	for scanner.Scan() {
		var line struct {
			Msg       string `json:"msg"`
			RequestID string `json:"request_id"`
		}
		if json.Unmarshal(scanner.Bytes(), &line) != nil || line.Msg != "panic" {
			continue
		}
		if line.RequestID != requestID {
			t.Fatalf("panic log request_id = %q, want %q", line.RequestID, requestID)
		}
		return
	}
	t.Fatalf("no panic log line written; log output:\n%s", buf.String())
}
