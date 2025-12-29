# Project-Wide TODO - Required Fixes

Comprehensive code review findings from December 2025. Issues are organized by priority and span all services (explore, users, mobile app).

---

## Critical Priority

### 1. Consolidate Duplicate Logging Package
**Files:**
- `services/explore/pkg/logging/logger.go` (53 lines)
- `services/users/pkg/logging/logger.go` (53 lines)

**Issue:** Identical logging code duplicated across services. This creates maintenance burden and potential drift.

**Current State:**
```go
// IDENTICAL CODE IN BOTH LOCATIONS:
package logging

import "log/slog"

type Config struct {
    Level  string
    Format string
}

func NewLogger(cfg Config) *slog.Logger {
    // ... identical implementation
}
```

**Implementation:**
1. Create shared package at repository root:
```bash
mkdir -p pkg/logging
mv services/explore/pkg/logging/* pkg/logging/
```

2. Update imports in both services:
```go
// Before
import "github.com/andrew-craig/cairn/explore/pkg/logging"

// After
import "github.com/andrew-craig/cairn/pkg/logging"
```

3. Update `go.mod` files to reference shared package
4. Delete duplicate from `services/users/pkg/logging/`
5. Run `go mod tidy` in all services

---

### 2. Standardize PostgreSQL Driver to pgx/v5
**Files:**
- `services/explore/fetcher/internal/db/config.go`
- `services/explore/recommender/cmd/recommender/main.go`
- All explore service database code

**Issue:** Explore services use `database/sql` + `lib/pq` while user service uses modern `pgx/v5`. This creates inconsistency in connection pooling, performance, and feature availability.

**Current State:**
```go
// Explore services
import _ "github.com/lib/pq"
db, err := sql.Open("postgres", connStr)

// User service
import "github.com/jackc/pgx/v5/pgxpool"
pool, err := pgxpool.NewWithConfig(context.Background(), config)
```

**Implementation:**
1. Update `services/explore/go.mod`:
```bash
go get github.com/jackc/pgx/v5
go get github.com/jackc/pgx/v5/pgxpool
```

2. Replace connection logic in `fetcher/internal/db/config.go`:
```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
)

func (c *Config) Connect(ctx context.Context) (*pgxpool.Pool, error) {
    connStr := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
        c.User, c.Password, c.Host, c.Port, c.DBName,
    )

    config, err := pgxpool.ParseConfig(connStr)
    if err != nil {
        return nil, fmt.Errorf("failed to parse connection string: %w", err)
    }

    // Connection pool settings
    config.MaxConns = 25
    config.MinConns = 5
    config.MaxConnLifetime = 5 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, config)
    if err != nil {
        return nil, fmt.Errorf("failed to connect: %w", err)
    }

    return pool, nil
}
```

3. Update all repository methods to use pgx types
4. Replace `*sql.DB` with `*pgxpool.Pool` throughout
5. Update queries to use pgx syntax (named parameters, etc.)
6. Run all tests to verify migration

**Benefits:**
- 2-3x faster query performance
- Better connection pooling with health checks
- Native PostgreSQL protocol support
- Modern, actively maintained library

---

### 3. Add OpenAPI/Swagger Specifications
**Files to Create:**
- `services/explore/api/openapi.yaml`
- `services/users/api/openapi.yaml`

**Issue:** No formal API documentation exists. APIs are documented only in CLAUDE.md and code comments, making client integration difficult.

**Implementation:**

1. Create directory structure:
```bash
mkdir -p services/explore/api
mkdir -p services/users/api
```

