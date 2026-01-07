# Middleware Package

This package provides HTTP middleware functions for the Cairn user service. All middleware are designed to work with the Gin web framework.

## Available Middleware

### Authentication & Authorization

#### `JWTAuth(jwtManager *auth.JWTManager)`
Validates JWT tokens from the Authorization header and extracts user ID into the request context.

**Usage:**
```go
protected := router.Group("/api")
protected.Use(middleware.JWTAuth(jwtManager))
{
    protected.GET("/profile", getProfile)
}
```

**Error Responses:**
- `401` - Missing or invalid token
- `401` - Expired token
- `401` - Invalid signature

#### `OptionalAuth(jwtManager *auth.JWTManager)`
Validates JWT tokens if present but doesn't require them. Useful for endpoints that behave differently for authenticated vs unauthenticated users.

#### `RequireSameUser()`
Ensures authenticated users can only access their own resources. Expects a URL parameter named `id`.

**Usage:**
```go
users := router.Group("/users")
users.Use(middleware.JWTAuth(jwtManager))
users.Use(middleware.RequireSameUser())
{
    users.GET("/:id", getUser)
    users.PATCH("/:id", updateUser)
}
```

**Error Responses:**
- `401` - Not authenticated
- `403` - Trying to access another user's resources

#### Helper Functions

- `GetUserIDFromContext(c *gin.Context)` - Extract user ID from context
- `MustGetUserID(c *gin.Context)` - Extract user ID or panic (use only after JWTAuth)
- `IsAuthenticated(c *gin.Context)` - Check if request is authenticated

### Rate Limiting

#### `RateLimit(limit int, window time.Duration)`
Limits requests per IP address.

**Usage:**
```go
// Allow 100 requests per minute per IP
auth := router.Group("/auth")
auth.Use(middleware.RateLimit(100, time.Minute))
{
    auth.POST("/login", login)
}
```

#### `RateLimitByUser(limit int, window time.Duration)`
Limits requests per authenticated user (falls back to IP-based for unauthenticated requests).

#### `RateLimitAuth(limit int, window time.Duration)`
Specialized rate limiter for authentication endpoints to prevent brute force attacks.

**Usage:**
```go
// Allow only 5 login attempts per minute per IP
router.POST("/auth/login", middleware.RateLimitAuth(5, time.Minute), login)
```

### Security

#### `RequireHTTPS()`
Enforces HTTPS connections by rejecting HTTP requests.

**Usage:**
```go
if cfg.IsProduction() {
    router.Use(middleware.RequireHTTPS())
}
```

#### `SecureHeaders()`
Adds security-related HTTP headers (HSTS, X-Frame-Options, CSP, etc.).

**Usage:**
```go
router.Use(middleware.SecureHeaders())
```

#### `PreventCaching()`
Prevents browser caching of sensitive responses.

**Usage:**
```go
router.GET("/users/:id", middleware.PreventCaching(), getUser)
```

#### `RequireJSON()`
Validates that request Content-Type is application/json.

**Usage:**
```go
router.POST("/auth/register", middleware.RequireJSON(), register)
```

### Logging

**Note:** Logging middleware has been moved to the shared `pkg/logging` package.

#### `logging.RequestLogger(logger *slog.Logger)`
Logs HTTP requests with structured logging using `log/slog`.

**Usage:**
```go
import "github.com/andrew-craig/cairn/pkg/logging"

logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
router.Use(logging.RequestLogger(logger))
```

**Features:**
- Structured logging with slog
- Request ID generation and propagation
- Duration, status, method, path logging
- User ID extraction from context
- Client IP tracking
- JSON output for production, text for development

For detailed documentation, see `pkg/logging/README.md` and `docs/LOGGING_STRATEGY.md`.

### CORS

#### `CORS()`
Enables CORS with default configuration (allows all origins).

**Usage:**
```go
router.Use(middleware.CORS())
```

#### `ProductionCORS(allowedOrigins []string)`
Enables strict CORS for production with specific allowed origins.

**Usage:**
```go
if cfg.IsProduction() {
    router.Use(middleware.ProductionCORS([]string{
        "https://app.example.com",
        "https://www.example.com",
    }))
} else {
    router.Use(middleware.DevelopmentCORS())
}
```

#### `CORSWithWildcard(allowedOrigins []string)`
Supports wildcard subdomain matching.

**Usage:**
```go
router.Use(middleware.CORSWithWildcard([]string{
    "*.example.com",
    "https://app.example.com",
}))
```

### Recovery

#### `Recovery()`
Recovers from panics and returns a 500 error without exposing details.

