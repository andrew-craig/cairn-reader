# Project-Wide TODO - Required Fixes

Comprehensive code review findings from December 2025. Issues are organized by priority and span all services (explore, users, mobile app).

---

## Critical Priority

### 1. Update the mobile app to use the new API endpoints
**Status:** Required after API v1 migration (completed 2026-01-01)

**Background:** All backend services have been migrated to standardized API v1 endpoints following `docs/API_MIGRATION_PLAN.md`. The mobile app currently uses the old endpoint structure and needs to be updated to use the new paths.

**Goal:** Update mobile app API client to use new v1 endpoints across all services.

**Key Changes Required:**

**User Service** (`apps/mobile/src/services/auth.ts` or similar):
```typescript
// OLD: POST /auth/login
// NEW: POST /api/v1/auth/login

// OLD: GET /user/{id}
// NEW: GET /api/v1/user/{user_id}

// OLD: /health
// NEW: /health/ready (for readiness checks)
```

**Explore Service** (`apps/mobile/src/services/explore.ts` or similar):
```typescript
// OLD: GET /explore/recommendation/{userID}
// NEW: GET /api/v1/explore/recommendation/{user_id}

// OLD: POST /explore/article/read (with article_id in body)
// NEW: POST /api/v1/explore/article/{article_id}/read (BREAKING: article_id now in path)

// OLD: POST /explore/article/{articleID}/vote
// NEW: POST /api/v1/explore/article/{article_id}/vote
```

**Read Service** (`apps/mobile/src/services/read.ts` or similar):
```typescript
// Content Service endpoints already use /api/v1 prefix
// Main change: {id} → {content_id} for consistency

// OLD: GET /api/v1/content/{id}
// NEW: GET /api/v1/content/{content_id}

// Ingest RSS Service (if used):
// OLD: /api/v1/user/{user_id}/feed/subscription
// NEW: /api/v1/source/rss/user/{user_id}/subscription
```

**Configuration Updates:**
- Update `apps/mobile/src/config/api.ts` with new base paths
- Ensure all services use `/api/v1` prefix
- Update path parameter names to snake_case (`user_id`, `article_id`, `content_id`)

**Breaking Changes:**
1. **Explore Service - Mark as Read**: Article ID moved from request body to URL path parameter
   ```typescript
   // OLD:
   fetch('/explore/article/read', {
     method: 'POST',
     body: JSON.stringify({ article_id: '...' })
   })

   // NEW:
   fetch(`/api/v1/explore/article/${articleId}/read`, {
     method: 'POST'
   })
   ```

2. **Health Checks**: Use `/health/ready` for service availability checks instead of `/health`

**Testing Requirements:**
1. Test all authentication flows (register, login, logout, refresh)
2. Test article recommendations and interactions (upvote, downvote, mark as read)
3. Test content management (save, retrieve, search, update metadata)
4. Test RSS feed subscriptions (subscribe, unsubscribe, list)
5. Verify error handling works with new JSON error response format

**Reference:**
- Full endpoint mapping: `docs/API_MIGRATION_PLAN.md` Appendix A
- Updated API documentation: `CLAUDE.md` API Endpoints section
- Health check standards: `docs/API_MIGRATION_PLAN.md` section 5

**Estimated Effort:** 4-6 hours

### 2. OpenAPI Generation
**Status:** Deferred from API migration (completed 2026-01-01)

**Background:** During the API v1 migration, all services were updated to use standardized endpoints following `docs/API_MIGRATION_PLAN.md`. OpenAPI spec generation was identified as desirable but requires additional tooling setup.

**Goal:** Generate OpenAPI 3.0 specifications automatically from code for all services.

**Current State:**
- All services have been migrated to API v1 endpoints
- Manual OpenAPI specs exist but may be outdated:
  - `services/explore/api/openapi.yaml` (needs splitting into fetcher/recommender)
  - `services/users/api/openapi.yaml`
  - `services/read/content/api/openapi.yaml`
  - `services/read/fetcher/api/openapi.yaml`

**Implementation Options:**

**Option 1: Code Annotations (Recommended for Go services)**
Use `swaggo/swag` for annotation-based spec generation:

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

Add annotations to handlers:
```go
// @Summary Get user recommendations
// @Description Returns 5 personalized article recommendations
// @Tags explore
// @Accept json
// @Produce json
// @Param user_id path string true "User ID (UUID)"
// @Success 200 {object} RecommendationResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/explore/recommendation/{user_id} [get]
// @Security BearerAuth
func (s *Server) handleRecommendations(w http.ResponseWriter, r *http.Request) {
    // handler code
}
```

Generate specs:
```bash
cd services/explore/recommender
swag init -g cmd/explore_recommender/main.go -o api/
```

**Option 2: Runtime Generation**
Use `go-chi/docgen` for chi-based services (Read services):
```bash
go get github.com/go-chi/docgen
```

**Option 3: Manual Maintenance**
Continue maintaining OpenAPI specs manually, using validation tools:
```bash
npx @apidevtools/swagger-cli validate services/*/api/openapi.yaml
```

**Tasks:**
1. Choose code generation approach (Option 1 recommended)
2. Split Explore service spec into separate fetcher/recommender specs
3. Add annotations to all handler functions across services
4. Create Makefile targets for spec generation
5. Set up CI/CD to validate specs match implementation
6. Update documentation with spec generation workflow

**Benefits:**
- Single source of truth (code drives documentation)
- Automatic API client generation
- Contract testing capabilities
- Always up-to-date API documentation

**Estimated Effort:** 8-12 hours across all services

---

## High Priority

### 2. Update response formats to follow a standard

Refer to the standard response format proposed in `docs/API_MIGRATION_PLAN.md`. Implement this for all API endpoints. Implement it by:
1. changing the backend service
2. changing the mobile app

There is no need to worry about backward compatability or migrations. Assume all users will be on the new mobile app version

### 3. Article Detail Screen
The current implementation has a placeholder for article navigation:
```typescript
const handleArticlePress = (article: Article) => {
  // TODO: Navigate to article detail screen
  console.log('Article pressed:', article.id);
};
```

**Required:**
- Create `ArticleDetailScreen.tsx` to display article content
- Fetch full article content including cleaned HTML
- Display reading position and allow scrolling
- Add navigation from ReadScreen to ArticleDetailScreen

### 4. Add Article Functionality
Remaining task to implement the Read section in the mobile app and connect it to the Read backend.

The add button currently has a placeholder:
```typescript
const handleAddPress = () => {
  // TODO: Navigate to add article screen or show add modal
  console.log('Add pressed');
};
```

