# [CRITICAL] Fix unchecked error returns in Explore Service - Recommender

Labels: bug, priority:critical, service:explore, go

## Problem

The Explore Service Recommender has **12 instances** of unchecked error returns that could lead to resource leaks, silent failures, and malformed HTTP responses.

## Impact

- **Resource Leaks**: Unchecked `Close()` and `Rollback()` calls can exhaust database connection pool
- **Silent Failures**: HTTP response encoding errors could send malformed responses without error notification
- **Data Corruption**: Transaction rollback errors are ignored, potentially leaving database in inconsistent state

## Affected Files

### Database Operations (6 instances)
- `recommender/internal/db/article_repository.go:82` - `tx.Rollback()` not checked (defer)
- `recommender/internal/db/article_repository.go:102` - `stmt.Close()` not checked
- `recommender/internal/db/article_repository.go:188` - `rows.Close()` not checked
- `recommender/internal/db/article_repository.go:220` - `rows.Close()` not checked
- `recommender/internal/db/article_repository.go:256` - `rows.Close()` not checked
- `recommender/internal/db/vote_repository.go:43` - `tx.Rollback()` not checked (defer)
- `recommender/internal/db/vote_repository.go:145` - `tx.Rollback()` not checked (defer)

### HTTP API Handlers (3 instances)
- `recommender/internal/api/handlers.go:17` - `json.Encode()` error not checked (health endpoint)
- `recommender/internal/api/handlers.go:41` - `json.Encode()` error not checked (articles endpoint)
- `recommender/internal/api/handlers.go:56` - `json.Encode()` error not checked (articles endpoint)

### Test Utilities (3 instances)
- `recommender/cmd/cleanup/main.go:47` - `database.Close()` not checked
- `recommender/integration_test.go:91` - `suite.database.Close()` not checked
- `recommender/integration_test.go:144` - `resp.Body.Close()` not checked
- `recommender/internal/db/article_repository_test.go:27` - `db.Close()` not checked
- `recommender/internal/db/article_repository_test.go:58` - `db.Close()` not checked

## Recommended Fixes

### For defer Rollback() calls:
```go
// Before
defer tx.Rollback()

// After
defer func() {
    if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
        log.Printf("error rolling back transaction: %v", err)
    }
}()
```

Note: Check for `sql.ErrTxDone` because Rollback() will error if transaction was already committed.

### For JSON encoding in HTTP handlers:
```go
// Before
json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})

// After
if err := json.NewEncoder(w).Encode(map[string]string{"status": "healthy"}); err != nil {
    log.Printf("error encoding response: %v", err)
    http.Error(w, "Internal server error", http.StatusInternalServerError)
    return
}
```

### For rows/stmt Close():
```go
// Before
defer rows.Close()

// After
defer func() {
    if err := rows.Close(); err != nil {
        log.Printf("error closing rows: %v", err)
    }
}()
```

## Testing

After fixes, verify with:
```bash
cd services/explore
golangci-lint run ./recommender/...
make test
curl http://localhost:8081/health  # Test health endpoint
```

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Section: "Unchecked Error Returns")
- Tool: `golangci-lint` (errcheck linter)
- Estimated Effort: **2-3 hours**

## Acceptance Criteria

- [ ] All 12 instances of unchecked errors are properly handled
- [ ] JSON encoding errors in HTTP handlers return proper error responses
- [ ] Transaction rollback errors are logged (excluding sql.ErrTxDone)
- [ ] Database resource cleanup errors are logged
- [ ] All tests pass
- [ ] `golangci-lint run ./recommender/...` reports no errcheck issues