**Usage:**
```go
router.Use(middleware.Recovery())
```

#### `RecoveryWithDetails()`
Recovers from panics and includes stack trace in response. **WARNING: Development only!**

**Usage:**
```go
if cfg.IsDevelopment() {
    router.Use(middleware.RecoveryWithDetails())
} else {
    router.Use(middleware.SafeRecovery())
}
```

#### `SafeRecovery()`
Production-ready recovery that logs full details but returns generic errors to clients.

## Complete Example

Here's how to set up a complete middleware stack:

```go
func setupRouter(cfg *config.Config, jwtManager *auth.JWTManager) *gin.Engine {
    // Create router (don't use gin.Default() since we're adding our own middleware)
    router := gin.New()

    // Global middleware (applied to all routes)
    router.Use(middleware.RequestID())           // Generate request IDs
    router.Use(middleware.RequestLogger())       // Log all requests
    router.Use(middleware.SafeRecovery())        // Recover from panics

    if cfg.IsProduction() {
        router.Use(middleware.RequireHTTPS())    // Enforce HTTPS
        router.Use(middleware.SecureHeaders())   // Add security headers
        router.Use(middleware.ProductionCORS(cfg.AllowedOrigins))
    } else {
        router.Use(middleware.DevelopmentCORS()) // Permissive CORS for dev
    }

    // Public endpoints
    health := router.Group("/health")
    {
        health.GET("", healthCheck)
        health.GET("/ready", readyCheck)
    }

    // Authentication endpoints (rate limited)
    auth := router.Group("/auth")
    auth.Use(middleware.RateLimitAuth(5, time.Minute))  // 5 attempts per minute
    auth.Use(middleware.RequireJSON())                   // Require JSON
    {
        auth.POST("/register", register)
        auth.POST("/register/mobile", registerMobile)
        auth.POST("/login", login)
        auth.POST("/login/mobile", loginMobile)
        auth.POST("/refresh", refresh)
        auth.POST("/logout", middleware.JWTAuth(jwtManager), logout)
        auth.POST("/logout-all", middleware.JWTAuth(jwtManager), logoutAll)
    }

    // Protected user endpoints
    users := router.Group("/users")
    users.Use(middleware.JWTAuth(jwtManager))           // Require authentication
    users.Use(middleware.RequireSameUser())             // Require ownership
    users.Use(middleware.RateLimitByUser(100, time.Minute)) // 100 req/min per user
    users.Use(middleware.PreventCaching())              // Don't cache sensitive data
    {
        users.GET("/:id", getUser)
        users.PATCH("/:id", middleware.RequireJSON(), updateUser)
        users.POST("/:id/upgrade", middleware.RequireJSON(), upgradeAccount)
        users.DELETE("/:id", deleteUser)
    }

    return router
}
```

## Best Practices

1. **Ordering Matters**: Apply middleware in the correct order:
   - Recovery (first, to catch all panics)
   - Request ID
   - Logging
   - Security headers
   - CORS
   - Rate limiting
   - Authentication
   - Authorization

2. **Rate Limiting**: Use different limits for different endpoint types:
   - Authentication: 5-10 requests per minute
   - API endpoints: 100-1000 requests per minute
   - Sensitive operations: 5-20 requests per minute

3. **Environment-Specific**: Use different middleware configurations for development and production:
   ```go
   if cfg.IsProduction() {
       router.Use(middleware.SafeRecovery())
   } else {
       router.Use(middleware.RecoveryWithDetails())
   }
   ```

4. **Security**: Always use these in production:
   - `RequireHTTPS()`
   - `SecureHeaders()`
   - `SafeRecovery()`
   - `RateLimit()` on authentication endpoints

5. **Logging**: Don't log sensitive data (passwords, tokens, etc.). The middleware is designed to avoid this, but be careful when customizing.

## Testing Middleware

Example test for authentication middleware:

```go
func TestJWTAuth(t *testing.T) {
    // Setup
    jwtManager := setupTestJWTManager(t)
    router := gin.New()
    router.Use(middleware.JWTAuth(jwtManager))
    router.GET("/test", func(c *gin.Context) {
        userID := middleware.MustGetUserID(c)
        c.JSON(200, gin.H{"user_id": userID})
    })

    // Test with valid token
    token, _ := jwtManager.GenerateToken(testUserID)
    req := httptest.NewRequest("GET", "/test", nil)
    req.Header.Set("Authorization", "Bearer "+token)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, 200, w.Code)

    // Test without token
    req = httptest.NewRequest("GET", "/test", nil)
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    assert.Equal(t, 401, w.Code)
}
```
