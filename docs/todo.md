# Project-Wide TODO - Required Fixes

Comprehensive code review findings from December 2025. Issues are organized by priority and span all services (explore, users, mobile app).

---

## Critical Priority

---

## High Priority

### 7. Remove Hardcoded Secrets from Docker Compose
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
