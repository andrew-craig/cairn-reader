# Explore Services - Required Fixes

Code review findings from December 2025. Issues are organized by priority.

---

## High Priority

### 1. Add Request Body Size Limits
**Files:** `recommender/internal/api/handlers.go`

Add `http.MaxBytesReader` to prevent denial-of-service attacks via large payloads.

**Recommended limits:**
| Endpoint | Limit | Rationale |
|----------|-------|-----------|
| `POST /explore/articles` | 10MB | Batch article submission with content |
| `POST /explore/articles/read` | 1KB | Simple JSON with article_id |
| `POST /explore/articles/:id/vote` | 1KB | Simple JSON with vote_type |

**Implementation:**
```go
// At the start of each handler:
r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
    // Check for MaxBytesError and log appropriately
    var maxBytesErr *http.MaxBytesError
    if errors.As(err, &maxBytesErr) {
        log.Printf("Request body too large: limit=%d", maxBytesErr.Limit)
        http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
        return
    }
    http.Error(w, "Invalid request body", http.StatusBadRequest)
    return
}
```

---

### 2. Validate Article Exists Before Recording Vote
**File:** `recommender/internal/db/vote_repository.go`

The `RecordVote` function updates article counters without verifying the article exists. If the article doesn't exist, the UPDATE silently affects 0 rows.

**Fix:** Check `RowsAffected()` after the UPDATE and return an error if 0:
```go
result, err := tx.ExecContext(ctx, updateQuery, articleID)
if err != nil {
    return fmt.Errorf("failed to update article vote counts: %w", err)
}
rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
    return fmt.Errorf("article not found: %s", articleID)
}
```

---

### 3. Delete Unused Gin Middleware
**File:** `pkg/logging/middleware.go`

This file imports and uses the Gin web framework, but both services use stdlib `net/http`. This is dead code.

**Action:** Delete the entire file. The recommender already has its own logging middleware in `internal/api/middleware.go`.

---

## Medium Priority

### 4. Replace O(n²) Sorting with Standard Library
**File:** `recommender/internal/recommend/engine.go:141-147`

The `selectHighQualityArticles` function uses bubble sort. Replace with `sort.Slice`:

```go
import "sort"

// Replace lines 141-147 with:
sort.Slice(scored, func(i, j int) bool {
    return scored[i].score > scored[j].score
})
```

---

### 5. Add Connection Pool Configuration to Fetcher
**File:** `fetcher/internal/db/config.go`

The fetcher doesn't configure connection pool settings. Add to `Connect()` method:

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

---

### 6. Standardize Logging to slog
**Files:** Multiple files use `log.Printf` instead of structured `slog`

Files to update:
- `recommender/internal/api/handlers.go` - uses `log.Printf`
- `recommender/internal/db/article_repository.go` - uses `log.Printf`
- `recommender/internal/db/vote_repository.go` - uses `log.Printf`
- `recommender/internal/recommend/engine.go` - uses `log.Printf`
- `fetcher/internal/fetcher/fetcher.go` - uses `log.Printf`
- `fetcher/internal/sync/feed_sync.go` - uses `log.Printf`

**Pattern to follow:**
```go
// Before
log.Printf("Error storing articles: %v", err)

// After
slog.Error("failed to store articles", slog.Any("error", err))
```

---

### 7. Make SSL Mode Configurable (Default: Require)
**Files:**
- `fetcher/internal/db/config.go:35`
- `recommender/cmd/recommender/main.go:149`

Add environment variable `DB_SSLMODE` with default `require`:

```go
sslMode := getEnv("DB_SSLMODE", "require")
connStr := fmt.Sprintf(
    "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
    c.Host, c.Port, c.User, c.Password, c.DBName, sslMode,
)
```

Update docker-compose files to explicitly set `DB_SSLMODE=disable` for local development.

---

### 8. Validate User IDs as UUIDs
**File:** `recommender/internal/db/user_repository.go`

Add UUID validation in `CreateOrGetUser`:

```go
import "github.com/google/uuid"

func (r *UserRepository) CreateOrGetUser(ctx context.Context, externalUserID string) (int, error) {
    // Validate UUID format
    if _, err := uuid.Parse(externalUserID); err != nil {
        return 0, fmt.Errorf("invalid user ID format (must be UUID): %w", err)
    }
    // ... rest of function
}
```

---

### 9. Extract URL Path Parsing Helper
**File:** `recommender/internal/api/handlers.go`

The manual string manipulation for path parameters is fragile. Extract a helper:

```go
// In handlers.go or a new file
func extractPathParam(path, prefix, suffix string) string {
    path = strings.TrimPrefix(path, prefix)
    path = strings.TrimSuffix(path, suffix)
    return strings.TrimSpace(path)
}

// Usage
articleID := extractPathParam(r.URL.Path, "/explore/articles/", "/vote")
```

---

## Low Priority

### 10. Implement Kubernetes-Style Health Endpoints
**File:** `recommender/internal/api/handlers.go`

Follow Kubernetes best practices with separate liveness and readiness probes:

- `/health` (liveness) - Returns 200 if process is running (current behavior is fine)
- `/ready` (readiness) - Returns 200 only if dependencies (DB, Vault) are healthy

**Add readiness handler:**
```go
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
    // Check database connectivity
    if err := s.db.PingContext(r.Context()); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "not ready",
            "error":  "database unavailable",
        })
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "status":  "ready",
        "service": "recommender",
    })
}
```

Register in `server.go`:
```go
mux.HandleFunc("/ready", s.handleReady)
```

---

### 11. Add Request ID Propagation
**File:** `recommender/internal/api/middleware.go`

Enhance logging middleware to generate/propagate X-Request-ID for distributed tracing:

```go
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        requestID := r.Header.Get("X-Request-ID")
        if requestID == "" {
            requestID = uuid.New().String()
        }
        w.Header().Set("X-Request-ID", requestID)

        // Add to context for downstream logging
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        r = r.WithContext(ctx)

        // ... rest of middleware
    })
}
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

### 13. Log Warning for Silent Vote Counter Failures
**File:** `recommender/internal/db/vote_repository.go:73-75, 79-81`

The `WHERE upvotes > 0` / `WHERE downvotes > 0` conditions prevent negative counts but fail silently. Add logging:

```go
result, err := tx.ExecContext(ctx, updateQuery, articleID)
if err != nil {
    return fmt.Errorf("failed to update article vote counts: %w", err)
}
rowsAffected, _ := result.RowsAffected()
if rowsAffected == 0 {
    log.Printf("Warning: vote counter update had no effect (article=%s, possibly already at 0)", articleID)
}
```

---

### 14. Cache Internal User ID in Recommendation Flow
**File:** `recommender/internal/recommend/engine.go`

The recommendation flow calls `CreateOrGetUser` multiple times for the same user (once per repository method). Consider:
- Passing internal user ID through the call chain
- Or caching at the repository level with a short TTL

---

## Notes

- All changes should include appropriate tests
- Run `make test` and `make lint` after each fix
- Update docker-compose.yml to set `DB_SSLMODE=disable` when SSL mode default changes