**Required:**
- Create modal or screen to add articles manually
- Allow user to paste URL
- Call `ReadService.addContentToUser()` with the URL
- Refresh article list after adding

### 5. Search Functionality
The search button currently has a placeholder:
```typescript
const handleSearchPress = () => {
  // TODO: Navigate to search screen
  console.log('Search pressed');
};
```

**Required:**
- Create search screen or modal
- Implement search input with debouncing
- Call `ReadService.searchUserContents()` with query
- Display search results

### 6. Article Actions
Users should be able to:
- Mark articles as favorite (swipe action or button)
- Delete articles (swipe action)
- Change article status (unread → reading → completed → archived)

**Implementation:**
- Add swipe actions to ArticleRow component
- Call `ReadService.updateUserContent()` for status/favorite changes
- Call `ReadService.deleteUserContent()` for deletion
- Update local state optimistically

### 7. Filter/Sort Options
The design shows filtering/sorting controls that are not yet implemented:
- Filter by status (unread, reading, completed, archived)
- Filter by favorites
- Sort by date added, title, etc.

**Implementation:**
- Add filter UI in header
- Update `ReadService.listUserContents()` calls with filter params
- Persist filter preferences

### 8. Sync with Explore Service
Currently, articles from the Explore service are separate from the Read service. Need to implement:
- When user saves article from Explore, add to Read service
- Integration point in ExploreScreen to call `ReadService.addContentToUser()`

---

## Medium Priority

### 9. Environment-Aware HTTPS Enforcement (Security - Phase 8)
**Status:** Required from security hardening assessment
**Priority:** Medium
**Effort:** 1-2 hours

**Background:** Phase 8 Security Hardening identified that the HTTPS enforcement middleware blocks non-HTTPS requests in all environments, including development. This prevents local development and testing without TLS certificates.

**Issue:** The `RequireHTTPS()` middleware in `services/users/internal/middleware/security.go` is applied globally without environment awareness, causing all HTTP requests to return 403 Forbidden in development environments.

**Current Code:**
```go
// services/users/internal/middleware/security.go:10-32
func RequireHTTPS() gin.HandlerFunc {
    return func(c *gin.Context) {
        if c.Request.TLS != nil {
            c.Next()
            return
        }
        if c.GetHeader("X-Forwarded-Proto") == "https" {
            c.Next()
            return
        }
        c.JSON(http.StatusForbidden, gin.H{"error": "HTTPS required"})
        c.Abort()
    }
}
```

**Implementation:**

Update middleware to check environment configuration:

```go
func RequireHTTPS(cfg *config.Config) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Skip HTTPS check in development
        if cfg.Server.Environment == "development" {
            c.Next()
            return
        }

        // Enforce HTTPS in production/staging
        if c.Request.TLS != nil {
            c.Next()
            return
        }
        if c.GetHeader("X-Forwarded-Proto") == "https" {
            c.Next()
            return
        }
        c.JSON(http.StatusForbidden, gin.H{"error": "HTTPS required"})
        c.Abort()
    }
}
```

Update router initialization in `services/users/internal/handlers/router.go`:

```go
// Apply HTTPS enforcement based on environment
if cfg.Server.Environment == "production" || cfg.Server.Environment == "staging" {
    router.Use(middleware.RequireHTTPS(cfg))
}
```

**Benefits:**
- Allows HTTP in development for easier local testing
- Maintains strict HTTPS enforcement in production
- Reduces friction in development workflow
- Aligns with 12-factor app principles

**Testing:**
1. Test development environment accepts HTTP requests
2. Test production environment rejects HTTP requests
3. Test X-Forwarded-Proto header handling for proxies
4. Verify HSTS headers still applied in production

**Reference:** See `services/users/SECURITY_ASSESSMENT.md` Section 7 for detailed analysis.

---

### 10. Vault Dependency Resilience (Security - Phase 8)
**Status:** Required from security hardening assessment
**Priority:** Medium (High impact for production reliability)
**Effort:** 4-6 hours

**Background:** Phase 8 Security Hardening identified that the User Service fails immediately if HashiCorp Vault is unavailable, creating a single point of failure in distributed deployments.

**Issue:** The service performs a hard exit if Vault initialization fails during startup, preventing graceful degradation or retry logic.

**Current Code:**
```go
// services/users/cmd/user-service/main.go:53-59
vaultClient, err := initializeVault(cfg)
if err != nil {
    slog.Error("failed to initialize vault", slog.Any("error", err))
    os.Exit(1)  // Hard failure - no retry
}
```

**Implementation:**

**Option 1: Retry Logic with Exponential Backoff (Recommended)**

```go
func initializeVaultWithRetry(cfg *config.Config) (*auth.VaultClient, error) {
    maxRetries := 5
    baseDelay := 1 * time.Second
    maxDelay := 30 * time.Second

    var lastErr error
    for attempt := 0; attempt < maxRetries; attempt++ {
        vaultClient, err := initializeVault(cfg)
        if err == nil {
            slog.Info("vault initialized successfully", slog.Int("attempt", attempt+1))
            return vaultClient, nil
        }

        lastErr = err
        if attempt < maxRetries-1 {
            delay := time.Duration(math.Pow(2, float64(attempt))) * baseDelay
            if delay > maxDelay {
                delay = maxDelay
            }
            slog.Warn("vault initialization failed, retrying",
                slog.Int("attempt", attempt+1),
                slog.Duration("retry_in", delay),
                slog.Any("error", err))
            time.Sleep(delay)
        }
    }

    return nil, fmt.Errorf("failed to initialize vault after %d attempts: %w", maxRetries, lastErr)
}
```

**Option 2: Graceful Degradation with Cached Keys**

```go
type VaultWrapper struct {
    client     *auth.VaultClient
    cachedKeys *auth.JWTKeyPair
    mu         sync.RWMutex
}

func (v *VaultWrapper) GetJWTKeys(ctx context.Context) (*auth.JWTKeyPair, error) {
    // Try Vault first
    if v.client != nil {
        keys, err := v.client.GetJWTKeys(ctx)
        if err == nil {
            v.cacheKeys(keys)
            return keys, nil
        }
        slog.Warn("vault unavailable, using cached keys", slog.Any("error", err))
    }

    // Fallback to cached keys
    v.mu.RLock()
    defer v.mu.RUnlock()
    if v.cachedKeys != nil {
        return v.cachedKeys, nil
    }

    return nil, fmt.Errorf("vault unavailable and no cached keys")
}
```

**Option 3: Health Check Degradation**

