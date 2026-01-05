# Project-Wide TODO - Required Fixes

Comprehensive code review findings from December 2025. Issues are organized by priority and span all services (explore, users, mobile app).

---

## Critical Priority

---

## High Priority

---

## Medium Priority

### 3. Complete API v1 Migration - Update OpenAPI Specifications
**Files:**
- `services/explore/api/openapi.yaml`
- `services/users/api/openapi.yaml`

**Issue:** OpenAPI specifications have not been updated to reflect the new API v1 structure that was implemented in the code. Service routes were successfully migrated, but documentation specs still show old paths.

**Impact:**
- API documentation is out of sync with implementation
- API consumers (including mobile app developers) see incorrect endpoint paths
- OpenAPI validation tools will fail against actual endpoints
- Developer onboarding is more difficult due to inaccurate documentation

**Current State:**
✅ **Service code fully migrated** (100% complete):
- All routes use `/api/v1` prefix
- Health checks use `/health/live` and `/health/ready`
- Path parameters use snake_case (`{user_id}`, `{article_id}`)

❌ **OpenAPI specs not updated**:

**Explore Service** (`services/explore/api/openapi.yaml`):
- Still uses `/health` instead of `/health/live`
- Still uses `/fetch` instead of `/api/v1/explore/feed/fetch`
- Still uses `/feeds/stats` instead of `/api/v1/explore/feed/stats`
- Still uses `/ready` instead of `/health/ready`
- Still uses `/explore/articles` instead of `/api/v1/explore/article`
- Still uses `{userID}` instead of `{user_id}`
- Still uses `{articleID}` instead of `{article_id}`

**User Service** (`services/users/api/openapi.yaml`):
- Still uses `/health` instead of `/health/live`
- Still uses `/ready` instead of `/health/ready`
- Missing `/api/v1` prefix on all auth and user endpoints

**Implementation:**

Update Explore Service OpenAPI spec:
```yaml
paths:
  # Health endpoints
  /health/live:
    get:
      summary: Liveness check (Fetcher)
      # ...

  /health/ready:
    get:
      summary: Readiness check (Fetcher)
      # ...

  # Fetcher endpoints
  /api/v1/explore/feed/fetch:
    post:
      summary: Trigger manual feed fetch
      # ...

  /api/v1/explore/feed/stats:
    get:
      summary: Get feed statistics
      # ...

  /api/v1/explore/feed/sync:
    post:
      summary: Sync feeds from Kagi Small Web
      # ...

  # Recommender endpoints
  /api/v1/explore/article:
    post:
      summary: Submit articles (internal)
      # ...

  /api/v1/explore/recommendation/{user_id}:
    get:
      summary: Get recommendations
      parameters:
        - name: user_id
          in: path
          # ...

  /api/v1/explore/article/{article_id}/read:
    post:
      summary: Mark article as read
      parameters:
        - name: article_id
          in: path
          # ...

  /api/v1/explore/article/{article_id}/vote:
    post:
      summary: Vote on article
      parameters:
        - name: article_id
          in: path
          # ...
    delete:
      summary: Remove vote
      # ...
    get:
      summary: Get vote counts
      # ...
```

Update User Service OpenAPI spec:
```yaml
paths:
  # Health endpoints
  /health/live:
    get:
      summary: Liveness check
      # ...

  /health/ready:
    get:
      summary: Readiness check
      # ...

  # Authentication endpoints
  /api/v1/auth/register:
    post:
      summary: Register with email/password
      # ...

  /api/v1/auth/register/mobile:
    post:
      summary: Register with device ID
      # ...

  /api/v1/auth/login:
    post:
      summary: Login with email/password
      # ...

  # ... continue for all auth endpoints

  # User management endpoints
  /api/v1/user/{user_id}:
    get:
      summary: Get user profile
      parameters:
        - name: user_id
          in: path
          schema:
            type: string
            format: uuid
          # ...
```

**Verification:**
```bash
# Validate updated specs
npx @apidevtools/swagger-cli validate services/explore/api/openapi.yaml
npx @apidevtools/swagger-cli validate services/users/api/openapi.yaml

# View specs in Swagger UI (optional)
docker run -p 8082:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/explore/api:/api swaggerapi/swagger-ui

docker run -p 8083:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/users/api:/api swaggerapi/swagger-ui
```

