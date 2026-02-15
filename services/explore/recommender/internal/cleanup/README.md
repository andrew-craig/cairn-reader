# Article Cleanup

This package implements automatic cleanup of old articles to maintain database health and performance.

## Overview

The article cleanup system provides two cleanup strategies:

1. **Soft Delete**: Marks articles older than N days as `deleted = true`
2. **Hard Delete**: Permanently removes articles that have been soft-deleted for an extended period

This two-phase approach maintains referential integrity while gradually removing old content.

## Automatic Cleanup

The recommender service automatically runs cleanup when started:

- **Frequency**: Once every 24 hours
- **Retention Period**: 90 days (configurable via `ARTICLE_RETENTION_DAYS` environment variable)
- **Soft Delete**: Articles older than retention period are marked as deleted
- **Hard Delete**: Articles soft-deleted for 30+ days beyond retention period are permanently removed

### Configuration

Set the retention period in `docker-compose.yml` or your environment:

```yaml
environment:
  - ARTICLE_RETENTION_DAYS=90  # Delete articles after 90 days
```

### Logs

The cleanup job logs its activity:

```
Starting article cleanup job (retention: 90 days, interval: 24h0m0s)
Running article cleanup (marking articles older than 90 days as deleted)...
Marked 42 articles as deleted
Hard deleted 15 articles (older than 120 days and already marked as deleted)
Article cleanup completed
```

## Manual Cleanup

For one-time cleanup or testing, use the standalone cleanup utility.

### Building

```bash
# Via Makefile
make build

# Or directly
go build -o bin/cleanup ./recommender/cmd/cleanup
```

### Running

```bash
# Use default retention period (90 days)
./bin/cleanup

# Or via Makefile
make run-cleanup

# Specify custom retention period
./bin/cleanup 60  # Delete articles older than 60 days

# Using environment variables
ARTICLE_RETENTION_DAYS=30 ./bin/cleanup
```

### Docker

To run cleanup in Docker:

```bash
# Connect to running recommender container
docker compose exec recommender /root/recommender --cleanup

# Or run the standalone utility
docker compose run --rm recommender sh -c "go run ./recommender/cmd/cleanup/main.go"
```

## Implementation Details

### Soft Delete Process

1. Finds articles where `created_at < NOW() - INTERVAL 'N days'`
2. Sets `deleted = true` and updates `updated_at`
3. Only affects articles not already marked as deleted
4. Deleted articles are excluded from recommendations

### Hard Delete Process

1. Finds articles where `created_at < NOW() - INTERVAL '(N+30) days'`
2. Only deletes articles already marked as `deleted = true`
3. Permanently removes articles and cascades to related tables
4. Frees up database space

### Why Two-Phase Deletion?

- **Referential Integrity**: Soft delete preserves vote and recommendation history
- **Gradual Cleanup**: Allows time to verify deleted articles before permanent removal
- **Data Recovery**: Soft-deleted articles can be restored if needed
- **Performance**: Hard delete runs on smaller dataset (only soft-deleted articles)

## Repository Methods

The cleanup functionality is implemented in [`article_repository.go`](../db/article_repository.go):

```go
// MarkOldArticlesAsDeleted sets deleted=true for articles older than N days
func (r *ArticleRepository) MarkOldArticlesAsDeleted(ctx context.Context, days int) (int, error)

// HardDeleteOldArticles removes articles older than N days (optional, for maintenance)
func (r *ArticleRepository) HardDeleteOldArticles(ctx context.Context, days int) (int, error)
```

## Testing

To test the cleanup functionality:

1. Insert test articles with old `created_at` timestamps
2. Run cleanup utility or wait for automatic cleanup
3. Verify articles are marked as deleted
4. Check logs for cleanup counts

```sql
-- Create test article with old timestamp
INSERT INTO articles (id, title, link, created_at)
VALUES ('test123', 'Old Article', 'https://example.com/old', NOW() - INTERVAL '100 days');

-- Check deleted status
SELECT id, title, deleted, created_at FROM articles WHERE id = 'test123';
```

## Monitoring

Monitor cleanup effectiveness through:

- **Service logs**: Check cleanup job output
- **Database queries**: Count deleted vs active articles
- **Admin API** (future): `/admin/stats` endpoint will show deleted article counts

```sql
-- Check article counts
SELECT
  COUNT(*) as total,
  COUNT(*) FILTER (WHERE deleted = false) as active,
  COUNT(*) FILTER (WHERE deleted = true) as deleted
FROM articles;
```
