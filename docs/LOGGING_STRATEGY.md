# Cairn Logging Strategy

This document defines a consistent logging strategy for all Go services in the Cairn project.

## Recommendation: Use `log/slog` (Standard Library)

All services should use Go's standard library `log/slog` package for structured logging.

### Why `log/slog`?

1. **Standard library** - No external dependencies, maintained by Go team
2. **Available in Go 1.21+** - Both services use Go 1.24
3. **Structured logging** - Native support for key-value pairs
4. **Flexible handlers** - JSON for production, text for development
5. **Log levels** - Built-in DEBUG, INFO, WARN, ERROR levels
6. **Context integration** - Works seamlessly with `context.Context`
7. **Performance** - Designed for high-performance production use

## Current State Issues

| Issue | Current Pattern | Impact |
|-------|-----------------|--------|
| Mixed logging methods | `log.Printf`, `fmt.Printf`, `log.Println` | Inconsistent output format |
| No log levels | All logs treated equally | Cannot filter by severity |
| Unstructured output | String interpolation | Hard to parse/search |
| Inconsistent prefixes | `[Recommendations]`, `[DB]`, `AUTH_SUCCESS` | No standard categorization |
| No JSON support | Text only | Not machine-parseable |

## Implementation Guide

### 1. Package Structure

Create a shared logging package that can be used across services:

```
pkg/
└── logging/
    ├── logger.go       # Logger initialization and configuration
    ├── middleware.go   # HTTP middleware for request logging
    └── context.go      # Context helpers for request-scoped logging
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
    ServiceName string // e.g., "user-service", "fetcher", "recommender"
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
// Before (current)
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
// Before (current)
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
// Before (current)
log.Printf("[Recommendations] Found %d eligible articles for user: %s", len(articles), userID)

// After (slog)
slog.Debug("found eligible articles",
    slog.String("component", "recommendations"),
    slog.String("user_id", userID),
    slog.Int("count", len(articles)),
)
```

#### Error Logging

```go
// Before (current)
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
// Before (current)
log.Printf("[DB] GetForRecommendation: userID=%s, limit=%d", userID, limit)

// After (slog)
slog.Debug("database query",
    slog.String("operation", "GetForRecommendation"),
    slog.String("user_id", userID),
    slog.Int("limit", limit),
)
```

### 5. HTTP Middleware (Gin)

```go
// pkg/logging/middleware.go
package logging

import (
    "log/slog"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"
)

// RequestLogger returns Gin middleware for structured request logging
func RequestLogger(logger *slog.Logger) gin.HandlerFunc {
    return func(c *gin.Context) {
        start := time.Now()

        // Generate or extract request ID
        requestID := c.GetHeader("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        c.Set("request_id", requestID)
        c.Header("X-Request-ID", requestID)

        // Create request-scoped logger
        reqLogger := logger.With(
            slog.String("request_id", requestID),
            slog.String("method", c.Request.Method),
            slog.String("path", c.Request.URL.Path),
            slog.String("client_ip", c.ClientIP()),
        )
        c.Set("logger", reqLogger)

        // Process request
        c.Next()

        // Log completed request
        duration := time.Since(start)
        status := c.Writer.Status()

        // Choose log level based on status code
        logFn := reqLogger.Info
        if status >= 500 {
            logFn = reqLogger.Error
        } else if status >= 400 {
            logFn = reqLogger.Warn
        }

        attrs := []any{
            slog.Int("status", status),
            slog.Duration("duration", duration),
            slog.Int("size", c.Writer.Size()),
        }

        // Add user ID if authenticated
        if userID, exists := c.Get("user_id"); exists {
            attrs = append(attrs, slog.Any("user_id", userID))
        }

        logFn("http request completed", attrs...)
    }
}

// GetLogger retrieves the request-scoped logger from context
func GetLogger(c *gin.Context) *slog.Logger {
    if logger, exists := c.Get("logger"); exists {
        if l, ok := logger.(*slog.Logger); ok {
            return l
        }
    }
    return slog.Default()
}
```

### 6. Context Integration (net/http)

```go
// pkg/logging/context.go
package logging

import (
    "context"
    "log/slog"
)

type contextKey string

const loggerKey contextKey = "logger"

// WithLogger adds a logger to the context
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves the logger from context, or returns default
func FromContext(ctx context.Context) *slog.Logger {
    if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
        return logger
    }
    return slog.Default()
}
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
| `service` | string | Service name (user-service, fetcher, recommender) |
| `component` | string | Logical component (auth, recommendations, db) |
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

## Migration Strategy

### Phase 1: Create Shared Package
1. Create `pkg/logging` with logger initialization and middleware
2. Add to both service module dependencies

### Phase 2: Service-by-Service Migration

**Priority order:**
1. **User Service** - Already has sophisticated middleware, highest visibility
2. **Recommender** - Core business logic with debugging needs
3. **Fetcher** - Background processing, simpler logging needs

**Per-service steps:**
1. Initialize logger in `main.go` with environment config
2. Replace middleware logging
3. Update handlers and services
4. Update repositories

### Phase 3: Cleanup
1. Remove old logging imports (`"log"`, `"fmt"` for logging)
2. Update documentation
3. Configure log aggregation (e.g., CloudWatch, Datadog)

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