```go
// Update health check to report degraded status
func (h *HealthHandler) Ready(c *gin.Context) {
    dbHealthy := h.checkDatabase()
    vaultHealthy := h.checkVault()

    status := "healthy"
    if !dbHealthy || !vaultHealthy {
        status = "degraded"
    }

    c.JSON(http.StatusOK, gin.H{
        "status": status,
        "checks": gin.H{
            "database": dbHealthy,
            "vault":    vaultHealthy,
        },
    })
}
```

**Recommended Approach:**
Combine Option 1 (retry logic) with Option 3 (health check degradation):
- Retry Vault connection on startup with exponential backoff
- Report degraded status if Vault unavailable but service running
- Log all Vault connectivity issues for monitoring

**Benefits:**
- Prevents cascading failures in distributed deployments
- Improves service availability during Vault maintenance
- Better alignment with cloud-native resilience patterns
- Allows service to continue operating during temporary Vault outages

**Testing:**
1. Test startup with Vault unavailable (should retry and eventually fail gracefully)
2. Test runtime Vault failure (should use cached keys if available)
3. Test health endpoint reports degraded status correctly
4. Verify monitoring alerts trigger on Vault connectivity issues

**Reference:** See `services/users/SECURITY_ASSESSMENT.md` Section 6 for detailed analysis.

---

### 11. Standardize Security Event Logging (Security - Phase 8)
**Status:** Required from security hardening assessment
**Priority:** Medium (High importance for security monitoring)
**Effort:** 3-4 hours

**Background:** Phase 8 Security Hardening identified that critical security events use unstructured `fmt.Printf` statements instead of structured logging, making it difficult to monitor security events in production.

**Issue:** Security-critical operations (token reuse detection, authorization failures, etc.) use inconsistent logging patterns that are hard to query and monitor.

**Current Examples:**
```go
// services/users/internal/auth/refresh_token.go:158-160
if err != nil {
    fmt.Printf("failed to revoke token family on reuse: %v\n", err)
}

// Missing structured logging for:
// - Token reuse detection
// - Authorization failures
// - Rate limit violations
// - Failed authentication attempts
// - Vault connectivity issues
```

**Implementation:**

**1. Replace fmt.Printf with structured slog:**

```go
// services/users/internal/auth/refresh_token.go
func (s *RefreshTokenService) ValidateAndRotateToken(...) {
    if s.isTokenReused(tokenModel) {
        // Log security event with structured data
        slog.Warn("token_reuse_detected",
            slog.String("user_id", tokenModel.UserID.String()),
            slog.String("token_family", tokenModel.TokenFamily.String()),
            slog.String("ip_address", ipAddress),
            slog.String("device_info", deviceInfo),
            slog.Time("last_used_at", *tokenModel.LastUsedAt),
            slog.Time("detected_at", time.Now()),
        )

        // Revoke family
        err := s.repo.RevokeTokenFamily(ctx, *tokenModel.TokenFamily)
        if err != nil {
            slog.Error("failed_to_revoke_token_family",
                slog.String("user_id", tokenModel.UserID.String()),
                slog.String("token_family", tokenModel.TokenFamily.String()),
                slog.Any("error", err),
            )
        } else {
            slog.Info("token_family_revoked",
                slog.String("user_id", tokenModel.UserID.String()),
                slog.String("token_family", tokenModel.TokenFamily.String()),
                slog.String("reason", "token_reuse"),
            )
        }

        return "", uuid.Nil, ErrTokenReused
    }
}
```

**2. Add structured logging to authentication handlers:**

```go
// services/users/internal/handlers/auth_handler.go
func (h *AuthHandler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        slog.Warn("login_invalid_request",
            slog.String("ip", c.ClientIP()),
            slog.Any("error", err),
        )
        c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
        return
    }

    response, err := h.authService.Login(c.Request.Context(), req.Email, req.Password, c.ClientIP(), "")
    if err != nil {
        slog.Warn("login_failed",
            slog.String("email", req.Email),
            slog.String("ip", c.ClientIP()),
            slog.Any("error", err),
        )
        c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid credentials"})
        return
    }

    slog.Info("login_successful",
        slog.String("user_id", response.User.ID.String()),
        slog.String("email", *response.User.Email),
        slog.String("ip", c.ClientIP()),
    )

    c.JSON(http.StatusOK, response)
}
```

**3. Add logging to authorization middleware:**

```go
// services/users/internal/middleware/authorization.go
func RequireSameUser(c *gin.Context) {
    requestingUserID, err := GetUserIDFromContext(c)
    if err != nil {
        slog.Warn("authorization_missing_user_id",
            slog.String("path", c.Request.URL.Path),
            slog.String("ip", c.ClientIP()),
        )
        c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
        c.Abort()
        return
    }

    targetUserIDStr := c.Param("user_id")
    targetUserID, err := uuid.Parse(targetUserIDStr)
    if err != nil {
        slog.Warn("authorization_invalid_user_id",
            slog.String("target_user_id", targetUserIDStr),
            slog.String("ip", c.ClientIP()),
        )
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user ID format"})
        c.Abort()
        return
    }

    if requestingUserID != targetUserID {
        slog.Warn("authorization_forbidden",
            slog.String("requesting_user_id", requestingUserID.String()),
            slog.String("target_user_id", targetUserID.String()),
            slog.String("path", c.Request.URL.Path),
            slog.String("ip", c.ClientIP()),
        )
        c.JSON(http.StatusForbidden, gin.H{"error": "access denied"})
        c.Abort()
        return
    }

    c.Next()
}
```

**4. Add logging to rate limiting middleware:**

```go
// services/users/internal/middleware/rate_limit.go
func (rl *RateLimiter) RateLimit(...) gin.HandlerFunc {
    return func(c *gin.Context) {
        key := keyFunc(c)
        if !rl.allow(key) {
            slog.Warn("rate_limit_exceeded",
                slog.String("key", key),
                slog.String("path", c.Request.URL.Path),
                slog.String("ip", c.ClientIP()),
            )
            c.JSON(http.StatusTooManyRequests, gin.H{
                "error": "rate limit exceeded",
                "retry_after": int(rl.window.Seconds()),
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

**5. Create security event constants:**

```go
// services/users/internal/security/events.go
package security

