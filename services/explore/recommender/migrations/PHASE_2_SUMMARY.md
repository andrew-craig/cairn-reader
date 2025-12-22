# Phase 2 Implementation Summary: Article Deduplication

## Overview
Phase 2 of the RECOMMENDER_PLAN has been successfully implemented. This phase implements proper article deduplication using PostgreSQL's `ON CONFLICT` clause, ensuring that duplicate articles (based on link) are properly handled while preserving engagement metrics.

## Implementation Date
December 19, 2025

## Changes Made

### 1. Updated Article Model
**File**: [pkg/models/article.go](../../pkg/models/article.go)

Added new fields to support voting and recommendation tracking:
```go
type Article struct {
    // ... existing fields ...
    Upvotes     int       `json:"upvotes"`
    Downvotes   int       `json:"downvotes"`
    Recommends  int       `json:"recommends"`
    Deleted     bool      `json:"deleted"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### 2. Updated ArticleRepository.Create()
**File**: [recommender/internal/db/article_repository.go](../../recommender/internal/db/article_repository.go:23-62)

Implemented Phase 2 deduplication logic:
```sql
INSERT INTO articles (
    id, title, link, description, content, author, published,
    feed_url, feed_title, categories, feed_id, created_at, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
ON CONFLICT (link) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    content = EXCLUDED.content,
    author = EXCLUDED.author,
    published = EXCLUDED.published,
    updated_at = NOW()
WHERE articles.deleted = false
```

