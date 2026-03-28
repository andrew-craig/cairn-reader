// Package middleware provides shared HTTP middleware components for all Cairn services.
//
// This package contains production-ready middleware extracted from the User Service,
// following the same patterns as [pkg/auth] for shared cross-cutting concerns.
//
// # Available Middleware
//
//   - Recovery: Panic recovery with structured logging (production-safe)
//   - CORS: Cross-Origin Resource Sharing with configurable origins
//   - Security: HTTPS enforcement, security headers, cache control, content type validation
//   - Rate Limiting: Token bucket rate limiter (per-IP or custom key)
//
// # Usage
//
// All middleware follow the standard [net/http] middleware pattern:
//
//	func(http.Handler) http.Handler
//
// Use with chi router:
//
//	r := chi.NewRouter()
//	r.Use(middleware.Recovery)
//	r.Use(middleware.SecureHeadersRelaxed)
//	r.Use(middleware.CORS(middleware.DefaultCORSConfig()))
//	r.Use(middleware.RateLimit(10, time.Minute))
package middleware
