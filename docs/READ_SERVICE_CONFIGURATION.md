# Configuration Reference

This document provides a complete reference for all configuration options in Cairn Backend services.

## Table of Contents

- [Configuration Methods](#configuration-methods)
- [Content Service Configuration](#content-service-configuration)
- [RSS Fetcher Service Configuration](#rss-fetcher-service-configuration)
- [Database Configuration](#database-configuration)
- [Worker Configuration](#worker-configuration)
- [Environment-Specific Configs](#environment-specific-configs)
- [Security Configuration](#security-configuration)

## Configuration Methods

Cairn services can be configured using:

1. **Environment Variables** (recommended)
2. **`.env` files** (for local development)
3. **Docker Compose environment** (for containerized deployments)

### Priority Order

Configuration values are read in this order (highest to lowest priority):
1. Environment variables
2. `.env` file
3. Default values (hardcoded)

## Content Service Configuration

### Required Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DATABASE_URL` | string | Yes | - | PostgreSQL connection string |
| `SERVER_PORT` | int | No | `8080` | HTTP server port |

### Optional Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `LOG_LEVEL` | string | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `MAX_CONTENT_SIZE_MB` | int | No | `5` | Maximum content size in MB |
| `ENABLE_AUTO_MIGRATIONS` | bool | No | `true` | Run migrations on startup |
| `DB_MAX_OPEN_CONNS` | int | No | `25` | Maximum open database connections |
| `DB_MAX_IDLE_CONNS` | int | No | `5` | Maximum idle database connections |
| `DB_CONN_MAX_LIFETIME_MINUTES` | int | No | `5` | Connection maximum lifetime in minutes |
| `HTTP_READ_TIMEOUT_SECONDS` | int | No | `30` | HTTP read timeout |
| `HTTP_WRITE_TIMEOUT_SECONDS` | int | No | `30` | HTTP write timeout |
| `SHUTDOWN_TIMEOUT_SECONDS` | int | No | `30` | Graceful shutdown timeout |

### Content Processing Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `CONTENT_FETCH_TIMEOUT_SECONDS` | int | No | `30` | Timeout for fetching content from URL |
| `ENABLE_READABILITY` | bool | No | `true` | Enable readability extraction |
| `ENABLE_SANITIZATION` | bool | No | `true` | Enable HTML sanitization |
| `MAX_CONTENT_SIZE_BYTES` | int | No | `5242880` | Max content size (5MB default) |

### Pagination Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DEFAULT_PAGE_SIZE` | int | No | `20` | Default pagination page size |
| `MAX_PAGE_SIZE` | int | No | `100` | Maximum pagination page size |

### Search Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `SEARCH_MAX_RESULTS` | int | No | `100` | Maximum search results to return |
| `SEARCH_LANGUAGE` | string | No | `english` | PostgreSQL text search language |

### Cleanup Job Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `ORPHANED_CONTENT_RETENTION_DAYS` | int | No | `90` | Days before deleting orphaned content |
| `CLEANUP_BATCH_SIZE` | int | No | `100` | Batch size for cleanup operations |
| `CLEANUP_SCHEDULE` | string | No | `0 2 * * *` | Cron schedule for cleanup (2 AM daily) |

### Example `.env` File

```bash
# Content Service Configuration
DATABASE_URL=postgres://cairn:password@localhost:5432/content_service?sslmode=disable
SERVER_PORT=8080
LOG_LEVEL=info

# Database Connection Pool
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_MINUTES=5

# Content Processing
MAX_CONTENT_SIZE_MB=5
CONTENT_FETCH_TIMEOUT_SECONDS=30
ENABLE_READABILITY=true
ENABLE_SANITIZATION=true

# Pagination
DEFAULT_PAGE_SIZE=20
MAX_PAGE_SIZE=100

# Cleanup
ORPHANED_CONTENT_RETENTION_DAYS=90
CLEANUP_BATCH_SIZE=100
CLEANUP_SCHEDULE="0 2 * * *"
```

## RSS Fetcher Service Configuration

### Required Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DATABASE_URL` | string | Yes | - | PostgreSQL connection string |
| `SERVER_PORT` | int | No | `8081` | HTTP server port |
| `CONTENT_SERVICE_URL` | string | Yes | - | Content Service base URL |

### Optional Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `LOG_LEVEL` | string | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `ENABLE_AUTO_MIGRATIONS` | bool | No | `true` | Run migrations on startup |
| `DB_MAX_OPEN_CONNS` | int | No | `25` | Maximum open database connections |
| `DB_MAX_IDLE_CONNS` | int | No | `5` | Maximum idle database connections |
| `DB_CONN_MAX_LIFETIME_MINUTES` | int | No | `5` | Connection maximum lifetime in minutes |
| `HTTP_READ_TIMEOUT_SECONDS` | int | No | `30` | HTTP read timeout |
| `HTTP_WRITE_TIMEOUT_SECONDS` | int | No | `30` | HTTP write timeout |
| `SHUTDOWN_TIMEOUT_SECONDS` | int | No | `30` | Graceful shutdown timeout |

### Feed Polling Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `FEED_FETCH_TIMEOUT_SECONDS` | int | No | `30` | Timeout for fetching RSS feeds |
| `FEED_USER_AGENT` | string | No | `Cairn/1.0` | User agent for feed requests |
| `MAX_FEED_REDIRECTS` | int | No | `10` | Maximum HTTP redirects to follow |
| `FEED_BATCH_SIZE` | int | No | `10` | Number of feeds to poll per batch |
| `FEED_POLL_INTERVAL_SECONDS` | int | No | `10` | Sleep between polling batches |

### Polling Tier Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `TIER_ACTIVE_INTERVAL_HOURS` | int | No | `1` | Active tier polling interval (hours) |
| `TIER_MODERATE_INTERVAL_HOURS` | int | No | `6` | Moderate tier polling interval (hours) |
| `TIER_QUIET_INTERVAL_HOURS` | int | No | `24` | Quiet tier polling interval (hours) |
| `TIER_ACTIVE_THRESHOLD_DAYS` | int | No | `7` | Days for active tier threshold |
| `TIER_MODERATE_THRESHOLD_DAYS` | int | No | `30` | Days for moderate tier threshold |
| `TIER_UPDATE_SCHEDULE` | string | No | `0 1 * * *` | Cron schedule for tier updates (1 AM daily) |

### Feed Error Handling

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `MAX_CONSECUTIVE_ERROR_DAYS` | int | No | `7` | Days of errors before auto-disable |
| `ENABLE_FEED_AUTO_DISABLE` | bool | No | `true` | Auto-disable feeds after consecutive errors |

### Feed Subscription Limits

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `MAX_FEEDS_PER_USER` | int | No | `100` | Maximum feeds per user |

### Worker Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `WORKER_CONCURRENCY` | int | No | `5` | Number of concurrent worker goroutines |
| `CONTENT_EXTRACTION_BATCH_SIZE` | int | No | `20` | Feed items to process per batch |
| `CONTENT_EXTRACTION_INTERVAL_SECONDS` | int | No | `10` | Sleep between extraction batches |
| `ARTICLE_FETCH_TIMEOUT_SECONDS` | int | No | `30` | Timeout for fetching article content |
| `MAX_EXTRACTION_RETRIES` | int | No | `3` | Max retries for failed extractions |

### Outbox Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `OUTBOX_BATCH_SIZE` | int | No | `10` | Outbox entries to process per batch |
| `OUTBOX_DELIVERY_INTERVAL_SECONDS` | int | No | `10` | Sleep between delivery batches |
| `OUTBOX_MAX_RETRIES` | int | No | `6` | Maximum delivery retry attempts |
| `OUTBOX_RETENTION_DAYS` | int | No | `7` | Days to keep delivered outbox entries |

### Retry Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `RETRY_MIN_DELAY_SECONDS` | int | No | `60` | Minimum retry delay (1 minute) |
| `RETRY_MAX_DELAY_SECONDS` | int | No | `43200` | Maximum retry delay (12 hours) |
| `RETRY_MULTIPLIER` | float | No | `2.0` | Exponential backoff multiplier |

### Circuit Breaker Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `CIRCUIT_BREAKER_MAX_REQUESTS` | int | No | `5` | Requests before opening circuit |
| `CIRCUIT_BREAKER_INTERVAL_SECONDS` | int | No | `60` | Interval for counting failures |
| `CIRCUIT_BREAKER_TIMEOUT_SECONDS` | int | No | `30` | Timeout before half-open state |
| `ENABLE_CIRCUIT_BREAKER` | bool | No | `true` | Enable circuit breaker |

### Cleanup Job Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `OUTBOX_CLEANUP_SCHEDULE` | string | No | `0 3 * * *` | Cron schedule (3 AM daily) |
| `FEED_ITEMS_CLEANUP_SCHEDULE` | string | No | `0 4 * * *` | Cron schedule (4 AM daily) |
| `FEED_ITEMS_RETENTION_DAYS` | int | No | `7` | Days to keep completed feed items |
| `FEED_ITEMS_FAILED_RETENTION_DAYS` | int | No | `30` | Days to keep failed feed items |
| `CLEANUP_BATCH_SIZE` | int | No | `100` | Batch size for cleanup operations |

### Example `.env` File

```bash
# RSS Fetcher Service Configuration
DATABASE_URL=postgres://cairn:password@localhost:5433/rss_fetcher_service?sslmode=disable
SERVER_PORT=8081
CONTENT_SERVICE_URL=http://localhost:8080
LOG_LEVEL=info

# Database Connection Pool
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME_MINUTES=5

# Feed Polling
FEED_FETCH_TIMEOUT_SECONDS=30
FEED_USER_AGENT="Cairn/1.0"
MAX_FEED_REDIRECTS=10
FEED_BATCH_SIZE=10
FEED_POLL_INTERVAL_SECONDS=10

# Polling Tiers
TIER_ACTIVE_INTERVAL_HOURS=1
TIER_MODERATE_INTERVAL_HOURS=6
TIER_QUIET_INTERVAL_HOURS=24
TIER_ACTIVE_THRESHOLD_DAYS=7
TIER_MODERATE_THRESHOLD_DAYS=30

# Feed Error Handling
MAX_CONSECUTIVE_ERROR_DAYS=7
ENABLE_FEED_AUTO_DISABLE=true

# Feed Limits
MAX_FEEDS_PER_USER=100

# Workers
WORKER_CONCURRENCY=5
CONTENT_EXTRACTION_BATCH_SIZE=20
CONTENT_EXTRACTION_INTERVAL_SECONDS=10
ARTICLE_FETCH_TIMEOUT_SECONDS=30

# Outbox
OUTBOX_BATCH_SIZE=10
OUTBOX_DELIVERY_INTERVAL_SECONDS=10
OUTBOX_MAX_RETRIES=6
OUTBOX_RETENTION_DAYS=7

# Circuit Breaker
CIRCUIT_BREAKER_MAX_REQUESTS=5
CIRCUIT_BREAKER_INTERVAL_SECONDS=60
CIRCUIT_BREAKER_TIMEOUT_SECONDS=30
ENABLE_CIRCUIT_BREAKER=true

# Cleanup Jobs
OUTBOX_CLEANUP_SCHEDULE="0 3 * * *"
FEED_ITEMS_CLEANUP_SCHEDULE="0 4 * * *"
FEED_ITEMS_RETENTION_DAYS=7
FEED_ITEMS_FAILED_RETENTION_DAYS=30
CLEANUP_BATCH_SIZE=100
```

## Database Configuration

### PostgreSQL Connection String Format

```
postgres://username:password@host:port/database?options
```

**Components**:
- `username`: Database user
- `password`: User password
- `host`: Database hostname or IP
- `port`: Database port (default: 5432)
- `database`: Database name
- `options`: Additional connection options

**Common Options**:
- `sslmode=disable` - Disable SSL (local dev only)
- `sslmode=require` - Require SSL (production)
- `connect_timeout=10` - Connection timeout in seconds
- `statement_timeout=30000` - Statement timeout in milliseconds
- `application_name=cairn-content` - Application name in logs

**Examples**:

```bash
# Local development (no SSL)
DATABASE_URL=postgres://cairn:password@localhost:5432/content_service?sslmode=disable

# Production (with SSL)
DATABASE_URL=postgres://cairn:password@db.example.com:5432/content_service?sslmode=require&connect_timeout=10

# With application name
DATABASE_URL=postgres://cairn:password@localhost:5432/content_service?sslmode=disable&application_name=cairn-content
```

### Connection Pool Settings

| Variable | Recommended | Description |
|----------|-------------|-------------|
| `DB_MAX_OPEN_CONNS` | 25 | Max concurrent connections |
| `DB_MAX_IDLE_CONNS` | 5 | Max idle connections kept open |
| `DB_CONN_MAX_LIFETIME_MINUTES` | 5 | Max connection lifetime |

**Tuning Guidelines**:

- **Small instance**: `MAX_OPEN_CONNS=10`, `MAX_IDLE_CONNS=2`
- **Medium instance**: `MAX_OPEN_CONNS=25`, `MAX_IDLE_CONNS=5`
- **Large instance**: `MAX_OPEN_CONNS=50`, `MAX_IDLE_CONNS=10`

**Formula**:
```
MAX_OPEN_CONNS = (Available CPU Cores * 2) + disk_spindles
```

## Worker Configuration

### Feed Polling Worker

**Key Settings**:
- `WORKER_CONCURRENCY`: Number of feeds processed concurrently
- `FEED_BATCH_SIZE`: Feeds fetched per query
- `FEED_POLL_INTERVAL_SECONDS`: Sleep between batches

**Recommendations**:
- **Low traffic**: Concurrency=3, Batch=5, Interval=30s
- **Medium traffic**: Concurrency=5, Batch=10, Interval=10s
- **High traffic**: Concurrency=10, Batch=20, Interval=5s

### Content Extraction Worker

**Key Settings**:
- `WORKER_CONCURRENCY`: Concurrent article extractions
- `CONTENT_EXTRACTION_BATCH_SIZE`: Feed items per batch
- `ARTICLE_FETCH_TIMEOUT_SECONDS`: Timeout for article fetch

**Recommendations**:
- Start with concurrency=5
- Increase if you have many pending feed items
- Monitor memory usage (article content can be large)

### Outbox Delivery Worker

**Key Settings**:
- `WORKER_CONCURRENCY`: Concurrent Content Service API calls
- `OUTBOX_BATCH_SIZE`: Outbox entries per batch
- `OUTBOX_MAX_RETRIES`: Max retry attempts

**Recommendations**:
- Keep concurrency moderate (5-10) to avoid overwhelming Content Service
- Circuit breaker will open if Content Service is down

## Environment-Specific Configs

### Development

```bash
# .env.development
LOG_LEVEL=debug
ENABLE_AUTO_MIGRATIONS=true
DB_MAX_OPEN_CONNS=10
WORKER_CONCURRENCY=3
FEED_BATCH_SIZE=5
```

### Staging

```bash
# .env.staging
LOG_LEVEL=info
ENABLE_AUTO_MIGRATIONS=true
DB_MAX_OPEN_CONNS=25
WORKER_CONCURRENCY=5
FEED_BATCH_SIZE=10
```

### Production

```bash
# .env.production
LOG_LEVEL=info
ENABLE_AUTO_MIGRATIONS=false  # Run migrations manually
DB_MAX_OPEN_CONNS=50
WORKER_CONCURRENCY=10
FEED_BATCH_SIZE=20
CIRCUIT_BREAKER_TIMEOUT_SECONDS=60
```

## Security Configuration

### Database Security

**Production**:
```bash
# Always use SSL in production
DATABASE_URL=postgres://user:pass@host:5432/db?sslmode=require

# Use strong passwords (min 32 characters)
# Generate with: openssl rand -base64 32
```

**Connection Limits**:
```sql
-- Set connection limit per database
ALTER DATABASE content_service CONNECTION LIMIT 100;

-- Set connection limit per user
ALTER USER cairn CONNECTION LIMIT 50;
```

### HTTP Timeouts

**Prevent resource exhaustion**:
```bash
HTTP_READ_TIMEOUT_SECONDS=30
HTTP_WRITE_TIMEOUT_SECONDS=30
SHUTDOWN_TIMEOUT_SECONDS=30
```

### Content Security

**Size limits**:
```bash
MAX_CONTENT_SIZE_MB=5  # Prevent excessive memory usage
MAX_CONTENT_SIZE_BYTES=5242880
```

**Sanitization**:
```bash
ENABLE_SANITIZATION=true  # Always enabled in production
```

## Docker Compose Configuration

### Example docker-compose.yml

```yaml
version: '3.8'

services:
  content-service:
    build: ./services/content-service
    environment:
      - DATABASE_URL=postgres://cairn:${DB_PASSWORD}@postgres-content:5432/content_service?sslmode=disable
      - SERVER_PORT=8080
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - DB_MAX_OPEN_CONNS=${DB_MAX_OPEN_CONNS:-25}
      - MAX_CONTENT_SIZE_MB=${MAX_CONTENT_SIZE_MB:-5}
    ports:
      - "8080:8080"
    depends_on:
      - postgres-content

  rss-fetcher-service:
    build: ./services/rss-fetcher-service
    environment:
      - DATABASE_URL=postgres://cairn:${DB_PASSWORD}@postgres-fetcher:5432/rss_fetcher_service?sslmode=disable
      - SERVER_PORT=8081
      - CONTENT_SERVICE_URL=http://content-service:8080
      - LOG_LEVEL=${LOG_LEVEL:-info}
      - WORKER_CONCURRENCY=${WORKER_CONCURRENCY:-5}
    ports:
      - "8081:8081"
    depends_on:
      - postgres-fetcher
      - content-service
```

### Environment Variable File (.env)

```bash
# Shared configuration
LOG_LEVEL=info
DB_PASSWORD=secure_random_password_here

# Content Service
DB_MAX_OPEN_CONNS=25
MAX_CONTENT_SIZE_MB=5

# RSS Fetcher
WORKER_CONCURRENCY=5
FEED_BATCH_SIZE=10
```

## Validation & Testing

### Validate Configuration

```bash
# Check if services start successfully
docker-compose up -d

# Check logs for configuration errors
docker-compose logs content-service | grep -i error
docker-compose logs rss-fetcher-service | grep -i error

# Test health endpoints
curl http://localhost:8080/health/ready
curl http://localhost:8081/health/ready
```

### Common Configuration Errors

**Database connection failed**:
- Check `DATABASE_URL` format
- Verify database is running
- Check network connectivity
- Verify credentials

**Worker not processing**:
- Check `WORKER_CONCURRENCY` > 0
- Verify `DATABASE_URL` is correct
- Check worker logs for errors

**High memory usage**:
- Reduce `WORKER_CONCURRENCY`
- Reduce `DB_MAX_OPEN_CONNS`
- Check for memory leaks in logs

## Configuration Best Practices

1. **Never commit secrets** to version control
2. **Use environment variables** for all sensitive data
3. **Use strong passwords** (min 32 characters, random)
4. **Enable SSL** in production (`sslmode=require`)
5. **Set resource limits** to prevent resource exhaustion
6. **Monitor logs** for configuration warnings
7. **Test configuration changes** in staging first
8. **Document custom configs** for your deployment
9. **Back up `.env` files** securely (encrypted)
10. **Rotate credentials** regularly

## Additional Resources

- [Deployment Guide](READ_SERVICE_DEPLOYMENT.md)
- [Troubleshooting Guide](READ_SERVICE_TROUBLESHOOTING.md)
- [Architecture Documentation](READ_SERVICE_ARCHITECTURE.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