2. Add OpenAPI 3.0 spec for Explore service (`services/explore/api/openapi.yaml`):
```yaml
openapi: 3.0.3
info:
  title: Cairn Explore Service API
  version: 1.0.0
  description: RSS feed fetching and article recommendation service

servers:
  - url: http://localhost:8080
    description: Fetcher service
  - url: http://localhost:8081
    description: Recommender service

paths:
  /health:
    get:
      summary: Health check
      responses:
        '200':
          description: Service is healthy

  /explore/recommendations/{userID}:
    get:
      summary: Get personalized article recommendations
      parameters:
        - name: userID
          in: path
          required: true
          schema:
            type: string
            format: uuid
      security:
        - bearerAuth: []
      responses:
        '200':
          description: List of recommended articles
          content:
            application/json:
              schema:
                type: array
                items:
                  $ref: '#/components/schemas/Article'

  /explore/articles/{articleID}/vote:
    post:
      summary: Vote on an article
      parameters:
        - name: articleID
          in: path
          required: true
          schema:
            type: string
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                vote_type:
                  type: string
                  enum: [upvote, downvote]
              required:
                - vote_type
      security:
        - bearerAuth: []
      responses:
        '200':
          description: Vote recorded
        '400':
          description: Invalid vote type
        '401':
          description: Unauthorized

components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    Article:
      type: object
      properties:
        id:
          type: string
        title:
          type: string
        link:
          type: string
          format: uri
        description:
          type: string
        published_at:
          type: string
          format: date-time
        author:
          type: string
        categories:
          type: array
          items:
            type: string
```

3. Add similar spec for User service with auth endpoints
4. Install swagger-ui for local viewing:
```bash
docker run -p 8082:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/explore/api:/api swaggerapi/swagger-ui
```

5. Add to CI/CD pipeline for validation:
```bash
npm install -g @apidevtools/swagger-cli
swagger-cli validate services/explore/api/openapi.yaml
```

---

### 4. Add Tests for Recommendation Engine
**File:** `services/explore/recommender/internal/recommend/engine.go`

**Issue:** The recommendation engine contains core business logic but has no test coverage. This is critical as it implements the quality scoring algorithm.

**Implementation:**

Create `services/explore/recommender/internal/recommend/engine_test.go`:

```go
package recommend

import (
    "context"
    "testing"
    "time"

    "github.com/andrew-craig/cairn/explore/pkg/models"
)

func TestQualityScoreCalculation(t *testing.T) {
    tests := []struct {
        name        string
        upvotes     int
        downvotes   int
        recommends  int
        wantScore   float64
    }{
        {
            name:       "high quality article",
            upvotes:    10,
            downvotes:  1,
            recommends: 20,
            wantScore:  0.65, // (10 + (1*3)) / 20 = 13/20
        },
        {
            name:       "low quality article",
            upvotes:    2,
            downvotes:  5,
            recommends: 30,
            wantScore:  0.567, // (2 + (5*3)) / 30 = 17/30
        },
        {
            name:       "no engagement",
            upvotes:    0,
            downvotes:  0,
            recommends: 10,
            wantScore:  0.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            article := &models.Article{
                Upvotes:    tt.upvotes,
                Downvotes:  tt.downvotes,
                Recommends: tt.recommends,
            }

            score := calculateQualityScore(article)

            if score != tt.wantScore {
                t.Errorf("calculateQualityScore() = %v, want %v", score, tt.wantScore)
            }
        })
    }
}

func TestRecommendationSelection(t *testing.T) {
    engine := &Engine{
        // mock repositories
    }

    articles := []*models.Article{
        {ID: "1", Upvotes: 10, Downvotes: 1, Recommends: 20},  // High quality
        {ID: "2", Upvotes: 5, Downvotes: 2, Recommends: 15},   // Medium quality
        {ID: "3", Upvotes: 2, Downvotes: 8, Recommends: 25},   // Low quality
        {ID: "4", Upvotes: 0, Downvotes: 0, Recommends: 1},    // New article
    }

    recommendations, err := engine.GetRecommendations(context.Background(), "user123")
    if err != nil {
        t.Fatalf("GetRecommendations() error = %v", err)
    }

    if len(recommendations) != 5 {
        t.Errorf("Expected 5 recommendations, got %d", len(recommendations))
    }

    // Verify 4 high-quality + 1 low-recommends article
    lowRecommendsCount := 0
    for _, rec := range recommendations {
        if rec.Recommends < 5 {
            lowRecommendsCount++
        }
    }

    if lowRecommendsCount != 1 {
        t.Errorf("Expected 1 low-recommends article, got %d", lowRecommendsCount)
    }
}

func TestRecommendationDiversity(t *testing.T) {
    // Test that same user doesn't get same articles twice
    // Test category diversity
    // Test recency bias
}

func TestFilterDeletedArticles(t *testing.T) {
    // Test that deleted articles are excluded
}
```

