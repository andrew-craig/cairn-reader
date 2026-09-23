package api

import (
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/andrew-craig/cairn-reader/pkg/middleware/middlewaretest"
)

// A panic must be logged with the request's real ID, which requires the
// request-ID middleware to wrap Recovery.
func TestRouter_PanicLogCarriesRequestID(t *testing.T) {
	router, _ := newTestRouterHandler(t)
	middlewaretest.AssertPanicLogCarriesRequestID(t, router.(*chi.Mux))
}
