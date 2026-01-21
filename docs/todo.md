# Project-Wide TODO - Required Fixes

Comprehensive code review findings from December 2025. Issues are organized by priority and span all services (explore, users, mobile app).

IMPORTANT: After implementing a task, move it to the completed section at the end of the file. Include only a 1-2 description 

---

## Critical Priority

---

## High Priority

---

## Medium Priority

### Task 6. Internal API Routes Without Authentication

Issue: services/read/content/internal/api/router.go:104-109:

// Internal API routes - used by internal services (Ingest RSS, etc.)
// These routes do NOT require authentication as they are internal-only
r.Route("/api/v1/internal", func(r chi.Router) {
    r.Post("/content/user/bulk", bulkHandler.BulkAddToUsersInternal)
})

Risk: If internal network is compromised, these endpoints are unprotected.

Recommendation: Use service-to-service authentication:

    Mutual TLS (mTLS)
    API keys validated against Vault
    JWT tokens with service-level claims


### Task 8. Error Messages May Leak Information

Issue: Detailed error messages returned to clients (gin_adapter.go:71-89):

"error": "token has expired"
"error": "invalid token signature"
"error": "invalid token issuer"

Recommendation: Return generic "unauthorized" message; log specific errors server-side.


---

## Low Priority

### Task 9. No JTI (JWT ID) for Token Revocation

Issue: JWTs don't include unique identifiers.

Impact: Cannot revoke individual access tokens if compromised.

Recommendation: Add jti claim for token blacklisting capability if needed.


### Task 10. Console Logging for Security Events

Issue: Security events logged to stdout (refresh_token.go:159):

fmt.Printf("failed to revoke token family on reuse: %v\n", err)

Recommendation: Use structured logging with proper log levels and security event tagging.


Concerns
### Task 11. No Vault Response Caching

Keys are fetched synchronously on startup. Consider caching with TTL.

### Task 12. Vault Connection Not Retried

If Vault is temporarily unavailable at startup, service fails immediately.

Recommendation: Implement retry with exponential backoff for Vault connection.

### Task 13. Align on Gin middleware

Framework variations: User Service uses Gin with pkg/auth.GinMiddleware, while Read/Explore use net/http with pkg/auth.Middleware. Both use the same underlying validator - acceptable.

### Task 14. Review OptionalAuth behaviour

Optional Auth behavior: When invalid token provided, OptionalAuth continues without authentication. This is documented but could allow requests that appear authenticated to proceed unauthenticated.




---

### Task 17. Add Package-Level Documentation
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

### Task 22. Add HTTP Framework Decision Documentation
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

### Task 18. Add Mobile App Test Infrastructure
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

### Task 19. Optimize N+1 Query in Recommendation Flow
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
- **Reduce Access Token Expiry** (Task #2) - Changed default JWT access token lifetime from 60 minutes to 15 minutes in `services/users/internal/config/config.go`. This reduces the attack window if tokens are compromised. The value remains configurable via `JWT_ACCESS_TOKEN_EXPIRY` environment variable.
- Missing kid (Key ID) Header for Key Rotation
- **Thread-Safe Key Updates in JWTManager** (Task #3) - Added `sync.RWMutex` to `JWTManager` struct in `services/users/internal/auth/jwt.go` to protect key fields during rotation. Updated `GenerateToken()`, `ValidateToken()`, `UpdateKeys()`, `GetPublicKey()`, and `GetKeyID()` methods to use appropriate read/write locks. This prevents race conditions where token generation could use mismatched keys during key rotation. Added comprehensive concurrent access tests that pass with Go's race detector.
- **Thread-Safe Key Updates in Validator** (Task #4) - Added `sync.RWMutex` to `Validator` struct in `pkg/auth/validator.go` to protect key fields during rotation. Updated `ValidateToken()`, `UpdatePublicKey()`, `GetPublicKey()`, and `GetKeyID()` methods to use appropriate read/write locks. Added comprehensive concurrent access tests that pass with Go's race detector.
- **Increase Token Reuse Grace Period** (Task #5) - Increased `TokenReuseGracePeriod` from 5 seconds to 15 seconds in `services/users/internal/auth/refresh_token.go`. This prevents false positive token reuse detection caused by network latency or retry logic.
- **Replace Panic-Based User ID Extraction with Error Handling** (Task #7) - Added `GetUserIDOrError()` function to `pkg/auth/middleware.go` and `GetUserIDFromGinContext()` update to `pkg/auth/gin_adapter.go` that return errors instead of panicking when user ID is not found in context. Updated all handler implementations in explore/recommender and read/content services to use the error-returning functions. This prevents service crashes from programming errors (e.g., missing middleware). Deprecated `MustGetUserID()` and `MustGetUserIDFromGin()` functions with documentation recommending the new approach. Added comprehensive tests for error handling scenarios.

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
- **Setup Testcontainers for Integration Tests** (Task #2) - Implemented testcontainers-go for PostgreSQL integration tests in the Explore Recommender service. Created `services/explore/recommender/internal/testutil/db.go` with `SetupTestDB()` helper function that automatically spins up PostgreSQL 16 containers, runs migrations, and provides pgxpool connections. Updated `article_repository_integration_test.go` to use testcontainers and migrated from database/sql to pgxpool for consistency. Integration tests now work without manual database setup - just require Docker. Added comprehensive testing documentation to services/explore/README.md covering unit tests, integration tests, and CI/CD considerations. Tests can be skipped with `-short` flag for environments without Docker. 