Add integration test:
```go
func TestRecommendationEngineIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test database
    // Insert test articles
    // Call engine.GetRecommendations()
    // Verify results
    // Verify recommends counter incremented
}
```

Run tests:
```bash
cd services/explore/recommender
go test ./internal/recommend/... -v
go test ./internal/recommend/... -v -short  # Skip integration tests
```

---

## High Priority

---

### 4. Move Shared Models to Root Package
**Files:**
- `services/explore/pkg/models/article.go`
- `services/explore/pkg/models/user.go`
- `services/explore/pkg/models/vote.go`

**Issue:** Shared domain models are located in the explore service package, but are imported by user service. This creates coupling where user service depends on explore service code.

**Current State:**
```go
// In services/users/go.mod
replace github.com/andrew-craig/cairn-core/user-service => ../users

// This forces user service to import from explore
import "github.com/andrew-craig/cairn/explore/pkg/models"
```

**Implementation:**
1. Create shared models package:
```bash
mkdir -p pkg/models
mv services/explore/pkg/models/* pkg/models/
```

2. Update imports across both services:
```go
// Before
import "github.com/andrew-craig/cairn/explore/pkg/models"

// After
import "github.com/andrew-craig/cairn/pkg/models"
```

3. Remove replace directive from go.mod files
4. Run `go mod tidy` in all services

---

### 5. Fix Module Dependency Architecture
**Files:**
- `services/explore/go.mod` (line 6)
- `services/explore/recommender/Dockerfile`

**Issue:** The explore service uses a replace directive pointing to local filesystem, creating tight coupling and special Dockerfile handling.

**Current State:**
```go
// services/explore/go.mod
replace github.com/andrew-craig/cairn-core/user-service => ../users
```

This forces the Dockerfile to copy both directories:
```dockerfile
# services/explore/recommender/Dockerfile
COPY users users/          # Must copy users dir for auth import
COPY explore explore/
WORKDIR /app/explore
```

**Implementation:**

**Option 1: Extract to Shared Package (Recommended)**
1. Move auth middleware to root:
```bash
mkdir -p pkg/auth
mv services/users/pkg/auth/* pkg/auth/
```

2. Update imports in all services
3. Remove replace directive
4. Simplify Dockerfile to only copy explore directory

**Option 2: Use Go Workspace**
1. Create `go.work` at repository root:
```bash
go work init
go work use services/explore services/users
```

2. This enables local development without replace directives
3. For production builds, use proper module versioning

---

### 6. Remove Hardcoded Secrets from Docker Compose
**File:** `infrastructure/docker/docker-compose.yml`

**Issue:** Secrets are hardcoded in version-controlled docker-compose file. This is a security risk if the repository becomes public or is compromised.

**Current State:**
```yaml
vault:
  environment:
    VAULT_DEV_ROOT_TOKEN_ID: dev-token

postgres:
  environment:
    POSTGRES_PASSWORD: cairn_password
```

**Implementation:**

1. Create `.env.example` template:
```bash
# .env.example
POSTGRES_PASSWORD=your_password_here
VAULT_DEV_ROOT_TOKEN_ID=your_vault_token_here
DB_PASSWORD=your_db_password_here
```

2. Add `.env` to `.gitignore`:
```bash
echo ".env" >> .gitignore
```

3. Update docker-compose.yml to use environment variables:
```yaml
postgres:
  environment:
    POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}

vault:
  environment:
    VAULT_DEV_ROOT_TOKEN_ID: ${VAULT_DEV_ROOT_TOKEN_ID}
```

4. Create local `.env` file for development (never commit):
```bash
cp .env.example .env
# Edit .env with actual values
```