**Effort:** 2-3 hours

**Note:** Read Service OpenAPI specs (content and fetcher) are already updated and compliant.

---

### 4. Complete API v1 Migration - Update Integration Tests
**File:** `services/explore/recommender/integration_test.go`

**Issue:** Integration tests use old API paths that don't match the current implementation, causing tests to fail.

**Impact:**
- Integration tests cannot verify the actual API endpoints
- False negatives (tests fail even though code works)
- Reduced confidence in deployment safety
- CI/CD pipeline may fail or provide misleading results

**Current Test Paths (❌ Incorrect):**
```go
// Line 144
resp, err := http.Post(suite.server.URL+"/explore/articles", ...)

// Line 171
resp, err = http.Post(suite.server.URL+"/explore/articles", ...)

// Line 327
resp, err := http.Post(suite.server.URL+"/explore/articles/"+article.ID+"/vote", ...)

// Line 395
resp, err := http.Post(suite.server.URL+"/explore/articles/"+badArticle.ID+"/vote", ...)

// Line 543
resp, err := http.Post(suite.server.URL+"/explore/articles", ...)

// Line 554
resp, err = http.Get(suite.server.URL + "/explore/recommendations/" + userID)

// Line 582
resp, err := http.Post(suite.server.URL+"/explore/articles/"+recommendations[0].ID+"/vote", ...)
```

**Implementation:**

Update all HTTP test requests to use v1 API paths:
```go
// Article submission (lines 144, 171, 543)
resp, err := http.Post(suite.server.URL+"/api/v1/explore/article", ...)

// Get recommendations (line 554)
resp, err = http.Get(suite.server.URL + "/api/v1/explore/recommendation/" + userID)

// Vote endpoints (lines 327, 395, 582)
resp, err := http.Post(suite.server.URL+"/api/v1/explore/article/"+article.ID+"/vote", ...)
```

**Note:** These tests also need JWT authentication to work with the current implementation. Consider:
1. Generating test JWT tokens using the auth middleware
2. Adding `Authorization: Bearer <token>` header to authenticated requests
3. Updating the mock auth middleware in test setup to properly validate tokens

**Verification:**
```bash
cd services/explore
go test -v ./recommender/integration_test.go
```

**Effort:** 1-2 hours

---

### 5. Complete API v1 Migration - Update Service README Files
**Files:**
- `services/explore/README.md`
- `services/users/README.md`

**Issue:** Service-specific README files have not been updated with the new API v1 endpoints and examples.

**Impact:**
- Developers reading service-specific docs see incorrect curl examples
- Service README files contradict CLAUDE.md (which is updated)
- Confusion during local development and testing
- Harder onboarding for new contributors

**Current State:**
- ❌ `services/explore/README.md` - No `/api/v1` paths found, no `/health/live` endpoints
- ❌ `services/users/README.md` - No `/api/v1` paths found
- ✅ `CLAUDE.md` - Already updated with all new paths

**Implementation:**

Update all example curl commands in each README to match the new API structure:

**Explore Service README:**
```markdown
# Example API Calls

## Fetcher Service (port 8080)

# Health checks
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# Manual feed fetch
curl -X POST http://localhost:8080/api/v1/explore/feed/fetch

# Feed statistics
curl http://localhost:8080/api/v1/explore/feed/stats

# Sync feeds
curl -X POST http://localhost:8080/api/v1/explore/feed/sync

## Recommender Service (port 8081)

# Health checks
curl http://localhost:8081/health/live
curl http://localhost:8081/health/ready

# Get recommendations (requires auth)
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/recommendation/user123

# Mark article as read (requires auth)
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/read

# Vote on article (requires auth)
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"vote_type":"upvote"}' \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote
```

