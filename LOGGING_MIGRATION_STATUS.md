# Logging Migration Status Report

**Date:** 2026-01-05
**Document:** Review of logging strategy migration from `docs/LOGGING_STRATEGY.md`

## Executive Summary

The migration to `log/slog` structured logging is **PARTIALLY COMPLETE** (~70% done).

- ✅ **Phase 1 (Complete)**: Shared `pkg/logging` package created with all recommended components
- 🟡 **Phase 2 (Partial)**: Service main.go files migrated, but internal components still use old patterns
- ❌ **Phase 3 (Not Started)**: Old logging imports and patterns not yet removed

## Detailed Findings

### Phase 1: Create Shared Package ✅ COMPLETE

The `pkg/logging` package exists with the exact structure specified in the migration document:

```
pkg/logging/
├── logger.go          ✅ Logger initialization and configuration
├── middleware.go      ✅ Gin HTTP middleware for request logging
├── chi_middleware.go  ✅ Chi HTTP middleware (bonus, not in spec)
└── context.go         ✅ Context helpers for request-scoped logging
```

All components match the specification from `docs/LOGGING_STRATEGY.md`.

### Phase 2: Service-by-Service Migration 🟡 PARTIAL

#### Migration Progress by Service

| Service | Main.go | Handlers | Repositories | Internal Components | Status |
|---------|---------|----------|--------------|---------------------|--------|
| **User Service** | ✅ | ✅ | ✅ | 🟡 (old middleware file remains) | **95%** |
| **Explore Recommender** | ✅ | ✅ | ✅ | 🟡 (cleanup utility unmigrated) | **90%** |
| **Explore Fetcher** | ✅ | ✅ | ✅ | 🟡 (client uses old logging) | **90%** |
| **Read Content Service** | ✅ | ✅ | ✅ | 🟡 (middleware unmigrated) | **85%** |
| **Read Ingest RSS** | ✅ | ✅ | 🟡 | ❌ (most internals unmigrated) | **40%** |

#### Specific Findings

**User Service** (3 files with old logging):
- ✅ `cmd/user-service/main.go` - Using slog
- ✅ `internal/handlers/router.go` - Using `pkg/logging.RequestLogger`
- ❌ `internal/middleware/logging.go` - **OLD FILE** still uses `log.Printf`, `AUTH_SUCCESS`, etc.
- ❌ `cmd/migrate/main.go` - Uses old logging
- ❌ `pkg/auth/examples/` - Example files use old logging (low priority)

**Explore Service** (3 files with old logging):
- ✅ Both main.go files migrated to slog
- ✅ Handlers and repositories using slog
- ❌ `recommender/cmd/explore_cleanup/main.go` - Uses `log.Printf`
- ❌ `recommender/internal/cleanup/article_cleanup.go` - Uses `log.Printf`
- ❌ `fetcher/internal/client/recommender_client.go` - Uses `log.Printf`

**Read Service** (18 files with old logging):
- ✅ Both main.go files migrated to slog
- ✅ Main handlers migrated to slog
- ❌ **Ingest RSS service internals** heavily unmigrated:
  - `internal/fetcher/feed_fetcher.go` - Uses `log.Printf` (lines 99, 123, 141, 144)
  - `internal/processor/item_processor.go`
  - `internal/processor/update_detector.go`
  - `internal/scheduler/poll_scheduler.go`
  - `internal/scheduler/tier_manager.go`
  - `internal/worker/feed_worker.go`
  - `internal/worker/outbox_worker.go`
  - `internal/jobs/*.go` (5 files)
  - `internal/api/middleware/recovery.go`
  - `internal/api/handlers/subscription_handler.go`
- ❌ Content service middleware: `internal/api/middleware/error_handler.go`

### Phase 3: Cleanup ❌ NOT STARTED

Old logging patterns still present:
- **214 occurrences** of `log.Printf`, `log.Println`, `fmt.Printf` across **32 files**
- **191 total Go files** in services directory
- ~**17% of files** still contain old logging patterns

