# Phase 2 Implementation - Testing Guide

## What Was Implemented

Phase 2 of the Fetcher implementation has been completed:

### New Components
1. **Feed Model** ([pkg/models/feed.go](pkg/models/feed.go))
   - Feed struct with health tracking fields
   - FetchHistory struct for monitoring

2. **Database Layer** ([fetcher/internal/db/](fetcher/internal/db/))
   - `config.go`: Database connection configuration
   - `feed_repository.go`: Feed CRUD operations with methods:
     - `GetNextFeed()`: Returns next feed to fetch (prioritizes never-fetched, then oldest)
     - `UpdateFetchResult()`: Records success/failure and manages health tracking
     - `ImportFeeds()`: Bulk import from Kagi list
     - `ListFeeds()`: Query all feeds with filtering
     - `RecordFetchHistory()`: Log fetch attempts
     - `GetFeedStats()`: Statistics for monitoring

3. **Feed Sync Service** ([fetcher/internal/sync/feed_sync.go](fetcher/internal/sync/feed_sync.go))
   - Fetches Kagi Small Web feed list daily
   - Parses feed URLs (one per line, supports comments)
   - Imports new feeds without duplicating existing ones
   - Runs on startup and every 24 hours

4. **Database Schema** ([fetcher/migrations/001_init_schema.sql](fetcher/migrations/001_init_schema.sql))
   - `feeds` table with health tracking
   - `fetch_history` table for monitoring
   - Performance indexes

5. **Docker Infrastructure** ([docker-compose.yml](docker-compose.yml))
   - New `fetcher_db` PostgreSQL container (port 5433)
   - Separate volume for fetcher data
   - Environment variables for database connection

6. **Updated Main** ([fetcher/cmd/fetcher/main.go](fetcher/cmd/fetcher/main.go))
   - Database initialization
   - Feed sync background process
   - New HTTP endpoints for monitoring

## Testing Instructions

### 1. Start Services

```bash
# Start all services (requires Docker)
docker-compose up --build

# Or start in detached mode
docker-compose up -d --build

# View logs
docker-compose logs -f fetcher
docker-compose logs -f fetcher_db
```

### 2. Verify Database Creation

The fetcher_db should be created automatically with the schema applied:

```bash
# Connect to fetcher database
docker exec -it $(docker ps -qf "name=fetcher_db") psql -U fetcher -d fetcher_db

# Check tables exist
\dt

# Should show:
# - feeds
# - fetch_history

# Check feeds table structure
\d feeds

# Exit
\q
```

### 3. Test Feed Sync

The feed sync runs automatically on startup. Check the logs:

```bash
docker-compose logs -f fetcher | grep "feed sync"
```

You should see:
- "Starting feed sync from https://raw.githubusercontent.com/..."
- "Imported X new feeds (total in list: Y)"
- "Feed sync complete - Total: X, Enabled: X, Disabled: 0, Never fetched: X"

### 4. Manually Trigger Feed Sync

```bash
# Trigger sync
curl -X POST http://localhost:8080/feeds/sync

# Response:
# {"status":"sync triggered"}
```

### 5. Check Feed Statistics

```bash
curl http://localhost:8080/feeds/stats

# Response:
# {"total":500,"enabled":500,"disabled":0,"never_fetched":500}
```

### 6. Query Feeds Directly

```bash
# Connect to database
docker exec -it $(docker ps -qf "name=fetcher_db") psql -U fetcher -d fetcher_db

# Count total feeds
SELECT COUNT(*) FROM feeds;

# Show first 10 feeds
SELECT id, url, enabled, last_fetched_at, consecutive_failures
FROM feeds
LIMIT 10;

# Show feeds that have been fetched
SELECT id, url, last_fetched_at, consecutive_failures
FROM feeds
WHERE last_fetched_at IS NOT NULL;

# Show disabled feeds
SELECT id, url, consecutive_failures
FROM feeds
WHERE enabled = false;

# Show next feed to fetch (same query used by GetNextFeed)
SELECT id, url, last_fetched_at
FROM feeds
WHERE enabled = true
ORDER BY last_fetched_at NULLS FIRST, last_fetched_at ASC
LIMIT 1;
```