**User Service README:**
```markdown
# Example API Calls

# Health checks
curl http://localhost:8082/health/live
curl http://localhost:8082/health/ready

# Register user
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securepass123"}' \
  http://localhost:8082/api/v1/auth/register

# Login
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securepass123"}' \
  http://localhost:8082/api/v1/auth/login

# Get user profile (requires auth)
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8082/api/v1/user/{user_id}

# Update user profile (requires auth)
curl -X PATCH \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"email":"newemail@example.com"}' \
  http://localhost:8082/api/v1/user/{user_id}
```

Also update any endpoint documentation tables and architecture diagrams.

**Verification:**
```bash
# Test each curl command manually to ensure it works
# Or create a script to test all endpoints

# services/explore/test-api.sh
#!/bin/bash
curl -f http://localhost:8080/health/live || echo "❌ Fetcher health check failed"
curl -f http://localhost:8081/health/ready || echo "❌ Recommender health check failed"
# ... etc
```

**Effort:** 1-2 hours

**Reference:** See `docs/API_MIGRATION_PLAN.md` (to be deleted after this task) for complete migration details.

---

### 6. Fix Type Safety Violations in Mobile App
**Files:**
- `src/components/ArticleListScreen.tsx:24` - `onArticlePress` parameter uses `any`
- `src/components/ArticleListScreen.tsx:29-39` - `onViewableItemsChanged` uses `any`
- `src/screens/ExploreScreen.tsx:139` - `onViewableItemsChanged` callback uses `any`

**Issue:** 3 instances of explicit `any` types that reduce TypeScript's type safety and ability to catch errors at compile time.

