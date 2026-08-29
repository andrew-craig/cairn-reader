# Troubleshooting Guide

This guide covers common issues, their causes, and solutions for Cairn Backend services.

## Table of Contents

- [Service Won't Start](#service-wont-start)
- [Database Issues](#database-issues)
- [Content Processing Issues](#content-processing-issues)
- [Feed Polling Issues](#feed-polling-issues)
- [Performance Issues](#performance-issues)
- [API Issues](#api-issues)
- [Worker Issues](#worker-issues)
- [Docker Issues](#docker-issues)
- [Migration Issues](#migration-issues)
- [Debugging Tips](#debugging-tips)

## Service Won't Start

### Issue: Content Service fails to start

**Symptoms**:
- Container exits immediately
- "connection refused" errors
- Health check failing

**Diagnosis**:
```bash
# Check container logs
docker compose logs content-service

# Check if database is ready
docker compose logs postgres-content

# Check port conflicts
netstat -an | grep 8080
lsof -i :8080
```

**Common Causes & Solutions**:

1. **Database not ready**
   ```bash
   # Wait for database health check
   docker compose ps postgres-content

   # Should show "healthy" status
   # If unhealthy, check database logs
   docker compose logs postgres-content
   ```

2. **Port already in use**
   ```bash
   # Find process using port 8080
   sudo lsof -i :8080

   # Kill the process or change port in docker-compose.yml
   ports:
     - "8090:8080"  # Use different external port
   ```

3. **Database connection string invalid**
   ```bash
   # Check environment variable
   docker compose exec content-service env | grep DATABASE_URL

   # Should be: postgres://user:pass@host:port/db?sslmode=disable
   # Fix in docker-compose.yml or .env file
   ```

4. **Migration failure**
   ```bash
   # Check migration logs
   docker compose logs content-service | grep -i migration

   # Manually run migrations
   docker compose exec postgres-content psql -U cairn -d content_service
   SELECT * FROM schema_migrations;

   # If stuck, see Migration Issues section
   ```

### Issue: RSS Fetcher Service fails to start

**Diagnosis**:
```bash
# Check logs
docker compose logs rss-fetcher-service

# Check dependencies
docker compose ps content-service
docker compose ps postgres-fetcher
```

**Common Causes & Solutions**:

1. **Content Service unavailable**
   ```bash
   # Verify Content Service is healthy
   curl http://localhost:8080/health/ready

   # If not healthy, fix Content Service first
   # Then restart RSS Fetcher
   docker compose restart rss-fetcher-service
   ```

2. **Circuit breaker open**
   ```bash
   # Check logs for circuit breaker messages
   docker compose logs rss-fetcher-service | grep -i "circuit"

   # Wait 30 seconds for half-open state
   # Or restart service
   docker compose restart rss-fetcher-service
   ```

## Database Issues

### Issue: "connection refused" or "could not connect to server"

**Diagnosis**:
```bash
# Check if PostgreSQL is running
docker compose ps postgres-content

# Try to connect manually
docker compose exec postgres-content psql -U cairn -d content_service

# Check network connectivity
docker compose exec content-service ping postgres-content
```

**Solutions**:

1. **Database container not running**
   ```bash
   # Start database
   docker compose up -d postgres-content

   # Wait for healthy status
   docker compose ps postgres-content
   ```

2. **Wrong connection parameters**
   ```bash
   # Check connection string
   # Format: postgres://user:pass@host:port/db?options

   # For Docker Compose, host should be container name
   DATABASE_URL=postgres://cairn:password@postgres-content:5432/content_service?sslmode=disable

   # NOT localhost:
   # ❌ postgres://cairn:password@localhost:5432/...
   ```

3. **Network issues**
   ```bash
   # Recreate network
   docker compose down
   docker compose up -d
   ```

### Issue: "too many connections"

**Symptoms**:
- "FATAL: remaining connection slots are reserved"
- "FATAL: sorry, too many clients already"

**Diagnosis**:
```bash
# Check current connections
docker compose exec postgres-content psql -U cairn -d content_service -c \
  "SELECT count(*) FROM pg_stat_activity WHERE datname='content_service';"

# Check max connections
docker compose exec postgres-content psql -U cairn -d content_service -c \
  "SHOW max_connections;"
```

**Solutions**:

1. **Reduce connection pool size**
   ```bash
   # In docker-compose.yml or .env
   DB_MAX_OPEN_CONNS=10  # Reduce from 25
   DB_MAX_IDLE_CONNS=2   # Reduce from 5

   # Restart service
   docker compose restart content-service
   ```

2. **Increase PostgreSQL max_connections**
   ```bash
   # In docker-compose.yml, add to postgres-content:
   command: postgres -c max_connections=200

   # Restart database
   docker compose restart postgres-content
   ```

3. **Kill idle connections**
   ```sql
   -- Connect to database
   docker compose exec postgres-content psql -U cairn -d content_service

   -- Kill idle connections
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE datname = 'content_service'
     AND state = 'idle'
     AND state_change < NOW() - INTERVAL '5 minutes';
   ```

### Issue: Slow database queries

**Diagnosis**:
```sql
-- Find slow queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query, state
FROM pg_stat_activity
WHERE (now() - pg_stat_activity.query_start) > interval '5 seconds'
  AND state != 'idle';

-- Check table sizes
SELECT schemaname, tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Check missing indexes
SELECT schemaname, tablename, attname, null_frac, avg_width, n_distinct
FROM pg_stats
WHERE schemaname = 'public'
  AND null_frac > 0.5;
```

**Solutions**:

1. **Analyze and vacuum tables**
   ```sql
   VACUUM ANALYZE contents;
   VACUUM ANALYZE user_contents;
   VACUUM ANALYZE feeds;
   VACUUM ANALYZE feed_items;
   ```

2. **Check query plans**
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM user_contents
   WHERE user_id = '550e8400-e29b-41d4-a716-446655440000'
   ORDER BY added_at DESC
   LIMIT 20;

   -- Look for "Seq Scan" - should use index
   -- If no index used, check migrations
   ```

3. **Monitor long-running queries**
   ```sql
   -- Set statement timeout (in postgresql.conf or per-session)
   SET statement_timeout = '30s';

   -- Or in docker-compose.yml:
   command: postgres -c statement_timeout=30000
   ```

## Content Processing Issues

### Issue: Content extraction fails

**Symptoms**:
- "failed to fetch content" errors
- "timeout" errors
- Content not appearing in user's list

**Diagnosis**:
```bash
# Check Content Service logs
docker compose logs content-service | grep -i error

# Test content creation manually
curl -X POST http://localhost:8080/api/v1/contents \
  -H "Content-Type: application/json" \
  -d '{
    "source_url": "https://example.com/article",
    "title": "Test Article"
  }'
```

**Common Causes & Solutions**:

1. **Timeout fetching URL**
   ```bash
   # Increase timeout
   CONTENT_FETCH_TIMEOUT_SECONDS=60

   # Or provide HTML directly instead of URL
   curl -X POST http://localhost:8080/api/v1/contents \
     -H "Content-Type: application/json" \
     -d '{
       "html": "<html>...</html>",
       "source_url": "https://example.com/article"
     }'
   ```

2. **Paywall or anti-scraping**
   ```bash
   # RSS Fetcher fallback to description is automatic
   # For manual content, provide HTML:

   # The service will use RSS description if full fetch fails
   # Check logs for fallback messages
   docker compose logs rss-fetcher-service | grep -i fallback
   ```

3. **Content too large**
   ```bash
   # Check content size limit
   MAX_CONTENT_SIZE_MB=5

   # Increase if needed (use with caution):
   MAX_CONTENT_SIZE_MB=10

   # Restart service
   docker compose restart content-service
   ```

### Issue: Sanitization removes too much content

**Symptoms**:
- Images missing
- Formatting lost
- Links removed

**Diagnosis**:
```bash
# Check sanitization settings
docker compose exec content-service env | grep SANITIZ

# Review bluemonday configuration in code:
# services/content-service/internal/processor/sanitizer.go
```

**Solutions**:

1. **Sanitization is working as designed** - it removes potentially dangerous content
2. **If you need to allow more tags**, edit `sanitizer.go`:
   ```go
   // Add more allowed tags
   policy.AllowElements("figure", "figcaption", "video", "audio")
   ```
3. **Disable sanitization** (NOT recommended for production):
   ```bash
   ENABLE_SANITIZATION=false  # Security risk!
   ```

## Feed Polling Issues

### Issue: Feeds not being polled

**Symptoms**:
- No new content appearing
- `last_fetched_at` not updating
- Workers not processing feeds

**Diagnosis**:
```bash
# Check worker logs
docker compose logs rss-fetcher-worker

# Check feed status in database
docker compose exec postgres-fetcher psql -U cairn -d rss_fetcher_service -c \
  "SELECT id, feed_url, status, polling_tier, last_fetched_at, next_poll_at
   FROM feeds
   ORDER BY next_poll_at ASC
   LIMIT 10;"
```

**Common Causes & Solutions**:

1. **Worker not running**
   ```bash
   # Check if worker container is running
   docker compose ps rss-fetcher-worker

   # If not running, start it
   docker compose up -d rss-fetcher-worker

   # Check logs for errors
   docker compose logs rss-fetcher-worker
   ```

2. **Feeds are disabled**
   ```sql
   -- Check disabled feeds
   SELECT id, feed_url, status, consecutive_error_days, last_error_message
   FROM feeds
   WHERE status = 'disabled';

   -- Re-enable a feed
   UPDATE feeds
   SET status = 'active',
       consecutive_error_days = 0,
       next_poll_at = NOW()
   WHERE id = 'feed-uuid-here';

   -- Or use API
   curl -X PATCH http://localhost:8081/api/v1/feeds/{feed_id}/enable
   ```

3. **All feeds in future polling**
   ```sql
   -- Check next poll times
   SELECT feed_url, next_poll_at, polling_tier
   FROM feeds
   WHERE status = 'active'
   ORDER BY next_poll_at ASC
   LIMIT 10;

   -- If all in future, wait or manually trigger:
   UPDATE feeds
   SET next_poll_at = NOW()
   WHERE status = 'active';
   ```

### Issue: Feed returns errors

**Symptoms**:
- `consecutive_error_days` incrementing
- Feed auto-disabled after 7 days
- Error messages in `last_error_message`

**Diagnosis**:
```sql
-- Check feed errors
SELECT feed_url, status, consecutive_error_days,
       last_error_at, last_error_message
FROM feeds
WHERE consecutive_error_days > 0
ORDER BY consecutive_error_days DESC;
```

**Common Causes & Solutions**:

1. **Feed URL is invalid or moved**
   ```bash
   # Test feed URL manually
   curl -I https://blog.example.com/feed.xml

   # If 404 or redirect, update feed URL:
   UPDATE feeds
   SET feed_url = 'https://blog.example.com/new-feed.xml',
       consecutive_error_days = 0,
       status = 'active'
   WHERE id = 'feed-uuid';
   ```

2. **Feed requires authentication**
   ```bash
   # Some feeds require auth - not currently supported
   # Alternative: Use a feed proxy service
   ```

3. **Network timeout**
   ```bash
   # Increase timeout
   FEED_FETCH_TIMEOUT_SECONDS=60

   # Restart worker
   docker compose restart rss-fetcher-worker
   ```

### Issue: Duplicate content appearing

**Symptoms**:
- Same article appears multiple times
- Users see repeated items

**Diagnosis**:
```sql
-- Check for duplicates by content_hash
SELECT content_hash, source_feed_id, COUNT(*)
FROM contents
GROUP BY content_hash, source_feed_id
HAVING COUNT(*) > 1;

-- Check feed items
SELECT feed_id, item_guid, COUNT(*)
FROM feed_items
GROUP BY feed_id, item_guid
HAVING COUNT(*) > 1;
```

**Solutions**:

1. **Should not happen** - deduplication is enforced by unique constraints
2. **If duplicates exist**, check constraints:
   ```sql
   -- Verify unique constraint exists
   SELECT conname, contype, conkey
   FROM pg_constraint
   WHERE conrelid = 'contents'::regclass
     AND contype = 'u';

   -- Should see: UNIQUE(content_hash, source_feed_id)
   ```

3. **Manual cleanup** (if duplicates somehow exist):
   ```sql
   -- Keep only first occurrence
   DELETE FROM contents a USING contents b
   WHERE a.id > b.id
     AND a.content_hash = b.content_hash
     AND a.source_feed_id = b.source_feed_id;
   ```

## Performance Issues

### Issue: High memory usage

**Diagnosis**:
```bash
# Check container memory usage
docker stats

# Check service logs for OOM
docker compose logs | grep -i "out of memory"

# Check Go heap profile (if metrics enabled)
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

**Solutions**:

1. **Reduce worker concurrency**
   ```bash
   WORKER_CONCURRENCY=3  # Reduce from 5
   CONTENT_EXTRACTION_BATCH_SIZE=10  # Reduce from 20

   docker compose restart rss-fetcher-worker
   ```

2. **Reduce database connection pool**
   ```bash
   DB_MAX_OPEN_CONNS=10  # Reduce from 25
   DB_MAX_IDLE_CONNS=2   # Reduce from 5

   docker compose restart content-service rss-fetcher-service
   ```

3. **Limit content size**
   ```bash
   MAX_CONTENT_SIZE_MB=3  # Reduce from 5

   docker compose restart content-service
   ```

4. **Add memory limits**
   ```yaml
   # In docker-compose.yml
   services:
     content-service:
       deploy:
         resources:
           limits:
             memory: 1G
   ```

### Issue: High CPU usage

**Diagnosis**:
```bash
# Check CPU usage
docker stats

# Check for CPU-intensive queries
SELECT pid, query, state,
       (now() - query_start) AS runtime
FROM pg_stat_activity
WHERE state != 'idle'
ORDER BY runtime DESC;
```

**Solutions**:

1. **Reduce polling frequency**
   ```bash
   # Increase polling intervals
   TIER_ACTIVE_INTERVAL_HOURS=2  # From 1
   TIER_MODERATE_INTERVAL_HOURS=12  # From 6
   TIER_QUIET_INTERVAL_HOURS=48  # From 24
   ```

2. **Reduce batch sizes**
   ```bash
   FEED_BATCH_SIZE=5  # Reduce from 10
   CONTENT_EXTRACTION_BATCH_SIZE=10  # Reduce from 20
   ```

3. **Add CPU limits**
   ```yaml
   # In docker-compose.yml
   services:
     rss-fetcher-worker:
       deploy:
         resources:
           limits:
             cpus: '1.5'
   ```

### Issue: Slow API responses

**Diagnosis**:
```bash
# Test API response time
time curl http://localhost:8080/api/v1/users/USER_ID/contents

# Check database query performance
# See "Slow database queries" section above
```

**Solutions**:

1. **Add database indexes** (should already exist from migrations)
2. **Reduce page size**
   ```bash
   # Request smaller pages
   curl "http://localhost:8080/api/v1/users/USER_ID/contents?limit=10"
   ```

3. **Use pagination cursor**
   ```bash
   # More efficient than large offsets
   curl "http://localhost:8080/api/v1/users/USER_ID/contents?cursor=CURSOR"
   ```

## API Issues

### Issue: 404 Not Found

**Diagnosis**:
```bash
# Check exact endpoint
curl -v http://localhost:8080/api/v1/contents/CONTENT_ID

# Check service is running
docker compose ps content-service

# Check routing
docker compose logs content-service | grep -i "route not found"
```

**Solutions**:

1. **Verify correct URL**
   ```bash
   # Correct: /api/v1/contents (plural)
   # Wrong: /api/v1/content (singular)

   # See OpenAPI specs for exact paths:
   # services/content-service/api/openapi.yaml
   ```

2. **Verify resource exists**
   ```bash
   # Check if content ID exists in database
   docker compose exec postgres-content psql -U cairn -d content_service -c \
     "SELECT id, title FROM contents WHERE id = 'CONTENT_ID';"
   ```

### Issue: 500 Internal Server Error

**Diagnosis**:
```bash
# Check service logs
docker compose logs content-service | tail -50

# Look for stack traces
docker compose logs content-service | grep -A 20 "panic\|FATAL\|ERROR"
```

**Solutions**:

1. **Check error message in logs** - follow specific error guidance
2. **Database connection issue** - see Database Issues section
3. **If panic/crash** - file a bug report with stack trace

### Issue: Request timeout

**Diagnosis**:
```bash
# Test with verbose output
curl -v --max-time 60 http://localhost:8080/api/v1/contents/CONTENT_ID

# Check for slow queries
# See Database Issues -> Slow queries
```

**Solutions**:

1. **Increase HTTP timeouts**
   ```bash
   HTTP_READ_TIMEOUT_SECONDS=60  # From 30
   HTTP_WRITE_TIMEOUT_SECONDS=60  # From 30

   docker compose restart content-service
   ```

2. **Optimize query** - see Performance Issues

## Worker Issues

### Issue: Outbox not delivering

**Symptoms**:
- Content not appearing in Content Service
- Outbox entries stuck in 'pending' or 'failed'
- `retry_count` incrementing

**Diagnosis**:
```bash
# Check outbox worker logs
docker compose logs rss-fetcher-worker | grep -i outbox

# Check outbox status
docker compose exec postgres-fetcher psql -U cairn -d rss_fetcher_service -c \
  "SELECT delivery_status, COUNT(*)
   FROM content_outbox
   GROUP BY delivery_status;"

# Check failed entries
docker compose exec postgres-fetcher psql -U cairn -d rss_fetcher_service -c \
  "SELECT id, delivery_status, retry_count, created_at
   FROM content_outbox
   WHERE delivery_status = 'failed'
   ORDER BY created_at DESC
   LIMIT 10;"
```

**Solutions**:

1. **Content Service is down**
   ```bash
   # Check Content Service health
   curl http://localhost:8080/health/ready

   # If down, fix Content Service first
   # Outbox will auto-retry when it's back up
   ```

2. **Circuit breaker is open**
   ```bash
   # Check logs for circuit breaker
   docker compose logs rss-fetcher-worker | grep -i circuit

   # Wait for half-open state (30s)
   # Or restart worker to reset
   docker compose restart rss-fetcher-worker
   ```

3. **Max retries exceeded**
   ```bash
   # Failed entries won't retry after 6 attempts
   # Manually retry or investigate error:

   # Check error details in logs
   # Then reset retry count:
   UPDATE content_outbox
   SET retry_count = 0,
       delivery_status = 'pending',
       next_retry_at = NOW()
   WHERE id = 'outbox-entry-id';
   ```

## Docker Issues

### Issue: "No space left on device"

**Diagnosis**:
```bash
# Check Docker disk usage
docker system df

# Check disk space
df -h
```

**Solutions**:

1. **Clean up Docker resources**
   ```bash
   # Remove unused containers, networks, images
   docker system prune -a

   # Remove unused volumes (WARNING: deletes data)
   docker volume prune

   # Remove specific stopped containers
   docker container prune
   ```

2. **Clean up logs**
   ```bash
   # Truncate logs
   truncate -s 0 /var/lib/docker/containers/*/*-json.log

   # Or configure log rotation in docker-compose.yml:
   logging:
     options:
       max-size: "10m"
       max-file: "3"
   ```

### Issue: "Cannot connect to Docker daemon"

**Diagnosis**:
```bash
# Check Docker status
sudo systemctl status docker

# Check Docker socket
ls -la /var/run/docker.sock
```

**Solutions**:

1. **Start Docker**
   ```bash
   sudo systemctl start docker
   sudo systemctl enable docker
   ```

2. **Add user to docker group**
   ```bash
   sudo usermod -aG docker $USER

   # Log out and log back in
   ```

## Migration Issues

### Issue: Migration fails

**Symptoms**:
- "migration version XXX dirty" error
- Service won't start
- Database schema mismatch

**Diagnosis**:
```sql
-- Check migration status
SELECT * FROM schema_migrations;

-- Check if migration is "dirty"
SELECT version, dirty FROM schema_migrations;
```

**Solutions**:

1. **Migration marked as dirty**
   ```bash
   # Manually fix migration state
   docker compose exec postgres-content psql -U cairn -d content_service -c \
     "UPDATE schema_migrations SET dirty = false WHERE version = XXX;"

   # Then retry migration
   make migrate-up
   ```

2. **Rollback and retry**
   ```bash
   # Rollback last migration
   make migrate-down

   # Reapply migration
   make migrate-up
   ```

3. **Manual intervention required**
   ```sql
   -- Connect to database
   docker compose exec postgres-content psql -U cairn -d content_service

   -- Check what's wrong
   \dt  -- List tables
   \d table_name  -- Describe specific table

   -- Manually apply SQL from migration file if needed
   -- Then update schema_migrations table
   ```

## Debugging Tips

### Enable Debug Logging

```bash
# In docker-compose.yml or .env
LOG_LEVEL=debug

# Restart services
docker compose restart content-service rss-fetcher-service
```

### View Real-Time Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f content-service

# Filter logs
docker compose logs -f | grep ERROR
docker compose logs -f content-service | grep "user_id=USER_ID"
```

### Execute Commands in Containers

```bash
# Get shell in container
docker compose exec content-service sh

# Run Go binary directly (for debugging)
docker compose exec content-service /app/content-service

# Check environment
docker compose exec content-service env
```

### Database Debugging

```sql
-- Connect to database
docker compose exec postgres-content psql -U cairn -d content_service

-- List all tables
\dt

-- Describe table
\d contents

-- View recent activity
SELECT * FROM pg_stat_activity WHERE datname = 'content_service';

-- Check locks
SELECT * FROM pg_locks WHERE NOT granted;

-- Kill a query
SELECT pg_cancel_backend(PID);

-- Terminate connection
SELECT pg_terminate_backend(PID);
```

### Network Debugging

```bash
# Test connectivity between containers
docker compose exec content-service ping postgres-content
docker compose exec rss-fetcher-service ping content-service

# Check DNS resolution
docker compose exec content-service nslookup postgres-content

# Test HTTP connectivity
docker compose exec rss-fetcher-service wget -O- http://content-service:8080/health/live
```

### Resource Monitoring

```bash
# Real-time resource usage
docker stats

# Detailed container info
docker inspect content-service

# Check container processes
docker compose top content-service
```

## Getting Help

If you're still experiencing issues:

1. **Check existing issues**: [GitHub Issues](https://github.com/andrew-craig/cairn-reader/services/read/issues)
2. **Review documentation**:
   - [README.md](../README.md)
   - [ARCHITECTURE.md](ARCHITECTURE.md)
   - [DEPLOYMENT.md](DEPLOYMENT.md)
   - [CONFIGURATION.md](CONFIGURATION.md)
3. **File a new issue** with:
   - Exact error message
   - Relevant logs
   - Steps to reproduce
   - Environment details (OS, Docker version, etc.)
   - Configuration (sanitize secrets!)

## Common Error Messages

| Error | Likely Cause | Solution |
|-------|--------------|----------|
| `connection refused` | Service not running or wrong host | Check service status, verify host in connection string |
| `too many connections` | Connection pool too large | Reduce `DB_MAX_OPEN_CONNS` |
| `timeout` | Operation taking too long | Increase timeout settings, check query performance |
| `not found` | Resource doesn't exist | Verify ID, check database |
| `duplicate key value` | Unique constraint violation | Check for existing record, may be OK (deduplication) |
| `migration dirty` | Migration failed mid-way | Reset dirty flag, retry migration |
| `circuit breaker open` | Dependency is failing | Wait for half-open, fix dependency, restart |
| `out of memory` | Memory limit exceeded | Reduce concurrency, add memory limits, reduce batch sizes |
| `permission denied` | File/DB permissions issue | Check ownership, user permissions |

## Prevention Best Practices

1. **Monitor health checks** regularly
2. **Set up log monitoring/alerting** for ERROR level
3. **Back up databases** regularly
4. **Test migrations** in staging first
5. **Use resource limits** to prevent runaway processes
6. **Monitor disk space** and set up alerts
7. **Keep dependencies updated** but test first
8. **Document custom configurations**
9. **Test disaster recovery** procedures
10. **Review logs** periodically for warnings