**Key Behaviors** (per RECOMMENDER_PLAN Phase 2):
- ✅ Duplicate detection based on article link (unique constraint from migration 003)
- ✅ Updates article content if feed re-publishes with changes
- ✅ Preserves vote counts (upvotes/downvotes) on updates
- ✅ Preserves recommends counter on updates
- ✅ Preserves deleted status (won't resurrect deleted articles)
- ✅ Updates the `updated_at` timestamp when article is modified
- ✅ Uses `ON CONFLICT (link)` instead of `ON CONFLICT (id)` for proper deduplication

### 3. Updated ArticleRepository.CreateBatch()
**File**: [recommender/internal/db/article_repository.go](../../recommender/internal/db/article_repository.go:64-122)

Applied the same deduplication logic to batch operations for consistency.

### 4. Updated All Repository Scan Methods
**Files**:
- [article_repository.go](../../recommender/internal/db/article_repository.go:123-165) - GetByID()
- [article_repository.go](../../recommender/internal/db/article_repository.go:167-184) - GetRecent()
- [article_repository.go](../../recommender/internal/db/article_repository.go:186-206) - GetUnreadForUser()
- [article_repository.go](../../recommender/internal/db/article_repository.go:208-248) - scanArticles()

All scan methods now include the new fields:
- upvotes, downvotes, recommends, deleted, created_at, updated_at

### 5. Comprehensive Test Coverage
**Files Created**:
- [article_repository_test.go](../../recommender/internal/db/article_repository_test.go) - Unit tests (requires test database)
- [article_repository_integration_test.go](../../recommender/internal/db/article_repository_integration_test.go) - Integration tests

## Test Results

### Integration Tests (All Passing ✅)
```bash
$ go test -v -tags=integration ./recommender/internal/db -run TestIntegration

=== RUN   TestIntegration_Create_NewArticle
--- PASS: TestIntegration_Create_NewArticle (0.03s)
=== RUN   TestIntegration_Create_DuplicateLink_UpdatesArticle
--- PASS: TestIntegration_Create_DuplicateLink_UpdatesArticle (0.12s)
=== RUN   TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated
--- PASS: TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated (0.14s)
=== RUN   TestIntegration_CreateBatch_WithDuplicates
--- PASS: TestIntegration_CreateBatch_WithDuplicates (0.02s)
PASS
ok      github.com/andrew-craig/cairn-explore/recommender/internal/db   0.769s
```

### Test Coverage

#### TestIntegration_Create_NewArticle
- ✅ Verifies new articles are inserted correctly
- ✅ Confirms default values (upvotes=0, downvotes=0, recommends=0, deleted=false)
- ✅ Validates all fields are stored properly

#### TestIntegration_Create_DuplicateLink_UpdatesArticle
- ✅ Tests core Phase 2 requirement: duplicate detection by link
- ✅ Verifies article content is updated (title, description, content, author, published)
- ✅ **Critical**: Confirms engagement metrics are preserved (upvotes, downvotes, recommends)
- ✅ Verifies updated_at timestamp is updated
- ✅ Confirms duplicate ID is not created (no second article)

#### TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated
- ✅ Tests Phase 2 requirement: don't update deleted articles
- ✅ Verifies deleted articles remain unchanged when duplicate link submitted
- ✅ Confirms deleted flag is preserved
- ✅ Validates updated_at timestamp doesn't change for deleted articles

#### TestIntegration_CreateBatch_WithDuplicates
- ✅ Tests batch operations with duplicate links
- ✅ Verifies deduplication works in batch context
- ✅ Confirms engagement metrics preserved during batch operations
- ✅ Validates new articles are created alongside updated ones

## Database Schema Dependencies

### Required Migration: 003_fetcher_schema_updates.sql
Phase 2 depends on migration 003 which added:
1. UNIQUE constraint on `articles.link` column (enables ON CONFLICT)
2. New columns: upvotes, downvotes, recommends, deleted
3. Index on deleted column for filtering

**Verification**:
```sql
-- Check unique constraint exists
SELECT conname, contype
FROM pg_constraint
WHERE conrelid = 'articles'::regclass AND contype = 'u';

-- Expected: articles_link_unique
```

## Deduplication Logic Flow

### New Article Submission
```
1. Fetcher sends article with link="https://example.com/article"
2. Repository executes INSERT with ON CONFLICT (link)
3. PostgreSQL checks if link exists

   Case A: Link doesn't exist
   - Article inserted as new
   - Default values: upvotes=0, downvotes=0, recommends=0, deleted=false
   - created_at and updated_at set to NOW()

   Case B: Link exists AND deleted=false
   - Article content updated (title, description, content, author, published)
   - Engagement metrics PRESERVED (upvotes, downvotes, recommends)
   - deleted flag PRESERVED
   - updated_at set to NOW()
   - created_at PRESERVED

   Case C: Link exists AND deleted=true
   - NO UPDATE occurs (WHERE clause prevents update)
   - Original article remains unchanged
   - No error returned to caller
```

### Example Scenario
```
Timeline:
1. Article submitted: link="https://blog.com/post", title="Draft Title"
   -> Created with ID=abc123, upvotes=0, downvotes=0, recommends=0

2. Users interact: upvotes=10, downvotes=2, recommends=15

3. Feed republishes with updated title="Final Title"
   -> ON CONFLICT triggers
   -> Title updated to "Final Title"
   -> upvotes=10, downvotes=2, recommends=15 PRESERVED ✅
   -> updated_at updated

4. Article soft-deleted: deleted=true

5. Feed republishes again
   -> ON CONFLICT triggers
   -> WHERE deleted=false prevents update
   -> Article remains deleted with original content ✅
```

## Success Criteria (All Met ✅)

Per RECOMMENDER_PLAN Phase 2:

- ✅ **Deduplication**: Articles deduplicated based on link (unique constraint)
- ✅ **Content Updates**: Article content updated when feed re-publishes
- ✅ **Preserve Votes**: Upvotes and downvotes preserved on updates
- ✅ **Preserve Recommends**: Recommends counter preserved on updates
- ✅ **Preserve Deleted**: Deleted status preserved (no resurrection)
- ✅ **Timestamp Updates**: updated_at timestamp updated on changes
- ✅ **No Deleted Updates**: Deleted articles are not updated
- ✅ **Batch Support**: Batch operations support same deduplication logic
- ✅ **Test Coverage**: Comprehensive integration tests passing

## Running the Tests

### Run Integration Tests
```bash
# Run all integration tests
go test -v -tags=integration ./recommender/internal/db

# Run specific test
go test -v -tags=integration ./recommender/internal/db -run TestIntegration_Create_DuplicateLink_UpdatesArticle
```

### Prerequisites
- Docker Compose running (`docker-compose up -d`)
- PostgreSQL accessible on localhost:5432
- Database migrations applied (001, 002, 003, 004)

## Architecture Notes

### Why ON CONFLICT (link) Instead of ON CONFLICT (id)?
The article ID is a SHA256 hash of the link, so theoretically duplicates would have the same ID. However:
- Using link is more explicit and clear in intent
- Migration 003 added UNIQUE constraint on link, not id
- Link is the natural deduplication key (source of truth)
- Handles edge cases where ID generation might differ

### Why Preserve Engagement Metrics?
Per RECOMMENDER_PLAN requirements:
- Users have voted on the article (upvotes/downvotes)
- Article has been recommended to users (recommends counter)
- Updating content shouldn't reset user engagement
- Quality score depends on these metrics: `(upvotes + (downvotes * 3)) / recommends`

### Why Not Update Deleted Articles?
- Deleted articles are intentionally removed from recommendations
- Resurrecting deleted articles would bypass content moderation
- Preserves admin/user deletion decisions
- Maintains data integrity and user trust

## Next Steps

According to [RECOMMENDER_PLAN.md](../../RECOMMENDER_PLAN.md), the implementation order continues with:

1. ✅ **Phase 1** - Database schema with users, votes, recommendations tables (COMPLETE)
2. ✅ **Phase 2** - Article deduplication (ON CONFLICT handling) (COMPLETE)
3. ⬜ **Phase 6** - Model updates (Vote, User, Recommendation models) - NEXT
4. ⬜ **Phase 3** - Voting API (upvote/downvote with user tracking)
5. ⬜ **Phase 4** - Enhanced recommendation algorithm (quality score + low-exposure)
6. ⬜ **Phase 8** - Article cleanup (90-day retention)
7. ⬜ **Phase 7** - Configuration updates
8. ⬜ **Phase 9** - Integration testing
9. ⬜ **Phase 10** - Admin endpoints (optional)

**Recommended Next Phase: Phase 6 - Model Updates**
1. Create `pkg/models/vote.go` - Vote model
2. Create `pkg/models/recommendation.go` - Recommendation model
3. Update `pkg/models/user.go` - Reflect new user structure (SERIAL id + TEXT user_id)
4. Create `recommender/internal/db/vote_repository.go` - Vote CRUD operations
5. Update `recommender/internal/db/user_repository.go` - Handle new user schema

## Files Modified

### Source Code
- ✅ [pkg/models/article.go](../../pkg/models/article.go) - Added voting/recommendation fields
- ✅ [recommender/internal/db/article_repository.go](../../recommender/internal/db/article_repository.go) - Deduplication logic

### Tests
- ✅ [recommender/internal/db/article_repository_test.go](../../recommender/internal/db/article_repository_test.go) - Unit tests (NEW)
- ✅ [recommender/internal/db/article_repository_integration_test.go](../../recommender/internal/db/article_repository_integration_test.go) - Integration tests (NEW)

### Documentation
- ✅ [recommender/migrations/PHASE_2_SUMMARY.md](PHASE_2_SUMMARY.md) - This document (NEW)

## Performance Considerations

### Index Usage
The deduplication relies on:
- UNIQUE constraint on `articles.link` (from migration 003)
- Index `idx_articles_deleted` for filtering deleted articles

### Query Performance
```sql
EXPLAIN ANALYZE INSERT INTO articles (...) VALUES (...)
ON CONFLICT (link) DO UPDATE SET ... WHERE deleted = false;

-- Uses Index: articles_link_unique (Unique Constraint)
-- Uses Index: idx_articles_deleted (WHERE clause)
```

### Batch Operations
- Batch inserts use prepared statements (efficient)
- Transaction wrapping ensures atomicity
- Each article checked independently for conflicts

## Known Limitations

1. **No Concurrent Update Protection**: If two processes submit the same link simultaneously, last write wins. This is acceptable for the RSS feed use case.

2. **No Conflict Notification**: Callers don't know if article was inserted or updated. This is intentional - the operation is idempotent.

3. **Test Database Required**: Integration tests require a running PostgreSQL database with migrations applied.

## Rollback Plan

If issues are discovered, rollback by:
1. Revert changes to `pkg/models/article.go`
2. Revert changes to `recommender/internal/db/article_repository.go`
3. Database schema changes (migration 003) should NOT be rolled back as they're required for Phase 1

No data migration needed - changes are backward compatible.
