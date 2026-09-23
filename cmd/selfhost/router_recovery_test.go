package main

import (
	"log/slog"
	"testing"

	"github.com/andrew-craig/cairn-reader/pkg/middleware/middlewaretest"
)

// A panic must be logged with the request's real ID, which requires the
// request-ID middleware to wrap Recovery.
func TestRouter_PanicLogCarriesRequestID(t *testing.T) {
	middlewaretest.AssertPanicLogCarriesRequestID(t, newMasterRouter(slog.Default()))
}
