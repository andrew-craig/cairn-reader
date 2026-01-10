# Project-Wide TODO - Required Fixes

Comprehensive code review findings from December 2025. Issues are organized by priority and span all services (explore, users, mobile app).

---

## Critical Priority

---

## High Priority

---

## Medium Priority

### 2. Setup Testcontainers for Integration Tests
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

## Low Priority

### 3. Clean Up Unused Dependency in Mobile App
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

### 4. Add Standardized Error Variables
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
import apperrors "github.com/cairn-app/cairn-reader/pkg/errors"

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

### 5. Improve Router for Path Parameter Handling
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

### 6. Consolidate getEnv Helper Functions
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
import "github.com/cairn-app/cairn-reader/pkg/env"

port := env.GetString("PORT", "8080")
timeout := env.GetDuration("FETCH_TIMEOUT", 30*time.Second)
maxRetries := env.GetInt("MAX_RETRIES", 3)
```

---

### 7. Add Package-Level Documentation
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

### 8. Add Mobile App Test Infrastructure
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

### 9. Optimize N+1 Query in Recommendation Flow
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

## Pre-Go Live

* Load testing
* Alerting
* Metrics

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
- **Add OpenAPI/Swagger Specifications** - Created comprehensive OpenAPI 3.0 specifications for both Explore and User services 
- **Add API Versioning** (Task #11) - Benefits include API evolution capability, backward compatibility support, clear API contracts, and consistent API design across all microservices.
- **Complete API v1 Migration - Update OpenAPI Specifications** - Updated OpenAPI specifications to reflect the new API v1 structure 
- **Complete API v1 Migration - Update Integration Tests** 
- **Complete API v1 Migration - Update Service README Files** 

### Code Organization & Maintainability
- **Consolidate Duplicate Logging Package** - Created shared logging package at repository root (`pkg/logging/`), updated imports in all services, eliminated duplicate code across explore and users services
- **Standardize Logging Library to log/slog in Read Service** 
- **Move Shared Models to Root Package** - Created `pkg/models/` at repository root and migrated all domain models (Article, User, Vote, Feed, RecommendationEvent) from `services/explore/pkg/models/`.
- **Fix Module Dependency Architecture** - Extracted auth middleware from `services/users/pkg/auth/` to shared `pkg/auth/` package at repository root. 
- **Consolidate Auth Middleware Implementations** - Created Gin-compatible wrappers in `pkg/auth/gin_adapter.go` with `NewGinMiddleware()` factory function, `JWTAuth()` and `OptionalAuth()` middleware methods, and context helpers (`GetUserIDFromGinContext()`, `MustGetUserIDFromGin()`, `IsAuthenticatedInGin()`). 
- **Rename testhelpers to testutil for Consistency** - 
- **Add Repository Interfaces to Explore Services** - Implemented repository interface pattern for all Explore service repositories, matching the pattern already used in User and Read services. 
- **Standardize Configuration Management** - Created shared `pkg/config/config.go` package with common configuration patterns including `DatabaseConfig`, `ServerConfig`, and `LoggingConfig` structs with validation methods. Implemented helper functions (`GetString`, `GetInt`, `GetBool`, `GetDuration`) for environment variable parsing. Created service-specific config packages for all services: 
- **Standardize Error Response Format** 

### Security & Reliability
- **Add Request Body Size Limits** - Implemented MaxBytesReader with appropriate limits (10MB for batch, 1KB for simple requests)
- **Validate Article Exists Before Recording Vote** - Added rowsAffected check and error handling
- **Make SSL Mode Configurable (Default: Require)** - DB_SSLMODE environment variable with "require" default
- **Remove Hardcoded Secrets from Docker Compose** - Migrated all hardcoded secrets from docker-compose files to environment variables.
- **Add Input Validation Library** (Task #1) - Implemented comprehensive request validation across all services using appropriate validation libraries. Stdlib services (Explore Recommender, Read Content, Read Ingest RSS) now use `github.com/go-ozzo/ozzo-validation/v4` with declarative validation methods on all DTOs. Gin service (User Service) already uses built-in Gin validation tags (`binding:"required,email"`). All handlers updated to call validation methods before processing requests. Benefits include consistent validation, better error messages, reduced boilerplate code, and improved maintainability. 

### Code Quality & Performance
- **Replace O(n²) Sorting with Standard Library** - Using sort.Slice for O(n log n) performance
- **Standardize Logging to slog** - Main service code migrated from log.Printf to structured slog
- **Extract URL Path Parsing Helper** - Implemented extractPathParam helper function
- **Log Warning for Silent Vote Counter Failures** - Added slog.Warn for rowsAffected == 0 cases
- **Fix Unchecked Error Returns in Explore Service - Fetcher and Recommender** 
- **Fix Type Safety Violations in Mobile App**
- **Fix Context Key Type Safety in User Service**

### Infrastructure & Architecture
- **Add Connection Pool Configuration to Fetcher** - Configured MaxOpenConns, MaxIdleConns, and ConnMaxLifetime
- **Implement Kubernetes-Style Health Endpoints** - Separate /health (liveness) and /ready (readiness) endpoints
- **Add Request ID Propagation** - X-Request-ID header generation and context propagation for distributed tracing
- **Validate User IDs as UUIDs** - UUID format validation in EnsureUserExists
- **Standardize Database Architecture Pattern** - Migrated infrastructure docker-compose.yml from single PostgreSQL instance with multiple databases to separate PostgreSQL instances per service (users-db:5432, recommender-db:5433, fetcher-db:5434). 

### Refactoring
- **Delete Unused Gin Middleware** - Removed from explore service (user service middleware is actively used)
- **Cache Internal User ID in Recommendation Flow** - Addressed by using external user ID from JWT token directly throughout the flow

### Testing & Quality Assurance
- **Add Tests for Recommendation Engine** 
