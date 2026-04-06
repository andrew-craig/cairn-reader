# HTTP Framework Decision: Chi vs stdlib

## Summary

All Cairn backend services use **Chi v5** (`github.com/go-chi/chi/v5`) for HTTP routing combined with Go's stdlib `net/http` for server implementation. Gin is not used in any service — its presence in one `go.mod` is an indirect/transitive dependency only.

## Decision

**Use Chi v5 for routing; use stdlib `net/http` for the HTTP server.**

This was chosen over:
- **Pure stdlib `net/http`**: Lacks URL parameter extraction, route grouping, and middleware chaining — all of which every service needs.
- **Gin**: More opinionated, heavier, with a custom context type (`*gin.Context`) that breaks compatibility with the standard `http.Handler`/`http.HandlerFunc` interface. This would prevent reuse of shared middleware across services.

Chi sits in the middle: it is a thin router that wraps stdlib, keeps the standard `http.Handler` interface, and adds the route grouping and URL param patterns that Gin provides — without the framework lock-in.

## Consistent Architecture Pattern

Every service follows the same shape:

```
cmd/<service>/main.go          → creates &http.Server{Handler: router}
internal/api/router.go         → chi.NewRouter(), middleware chain, route groups
pkg/middleware/                → shared middleware (Recovery, CORS, security, rate limit)
pkg/auth/middleware.go         → JWT authentication (RequireAuth, OptionalAuth)
pkg/auth/internal_auth.go      → service-to-service auth (RequireInternalAPIKey)
pkg/logging/chi_middleware.go  → structured request logging (ChiRequestLogger)
```

## Middleware Stack

### Global Middleware (all routes)

Applied via `r.Use(...)` at the top of each router:

| Middleware | Package | Purpose |
|---|---|---|
| `Recovery` | `pkg/middleware` | Catches panics; logs with request ID; returns `500` JSON — never leaks internals |
| `ChiRequestLogger` | `pkg/logging` | Structured `slog` logging; generates `X-Request-ID`; logs status, duration, size |
| `SecureHeadersRelaxed` | `pkg/middleware` | Sets `X-Frame-Options: DENY`, `X-Content-Type-Options: nosniff`, `X-XSS-Protection`, `Strict-Transport-Security`, `Referrer-Policy` |

### Route-Group Middleware

Applied via `r.Use(...)` inside `r.Route(...)` blocks:

| Middleware | Package | Purpose |
|---|---|---|
| `RequireHTTPS` | `pkg/middleware` | Checks `r.TLS` and `X-Forwarded-Proto`; returns `403` if not HTTPS |
| `CORS` / `DevelopmentCORS` | `pkg/middleware` | Handles CORS preflight and response headers; strict config in prod |
| `RateLimit` | `pkg/middleware` | Token-bucket rate limiter keyed by client IP; configured per route group |
| `RequireAuth` | `pkg/auth` | Validates RS256 JWT; stores `user_id` UUID in context via `UserIDContextKey` |
| `OptionalAuth` | `pkg/auth` | Validates JWT if present; continues unauthenticated if absent |
| `RequireInternalAPIKey` | `pkg/auth` | Validates `X-Internal-API-Key` header using constant-time compare |
| `ValidateJSON` | service-local | Rejects non-JSON content types on write methods |

### Health Check Exemption

Health routes (`/health/live`, `/health/ready`) are registered outside all security middleware groups to allow HTTP health checks from Docker and Kubernetes probes.

## Per-Service Configuration

| Service | Port | Router file | Auth type | Extra middleware |
|---|---|---|---|---|
| User Service | 8082 | `services/users/internal/handlers/router.go` | JWT (`RequireAuth`) | Rate limit on `/auth/*` (10 req/min), CORS |
| Content Service | 8083 | `services/read/content/internal/api/router.go` | JWT + Internal API key | `ValidateJSON` globally |
| Ingest RSS Service | 8085 | `services/read/fetcher/internal/api/router.go` | None (internal) | 60s request timeout |
| Email Ingest Service | 8087 | `services/read/email/internal/api/router.go` | JWT + API key | RequestID, RealIP (chi built-ins) |
| Explore Fetcher | 8080 | `services/explore/fetcher/...` | None (internal) | Security headers |
| Explore Recommender | 8081 | `services/explore/recommender/internal/api/server.go` | JWT (`RequireAuth`) | — |

## Key Shared Components

### `pkg/middleware`

```go
middleware.Recovery                          // panic recovery
middleware.ChiRequestLogger(logger)          // request logging (in pkg/logging)
middleware.SecureHeadersRelaxed              // security headers for APIs
middleware.SecureHeaders                     // stricter CSP for browser-facing routes
middleware.RequireHTTPS                      // enforce HTTPS
middleware.RequireHTTPSWithRedirect          // redirect HTTP → HTTPS (staging)
middleware.CORS(config)                      // configurable CORS
middleware.DevelopmentCORS()                 // wildcard CORS (dev only)
middleware.RateLimit(limit, window)          // per-IP token bucket
middleware.RateLimitWithKey(limit, w, fn)   // custom key function
middleware.PreventCaching                    // no-store headers
middleware.RequireJSON()                     // enforce application/json
```

### `pkg/auth`

```go
// JWT middleware
auth.NewMiddleware(validator).RequireAuth    // require valid JWT → sets user_id in context
auth.NewMiddleware(validator).OptionalAuth   // validate JWT if present

// Context helpers
auth.GetUserIDFromContext(ctx)              // returns (uuid.UUID, bool)
auth.GetUserIDOrError(ctx)                 // returns (uuid.UUID, error) — preferred
auth.SetUserIDInContext(ctx, userID)        // used by middleware

// Service-to-service
auth.NewInternalAuthMiddleware(key).RequireInternalAPIKey  // X-Internal-API-Key header
```

### `pkg/logging`

```go
logging.ChiRequestLogger(logger)   // chi-compatible structured request logger
                                   // generates X-Request-ID, logs at Info/Warn/Error
                                   // by status code (4xx → Warn, 5xx → Error)
```

## Why Not Pure Stdlib

Pure `net/http` requires writing URL param extraction by hand (e.g. parsing `/user/{id}` from `r.URL.Path`) and middleware composition with nested closures. As the number of services and routes grew, the repeated boilerplate became error-prone. Chi adds exactly the missing primitives — `chi.URLParam`, `r.Route`, `r.With`, `r.Use` — without changing the `http.Handler` contract.

## Why Not Gin

Gin uses `*gin.Context` instead of the stdlib `(http.ResponseWriter, *http.Request)` pair. This means every middleware and handler in the shared `pkg/` packages would need to import Gin and accept its custom type. This would couple all shared infrastructure to Gin's release cycle and API surface, and would prevent mixing Gin-based and stdlib-based handlers. Chi avoids this entirely.

## Consequences

- All middleware in `pkg/middleware`, `pkg/auth`, and `pkg/logging` is written against the stdlib interface and is reusable across every service without modification.
- New services follow the same router template: `chi.NewRouter()` → global middleware → `r.Route("/api/v1/...", ...)` → route-specific middleware → handlers.
- The `chi/v5/middleware` package (RequestID, RealIP, etc.) is available when needed but most services use the shared `pkg/` equivalents for consistency.