Old logging prefixes still in use:
- ❌ `AUTH_SUCCESS`, `AUTH_FAILURE` (in `services/users/internal/middleware/logging.go`)
- ❌ Various `log.Printf` patterns throughout internal packages

## Migration Impact

### What's Working Well

1. **All service entry points** use structured logging with proper initialization
2. **HTTP middleware** uses slog for request/response logging
3. **Main handlers and repositories** in core services migrated
4. **Consistent service naming** in logs (user-service, recommender, fetcher, etc.)

### What Needs Work

1. **Orphaned middleware file**: `services/users/internal/middleware/logging.go` should be deleted
2. **Read service internals**: Ingest RSS service has extensive old logging in workers, jobs, and processors
3. **Cleanup utilities**: Background jobs and cleanup scripts not migrated
4. **Example files**: Low-priority but present in pkg/auth/examples/

## Remaining Work

### High Priority

1. **Migrate Read/Ingest RSS internal components** (18 files):
   - `internal/fetcher/feed_fetcher.go` - Replace `log.Printf` with `slog.Info/Error`
   - `internal/processor/*.go` - Migrate item processor and update detector
   - `internal/scheduler/*.go` - Migrate poll scheduler and tier manager
   - `internal/worker/*.go` - Migrate feed and outbox workers
   - `internal/jobs/*.go` - Migrate all 5 job files

2. **Delete obsolete middleware**:
   - Remove `services/users/internal/middleware/logging.go` (replaced by `pkg/logging`)
   - Verify no references to old middleware functions

3. **Migrate Explore service utilities**:
   - `services/explore/recommender/cmd/explore_cleanup/main.go`
   - `services/explore/recommender/internal/cleanup/article_cleanup.go`
   - `services/explore/fetcher/internal/client/recommender_client.go`

### Medium Priority

4. **Migrate remaining middleware**:
   - `services/read/content/internal/api/middleware/error_handler.go`
   - `services/read/fetcher/internal/api/middleware/recovery.go`

5. **Update migration utilities**:
   - `services/users/cmd/migrate/main.go`

### Low Priority

6. **Example files** (can remain as-is or update later):
   - `pkg/auth/examples/basic/main.go`
   - `pkg/auth/examples/explore-service/main.go`

## Recommendations

### Immediate Actions

1. **Complete Read/Ingest RSS migration** - This service has the most remaining work
2. **Delete obsolete files** - Remove old middleware that's been replaced
3. **Run tests** - Ensure no functionality broken during migration

### Pattern to Follow

For each file with old logging:

```go
// Before
log.Printf("Fetching feed %s (%s)", feed.ID, feed.FeedURL)
log.Printf("Error processing feed item %s: %v", item.GUID, err)

// After
slog.Info("fetching feed",
    slog.String("feed_id", feed.ID),
    slog.String("feed_url", feed.FeedURL),
)
slog.Error("failed to process feed item",
    slog.String("feed_item_guid", item.GUID),
    slog.Any("error", err),
)
```

### Validation Checklist

After migration completion:

- [ ] Run `grep -r "log\.Printf\|log\.Println" services --include="*.go" | wc -l` should return 0 (or minimal)
- [ ] All services use `pkg/logging.NewLogger()` in main.go
- [ ] All HTTP requests logged with structured attributes (request_id, method, path, status, duration)
- [ ] No `AUTH_SUCCESS`, `[DB]`, `[Recommendations]` prefixes in logs
- [ ] Production logs output JSON format
- [ ] Development logs output text format

## Conclusion

The migration foundation is solid with the `pkg/logging` package fully implemented and all service entry points migrated. However, **significant work remains** in migrating internal components, particularly in the Read/Ingest RSS service.

**Estimated remaining effort**: 4-6 hours to complete migration of all remaining files and cleanup.

**Risk**: Low - The migration is backwards compatible and can be completed incrementally without breaking existing functionality.
