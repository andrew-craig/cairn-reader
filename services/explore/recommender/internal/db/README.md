# Article Repository Tests

This directory contains comprehensive tests for the article repository, specifically testing the Phase 2 deduplication logic.

## Test Files

### article_repository_test.go
Unit tests that require a dedicated test database (`cairn_test`). These tests are currently **skipped** when the test database is not available.

**Tests included:**
- TestCreate_NewArticle - Verify new article insertion
- TestCreate_DuplicateLink_UpdatesArticle - Test deduplication
- TestCreate_DuplicateLink_DeletedArticle_NotUpdated - Test deleted article protection
- TestCreateBatch_NewArticles - Test batch insertion
- TestCreateBatch_WithDuplicates - Test batch deduplication
- TestCreateBatch_EmptySlice - Test edge case
- TestGetRecent - Test retrieval
- TestGetByID_NotFound - Test error handling

### article_repository_integration_test.go
Integration tests that use the existing development database. These tests run with the `-tags=integration` flag.

**Tests included:**
- TestIntegration_Create_NewArticle
- TestIntegration_Create_DuplicateLink_UpdatesArticle
- TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated
- TestIntegration_CreateBatch_WithDuplicates

## Running the Tests

### Quick Run (Integration Tests Only)
```bash
# Run from project root
go test -v -tags=integration ./recommender/internal/db

# Run specific test
go test -v -tags=integration ./recommender/internal/db -run TestIntegration_Create_DuplicateLink
```

### Prerequisites
1. Docker Compose running:
   ```bash
   docker compose up -d
   ```

2. Database migrations applied (migrations 001-004):
   ```bash
   # Check migration status
   docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -c "\dt"
   ```

3. Environment variables (optional, defaults work with docker-compose):
   - `DB_HOST=localhost`
   - `DB_PORT=5432`
   - `DB_USER=cairn`
   - `DB_PASSWORD=cairn_password`
   - `DB_NAME=cairn_db`

### Unit Tests (Requires Test Database)
To run the unit tests, you need a separate test database:

```bash
# Create test database
docker exec -i cairn-explore-postgres-1 psql -U cairn -d postgres -c "CREATE DATABASE cairn_test;"

# Apply migrations to test database
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_test < recommender/migrations/001_init.sql
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_test < recommender/migrations/002_add_feed_id_to_articles.sql
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_test < recommender/migrations/003_fetcher_schema_updates.sql
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_test < recommender/migrations/004_voting_and_recommendations.sql

# Run unit tests
go test -v ./recommender/internal/db
```

## Test Coverage

### Phase 2 Requirements Coverage

All Phase 2 requirements from RECOMMENDER_PLAN.md are tested:

#### ✅ Deduplication Based on Link
- **Test**: TestIntegration_Create_DuplicateLink_UpdatesArticle
- **Validates**: Articles with same link are deduplicated, not duplicated

#### ✅ Content Updates on Republish
- **Test**: TestIntegration_Create_DuplicateLink_UpdatesArticle
- **Validates**: Title, description, content, author, published are updated

#### ✅ Preserve Vote Counts
- **Test**: TestIntegration_Create_DuplicateLink_UpdatesArticle
- **Validates**: upvotes and downvotes remain unchanged during update

#### ✅ Preserve Recommends Counter
- **Test**: TestIntegration_Create_DuplicateLink_UpdatesArticle
- **Validates**: recommends counter remains unchanged during update

#### ✅ Preserve Deleted Status
- **Test**: TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated
- **Validates**: Deleted articles are NOT updated (no resurrection)

#### ✅ Update Timestamp
- **Test**: TestIntegration_Create_DuplicateLink_UpdatesArticle
- **Validates**: updated_at timestamp is updated when article is modified

#### ✅ Batch Operations
- **Test**: TestIntegration_CreateBatch_WithDuplicates
- **Validates**: Batch operations handle deduplication correctly

## Test Data

All tests use the `test-*` prefix for IDs to avoid conflicts with production data. The integration tests clean up after themselves by deleting all records with IDs starting with `test-`.

### Test Article Structure
```go
models.Article{
    ID:          "test-new-article-1",
    Title:       "Test Article 1",
    Link:        "https://example.com/test/article1",
    Description: "Test description",
    Content:     "Test content for integration testing",
    Author:      "Test Author",
    Published:   time.Now(),
    FeedURL:     "https://example.com/feed",
    FeedTitle:   "Test Feed",
    Categories:  []string{"tech", "programming"},
}
```

## Debugging Tests

### View Test Data
```bash
# View all test articles
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -c "SELECT id, title, link, upvotes, downvotes, recommends, deleted FROM articles WHERE id LIKE 'test-%';"

# Clean up test data manually
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -c "DELETE FROM articles WHERE id LIKE 'test-%';"
```

### Enable SQL Logging
Add to test setup:
```go
db.SetLogger(log.New(os.Stdout, "SQL: ", log.LstdFlags))
```

### Check Unique Constraint
```bash
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -c "SELECT conname, contype FROM pg_constraint WHERE conrelid = 'articles'::regclass AND contype = 'u';"
```

## Common Issues

### Issue: Tests Skip Due to Missing Database
**Symptom**: "Skipping test: could not ping test database"
**Solution**:
1. Ensure Docker Compose is running
2. Check database connectivity: `psql -h localhost -U cairn -d cairn_db`
3. For unit tests, create `cairn_test` database

### Issue: Foreign Key Violations
**Symptom**: "violates foreign key constraint"
**Solution**: Ensure migrations 001-004 are applied in order

### Issue: Unique Constraint Missing
**Symptom**: Tests fail with unexpected duplicate articles
**Solution**: Verify migration 003 was applied (adds UNIQUE constraint on link)
```bash
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -c "\d articles"
```

## Performance Notes

### Test Execution Times (Approximate)
- TestIntegration_Create_NewArticle: ~30ms
- TestIntegration_Create_DuplicateLink_UpdatesArticle: ~120ms
- TestIntegration_Create_DuplicateLink_DeletedArticle_NotUpdated: ~140ms
- TestIntegration_CreateBatch_WithDuplicates: ~20ms

**Total**: ~310ms for all integration tests

### Database Connections
Each test creates a new database connection and cleans up after itself. For faster testing, consider connection pooling or test database setup/teardown optimization.

## Related Documentation

- [Phase 2 Implementation Summary](../../migrations/PHASE_2_SUMMARY.md)
- [RECOMMENDER_PLAN.md](../../../RECOMMENDER_PLAN.md)
- [Database Migrations README](../../migrations/README.md)
