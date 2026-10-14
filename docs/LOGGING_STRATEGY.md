# Cairn Logging Strategy

This document defines a consistent logging strategy for all Go services in the Cairn project.

## Recommendation: Use `log/slog` (Standard Library)

All services should use Go's standard library `log/slog` package for structured logging.

### Why `log/slog`?

1. **Standard library** - No external dependencies, maintained by Go team
2. **Available in Go 1.21+** - All services use Go 1.25 (see each module's `go.mod`)
3. **Structured logging** - Native support for key-value pairs
4. **Flexible handlers** - JSON for production, text for development
5. **Log levels** - Built-in DEBUG, INFO, WARN, ERROR levels
6. **Context integration** - Works seamlessly with `context.Context`
7. **Performance** - Designed for high-performance production use

## Current State

The shared `pkg/logging` package exists and all HTTP services use `log/slog` with `LOG_LEVEL` / `LOG_FORMAT` (JSON or text), a request ID, and request-scoped loggers. The original issues (mixed `log.Printf`/`fmt.Printf`, no levels, no JSON, ad-hoc prefixes) are resolved for service code. Remaining items:

| Item | Where | Status |
|------|-------|--------|
| `log.Printf` in env-parsing helpers | `services/read/content/cmd/worker/main.go` (lines 163, 177) | Open - runs after the slog logger is configured, should use `slog.Warn` |
| `log.Printf` / `log.Fatalf` in example program | `pkg/auth/examples/basic/main.go` | Open, low priority - sample code, not a service |
| `log.*` / `fmt.Print*` in CLI tools | `services/read/email/cmd/manage_keys/main.go`, `services/users/cmd/migrate/main.go`, `cmd/selfhost/version.go` | Intentional - interactive command output, not service logs |
| Log aggregation (e.g., CloudWatch, Datadog) | - | Not configured |

## Implementation Guide

### 1. Package Structure

The shared logging package (`pkg/logging`, its own Go module) is used across services:

```
pkg/
└── logging/
    ├── logger.go           # Logger initialization and configuration
    ├── chi_middleware.go   # chi middleware for request logging
    └── context.go          # Context helpers for request-scoped logging
```

### 2. Logger Initialization

```go
// pkg/logging/logger.go
package logging

import (
    "log/slog"
    "os"
)

// Config holds logger configuration
type Config struct {
    Level       string // debug, info, warn, error
    Format      string // json, text
    ServiceName string // e.g., "user-service", "fetcher", "content-service"
}

// NewLogger creates a configured slog.Logger
func NewLogger(cfg Config) *slog.Logger {
    var level slog.Level
    switch cfg.Level {
    case "debug":
        level = slog.LevelDebug
    case "warn":
        level = slog.LevelWarn
    case "error":
        level = slog.LevelError
    default:
        level = slog.LevelInfo
    }

    opts := &slog.HandlerOptions{
        Level: level,
        // Add source file info for debug/error levels
        AddSource: level <= slog.LevelDebug || level >= slog.LevelError,
    }

    var handler slog.Handler
    if cfg.Format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    // Add service name as a default attribute
    return slog.New(handler).With(
        slog.String("service", cfg.ServiceName),
    )
}

// SetDefault sets the default logger for the application
func SetDefault(logger *slog.Logger) {
    slog.SetDefault(logger)
}
```

### 3. Environment Configuration

```bash
# Development
LOG_LEVEL=debug
LOG_FORMAT=text

# Production
LOG_LEVEL=info
LOG_FORMAT=json
```

### 4. Logging Patterns

#### Service Startup

```go
// Before (legacy)
log.Printf("Starting service on port %s", port)
log.Println("✓ Database connected")

// After (slog)
slog.Info("starting service",
    slog.String("port", port),
    slog.String("env", env),
)

slog.Info("component initialized",
    slog.String("component", "database"),
    slog.String("host", dbHost),
)
```

#### HTTP Request Logging

```go
// Before (legacy)
log.Printf("[%s] %s %s %d %v %s user=%s",
    requestID, method, path, statusCode, duration, clientIP, userID)

// After (slog)
slog.Info("http request",
    slog.String("request_id", requestID),
    slog.String("method", method),
    slog.String("path", path),
    slog.Int("status", statusCode),
    slog.Duration("duration", duration),
    slog.String("client_ip", clientIP),
    slog.String("user_id", userID),
)
```

#### Business Logic

```go
// Before (legacy)
log.Printf("[Fetcher] Found %d new items for feed: %s", len(items), feedID)

// After (slog)
slog.Debug("found new feed items",
    slog.String("component", "fetcher"),
    slog.String("feed_id", feedID),
    slog.Int("count", len(items)),
)
```

#### Error Logging

```go
// Before (legacy)
log.Printf("ERROR: Failed to get articles: %v", err)
fmt.Printf("warning: failed to update timestamp: %v\n", err)

// After (slog)
slog.Error("failed to get articles",
    slog.String("user_id", userID),
    slog.Any("error", err),
)

slog.Warn("failed to update timestamp",
    slog.String("user_id", userID),
    slog.Any("error", err),
)
```

#### Database Operations

```go
// Before (legacy)
log.Printf("[DB] ListUserContent: userID=%s, limit=%d", userID, limit)

// After (slog)
slog.Debug("database query",
    slog.String("operation", "ListUserContent"),
    slog.String("user_id", userID),
    slog.Int("limit", limit),
)
```

### 5. HTTP Middleware (chi)

```go
// pkg/logging/chi_middleware.go
package logging

import (
    "log/slog"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5/middleware"
    "github.com/google/uuid"
)

// ChiRequestLogger returns Chi middleware for structured request logging
func ChiRequestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            start := time.Now()

            // Generate or extract request ID
            requestID := r.Header.Get(HeaderXRequestID)
            if requestID == "" {
                requestID = uuid.New().String()
            }
            w.Header().Set(HeaderXRequestID, requestID)

            // Create request-scoped logger
            reqLogger := logger.With(
                slog.String("request_id", requestID),
                slog.String("method", r.Method),
                slog.String("path", r.URL.Path),
                slog.String("client_ip", r.RemoteAddr),
            )

            // Wrap the response writer to capture status code and size
            ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

            // Store logger and request ID in context
            ctx := WithLogger(r.Context(), reqLogger)
            ctx = WithRequestID(ctx, requestID)
            r = r.WithContext(ctx)

            next.ServeHTTP(ww, r)

            // Choose log level based on status code
            status := ww.Status()
            logFn := reqLogger.Info
            if status >= 500 {
                logFn = reqLogger.Error
            } else if status >= 400 {
                logFn = reqLogger.Warn
            }

            logFn("http request completed",
                slog.Int("status", status),
                slog.Duration("duration", time.Since(start)),
                slog.Int("size", ww.BytesWritten()),
            )
        })
    }
}
```

Register it first on the router (`r.Use(logging.ChiRequestLogger(logger))`, as in each service's `router.go`). Handlers and services then retrieve the request-scoped logger with `logging.FromContext(r.Context())` (see section 6) rather than a framework-specific getter.

### 6. Context Integration (net/http)

```go
// pkg/logging/context.go (abridged)
package logging

const HeaderXRequestID = "X-Request-ID"

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context

// FromContext retrieves the logger from context, or returns slog.Default()
func FromContext(ctx context.Context) *slog.Logger

// WithRequestID / GetRequestIDFromContext store and read the request ID
func WithRequestID(ctx context.Context, requestID string) context.Context
func GetRequestIDFromContext(ctx context.Context) string

// SetRequestIDHeader copies the request ID from ctx onto an outbound request.
// Call before every service-to-service HTTP call so IDs propagate.
func SetRequestIDHeader(ctx context.Context, req *http.Request)
```

Usage in a handler (from `services/users/internal/handlers/auth_handler.go`):

```go
logging.FromContext(r.Context()).Warn("refresh request: body too large", slog.String("ip", ipAddress))
```

### 7. Log Level Guidelines

| Level | When to Use | Examples |
|-------|-------------|----------|
| **DEBUG** | Detailed diagnostic info | DB queries, internal state, article IDs |
| **INFO** | Normal operations | Request completed, service started, job ran |
| **WARN** | Recoverable issues | Non-critical failures, degraded performance |
| **ERROR** | Failures requiring attention | DB connection lost, external API failed |

### 8. Standard Attribute Names

Use consistent attribute names across all services:

| Attribute | Type | Description |
|-----------|------|-------------|
| `service` | string | Service name (user-service, fetcher, content-service) |
| `component` | string | Logical component (auth, fetcher, db) |
| `request_id` | string | UUID for request tracing |
| `user_id` | string | User identifier |
| `article_id` | string | Article identifier |
| `duration` | duration | Operation duration |
| `error` | error | Error object |
| `status` | int | HTTP status code |
| `method` | string | HTTP method |
| `path` | string | URL path |
| `client_ip` | string | Client IP address |
| `count` | int | Count of items |
| `operation` | string | DB/API operation name |

## Migration Status

### Phase 1: Create Shared Package - Done
`pkg/logging` provides logger initialization, chi middleware and context helpers; service modules depend on it.

### Phase 2: Service-by-Service Migration - Done
User, Content, Fetcher and Email services initialize the logger in `main.go` from `LOG_LEVEL` / `LOG_FORMAT`, use `logging.ChiRequestLogger`, and log through `logging.FromContext`.

### Phase 3: Cleanup - Partly done
1. Remaining `"log"` usage in service code: `services/read/content/cmd/worker/main.go` (see Current State)
2. Documentation updated (this file)
3. Log aggregation (e.g., CloudWatch, Datadog) - open

## Output Examples

### Development (text format)

```
time=2024-01-15T10:30:45.123Z level=INFO msg="starting service" service=user-service port=8080 env=development
time=2024-01-15T10:30:45.456Z level=INFO msg="component initialized" service=user-service component=database host=localhost
time=2024-01-15T10:30:46.789Z level=INFO msg="http request completed" service=user-service request_id=abc-123 method=POST path=/auth/login status=200 duration=45ms user_id=user-456
time=2024-01-15T10:30:47.012Z level=WARN msg="failed to update timestamp" service=user-service user_id=user-456 error="connection timeout"
```

### Production (JSON format)

```json
{"time":"2024-01-15T10:30:45.123Z","level":"INFO","msg":"starting service","service":"user-service","port":"8080","env":"production"}
{"time":"2024-01-15T10:30:46.789Z","level":"INFO","msg":"http request completed","service":"user-service","request_id":"abc-123","method":"POST","path":"/auth/login","status":200,"duration":"45ms","user_id":"user-456"}
{"time":"2024-01-15T10:30:47.012Z","level":"WARN","msg":"failed to update timestamp","service":"user-service","user_id":"user-456","error":"connection timeout"}
```

## Benefits

1. **Searchability**: Query logs by any attribute (`user_id=xyz`, `status>=500`)
2. **Consistency**: Same format across all services
3. **Flexibility**: JSON for production, text for development
4. **Performance**: Lazy evaluation, efficient memory use
5. **Traceability**: Request IDs propagate across services
6. **Maintainability**: Clear patterns, no external dependencies