### 7. Test Repository Methods (via Go tests)

Once Go is available:

```bash
# Run all tests
go test ./fetcher/internal/db/...

# Run with verbose output
go test -v ./fetcher/internal/db/...
```

### 8. Monitor Fetch History

After the fetcher starts working (Phase 3), you can monitor fetch history:

```bash
# Connect to database
docker exec -it $(docker ps -qf "name=fetcher_db") psql -U fetcher -d fetcher_db

# View recent fetch attempts
SELECT
    fh.id,
    f.url,
    fh.success,
    fh.articles_found,
    fh.articles_sent,
    fh.error_message,
    fh.fetch_started_at
FROM fetch_history fh
JOIN feeds f ON fh.feed_id = f.id
ORDER BY fh.created_at DESC
LIMIT 20;

# View success rate by feed
SELECT
    f.url,
    COUNT(*) as total_fetches,
    SUM(CASE WHEN fh.success THEN 1 ELSE 0 END) as successful,
    SUM(CASE WHEN NOT fh.success THEN 1 ELSE 0 END) as failed
FROM fetch_history fh
JOIN feeds f ON fh.feed_id = f.id
GROUP BY f.url
ORDER BY failed DESC
LIMIT 10;
```

## Expected Behavior

### On Startup
1. Fetcher connects to fetcher_db
2. Feed sync runs immediately
3. Kagi feed list is fetched (500+ feeds)
4. New feeds are imported to database
5. Stats are logged

### Every 24 Hours
1. Feed sync runs again
2. New feeds from Kagi list are added
3. Existing feeds are preserved (no duplicates)
4. Stats are logged

### Current Limitations (Phase 2)
- Fetcher still uses hardcoded feeds for actual fetching (Phase 3 will fix this)
- Feed health tracking is in place but not yet utilized
- Fetch history table exists but won't have data until Phase 3

## Troubleshooting

### Database Connection Failed
```bash
# Check fetcher_db is running
docker ps | grep fetcher_db

# Check fetcher_db health
docker inspect $(docker ps -qf "name=fetcher_db") | grep Health -A 10

# View database logs
docker-compose logs fetcher_db
```

### Feed Sync Failed
```bash
# Check fetcher logs
docker-compose logs -f fetcher | grep sync

# Common issues:
# - Network timeout (check KAGI_FEED_URL is accessible)
# - Database connection (check fetcher_db is healthy)
# - Parsing error (check Kagi feed format hasn't changed)
```

### No Feeds in Database
```bash
# Manually trigger sync
curl -X POST http://localhost:8080/feeds/sync

# Check if URL is accessible
curl https://raw.githubusercontent.com/kagisearch/smallweb/main/smallweb.txt | head -20

# Check fetcher logs for errors
docker-compose logs fetcher | grep -i error
```

## Verification Checklist

- [ ] fetcher_db container starts successfully
- [ ] Database schema is created (feeds and fetch_history tables)
- [ ] Feed sync runs on startup
- [ ] Feeds are imported from Kagi list
- [ ] `/feeds/stats` endpoint returns valid JSON
- [ ] `/feeds/sync` endpoint triggers sync
- [ ] Database contains 500+ feeds
- [ ] All feeds have `enabled = true` initially
- [ ] All feeds have `last_fetched_at = NULL` initially
- [ ] All feeds have `consecutive_failures = 0` initially

## Next Steps (Phase 3)

✅ **Phase 3 has been completed!** All planned functionality is now implemented:
1. ✅ Updated fetcher to use database instead of hardcoded feeds
2. ✅ Implemented one-feed-per-minute fetching strategy
3. ✅ Utilizing GetNextFeed() to prioritize feeds
4. ✅ Recording fetch results using UpdateFetchResult()
5. ✅ Logging all fetches to fetch_history table
6. ✅ Auto-disabling feeds after 10 consecutive failures
7. ✅ Comprehensive test suite (39 tests)

All fetcher implementation is complete. See [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) for remaining work on the Recommender service.
