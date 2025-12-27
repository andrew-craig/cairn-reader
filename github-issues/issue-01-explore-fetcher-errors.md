# [CRITICAL] Fix unchecked error returns in Explore Service - Fetcher

Labels: bug, priority:critical, service:explore, go

## Problem

The Explore Service Fetcher has **20 instances** of unchecked error returns that could lead to resource leaks, silent failures, and data inconsistencies.

## Impact

- **Resource Leaks**: Unchecked `Close()` calls on HTTP response bodies and database connections can exhaust connection pools
- **Silent Failures**: Database operations (`UpdateFetchResult`, `RecordFetchHistory`) fail silently, leading to incorrect state
- **Data Loss**: Errors are ignored in critical paths like feed fetching and article submission

## Affected Files

### Database Operations (5 instances)
- `fetcher/internal/db/feed_repository.go:133` - `stmt.Close()` not checked
- `fetcher/internal/db/feed_repository_test.go:14` - `db.Close()` not checked

### HTTP Operations (2 instances)
- `fetcher/internal/client/recommender_client.go:60` - `resp.Body.Close()` not checked
- `fetcher/internal/sync/feed_sync.go:74` - `resp.Body.Close()` not checked

### Business Logic (6 instances)
- `fetcher/internal/fetcher/fetcher.go:71` - `UpdateFetchResult()` error ignored
- `fetcher/internal/fetcher/fetcher.go:72` - `RecordFetchHistory()` error ignored
- `fetcher/internal/fetcher/fetcher.go:88` - `UpdateFetchResult()` error ignored
- `fetcher/internal/fetcher/fetcher.go:89` - `RecordFetchHistory()` error ignored
- `fetcher/internal/fetcher/fetcher.go:98` - `UpdateFetchResult()` error ignored
- `fetcher/internal/fetcher/fetcher.go:99` - `RecordFetchHistory()` error ignored

### Goroutines (2 instances)
- `fetcher/cmd/fetcher/main.go:71` - `FetchSingleFeed()` error ignored in goroutine
- `fetcher/cmd/fetcher/main.go:90` - `SyncOnce()` error ignored in goroutine

### Test Utilities (5 instances)
- `fetcher/internal/sync/feed_sync_test.go:16` - `database.Close()` not checked
- `fetcher/internal/sync/feed_sync_test.go:24` - `w.Write()` not checked
- `fetcher/internal/sync/feed_sync_test.go:59` - `database.Close()` not checked
- `fetcher/internal/sync/feed_sync_test.go:67` - `w.Write()` not checked
- `fetcher/internal/sync/feed_sync_test.go:108` - `w.Write()` not checked

## Recommended Fixes

### For defer Close() calls:
```go
// Before
defer resp.Body.Close()

// After
defer func() {
    if err := resp.Body.Close(); err != nil {
        log.Printf("error closing response body: %v", err)
    }
}()
```

### For database operations:
```go
// Before
f.feedRepo.UpdateFetchResult(ctx, feed.ID, false)

// After
if err := f.feedRepo.UpdateFetchResult(ctx, feed.ID, false); err != nil {
    log.Printf("error updating fetch result for feed %d: %v", feed.ID, err)
    // Consider: return error or implement retry logic
}
```

### For goroutines:
```go
// Before
go feedFetcher.FetchSingleFeed(ctx)

// After
go func() {
    if err := feedFetcher.FetchSingleFeed(ctx); err != nil {
        log.Printf("error in fetch goroutine: %v", err)
    }
}()
```

## Testing

After fixes, verify with:
```bash
cd services/explore
golangci-lint run ./fetcher/...
make test
```

## References

- Code Review Report: `CODE_REVIEW_REPORT.md` (Section: "Unchecked Error Returns")
- Tool: `golangci-lint` (errcheck linter)
- Estimated Effort: **3-4 hours**

## Acceptance Criteria

- [ ] All 20 instances of unchecked errors are properly handled
- [ ] Errors are logged with appropriate context
- [ ] Critical errors in business logic have proper error propagation or retry logic
- [ ] All tests pass
- [ ] `golangci-lint run ./fetcher/...` reports no errcheck issues