const (
    EventLoginSuccess        = "login_successful"
    EventLoginFailed         = "login_failed"
    EventTokenReused         = "token_reuse_detected"
    EventTokenFamilyRevoked  = "token_family_revoked"
    EventAuthorizationFailed = "authorization_forbidden"
    EventRateLimitExceeded   = "rate_limit_exceeded"
    EventAccountUpgraded     = "account_upgraded"
    EventAccountDeleted      = "account_deleted"
)
```

**Benefits:**
- Security events queryable in log aggregation systems (e.g., `event=token_reuse_detected`)
- Easy to create alerts for specific security events
- Structured data enables security analytics and dashboards
- Consistent logging format across all security events
- Better compliance with security logging requirements

**Files to Update:**
- `services/users/internal/auth/refresh_token.go` - Token reuse detection
- `services/users/internal/handlers/auth_handler.go` - Authentication events
- `services/users/internal/handlers/user_handler.go` - Account management events
- `services/users/internal/middleware/authorization.go` - Authorization failures
- `services/users/internal/middleware/rate_limit.go` - Rate limit violations

**Testing:**
1. Verify all security events emit structured logs
2. Test log output is JSON-formatted in production mode
3. Verify sensitive data (passwords, tokens) not logged
4. Create test scenarios for each security event type

**Reference:** See `services/users/SECURITY_ASSESSMENT.md` Section 8 for detailed analysis.

---

### 12. Standardize Configuration Management
**Files:**
- `services/users/internal/config/config.go` (219 lines, sophisticated)
- `services/explore/fetcher/cmd/fetcher/main.go` (inline getEnv calls)
- `services/explore/recommender/cmd/recommender/main.go` (inline getEnv calls)

**Issue:** User service has comprehensive configuration with validation, while explore services use simple inline environment variable reads.

**Implementation:**

Create `pkg/config/config.go` with shared patterns:
```go
package config

import (
    "fmt"
    "os"
    "strconv"
)

type DatabaseConfig struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    SSLMode  string
}

func (c *DatabaseConfig) Validate() error {
    if c.Host == "" {
        return fmt.Errorf("DB_HOST is required")
    }
    if c.Port == "" {
        return fmt.Errorf("DB_PORT is required")
    }
    // ... other validations
    return nil
}