**Impact:**
- Reduced type safety (bypasses TypeScript's type checking)
- Runtime errors that could be caught at compile time may only surface at runtime
- Poor IDE support (autocomplete and refactoring tools work less effectively)

**Implementation:**

Add imports at the top of each file:
```typescript
import { ViewToken } from 'react-native';
import { Article } from '../types/article';
```

Update ArticleListScreen.tsx:
```typescript
// Before
onArticlePress?: (article: any) => void;
onViewableItemsChanged?: (info: any) => void;

// After
onArticlePress?: (article: Article) => void;
onViewableItemsChanged?: (info: {
  viewableItems: ViewToken[];
  changed: ViewToken[];
}) => void;
```

Update ExploreScreen.tsx:
```typescript
// Before
onViewableItemsChanged={(info: any) => {
  handleViewableItemsChanged(info);
}}

// After
onViewableItemsChanged={(info: {
  viewableItems: ViewToken[];
  changed: ViewToken[];
}) => {
  handleViewableItemsChanged(info);
}}
```

**Verification:**
```bash
cd apps/mobile
npm run type-check  # Should pass without any type errors
npm run lint        # Should not show warnings for explicit any
```

**Effort:** 1-2 hours

---

### 4. Fix React Hook Dependency Warning in ExploreScreen
**File:** `src/screens/ExploreScreen.tsx:35`

**Issue:** `useEffect` hook has a missing dependency (`loadExploreArticles`) that could cause stale closures or missed re-renders.

**Impact:**
- Stale closures (effect may capture old version of function)
- Incorrect behavior (changes to dependencies won't trigger effect re-run)
- React warnings in development mode

**Implementation:**

Wrap function in useCallback (Recommended):
```typescript
const loadExploreArticles = useCallback(async (minArticles?: number) => {
  if (loadingRef.current) return;

  setLoading(true);
  loadingRef.current = true;
  setError(null);

  try {
    // ... existing logic
  } catch (err) {
    // ... existing error handling
  } finally {
    setLoading(false);
    loadingRef.current = false;
  }
}, [/* add dependencies that loadExploreArticles uses */]);

useEffect(() => {
  loadExploreArticles();
}, [loadExploreArticles]);
```

**Verification:**
```bash
cd apps/mobile
npm run lint        # Should show no React hooks warnings
npm start           # Test in development mode
```

**Effort:** 30 minutes

---

### 5. Remove Unused Functions in User Service
**Files:**
- `pkg/auth/examples/explore-service/main.go:90` - Unused variable `pathUserID` (BLOCKS COMPILATION)
- `internal/database/user_repository_test.go:56` - `cleanupTestUserByEmail` function unused
- `internal/database/user_repository_test.go:64` - `cleanupTestUserByDeviceID` function unused
- `internal/middleware/auth.go:161` - `extractTokenFromHeader` function unused

**Issue:** 4 unused functions and 1 unused variable that contribute to code clutter and can confuse developers. One unused variable prevents compilation.

**Impact:**
- Dead code increases codebase size without adding value
- Maintenance burden (developers may waste time reading/updating dead code)
- **Compilation error** in example code (unused variable blocks build)
- Confusion (developers may assume these functions are used somewhere)

**Implementation:**

Fix compilation error (Priority 1):
```go
// Option 1: Remove if not needed
// Delete line 90: pathUserID := r.PathValue("id")

// Option 2: Use the variable
pathUserID := r.PathValue("id")
slog.Info("processing request", "user_id", pathUserID)
// ... use pathUserID in the handler logic
```

For test cleanup functions:
```go
// Option 1: Remove unused functions (delete lines 56-69)

// Option 2: Add to test cleanup (if useful)
func TestCreateUser(t *testing.T) {
    // ... test code ...
    t.Cleanup(func() {
        cleanupTestUserByEmail(t, db, "test@example.com")
    })
}
```

For middleware helper:
```go
// If truly unused, remove the entire function (delete lines 161-174)
```

**Verification:**
```bash
cd services/users
go build ./...        # Check compilation
staticcheck ./...     # Check for unused code
make test
```

**Effort:** 30 minutes

---

### 6. Fix Context Key Type Safety in User Service
**Files:**
- `pkg/auth/middleware_test.go:345`
- `pkg/auth/middleware_test.go:352`

**Issue:** Middleware tests use built-in `string` type as context keys, which can cause collisions if different packages use the same string key.

**Impact:**
- Potential collisions (different packages using same string key will overwrite each other's values)
- Subtle bugs (context value collisions are hard to debug)
- Best practice violation (Go documentation recommends custom types for context keys)

**Implementation:**

Create custom context key type in `pkg/auth/middleware.go` or `pkg/auth/context.go`:
```go
// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context key constants
const (
    userIDKey contextKey = "userID"
)

// SetUserIDInContext adds the user ID to the request context
func SetUserIDInContext(ctx context.Context, userID string) context.Context {
    return context.WithValue(ctx, userIDKey, userID)
}

// GetUserIDFromContext retrieves the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (string, bool) {
    userID, ok := ctx.Value(userIDKey).(string)
    return userID, ok
}
```

Update middleware code:
```go
// Before
ctx := context.WithValue(r.Context(), "userID", userID)

// After
ctx := SetUserIDInContext(r.Context(), userID)
```

Update code that reads from context:
```go
// Before
userID := r.Context().Value("userID").(string)

// After
userID, ok := GetUserIDFromContext(r.Context())
if !ok {
    // handle missing user ID
}
```

**Verification:**
```bash
cd services/users
staticcheck ./...     # Should show no SA1029 warnings
make test
```

**Effort:** 30 minutes

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

### 15. Add API Versioning
**Files:** All API handlers in explore and user services

**Issue:** No API versioning strategy exists. Current endpoints have no version prefix.

**Current:**
```
/auth/login
/explore/recommendations/{userID}
```

**Implementation:**

1. Add `/v1/` prefix to all endpoints:
```
/v1/auth/login
/v1/explore/recommendations/{userID}
```

2. Update route registration:
```go
// Before
mux.HandleFunc("/explore/recommendations/", s.handleRecommendations)

// After
v1 := http.NewServeMux()
v1.HandleFunc("/explore/recommendations/", s.handleRecommendations)
mux.Handle("/v1/", http.StripPrefix("/v1", v1))
```

3. For Gin (user service):
```go
v1 := router.Group("/v1")
{
    auth := v1.Group("/auth")
    {
        auth.POST("/login", authHandler.Login)
        auth.POST("/register", authHandler.Register)
    }
}
```

4. Keep current routes as aliases temporarily for backward compatibility
5. Update all documentation and mobile app to use versioned endpoints

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

### 9. Complete Logging Migration to log/slog
**Reference:** `docs/LOGGING_STRATEGY.md` - See detailed migration strategy and implementation patterns

**Status:** ~70% complete - Phase 1 complete, Phase 2 partial, Phase 3 not started

**Issue:** Migration to structured logging with `log/slog` is incomplete. While main service entry points and the `pkg/logging` package are fully implemented per spec, many internal components still use old logging patterns (`log.Printf`, `log.Println`, `fmt.Printf`).

**Current State:**
- ✅ **Phase 1 Complete** - Shared `pkg/logging` package created with all recommended components:
  - `pkg/logging/logger.go` - Logger initialization and configuration
  - `pkg/logging/middleware.go` - Gin HTTP middleware
  - `pkg/logging/chi_middleware.go` - Chi HTTP middleware
  - `pkg/logging/context.go` - Context helpers
- 🟡 **Phase 2 Partial** - Service main.go files migrated, internals incomplete:
  - User Service: 95% (3 files remaining)
  - Explore Services: 90% (3 files remaining)
  - Read Content Service: 85% (2 files remaining)
  - Read Ingest RSS: 40% (18 files remaining)
- ❌ **Phase 3 Not Started** - Old logging patterns not cleaned up:
  - **214 occurrences** of old logging across **32 files**
  - Old prefixes still in use (`AUTH_SUCCESS`, `[DB]`, etc.)

**Impact:**
- Inconsistent log formatting across services
- Cannot filter logs by severity level in production
- Logs not machine-parseable (no JSON support in old code)
- Harder to search and correlate logs across services
- Old middleware files add confusion and maintenance burden

**Files Requiring Migration:**

**User Service** (3 files):
- `services/users/internal/middleware/logging.go` - **DELETE** (replaced by `pkg/logging`)
  - Contains old `log.Printf` patterns with `AUTH_SUCCESS`, `AUTH_FAILURE` prefixes
  - Superseded by `pkg/logging.RequestLogger()` already in use
- `services/users/cmd/migrate/main.go` - Migrate to slog
- `services/users/pkg/auth/examples/` - Low priority example files

**Explore Service** (3 files):
- `services/explore/recommender/cmd/explore_cleanup/main.go` - Uses `log.Printf`
- `services/explore/recommender/internal/cleanup/article_cleanup.go` - Uses `log.Printf`
- `services/explore/fetcher/internal/client/recommender_client.go` - Uses `log.Printf`

**Read Service - Content** (2 files):
- `services/read/content/internal/api/middleware/error_handler.go` - Uses `log.Printf`
- `services/read/content/internal/database/connection.go` - Uses `log.Printf`

**Read Service - Ingest RSS** (18 files - HIGHEST PRIORITY):
- `internal/fetcher/feed_fetcher.go:99,123,141,144` - Multiple `log.Printf` calls
- `internal/processor/item_processor.go`
- `internal/processor/update_detector.go`
- `internal/scheduler/poll_scheduler.go`
- `internal/scheduler/tier_manager.go`
- `internal/worker/feed_worker.go`
- `internal/worker/outbox_worker.go`
- `internal/jobs/content_extraction_job.go`
- `internal/jobs/feed_items_cleanup_job.go`
- `internal/jobs/feed_items_cleanup_scheduler.go`
- `internal/jobs/outbox_cleanup_job.go`
- `internal/jobs/outbox_cleanup_scheduler.go`
- `internal/api/middleware/recovery.go`
- `internal/api/handlers/subscription_handler.go`
- `cmd/ingest_rss_worker/main.go`
- `cmd/worker/main.go`

**Implementation:**

Follow patterns from `docs/LOGGING_STRATEGY.md`:

```go
// Before (old logging)
log.Printf("Fetching feed %s (%s)", feed.ID, feed.FeedURL)
log.Printf("Error processing feed item %s: %v", item.GUID, err)
fmt.Printf("warning: failed to update timestamp: %v\n", err)

// After (structured slog)
slog.Info("fetching feed",
    slog.String("feed_id", feed.ID),
    slog.String("feed_url", feed.FeedURL),
)
slog.Error("failed to process feed item",
    slog.String("feed_item_guid", item.GUID),
    slog.Any("error", err),
)
slog.Warn("failed to update timestamp",
    slog.Any("error", err),
)
```

For old middleware file:
```bash
# Delete the obsolete middleware file
rm services/users/internal/middleware/logging.go

# Verify no imports reference it
grep -r "internal/middleware" services/users/
```

**Standard Attribute Names** (use consistently):
- `service` - Service name (user-service, fetcher, recommender)
- `component` - Logical component (auth, recommendations, db)
- `request_id` - UUID for request tracing
- `user_id` - User identifier
- `article_id` - Article identifier
- `feed_id` - Feed identifier
- `duration` - Operation duration
- `error` - Error object
- `status` - HTTP status code
- `method` - HTTP method
- `path` - URL path
- `client_ip` - Client IP address
- `count` - Count of items
- `operation` - DB/API operation name

**Verification:**
```bash
# Check for remaining old patterns (should return 0 or minimal)
grep -r "log\.Printf\|log\.Println" services --include="*.go" | wc -l

# Verify all services use pkg/logging
grep -r "pkg/logging" services/*/cmd/*/main.go

# Test services still work
cd infrastructure/docker && docker-compose up --build
```

**Effort:** 4-6 hours
- User Service: 30 minutes (mostly deletion)
- Explore Services: 1 hour
- Read Content Service: 30 minutes
- Read Ingest RSS: 3 hours (most work)

**Priority Order:**
1. **Read/Ingest RSS internals** (highest impact - 18 files)
2. **Delete obsolete User Service middleware** (quick win)
3. **Explore Service utilities** (cleanup jobs)
4. **Read Content Service middleware** (minor fixes)
5. **Example files** (lowest priority)

**Reference:** See `docs/LOGGING_STRATEGY.md` for:
- Complete migration strategy
- Log level guidelines (DEBUG, INFO, WARN, ERROR)
- Environment configuration (LOG_LEVEL, LOG_FORMAT)
- Output examples (JSON for prod, text for dev)

---

## Low Priority

### 7. Clean Up Unused Dependency in Mobile App
**File:**
- `apps/mobile/package.json:27` - Unused dependency `expo-linking`

**Issue:** The `expo-linking` package is installed but never imported or used in the codebase. This increases node_modules size and installation time.

**Status:**
- ✅ **FIXED:** Unused imports (ArticleRow.tsx) - removed
- ✅ **FIXED:** Variable declaration (ExploreScreen.tsx) - fixed/removed
- ✅ **FIXED:** Unused file (src/navigation/index.ts) - deleted
- ❌ **REMAINING:** expo-linking dependency still present

**Impact:**
- Increases node_modules size unnecessarily (~1-2MB)
- Increases npm install time
- Minor bundle size impact (if tree-shaking doesn't eliminate it)

**Implementation:**

**Option 1: Remove if truly unused (recommended)**
```bash
cd apps/mobile
npm uninstall expo-linking
```

**Option 2: Document if reserved for future use**
Add comment to package.json or create a file documenting planned features:
```json
{
  "dependencies": {
    "expo-linking": "~8.0.10"  // Reserved for deep linking feature (planned)
  }
}
```

**Note:** expo-linking is typically used for:
- Deep linking (opening app from URLs)
- Universal links (iOS/Android app links)
- URL parsing and validation

If these features are planned, keep the dependency and document it. Otherwise, remove to reduce bundle size.

**Verification:**
```bash
cd apps/mobile
npm run type-check  # Should pass
npm run lint        # Should pass
npm start           # App should work normally
npm run ios         # Test deep linking if keeping dependency
```

**Effort:** 15 minutes

---

### 8. Apply Code Optimizations in Explore Service
**Files:**
- `fetcher/internal/fetcher/fetcher.go:166` - Loop can be simplified
- `fetcher/internal/fetcher/fetcher_test.go:487` - Potential nil pointer dereference
- `recommender/internal/api/server.go:56` - Use tagged switch instead of if-else chain

**Issue:** 3 code optimization opportunities identified by staticcheck that would improve code quality and maintainability.

**Impact:**
- Readability (clearer, more idiomatic Go code)
- Performance (minor improvements from loop optimization)
- Bug prevention (fixing potential nil pointer dereference)

**Implementation:**

Loop simplification (fetcher.go:166):
```go
// Before (4 lines, loop overhead)
categories := make([]string, 0, len(item.Categories))
for _, cat := range item.Categories {
    categories = append(categories, cat)
}

// After (1 line, no loop)
categories := append([]string(nil), item.Categories...)
```

Nil safety (fetcher_test.go:484-487):
```go
// Before (potential panic)
if lastFetchedAfter == nil {
    t.Error("...")
}
if !lastFetchedAfter.After(lastFetch) {  // Could panic if nil!
    t.Errorf("...")
}

// After (safe)
if lastFetchedAfter == nil {
    t.Fatal("...")  // Stops test immediately
}
if !lastFetchedAfter.After(lastFetch) {  // Safe - can't be nil here
    t.Errorf("...")
}
```

Switch clarity (server.go:56):
```go
// Before (if-else chain)
if r.Method == http.MethodPost {
    handleVoteSubmit(w, r)
} else if r.Method == http.MethodDelete {
    handleVoteRemove(w, r)
} else if r.Method == http.MethodGet {
    handleVoteQuery(w, r)
} else {
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}

// After (clean switch)
switch r.Method {
case http.MethodPost:
    handleVoteSubmit(w, r)
case http.MethodDelete:
    handleVoteRemove(w, r)
case http.MethodGet:
    handleVoteQuery(w, r)
default:
    http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
```

**Verification:**
```bash
cd services/explore
make test
staticcheck ./...  # Should show no S1011, SA5011, QF1003 warnings
```

**Effort:** 1 hour

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
- **Standardize Configuration Management** - Created shared `pkg/config/config.go` package with common configuration patterns including `DatabaseConfig`, `ServerConfig`, and `LoggingConfig` structs with validation methods. Implemented helper functions (`GetString`, `GetInt`, `GetBool`, `GetDuration`) for environment variable parsing. Created service-specific config packages for all services: `services/explore/fetcher/internal/config`, `services/explore/recommender/internal/config`, `services/read/content/internal/config`, and `services/read/fetcher/internal/config`. Each service config package uses the shared `pkg/config` package and adds service-specific configuration options. Updated all main.go files to use centralized configuration with validation at startup. Removed duplicate `getEnv` helper functions from all services. Added replace directives in service go.mod files to reference the shared config package. All services (explore/fetcher, explore/recommender, read/content, read/fetcher) build successfully. Benefits include centralized configuration management, validation at startup to prevent runtime errors, consistent patterns across all services, easier testing of configuration logic, and elimination of code duplication. This implementation combines tasks #12 (Standardize Configuration Management for Explore services) and #25 (Centralize Configuration Management in Read Service).

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
- **Fix Unchecked Error Returns in Explore Service - Fetcher** - Fixed unchecked error returns that could lead to resource leaks, silent failures, and data inconsistencies. Added proper error handling for transaction rollback in feed_repository.go with pgx.ErrTxClosed check. Migrated recommender_client.go from log.Printf to slog.Warn for structured logging of response body close errors. Added error handling for json.NewEncoder().Encode() in main.go health check handlers (liveness and readiness). All services compile successfully. Benefits include prevention of resource leaks from unchecked transaction rollbacks, consistent error logging with structured logging (slog), and prevention of silent JSON encoding failures in HTTP responses.
- **Fix Unchecked Error Returns in Explore Service - Recommender** - Fixed unchecked error returns and compilation issues in the recommender service. Key changes include: (1) Migrated article_repository_test.go from database/sql to pgxpool.Pool for consistency with the repository interface after pgx migration; (2) Fixed 6 unchecked resp.Body.Close() errors in integration_test.go by adding proper error handling with defer functions; (3) Fixed unchecked db.Close() error in migrate.go by adding error logging; (4) Fixed context key type safety issue in middleware.go by creating a custom contextKey type to avoid collisions; (5) Removed unused articleScanner interface and pgx import from interfaces.go. All golangci-lint issues resolved (0 issues), and all unit tests pass. Benefits include prevention of resource leaks from unchecked Close() calls, better type safety for context values, and cleaner codebase without unused code.

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
