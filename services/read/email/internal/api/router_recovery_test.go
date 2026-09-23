package api

import (
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/andrew-craig/cairn-reader/pkg/middleware/middlewaretest"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/api/middleware"
)

// A panic must be logged with the request's real ID, which requires the
// request-ID middleware to wrap Recovery.
func TestRouter_PanicLogCarriesRequestID(t *testing.T) {
	// The route tree dereferences JWTAuth while being built; nothing else is
	// touched until a real route is hit.
	router := NewRouter(RouterDeps{JWTAuth: middleware.NewJWTAuthFromValidator(nil)})
	middlewaretest.AssertPanicLogCarriesRequestID(t, router.(*chi.Mux))
}