func LoadDatabaseConfig() (*DatabaseConfig, error) {
    cfg := &DatabaseConfig{
        Host:     getEnv("DB_HOST", "localhost"),
        Port:     getEnv("DB_PORT", "5432"),
        User:     getEnv("DB_USER", ""),
        Password: getEnv("DB_PASSWORD", ""),
        DBName:   getEnv("DB_NAME", ""),
        SSLMode:  getEnv("DB_SSLMODE", "require"),
    }

    if err := cfg.Validate(); err != nil {
        return nil, err
    }

    return cfg, nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

Use in all services for consistent configuration handling with validation.

---

### 13. Consolidate Auth Middleware Implementations
**Files:**
- `services/users/internal/middleware/auth.go` (Gin framework)
- `services/users/pkg/auth/middleware.go` (stdlib http)

**Issue:** Two different auth middleware implementations exist with slight differences. 

**Current State:**
```go
// users/internal/middleware/auth.go - Gin version
func JWTAuth(jwtManager *auth.JWTManager) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Gin-specific implementation
    }
}

// users/pkg/auth/middleware.go - stdlib version
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Stdlib implementation
    })
}
```

**Implementation:**

To be confirmed - the read service users a third implementation. Validate and update before implementing

Keep only `pkg/auth/middleware.go` (stdlib version) for reusability:
1. Move stdlib middleware to `pkg/auth/middleware.go`
2. Delete `internal/middleware/auth.go`
3. Add Gin adapter if needed:
```go
// pkg/auth/gin_adapter.go
func GinAuthMiddleware(m *Middleware) gin.HandlerFunc {
    return func(c *gin.Context) {
        // Wrap stdlib middleware for Gin
        m.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            c.Request = r
            c.Next()
        })).ServeHTTP(c.Writer, c.Request)
    }
}
```

---

### 14. Standardize Error Response Format
**Files:**
- `services/users/internal/api/handlers.go` (Gin JSON responses)
- `services/explore/recommender/internal/api/handlers.go` (plain text responses)

**Issue:** Inconsistent error response formats between services.

**User service (JSON):**
```go
c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
```

**Recommender (plain text):**
```go
http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
```

**Implementation:**

Create `pkg/api/errors.go`:
```go
package api

type ErrorResponse struct {
    Error   string            `json:"error"`
    Code    string            `json:"code,omitempty"`
    Details map[string]string `json:"details,omitempty"`
}

func WriteError(w http.ResponseWriter, status int, message, code string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    json.NewEncoder(w).Encode(ErrorResponse{
        Error: message,
        Code:  code,
    })
}

// Usage:
api.WriteError(w, http.StatusUnauthorized, "missing token", "AUTH_TOKEN_MISSING")
```

Update all handlers to use consistent JSON error responses.

---

### 16. Add Input Validation Library
**Files:** All API handlers

**Issue:** Manual validation in each handler is error-prone and inconsistent.

**Current pattern:**
```go
if payload.ArticleID == "" {
    http.Error(w, "article_id is required", http.StatusBadRequest)
    return
}
if payload.VoteType != "upvote" && payload.VoteType != "downvote" {
    http.Error(w, "invalid vote_type", http.StatusBadRequest)
    return
}
```

**Implementation:**

**For stdlib services (explore):**
Add `github.com/go-ozzo/ozzo-validation`:
```go
import (
    validation "github.com/go-ozzo/ozzo-validation/v4"
    "github.com/go-ozzo/ozzo-validation/v4/is"
)

type VotePayload struct {
    VoteType string `json:"vote_type"`
}

func (p VotePayload) Validate() error {
    return validation.ValidateStruct(&p,
        validation.Field(&p.VoteType, validation.Required, validation.In("upvote", "downvote")),
    )
}

// In handler:
var payload VotePayload
if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
    api.WriteError(w, http.StatusBadRequest, "invalid request body", "INVALID_JSON")
    return
}
if err := payload.Validate(); err != nil {
    api.WriteError(w, http.StatusBadRequest, err.Error(), "VALIDATION_ERROR")
    return
}
```

**For Gin service (users):**
Use built-in validation:
```go
type RegisterRequest struct {
    Email    string `json:"email" binding:"required,email"`
    Password string `json:"password" binding:"required,min=8"`
}

func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    // ... process request
}
```

---

### 17. Setup Testcontainers for Integration Tests
**Files:** All `*_integration_test.go` files

**Issue:** Integration tests require manually running PostgreSQL, causing test failures in CI/CD.

**Current:**
```bash
# Test output:
Failed to ping test database: dial tcp 127.0.0.1:5433: connect: connection refused
```

**Implementation:**

1. Add testcontainers dependency:
```bash
go get github.com/testcontainers/testcontainers-go
go get github.com/testcontainers/testcontainers-go/modules/postgres
```

2. Create test helper `internal/db/testing.go`:
```go
package db

import (
    "context"
    "testing"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/modules/postgres"
    "github.com/testcontainers/testcontainers-go/wait"
)

func SetupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
    ctx := context.Background()

    postgresContainer, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("test_db"),
        postgres.WithUsername("test_user"),
        postgres.WithPassword("test_password"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(5*time.Second),
        ),
    )
    if err != nil {
        t.Fatalf("Failed to start postgres container: %v", err)
    }

    connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
    if err != nil {
        t.Fatalf("Failed to get connection string: %v", err)
    }

    pool, err := pgxpool.New(ctx, connStr)
    if err != nil {
        t.Fatalf("Failed to connect to test database: %v", err)
    }

    // Run migrations
    if err := runMigrations(pool); err != nil {
        t.Fatalf("Failed to run migrations: %v", err)
    }

    cleanup := func() {
        pool.Close()
        if err := postgresContainer.Terminate(ctx); err != nil {
            t.Logf("Failed to terminate container: %v", err)
        }
    }

    return pool, cleanup
}
```

3. Update integration tests:
```go
func TestArticleRepository_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    pool, cleanup := SetupTestDB(t)
    defer cleanup()

    repo := NewArticleRepository(pool)

    // ... test code
}
```

4. Run tests:
```bash
go test ./... -v              # Run all tests including integration
go test ./... -v -short       # Skip integration tests
```

---
### 18. Update Postgres

Update all Postgres databases (for all services) to Postgres 18. Resolve any conflicts during this migration.

---

## Low Priority

### 25. Centralize Configuration Management in Read Service
**Files:**
- `services/read/content/cmd/content/main.go:89-126`
- `services/read/fetcher/cmd/fetcher/main.go:89-126`

**Issue:** Read Service implements configuration directly in main.go without centralized config package or validation. Engineering Principles recommend grouped config in `internal/config/config.go` following User Service pattern.

**Current State:**
```go
// services/read/content/cmd/content/main.go
type Config struct {
    Port string
    DB   database.Config
}

func loadConfig() Config {
    return Config{
        Port: getEnv("PORT", "8080"),
        DB: database.Config{
            Host:     getEnv("DB_HOST", "localhost"),
            // ...
        },
    }
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
```

**Implementation:**

Create `services/read/content/internal/config/config.go`:
```go
package config

import (
    "fmt"
    "os"
    "strconv"

    "github.com/andrew-craig/cairn/services/read/content/internal/database"
)

type Config struct {
    Server   ServerConfig
    Database database.Config
}

type ServerConfig struct {
    Port string
}

func Load() (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            Port: getEnv("PORT", "8080"),
        },
        Database: database.Config{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnvInt("DB_PORT", 5432),
            User:     getEnv("DB_USER", "cairn_content"),
            Password: getEnv("DB_PASSWORD", "cairn_content_pass"),
            DBName:   getEnv("DB_NAME", "cairn_content"),
            SSLMode:  getEnv("DB_SSL_MODE", "disable"),
        },
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return cfg, nil
}

func (c *Config) Validate() error {
    if c.Server.Port == "" {
        return fmt.Errorf("PORT is required")
    }
    if c.Database.Host == "" {
        return fmt.Errorf("DB_HOST is required")
    }
    if c.Database.DBName == "" {
        return fmt.Errorf("DB_NAME is required")
    }
    return nil
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return defaultValue
}
```

Update main.go:
```go
package main

import (
    "github.com/andrew-craig/cairn/services/read/content/internal/config"
    "github.com/andrew-craig/cairn/services/read/content/internal/database"
)

func main() {
    // Load and validate configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatalf("Failed to load configuration: %v", err)
    }

    // Initialize database with validated config
    db, err := database.NewConnection(cfg.Database)
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    defer db.Close()

    // Use cfg.Server.Port instead of cfg.Port
    server := &http.Server{
        Addr: ":" + cfg.Server.Port,
        // ...
    }
}
```

Repeat for fetcher service.

**Benefits:**
- Centralized configuration management
- Validation at startup prevents runtime errors
- Consistent with User Service pattern
- Easier to test configuration logic
- Better organization

**Note:** This is a low-priority improvement since the current approach is functional. Implement when refactoring or adding new configuration options.

---

### 12. Define Sentinel Errors for Common Cases
**File:** `recommender/internal/db/article_repository.go`

Replace string errors with sentinel errors for programmatic checking:

```go
var (
    ErrArticleNotFound = errors.New("article not found")
    ErrUserNotFound    = errors.New("user not found")
)

// Usage
if err == sql.ErrNoRows {
    return nil, ErrArticleNotFound
}
```

---

### 13. Add Standardized Error Variables
**Files:** All repository files in explore and user services

**Issue:** String errors are used inline instead of defined error variables, preventing programmatic error checking.

**Current pattern:**
```go
if err == sql.ErrNoRows {
    return fmt.Errorf("article not found")
}
```

**Implementation:**

Create `pkg/errors/errors.go`:
```go
package errors

import "errors"

// Common repository errors
var (
    ErrNotFound         = errors.New("resource not found")
    ErrAlreadyExists    = errors.New("resource already exists")
    ErrInvalidInput     = errors.New("invalid input")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrForbidden        = errors.New("forbidden")
)

// Domain-specific errors
var (
    ErrArticleNotFound   = errors.New("article not found")
    ErrUserNotFound      = errors.New("user not found")
    ErrFeedNotFound      = errors.New("feed not found")
    ErrInvalidVoteType   = errors.New("invalid vote type")
)
```

Use in repositories:
```go
import apperrors "github.com/andrew-craig/cairn/pkg/errors"

func (r *ArticleRepository) GetByID(ctx context.Context, id string) (*models.Article, error) {
    var article models.Article
    err := r.db.QueryRowContext(ctx, query, id).Scan(...)
    if err == sql.ErrNoRows {
        return nil, apperrors.ErrArticleNotFound
    }
    return &article, nil
}

// In handlers, check errors:
article, err := repo.GetByID(ctx, id)
if errors.Is(err, apperrors.ErrArticleNotFound) {
    api.WriteError(w, http.StatusNotFound, "article not found", "ARTICLE_NOT_FOUND")
    return
}
```

---

### 19. Improve Router for Path Parameter Handling
**File:** `services/explore/recommender/internal/api/handlers.go`

**Issue:** Manual string manipulation for extracting path parameters is fragile and error-prone.

**Current:**
```go
// Lines 205, 264
articleID := strings.TrimPrefix(r.URL.Path, "/explore/articles/")
articleID = strings.TrimSuffix(articleID, "/vote")
```

**Implementation:**

Add lightweight router library (chi or httprouter):

**Option 1: chi (recommended - stdlib-style)**
```bash
go get github.com/go-chi/chi/v5
```

```go
import "github.com/go-chi/chi/v5"

func (s *Server) setupRoutes() {
    r := chi.NewRouter()

    // Middleware
    r.Use(s.loggingMiddleware)
    r.Use(s.authMiddleware)

    // Routes with path parameters
    r.Get("/health", s.handleHealth)
    r.Get("/explore/recommendations/{userID}", s.handleRecommendations)
    r.Post("/explore/articles/{articleID}/vote", s.handleVote)
    r.Delete("/explore/articles/{articleID}/vote", s.handleDeleteVote)

    s.router = r
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
    articleID := chi.URLParam(r, "articleID")  // Clean extraction
    // ... rest of handler
}
```

**Option 2: httprouter (faster, minimal)**
```bash
go get github.com/julienschmidt/httprouter
```

```go
import "github.com/julienschmidt/httprouter"

router := httprouter.New()
router.GET("/health", s.handleHealth)
router.POST("/explore/articles/:id/vote", s.handleVoteWithParams)

func (s *Server) handleVoteWithParams(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    articleID := ps.ByName("id")
    // ... rest of handler
}
```

---

### 20. Consolidate getEnv Helper Functions
**Files:**
- `services/explore/fetcher/cmd/fetcher/main.go:173-179`
- `services/explore/recommender/cmd/recommender/main.go:182-187`
- `services/users/internal/config/config.go:186-218`

**Issue:** Same `getEnv()` helper function duplicated across services with slight variations.

**Implementation:**

Extract to `pkg/env/env.go`:
```go
package env

import (
    "os"
    "strconv"
    "time"
)

// GetString returns the environment variable value or default
func GetString(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// GetInt returns the environment variable as int or default
func GetInt(key string, defaultValue int) int {
    if value := os.Getenv(key); value != "" {
        if intVal, err := strconv.Atoi(value); err == nil {
            return intVal
        }
    }
    return defaultValue
}

// GetBool returns the environment variable as bool or default
func GetBool(key string, defaultValue bool) bool {
    if value := os.Getenv(key); value != "" {
        if boolVal, err := strconv.ParseBool(value); err == nil {
            return boolVal
        }
    }
    return defaultValue
}

// GetDuration returns the environment variable as duration or default
func GetDuration(key string, defaultValue time.Duration) time.Duration {
    if value := os.Getenv(key); value != "" {
        if duration, err := time.ParseDuration(value); err == nil {
            return duration
        }
    }
    return defaultValue
}

// MustGetString returns the value or panics if not set
func MustGetString(key string) string {
    value := os.Getenv(key)
    if value == "" {
        panic("required environment variable not set: " + key)
    }
    return value
}
```

Use in all services:
```go
import "github.com/andrew-craig/cairn/pkg/env"

port := env.GetString("PORT", "8080")
timeout := env.GetDuration("FETCH_TIMEOUT", 30*time.Second)
maxRetries := env.GetInt("MAX_RETRIES", 3)
```

---

### 21. Add Package-Level Documentation
**Files:** Various package files across services

**Issue:** User service has excellent package documentation, while explore services have minimal package comments.

**Good example (users/internal/database/db.go):**
```go
// Package database provides PostgreSQL database connectivity and repository
// implementations for the user service. It uses pgx for high-performance
// PostgreSQL operations with connection pooling.
package database
```

**Minimal example (explore/fetcher/internal/db/config.go):**
```go
package db
```

**Implementation:**

Add comprehensive package documentation to all packages:

```go
// Package db provides PostgreSQL database connectivity for the fetcher service.
// It manages feed sources and tracks fetch history using the lib/pq driver.
//
// The main types are:
//   - Config: Database connection configuration
//   - FeedRepository: CRUD operations for RSS feed sources
//   - HistoryRepository: Tracking of fetch attempts
//
// Example usage:
//
//    cfg := &db.Config{
//        Host: "localhost",
//        Port: "5432",
//        // ... other fields
//    }
//    conn, err := cfg.Connect(ctx)
//    if err != nil {
//        log.Fatal(err)
//    }
//    defer conn.Close()
//
//    repo := db.NewFeedRepository(conn)
//    feeds, err := repo.List(ctx)
package db
```

Add to all packages in:
- `services/explore/fetcher/internal/db/`
- `services/explore/fetcher/internal/fetcher/`
- `services/explore/fetcher/internal/sync/`
- `services/explore/recommender/internal/db/`
- `services/explore/recommender/internal/recommend/`
- `services/explore/recommender/internal/api/`

---

### 22. Add HTTP Framework Decision Documentation
**File:** Create `docs/architecture/http-frameworks.md`

**Issue:** Different HTTP frameworks used across services (Gin vs stdlib) without documented rationale.

**Implementation:**

Create `docs/architecture/http-frameworks.md`:
```markdown
# HTTP Framework Decisions

## Current State

### User Service: Gin Web Framework
**Location:** `services/users/`

**Rationale:**
- Complex routing requirements (nested route groups, parameter validation)
- Built-in middleware ecosystem (CORS, recovery, logging)
- Better developer experience for auth-heavy service
- Excellent JSON binding and validation
- Performance optimized for REST APIs

**Trade-offs:**
- Additional dependency (~10MB)
- Framework-specific patterns
- Potential lock-in

### Explore Services: stdlib net/http
**Location:** `services/explore/fetcher/`, `services/explore/recommender/`

**Rationale:**
- Simple API surface (few endpoints)
- Minimal dependencies preferred
- Educational value (explicit HTTP handling)
- Lower memory footprint
- No framework lock-in

**Trade-offs:**
- Manual route parameter extraction
- More boilerplate code
- No built-in validation

## Decision

**Status:** Accepted

We maintain different frameworks based on service complexity:
- **Complex services with many endpoints** → Gin
- **Simple services with few endpoints** → stdlib

## Future Considerations

If explore services grow significantly (>10 endpoints), consider:
1. Migrating to chi (stdlib-compatible router)
2. Extracting common patterns to shared middleware
3. Re-evaluating Gin adoption

## Related

- See `pkg/api/` for shared HTTP utilities
- See `CLAUDE.md` for API conventions
```

---

### 23. Add Mobile App Test Infrastructure
**Files:** `apps/mobile/src/`

**Issue:** Mobile app has no test files (0 test coverage).

**Implementation:**

1. Install testing dependencies:
```bash
cd apps/mobile
npm install --save-dev @testing-library/react-native @testing-library/jest-native jest
```

2. Create `jest.config.js`:
```javascript
module.exports = {
  preset: 'react-native',
  setupFilesAfterEnv: ['<rootDir>/jest-setup.js'],
  transformIgnorePatterns: [
    'node_modules/(?!(react-native|@react-native|@react-navigation|expo)/)',
  ],
  collectCoverageFrom: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.d.ts',
    '!src/types/**',
  ],
};
```

3. Create example test `src/components/common/__tests__/Button.test.tsx`:
```typescript
import React from 'react';
import { render, fireEvent } from '@testing-library/react-native';
import { Button } from '../Button';

describe('Button', () => {
  it('renders correctly', () => {
    const { getByText } = render(<Button title="Test" onPress={() => {}} />);
    expect(getByText('Test')).toBeTruthy();
  });

  it('calls onPress when pressed', () => {
    const mockOnPress = jest.fn();
    const { getByText } = render(<Button title="Test" onPress={mockOnPress} />);

    fireEvent.press(getByText('Test'));
    expect(mockOnPress).toHaveBeenCalledTimes(1);
  });

  it('is disabled when disabled prop is true', () => {
    const mockOnPress = jest.fn();
    const { getByText } = render(
      <Button title="Test" onPress={mockOnPress} disabled />
    );

    fireEvent.press(getByText('Test'));
    expect(mockOnPress).not.toHaveBeenCalled();
  });
});
```

4. Add test script to `package.json`:
```json
{
  "scripts": {
    "test": "jest",
    "test:watch": "jest --watch",
    "test:coverage": "jest --coverage"
  }
}
```

5. Run tests:
```bash
npm test
```

---

### 24. Optimize N+1 Query in Recommendation Flow
**File:** `services/explore/recommender/internal/db/article_repository.go:327`

**Issue:** Recording recommendations happens in a loop, creating N database calls instead of 1 batch operation.

**Current:**
```go
for _, article := range recommendations {
    if err := r.RecordRecommendation(ctx, userID, article.ID); err != nil {
        // Individual INSERT for each article
    }
}
```

**Implementation:**

Add batch method to `article_repository.go`:
```go
func (r *ArticleRepository) RecordRecommendationsBatch(ctx context.Context, userID string, articleIDs []string) error {
    if len(articleIDs) == 0 {
        return nil
    }

    // Build batch INSERT
    query := `
        INSERT INTO user_article_recommendations (user_id, article_id, recommended_at)
        VALUES `

    values := make([]interface{}, 0, len(articleIDs)*2)
    placeholders := make([]string, 0, len(articleIDs))

    for i, articleID := range articleIDs {
        placeholders = append(placeholders, fmt.Sprintf("($%d, $%d, NOW())", i*2+1, i*2+2))
        values = append(values, userID, articleID)
    }

    query += strings.Join(placeholders, ", ")
    query += " ON CONFLICT (user_id, article_id) DO NOTHING"

    _, err := r.db.ExecContext(ctx, query, values...)
    if err != nil {
        return fmt.Errorf("failed to record recommendations batch: %w", err)
    }

    // Also batch increment recommends counter
    return r.incrementRecommendsCountBatch(ctx, articleIDs)
}

func (r *ArticleRepository) incrementRecommendsCountBatch(ctx context.Context, articleIDs []string) error {
    query := `
        UPDATE articles
        SET recommends = recommends + 1
        WHERE id = ANY($1)`

    _, err := r.db.ExecContext(ctx, query, articleIDs)
    return err
}
```

Use in recommendation engine:
```go
articleIDs := make([]string, len(recommendations))
for i, article := range recommendations {
    articleIDs[i] = article.ID
}

if err := r.articleRepo.RecordRecommendationsBatch(ctx, userID, articleIDs); err != nil {
    return nil, fmt.Errorf("failed to record recommendations: %w", err)
}
```

**Performance impact:**
- Before: N queries (5 INSERTs + 5 UPDATEs = 10 queries)
- After: 2 queries (1 batch INSERT + 1 batch UPDATE)
- 5x reduction in database round trips

---

## Notes

- All changes should include appropriate tests
- Run `make test` and `make lint` after each fix
- Update docker-compose.yml to set `DB_SSLMODE=disable` when SSL mode default changes

---

## Summary

**Total Issues:** 45 (4 Critical, 7 High, 19 Medium, 15 Low)

**Quick Wins (Can be done in <1 hour each):**
- Critical #1: Consolidate logging package
- High #4: Move shared models
- High #5: Standardize Read Service logging to slog
- High #6: Remove hardcoded secrets
- Medium #8: Rename testhelpers to testutil
- Low #20: Consolidate getEnv helpers
- Low #21: Add package documentation

**High Impact (Address first):**
- Critical #2: Standardize on pgx/v5 driver
- Critical #3: Add OpenAPI specs
- Critical #4: Add recommendation engine tests
- High #5: Standardize Read Service logging (consistency across all services)
- Medium #7: Document Read Service tech stack (knowledge sharing)
- Medium #15: Add API versioning
- Medium #16: Add input validation

**Long-term Improvements:**
- Medium #10-14: Architecture standardization
- Medium #17: Testcontainers setup
- Low #18-24: Code quality improvements

---

## Completed

The following items have been successfully implemented and verified:

### Performance & Modernization
- **Standardize PostgreSQL Driver to pgx/v5** - Migrated explore services (fetcher and recommender) from `database/sql` + `lib/pq` to modern `pgx/v5/pgxpool`. Updated all repository methods, connection pooling, and transaction handling. Benefits include 2-3x faster query performance, better connection pooling with health checks, and native PostgreSQL protocol support. Both services build and compile successfully.

### Documentation & API
- **Add OpenAPI/Swagger Specifications** - Created comprehensive OpenAPI 3.0 specifications for both Explore and User services (`services/explore/api/openapi.yaml` and `services/users/api/openapi.yaml`). Documented all endpoints with request/response schemas, authentication requirements, and examples. Added documentation section to CLAUDE.md with instructions for viewing specs with Swagger UI and validation commands. Both specs validated successfully with @apidevtools/swagger-cli.

### Code Organization & Maintainability
- **Consolidate Duplicate Logging Package** - Created shared logging package at repository root (`pkg/logging/`), updated imports in all services, eliminated duplicate code across explore and users services
- **Standardize Logging Library to log/slog in Read Service** - Migrated Read service from `go.uber.org/zap` to stdlib `log/slog`. Updated test file `cleanup_job_test.go` to use slog instead of zap for logger instantiation. Removed zap dependency from go.mod using `go mod edit -dropreplace` and `go mod tidy`. All tests pass and both services (content and fetcher) build successfully. Aligns with Engineering Principles preference for stdlib over external dependencies.
- **Move Shared Models to Root Package** - Created `pkg/models/` at repository root and migrated all domain models (Article, User, Vote, Feed, RecommendationEvent) from `services/explore/pkg/models/`. Updated imports across explore service. Created go.mod for the new shared package. All services (explore, users, read) build successfully.
- **Fix Module Dependency Architecture** - Extracted auth middleware from `services/users/pkg/auth/` to shared `pkg/auth/` package at repository root. This eliminates the coupling where explore service depended on users service code. Removed the `replace github.com/andrew-craig/cairn/services/users => ../users` directive from explore/go.mod. Updated imports in explore service (recommender) to use `pkg/auth` instead of `services/users/pkg/auth`. Added proper replace directives for pkg/auth, pkg/models, and pkg/logging in all service go.mod files. All services build successfully, eliminating the need for complex Dockerfile workarounds.
- **Rename testhelpers to testutil for Consistency** - Renamed `internal/testhelpers/` to `internal/testutil/` in both Read Service components (content and fetcher). Updated package declarations and all import statements in integration test files. This aligns Read Service with documented standards and patterns used in User Service and Explore Service. All tests pass (content service tests pass completely; fetcher service has a pre-existing test failure in worker package unrelated to this change). Benefits include consistency across services, easier test utility discovery, and alignment with Go community conventions.
- **Add Repository Interfaces to Explore Services** - Implemented repository interface pattern for all Explore service repositories, matching the pattern already used in User and Read services. Created `internal/db/interfaces.go` files defining `FeedRepositoryInterface` (fetcher), `ArticleRepositoryInterface`, `VoteRepositoryInterface`, and `UserRepositoryInterface` (recommender). Renamed concrete struct types to unexported names (e.g., `FeedRepository` → `feedRepository`) while keeping exported constructor functions that return interfaces. Updated all callers including handlers, recommendation engine, cleanup jobs, and sync components to use interface types instead of concrete types. Updated recommender cleanup command and integration test to use pgxpool.Pool for consistency with interface requirements. All services build successfully. Benefits include easier unit testing with mocks, better dependency injection, cleaner API surface, and consistency with established patterns across the codebase. Note: Read Service already follows this pattern correctly.

### Security & Reliability
- **Add Request Body Size Limits** - Implemented MaxBytesReader with appropriate limits (10MB for batch, 1KB for simple requests)
- **Validate Article Exists Before Recording Vote** - Added rowsAffected check and error handling
- **Make SSL Mode Configurable (Default: Require)** - DB_SSLMODE environment variable with "require" default
- **Remove Hardcoded Secrets from Docker Compose** - Migrated all hardcoded secrets from docker-compose files to environment variables. Created `.env.example` files in three locations: `infrastructure/docker/.env.example`, `services/explore/.env.example`, and `services/read/.env.example`. Updated all docker-compose.yml files (`infrastructure/docker/docker-compose.yml`, `services/explore/docker-compose.yml`, `services/read/docker-compose.yml`) to use environment variable substitution for sensitive values (PostgreSQL passwords, Vault tokens, database credentials). Added comprehensive documentation in README.md with setup instructions and security best practices. All `.env` files are already in `.gitignore` to prevent accidental commits.

### Code Quality & Performance
- **Replace O(n²) Sorting with Standard Library** - Using sort.Slice for O(n log n) performance
- **Standardize Logging to slog** - Main service code migrated from log.Printf to structured slog
- **Extract URL Path Parsing Helper** - Implemented extractPathParam helper function
- **Log Warning for Silent Vote Counter Failures** - Added slog.Warn for rowsAffected == 0 cases

### Infrastructure & Architecture
- **Add Connection Pool Configuration to Fetcher** - Configured MaxOpenConns, MaxIdleConns, and ConnMaxLifetime
- **Implement Kubernetes-Style Health Endpoints** - Separate /health (liveness) and /ready (readiness) endpoints
- **Add Request ID Propagation** - X-Request-ID header generation and context propagation for distributed tracing
- **Validate User IDs as UUIDs** - UUID format validation in EnsureUserExists
- **Standardize Database Architecture Pattern** - Migrated infrastructure docker-compose.yml from single PostgreSQL instance with multiple databases to separate PostgreSQL instances per service (users-db:5432, recommender-db:5433, fetcher-db:5434). This aligns with the microservices isolation pattern used in explore and read services. Benefits include independent scaling, better resource isolation, simplified deployment, and proper microservices boundaries. Updated .env.example with separate credentials for each database. Created MIGRATION.md guide for users upgrading from the old pattern. All services correctly depend on their specific database instances with proper health checks.

### Refactoring
- **Delete Unused Gin Middleware** - Removed from explore service (user service middleware is actively used)
- **Cache Internal User ID in Recommendation Flow** - Addressed by using external user ID from JWT token directly throughout the flow

### Testing & Quality Assurance
- **Add Tests for Recommendation Engine** - Created comprehensive test suite for `services/explore/recommender/internal/recommend/engine.go` including:
  - Unit tests for quality score calculation (8 test cases covering edge cases)
  - Unit tests for high-quality article selection (4 test cases)
  - Integration tests for GetRecommendations (5 test scenarios):
    - Returns correct number of recommendations when fewer than 5 articles available
    - Includes exploration article (low exposure) along with 4 high-quality articles
    - Filters out deleted articles from recommendations
    - Prevents duplicate recommendations to the same user
    - Properly increments recommends counter for each recommendation
  - All unit tests pass in CI/CD without database dependency (use `-short` flag)
  - Integration tests validate end-to-end recommendation algorithm with real database

### Documentation & Knowledge Sharing
- **Document Read Service Technology Stack in Engineering Principles** - Added comprehensive documentation for Read Service specific libraries to `docs/ENGINEERING_PRINCIPLES.md`. Documented chi/v5 HTTP router, go-readability content extraction, bluemonday HTML sanitization, gobreaker circuit breaker, and robfig/cron job scheduling libraries. Updated HTTP Framework section to include all three frameworks (stdlib net/http, Gin, chi/v5) with rationale for each choice. Added "Why chi/v5?" section explaining architectural decision for lightweight router vs full framework.