5. Update documentation in README.md:
```markdown
## Setup

1. Copy `.env.example` to `.env`
2. Update `.env` with your local configuration
3. Run `docker-compose up`
```

---

## Medium Priority

### 4. Standardize Database Architecture Pattern
**Files:**
- `infrastructure/docker/docker-compose.yml`
- `services/explore/docker-compose.yml`

**Issue:** Two different deployment patterns exist for PostgreSQL databases, creating inconsistency.

**Current State:**

Explore service (separate instances - good pattern):
```yaml
postgres:           # Recommender DB
  image: postgres:16-alpine
  environment:
    POSTGRES_DB: cairn_db

fetcher_db:        # Separate instance for fetcher
  image: postgres:16-alpine
  environment:
    POSTGRES_DB: fetcher_db
```

Infrastructure compose (single instance):
```yaml
postgres:          # Single instance
  image: postgres:16-alpine
  environment:
    POSTGRES_DB: postgres
  # Multiple databases created via init script
```

**Implementation:**
Standardize on the explore service pattern (separate instances). Update infrastructure/docker-compose.yml to use separate PostgreSQL instances for each service, ensuring proper microservices isolation.

---

### 11. Add Repository Interfaces to Explore Services
**Files:**
- `services/explore/fetcher/internal/db/feed_repository.go`
- `services/explore/recommender/internal/db/article_repository.go`
- `services/explore/recommender/internal/db/vote_repository.go`

**Issue:** User service defines repository interfaces for better testability, but explore services use concrete implementations directly.

**User service pattern (good):**
```go
// users/internal/database/user_repository.go
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id string) (*models.User, error)
    // ...
}

type postgresUserRepository struct {
    db *pgxpool.Pool
}
```

**Explore services (direct structs):**
```go
type FeedRepository struct {
    db *sql.DB
}
```

**Implementation:**

1. Define interfaces in `internal/db/interfaces.go`:
```go
package db

type FeedRepository interface {
    Create(ctx context.Context, feed *models.Feed) error
    List(ctx context.Context) ([]*models.Feed, error)
    // ... other methods
}

type ArticleRepository interface {
    Store(ctx context.Context, articles []*models.Article) error
    GetRecommendations(ctx context.Context, userID string, limit int) ([]*models.Article, error)
    // ... other methods
}
```

2. Rename existing structs to implementation names:
```go
type postgresFeedRepository struct {
    db *pgxpool.Pool
}

func NewFeedRepository(db *pgxpool.Pool) FeedRepository {
    return &postgresFeedRepository{db: db}
}
```

3. Update handlers to use interfaces instead of concrete types

**Benefits:**
- Easier unit testing with mocks
- Better dependency injection
- Cleaner API surface

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

## Low Priority

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

**Total Issues:** 41 (4 Critical, 6 High, 17 Medium, 14 Low)

**Quick Wins (Can be done in <1 hour each):**
- Critical #1: Consolidate logging package
- High #4: Move shared models
- High #6: Remove hardcoded secrets
- Low #20: Consolidate getEnv helpers
- Low #21: Add package documentation

**High Impact (Address first):**
- Critical #2: Standardize on pgx/v5 driver
- Critical #3: Add OpenAPI specs
- Critical #4: Add recommendation engine tests
- Medium #15: Add API versioning
- Medium #16: Add input validation

**Long-term Improvements:**
- Medium #10-14: Architecture standardization
- Medium #17: Testcontainers setup
- Low #18-24: Code quality improvements

---

## Completed

The following items have been successfully implemented and verified:

### Security & Reliability
- **Add Request Body Size Limits** - Implemented MaxBytesReader with appropriate limits (10MB for batch, 1KB for simple requests)
- **Validate Article Exists Before Recording Vote** - Added rowsAffected check and error handling
- **Make SSL Mode Configurable (Default: Require)** - DB_SSLMODE environment variable with "require" default

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

### Refactoring
- **Delete Unused Gin Middleware** - Removed from explore service (user service middleware is actively used)
- **Cache Internal User ID in Recommendation Flow** - Addressed by using external user ID from JWT token directly throughout the flow
