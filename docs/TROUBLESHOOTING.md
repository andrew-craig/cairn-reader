# Troubleshooting Guide

This guide covers common issues, their causes, and solutions for Cairn Backend services. Commands assume the dev stack (`infrastructure/docker/dev`); service names, host ports and database names come from its `docker-compose.yml` (see [infrastructure/docker/README.md](../infrastructure/docker/README.md)). Quick checks, deployment steps and backup/restore live in [DEPLOYMENT.md](DEPLOYMENT.md); environment variables are in [CONFIGURATION.md](CONFIGURATION.md).

Compose services: `cairn-db`, `vault`, `user-service`, `content-service`, `content-worker`, `ingest-rss`, `ingest-rss-worker`, `email-ingest`, `email-ingest-worker`. Each service has its own database inside `cairn-db` (`content_service`, `rss_fetcher_service`, `ingest_email`, `cairn_users` by default). The `psql` examples use the admin user from the compose `.env` (`POSTGRES_ADMIN_USER`, `cairn_admin` in `.env.example`).

## Table of Contents

- [Service Won't Start](#service-wont-start)
- [Database Issues](#database-issues)
- [Feed Polling Issues](#feed-polling-issues)
- [Performance Issues](#performance-issues)
- [API Issues](#api-issues)
- [Worker Issues](#worker-issues)
- [Docker Issues](#docker-issues)
- [Migration Issues](#migration-issues)
- [Debugging Tips](#debugging-tips)

## Service Won't Start

### Issue: A service exits immediately or fails its health check

**Diagnosis**:
```bash
docker compose ps
docker compose logs content-service        # or user-service, ingest-rss, email-ingest, ...
```

Services log `failed to load configuration` and exit when validation fails. Look for `invalid configuration: ...`.

**Common Causes & Solutions**:

1. **Missing required configuration**: `INTERNAL_API_KEY` (Content, Fetcher), Vault credentials (Content, User), `DB_PASSWORD` and `INGEST_API_KEY` (Email Ingest). See [CONFIGURATION.md](CONFIGURATION.md) for the full list of required variables per service.

2. **Database not ready or wrong credentials**
   ```bash
   docker compose ps cairn-db        # should be "healthy"
   docker compose logs cairn-db
   ```
   Every service connects with discrete `DB_HOST` / `DB_PORT` / `DB_USER` / `DB_PASSWORD` / `DB_NAME` variables (there is no `DATABASE_URL`). Inside the compose network `DB_HOST` is `cairn-db`, not `localhost`.

3. **Database SSL error**: the bundled `cairn-db` has no TLS, but the Content and Fetcher APIs default `DB_SSLMODE` to `require`. Set `DB_SSLMODE=disable`. See [CONFIGURATION.md](CONFIGURATION.md#common-configuration-errors).

4. **Vault unreachable or keys missing** (User, Content, Email Ingest): see [CONFIGURATION.md](CONFIGURATION.md#troubleshooting) and the Vault entries in [DEPLOYMENT.md](DEPLOYMENT.md#troubleshooting).

5. **Port already in use**
   ```bash
   sudo lsof -i :8083
   ```
   Change the host side of the `ports:` mapping in the compose file (for example `"8090:8080"`).

6. **Migration failure**: APIs run their migrations on startup.
   ```bash
   docker compose logs content-service | grep -i migration
   ```
   See [Migration Issues](#migration-issues).

### Issue: Fetcher worker cannot deliver to the Content Service

```bash
curl http://localhost:8083/health/ready        # Content Service
docker compose logs ingest-rss-worker | grep -i "circuit"
```

The worker's Content Service client opens its circuit breaker after at least 5 requests with a 60% or higher failure ratio, logs `Circuit breaker state changed`, and tries again (half-open) after 30 seconds. Fix the Content Service first; the worker recovers on its own, or restart it: `docker compose restart ingest-rss-worker`.

## Database Issues

### Issue: "connection refused" or "could not connect to server"

**Diagnosis**:
```bash
docker compose ps cairn-db
docker compose exec cairn-db psql -U cairn_admin -d postgres -c "SELECT 1;"
docker compose exec content-service ping cairn-db
```

**Solutions**:

1. **Database container not running**: `docker compose up -d cairn-db`, then wait for `healthy`.
2. **Wrong connection parameters**: in Compose, `DB_HOST` must be the container name `cairn-db`; `localhost` only works for services run on the host (published port 5432).
3. **Network issues**: `docker compose down && docker compose up -d`.

### Issue: "too many connections"

**Symptoms**:
- `FATAL: remaining connection slots are reserved`
- `FATAL: sorry, too many clients already`

**Diagnosis**:
```bash
docker compose exec cairn-db psql -U cairn_admin -d postgres -c \
  "SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;"
docker compose exec cairn-db psql -U cairn_admin -d postgres -c "SHOW max_connections;"
```

**Solutions**:

1. Reduce `DB_MAX_OPEN_CONNS` / `DB_MAX_IDLE_CONNS` for the services involved. The sum across every process must stay below PostgreSQL `max_connections`; the production compose file documents its budget in a comment above `cairn-db`. See [CONFIGURATION.md](CONFIGURATION.md#database-configuration).
2. Raise `max_connections` in the `cairn-db` `command:` in the compose file, then restart `cairn-db`.
3. Terminate idle connections:
   ```sql
   SELECT pg_terminate_backend(pid)
   FROM pg_stat_activity
   WHERE datname = 'content_service'
     AND state = 'idle'
     AND state_change < NOW() - INTERVAL '5 minutes';
   ```

### Issue: Slow database queries

Each service sets a per-connection statement timeout (`DB_STATEMENT_TIMEOUT`, default `30s`), so slow queries surface as `canceling statement due to statement timeout`.

**Diagnosis** (run in the affected database, e.g. `docker compose exec cairn-db psql -U cairn_admin -d content_service`):
```sql
SELECT pid, now() - query_start AS duration, query, state
FROM pg_stat_activity
WHERE (now() - query_start) > interval '5 seconds'
  AND state != 'idle';

SELECT relname, pg_size_pretty(pg_total_relation_size(relid)) AS size
FROM pg_catalog.pg_statio_user_tables
ORDER BY pg_total_relation_size(relid) DESC;
```

**Solutions**:

1. **Analyze and vacuum** (content database: `contents`, `user_contents`; fetcher database `rss_fetcher_service`: `feeds`, `feed_items`, `content_outbox`):
   ```sql
   VACUUM ANALYZE contents;
   VACUUM ANALYZE user_contents;
   ```
2. **Check query plans** for a `Seq Scan` where an index is expected:
   ```sql
   EXPLAIN ANALYZE
   SELECT * FROM user_contents
   WHERE user_id = '550e8400-e29b-41d4-a716-446655440000' AND list = 'reads'
   ORDER BY added_at DESC
   LIMIT 20;
   ```
   Indexes come from the migrations in `services/read/content/migrations`.

## Feed Polling Issues

### Issue: Feeds not being polled

**Symptoms**:
- No new content appearing
- `last_fetched_at` not updating

**Diagnosis**:
```bash
docker compose logs ingest-rss-worker
docker compose exec cairn-db psql -U cairn_admin -d rss_fetcher_service -c \
  "SELECT id, feed_url, status, polling_tier, last_fetched_at, next_poll_at
   FROM feeds ORDER BY next_poll_at ASC LIMIT 10;"
```

**Common Causes & Solutions**:

1. **Worker not running**: `docker compose ps ingest-rss-worker`, then `docker compose up -d ingest-rss-worker`. Its health port is `8086` in dev.

2. **Feeds are disabled**: a feed is disabled after 7 consecutive error days (`services/read/fetcher/internal/scheduler/poll_scheduler.go`). There is no API to re-enable it; update the row:
   ```sql
   SELECT id, feed_url, consecutive_error_days, last_error_message
   FROM feeds WHERE status = 'disabled';

   UPDATE feeds
   SET status = 'active', consecutive_error_days = 0, next_poll_at = NOW()
   WHERE id = 'feed-uuid-here';
   ```

3. **All feeds scheduled in the future**: wait for `next_poll_at`, or force a poll:
   ```sql
   UPDATE feeds SET next_poll_at = NOW() WHERE status = 'active';
   ```

### Issue: Feed returns errors

**Symptoms**: `consecutive_error_days` incrementing, `last_error_message` populated.

```sql
SELECT feed_url, status, consecutive_error_days, last_error_at, last_error_message
FROM feeds
WHERE consecutive_error_days > 0
ORDER BY consecutive_error_days DESC;
```

1. **Feed URL invalid or moved**: test with `curl -I <feed_url>`; if it moved, update `feeds.feed_url` and reset `consecutive_error_days = 0, status = 'active'`.
2. **Feed requires authentication**: not supported.
3. **Network timeouts**: check the worker logs (`failed to fetch feed` entries carry the `feed_id` and error).

### Issue: Duplicate content appearing

Deduplication is enforced by unique indexes, so duplicates should not occur.

```sql
-- fetcher database
SELECT feed_id, item_guid, COUNT(*)
FROM feed_items GROUP BY feed_id, item_guid HAVING COUNT(*) > 1;

-- content database
SELECT content_hash, source_feed_id, COUNT(*)
FROM contents GROUP BY content_hash, source_feed_id HAVING COUNT(*) > 1;

SELECT indexname FROM pg_indexes
WHERE tablename = 'contents' AND indexname IN ('idx_contents_rss_dedup', 'idx_contents_nonrss_dedup');
```

If either query returns rows, or the indexes are missing, check that all content migrations (`000004_unique_content_dedup`) were applied (see [Migration Issues](#migration-issues)) and file a bug.

## Performance Issues

### Issue: High memory or CPU usage

```bash
docker stats
docker compose logs | grep -i "out of memory"
```

**Solutions**:

1. **Reduce worker load** (Fetcher worker): `FEED_WORKER_COUNT`, `FEED_WORKER_QUEUE_SIZE`, `POLL_BATCH_SIZE`, `POLL_WORKER_POOL_SIZE`, `POLL_INTERVAL`. Defaults are in [CONFIGURATION.md](CONFIGURATION.md#rss-fetcher-service-configuration); restart with `docker compose restart ingest-rss-worker`.
2. **Reduce the database connection pool**: `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`.
3. **Add container limits**:
   ```yaml
   services:
     content-service:
       deploy:
         resources:
           limits:
             memory: 1G
   ```

### Issue: Slow API responses

```bash
time curl -H "Authorization: Bearer $TOKEN" "http://localhost:8083/api/v1/content/user/USER_ID?limit=10"
```

1. Check for slow queries (see [Slow database queries](#issue-slow-database-queries)).
2. Request smaller pages and use the pagination cursor rather than large offsets. See `services/read/content/api/openapi.yaml` for the list parameters.

## API Issues

### Issue: 404 Not Found

```bash
curl -v http://localhost:8083/health/live
docker compose ps content-service
```

Content Service paths are under `/api/v1/content` (singular), e.g. `/api/v1/content/user/{user_id}`. Fetcher subscription paths are under `/api/v1/source/rss/...`. Check the exact path in the service OpenAPI spec (`services/read/content/api/openapi.yaml`, `services/read/fetcher/api/openapi.yaml`, `services/users/api/openapi.yaml`, `services/read/email/api/openapi.yaml`).

### Issue: 401 / 403

User endpoints require `Authorization: Bearer <token>` (RS256 JWT) and only allow access to the caller's own data. A service that cannot fetch the JWT public key from Vault rejects tokens; see the Vault checks in [CONFIGURATION.md](CONFIGURATION.md#troubleshooting). Internal endpoints use `INTERNAL_API_KEY`.

### Issue: 500 Internal Server Error

```bash
docker compose logs content-service | tail -50
docker compose logs content-service | grep -A 20 "panic\|ERROR"
```

1. Follow the specific error in the logs.
2. Database connection issues: see [Database Issues](#database-issues).
3. If a panic/crash occurs, file a bug with the stack trace.

## Worker Issues

### Issue: Outbox not delivering

**Symptoms**:
- Fetched items not appearing in the Content Service
- `content_outbox` rows stuck in `pending`, `sending` or `failed`; `retry_count` incrementing

**Diagnosis**:
```bash
docker compose logs ingest-rss-worker | grep -i outbox

docker compose exec cairn-db psql -U cairn_admin -d rss_fetcher_service -c \
  "SELECT delivery_status, COUNT(*) FROM content_outbox GROUP BY delivery_status;"

docker compose exec cairn-db psql -U cairn_admin -d rss_fetcher_service -c \
  "SELECT id, delivery_status, retry_count, last_error, created_at
   FROM content_outbox WHERE delivery_status = 'failed'
   ORDER BY created_at DESC LIMIT 10;"
```

**Solutions**:

1. **Content Service is down**: check `curl http://localhost:8083/health/ready`. Pending entries retry automatically once it is back.
2. **Circuit breaker is open**: see [Fetcher worker cannot deliver](#issue-fetcher-worker-cannot-deliver-to-the-content-service).
3. **Rejected by the Content Service** (`401` in `last_error`): the worker's `INTERNAL_API_KEY` is unset or does not match the Content Service's.
4. **Max retries exceeded**: entries are marked `failed` after 6 attempts (`services/read/fetcher/internal/worker/outbox_worker.go`). Fix the cause, then reset:
   ```sql
   UPDATE content_outbox
   SET retry_count = 0, delivery_status = 'pending', next_retry_at = NOW()
   WHERE id = 'outbox-entry-id';
   ```

The Email Ingest worker has an equivalent outbox (`OUTBOX_MAX_RETRIES`, default 6) in the `ingest_email` database; check `docker compose logs email-ingest-worker`.

## Docker Issues

### Issue: "No space left on device"

```bash
docker system df
df -h
docker system prune -a        # remove unused containers, networks, images
docker volume prune           # WARNING: deletes data in unused volumes
```

To bound log growth, configure log rotation in the compose file:
```yaml
logging:
  options:
    max-size: "10m"
    max-file: "3"
```

### Issue: "Cannot connect to Docker daemon"

```bash
sudo systemctl status docker
sudo systemctl start docker
sudo usermod -aG docker $USER      # then log out and back in
```

## Migration Issues

APIs run their embedded migrations on startup (golang-migrate, tracked in the `schema_migrations` table of each service database); see [DEPLOYMENT.md](DEPLOYMENT.md#automatic-migrations).

### Issue: Migration fails

**Symptoms**:
- Service exits at startup with `failed to run migrations`
- `Dirty database version N. Fix and force version.`

**Diagnosis** (in the failing service's database):
```sql
SELECT version, dirty FROM schema_migrations;
```

**Solutions**:

1. **Dirty version**: a migration failed part-way. Inspect the schema, finish or undo the partial change by hand, then clear the flag (set `version` to the last fully applied migration):
   ```bash
   docker compose exec cairn-db psql -U cairn_admin -d content_service -c \
     "UPDATE schema_migrations SET dirty = false WHERE version = N;"
   docker compose restart content-service
   ```
2. **Run or roll back manually** (Read services, from `services/read`; the Makefile targets use the local `localhost:5433` / `5434` URLs):
   ```bash
   make migrate-up-content        # or migrate-up-fetcher
   make migrate-down-content      # rolls back one migration
   ```
   User Service: `make migrate-up` / `migrate-down` / `migrate-version` in `services/users`. Migration files live in each service's `migrations/` directory.

## Debugging Tips

### Enable Debug Logging

```bash
# in the compose .env or service environment
LOG_LEVEL=debug
docker compose restart content-service ingest-rss-worker
```

### View Real-Time Logs

```bash
docker compose logs -f
docker compose logs -f content-service
docker compose logs -f | grep -i error
```

### Execute Commands in Containers

```bash
docker compose exec content-service sh
docker compose exec content-service env
```

### Database Debugging

```bash
docker compose exec cairn-db psql -U cairn_admin -d content_service
```
```sql
\dt                         -- list tables
\d contents                 -- describe a table
SELECT * FROM pg_locks WHERE NOT granted;
SELECT pg_cancel_backend(PID);       -- cancel a query
SELECT pg_terminate_backend(PID);    -- terminate a connection
```

### Network Debugging

```bash
docker compose exec content-service ping cairn-db
docker compose exec ingest-rss-worker wget -O- http://content-service:8080/health/live
```

Dev host ports (user 8082, content 8083, ingest-rss 8085, email-ingest 8089; worker health 8084, 8086, 8090) are listed in [infrastructure/docker/README.md](../infrastructure/docker/README.md).

### Resource Monitoring

```bash
docker stats
docker compose top content-service
```

## Getting Help

If you're still experiencing issues:

1. **Check existing issues**: [GitHub Issues](https://github.com/andrew-craig/cairn-reader/issues)
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
| `connection refused` | Service not running or wrong host | Check service status, verify `DB_HOST` / service URLs |
| `too many connections` | Connection pools too large | Reduce `DB_MAX_OPEN_CONNS` |
| `invalid configuration: ... is required` | Required variable missing | See [CONFIGURATION.md](CONFIGURATION.md#common-configuration-errors) |
| `canceling statement due to statement timeout` | Query exceeded `DB_STATEMENT_TIMEOUT` | Check query plan and indexes |
| `duplicate key value` | Unique constraint violation | Check for existing record, may be OK (deduplication) |
| `Dirty database version` | Migration failed mid-way | Reset dirty flag, retry migration |
| `Circuit breaker state changed` | Content Service calls failing | Fix the Content Service; worker recovers after 30s |
