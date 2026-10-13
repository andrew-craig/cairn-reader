# Cairn Configuration Reference

This document provides a complete reference for all configuration options across all Cairn backend services.

## Table of Contents

- [Configuration Methods](#configuration-methods)
- [User Service Configuration](#user-service-configuration)
- [Read Service Configuration](#read-service-configuration)
- [Database Configuration](#database-configuration)
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

---

## User Service Configuration

The User Service handles authentication and account management, requiring HashiCorp Vault for JWT key management and PostgreSQL for user data storage.

### Required Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `VAULT_ADDR` | string | Yes | `http://localhost:8200` | HashiCorp Vault server address |
| `VAULT_TOKEN` | string | Yes* | - | Vault access token (dev mode) |
| `VAULT_ROLE_ID` | string | Yes* | - | Vault AppRole role ID (production) |
| `VAULT_SECRET_ID` | string | Yes* | - | Vault AppRole secret ID (production) |
| `DB_HOST` | string | Yes | `localhost` | PostgreSQL host |
| `DB_PORT` | string | Yes | `5432` | PostgreSQL port |
| `DB_USER` | string | Yes** | - | Database username |
| `DB_PASSWORD` | string | Yes** | - | Database password |
| `DB_NAME` | string | Yes | `cairn_users` | Database name |

**\* Authentication**: Use `VAULT_TOKEN` for dev mode OR `VAULT_ROLE_ID` + `VAULT_SECRET_ID` for production AppRole auth.

**\*\* Database Credentials**: Config validation only requires `DB_USER` when neither `VAULT_TOKEN` nor `VAULT_ROLE_ID` is set, but the service always connects with `DB_USER` / `DB_PASSWORD`; credentials are not fetched from Vault. In `ENVIRONMENT=production`, Vault credentials (token, or role ID plus secret ID) are required.

### Server Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `PORT` | string | No | `8080` | HTTP server port |
| `ENVIRONMENT` | string | No | `development` | Environment: `development`, `staging`, `production` |
| `SHUTDOWN_TIMEOUT` | duration | No | `30s` | Graceful shutdown timeout |

### Database Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DB_SSLMODE` | string | No | `disable` | SSL mode: `disable`, `require`, `verify-ca`, `verify-full` |
| `DB_MAX_OPEN_CONNS` | int | No | `25` | Maximum open database connections |
| `DB_MAX_IDLE_CONNS` | int | No | `5` | Maximum idle database connections |
| `DB_CONN_MAX_LIFETIME` | duration | No | `30m` | Maximum connection lifetime |
| `DB_STATEMENT_TIMEOUT` | duration | No | `30s` | Per-connection statement timeout (`0` disables) |

### Vault Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `VAULT_NAMESPACE` | string | No | - | Vault namespace (Enterprise only) |
| `VAULT_AUTH_PATH` | string | No | `approle` | Vault AppRole auth mount path |
| `VAULT_DB_CREDS_PATH` | string | No | `secret/data/database/credentials` | Read into config but not currently used; database credentials always come from `DB_USER` / `DB_PASSWORD` |
| `VAULT_TOKEN_RENEWAL_INTERVAL` | duration | No | `1h` | Token renewal check interval |

### JWT Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `JWT_PRIVATE_KEY_PATH` | string | No | `secret/data/jwt/private-key` | Vault path to RSA private key |
| `JWT_PUBLIC_KEY_PATH` | string | No | `secret/data/jwt/public-key` | Vault path to RSA public key |
| `JWT_ACCESS_TOKEN_EXPIRY` | duration | No | `60m` | Access token lifetime (recommended: 15m-60m) |
| `JWT_REFRESH_TOKEN_EXPIRY` | duration | No | `720h` (30 days) | Refresh token lifetime |
| `JWT_KEY_ROTATION_INTERVAL` | duration | No | `24h` | How often to check Vault for new keys |

**JWT Key Format**: RSA 2048-bit keys stored in Vault as PEM-encoded strings in the `value` field.

### Security Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `BCRYPT_COST` | int | No | `12` | Bcrypt cost factor (10-14, higher = slower) |
| `MIN_PASSWORD_LENGTH` | int | No | `8` | Minimum password length |
| `REQUIRE_PASSWORD_COMPLEXITY` | bool | No | `true` | Require uppercase, lowercase, digit, special char |
| `RATE_LIMIT_REQUESTS` | int | No | `100` | Max requests per window |
| `RATE_LIMIT_WINDOW` | duration | No | `1m` | Rate limit time window |

**Rate Limiting**: Applied to authentication endpoints (`/auth/*`) to prevent brute force attacks.

### Redis Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `REDIS_HOST` | string | No | - | Redis host |
| `REDIS_PORT` | string | No | `6379` | Redis port |
| `REDIS_PASSWORD` | string | No | - | Redis password |
| `REDIS_DB` | int | No | `0` | Redis database index |

These are parsed into config but nothing reads them yet; rate limiting is in-memory only.

### Logging Configuration

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `LOG_LEVEL` | string | No | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | string | No | `text` | Log format: `text`, `json` |

### Example `.env` File

#### Development

```bash
# User Service Configuration - Development

# Server
PORT=8082
ENVIRONMENT=development
SHUTDOWN_TIMEOUT=30s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=cairn
DB_PASSWORD=cairn_password
DB_NAME=cairn_users
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
DB_CONN_MAX_LIFETIME=5m

# Vault (Dev Mode)
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-token
VAULT_NAMESPACE=

# JWT Configuration
JWT_PRIVATE_KEY_PATH=secret/data/jwt/private-key
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
JWT_ACCESS_TOKEN_EXPIRY=60m
JWT_REFRESH_TOKEN_EXPIRY=720h  # 30 days
JWT_KEY_ROTATION_INTERVAL=24h

# Security
BCRYPT_COST=12
MIN_PASSWORD_LENGTH=8
REQUIRE_PASSWORD_COMPLEXITY=true
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_WINDOW=1m

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
```

#### Production

```bash
# User Service Configuration - Production

# Server
PORT=8082
ENVIRONMENT=production
SHUTDOWN_TIMEOUT=30s

# Database
DB_HOST=postgres.production.internal
DB_PORT=5432
DB_USER=<db-user>
DB_PASSWORD=<db-password>
DB_NAME=cairn_users
DB_SSLMODE=require
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10
DB_CONN_MAX_LIFETIME=5m

# Vault (AppRole Authentication)
VAULT_ADDR=https://vault.production.internal:8200
VAULT_ROLE_ID=<your-role-id>
VAULT_SECRET_ID=<your-secret-id>
VAULT_NAMESPACE=cairn
VAULT_AUTH_PATH=approle
VAULT_TOKEN_RENEWAL_INTERVAL=1h

# JWT Configuration
JWT_PRIVATE_KEY_PATH=secret/data/jwt/private-key
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key
JWT_ACCESS_TOKEN_EXPIRY=15m  # Shorter in production
JWT_REFRESH_TOKEN_EXPIRY=168h  # 7 days in production
JWT_KEY_ROTATION_INTERVAL=24h

# Security
BCRYPT_COST=12
MIN_PASSWORD_LENGTH=12  # Stricter in production
REQUIRE_PASSWORD_COMPLEXITY=true
RATE_LIMIT_REQUESTS=10  # Stricter rate limiting
RATE_LIMIT_WINDOW=1m

# Logging
LOG_LEVEL=info
LOG_FORMAT=json
```

### Vault Setup

#### 1. Generate JWT Key Pair

```bash
# Generate RSA 2048-bit key pair
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem
```

#### 2. Store Keys in Vault

**Development (Token Auth)**:
```bash
export VAULT_ADDR=http://localhost:8200
export VAULT_TOKEN=dev-token

# Store private key
vault kv put secret/jwt/private-key value=@private.pem

# Store public key
vault kv put secret/jwt/public-key value=@public.pem
```

**Production (AppRole Auth)**:
```bash
export VAULT_ADDR=https://vault.production.internal:8200

# Login with AppRole
vault write auth/approle/login \
  role_id="$VAULT_ROLE_ID" \
  secret_id="$VAULT_SECRET_ID"

# Store keys (same commands as dev)
vault kv put secret/jwt/private-key value=@private.pem
vault kv put secret/jwt/public-key value=@public.pem
```

### Security Best Practices

**Production Checklist**:
- ✅ Use `ENVIRONMENT=production`
- ✅ Enable SSL for database (`DB_SSLMODE=require`)
- ✅ Use AppRole authentication for Vault (not token)
- ✅ Shorter access token expiry (15 minutes recommended)
- ✅ Stricter rate limiting (10 requests/minute on auth endpoints)
- ✅ Use `LOG_FORMAT=json` for structured logging
- ✅ Rotate JWT keys periodically (manual or automated)
- ✅ Use strong passwords for database (min 32 characters)

**JWT Key Rotation**:
1. Generate new key pair
2. Store in Vault at the same paths
3. Service automatically detects and loads new keys within `JWT_KEY_ROTATION_INTERVAL`
4. Old tokens remain valid until expiration

### Troubleshooting

**Vault Connection Failed**:
```bash
# Check Vault status
curl $VAULT_ADDR/v1/sys/health

# Verify Vault token (dev)
vault token lookup

# Verify AppRole (production)
vault read auth/approle/role/<role-name>
```

**Database Connection Failed**:
```bash
# Test database connection
psql "host=$DB_HOST port=$DB_PORT dbname=$DB_NAME user=$DB_USER password=$DB_PASSWORD sslmode=$DB_SSLMODE"
```

**Cannot Read JWT Keys from Vault**:
```bash
# Verify keys exist
vault kv get secret/jwt/private-key
vault kv get secret/jwt/public-key

# Check key format (should be PEM)
vault kv get -field=value secret/jwt/private-key | head -1
# Expected: -----BEGIN RSA PRIVATE KEY-----
```

**Rate Limiting Too Aggressive**:
- Increase `RATE_LIMIT_REQUESTS` or `RATE_LIMIT_WINDOW`
- Default: 100 requests per minute (production: 10 requests per minute on auth endpoints)

### Service-Specific Notes

**Account Types**: The service supports three account types:
- **Mobile-only**: Authenticates with Expo device ID only
- **Email-only**: Authenticates with email/password only
- **Hybrid**: Upgraded mobile account, authenticates with email/password (device login disabled)

**Token Management**:
- Refresh tokens are automatically rotated on each use
- Token reuse detection triggers automatic revocation of all user tokens
- Token family tracking prevents replay attacks

**Authorization**: All user-specific endpoints enforce authorization (users can only access their own data).

---

## Read Service Configuration

The Read Service consists of three microservices, each with an API process and (except the Content API) a separate worker process: the Content Service (`services/read/content`), the RSS Fetcher Service (`services/read/fetcher`, `ingest-rss` in compose) and the Email Ingest Service (`services/read/email`). The code is the source of truth; defaults below are the values the code uses when a variable is unset.

All three API processes (and the email worker) share the same database, server and logging variables where noted. The Content and Fetcher worker processes read their own, slightly different, variable set. Only the variables listed here are read; there is no `DATABASE_URL`, no `SERVER_PORT` and no `ENABLE_AUTO_MIGRATIONS` (the Content API always runs its migrations on startup).

### Shared API Variables (Content and Fetcher APIs)

Loaded by `pkg/config`.

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `PORT` | string | No | `8080` (Content), `8081` (Fetcher) | HTTP server port |
| `ENVIRONMENT` | string | No | `development` | `development`, `staging`, `production` |
| `SHUTDOWN_TIMEOUT` | duration | No | `30s` | Graceful shutdown timeout |
| `DB_HOST` | string | No | `localhost` | PostgreSQL host |
| `DB_PORT` | string | No | `5432` | PostgreSQL port |
| `DB_USER` | string | No | `cairn_content` (Content), `cairn_rss` (Fetcher) | Database user |
| `DB_PASSWORD` | string | No | `cairn_content_pass` (Content), `cairn_rss_pass` (Fetcher) | Database password. Must be non-empty, but the built-in default satisfies that, so override it in any real deployment |
| `DB_NAME` | string | No | `cairn_content` (Content), `cairn_rss` (Fetcher) | Database name |
| `DB_SSLMODE` | string | No | `require` | `disable`, `require`, `verify-ca`, `verify-full` |
| `DB_MAX_OPEN_CONNS` | int | No | `25` | Maximum open connections |
| `DB_MAX_IDLE_CONNS` | int | No | `5` | Maximum idle connections |
| `DB_CONN_MAX_LIFETIME` | duration | No | `30m` | Maximum connection lifetime |
| `DB_STATEMENT_TIMEOUT` | duration | No | `30s` | Per-connection statement timeout (`0` disables) |
| `LOG_LEVEL` | string | No | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | string | No | `text` | `text`, `json` |
| `INTERNAL_API_KEY` | string | Yes | - | Shared key for service-to-service calls (Fetcher worker and Content API) |

Duration variables accept Go duration strings (`30s`, `5m`) or a bare number of seconds.

### Content Service Configuration

#### API (`cmd/content`)

In addition to the shared variables above:

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `VAULT_ADDR` | string | Yes | `http://localhost:8200` | Vault address (used to fetch the JWT public key) |
| `VAULT_TOKEN` | string | Yes* | - | Vault token (dev) |
| `VAULT_ROLE_ID` | string | Yes* | - | AppRole role ID (production) |
| `VAULT_SECRET_ID` | string | Yes* | - | AppRole secret ID (production) |
| `VAULT_AUTH_PATH` | string | No | `approle` | AppRole auth mount path |
| `JWT_PUBLIC_KEY_PATH` | string | Yes | `secret/data/jwt/public-key` | Vault path to the RS256 public key |
| `JWT_PUBLIC_KEY_REFRESH_INTERVAL` | duration | No | `5m` | How often the public key is re-fetched (picks up rotation without restart) |
| `INGEST_RSS_SERVICE_URL` | string | No | `http://localhost:8085` | Fetcher API base URL (RSS feed subscribe/unsubscribe) |
| `EMAIL_INGEST_SERVICE_URL` | string | No | `http://localhost:8087` | Email Ingest API base URL |

\* Either `VAULT_TOKEN` or both `VAULT_ROLE_ID` and `VAULT_SECRET_ID` must be set.

#### Worker (`cmd/worker`)

The worker reads its own variables directly (not `pkg/config`). Note `DB_SSL_MODE` here, versus `DB_SSLMODE` on the API.

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DB_HOST` | string | No | `localhost` | PostgreSQL host |
| `DB_PORT` | int | No | `5433` | PostgreSQL port |
| `DB_USER` | string | No | `cairn_content` | Database user |
| `DB_PASSWORD` | string | No | `cairn_content_pass` | Database password |
| `DB_NAME` | string | No | `cairn_content` | Database name |
| `DB_SSL_MODE` | string | No | `disable` | SSL mode |
| `DB_MAX_OPEN_CONNS` | int | No | `5` | Maximum open connections |
| `DB_MAX_IDLE_CONNS` | int | No | `2` | Maximum idle connections |
| `DB_CONN_MAX_LIFETIME` | duration | No | `5m` | Maximum connection lifetime |
| `DB_CONN_MAX_IDLE_TIME` | duration | No | `2m` | Maximum connection idle time |
| `DB_STATEMENT_TIMEOUT` | duration | No | `30s` | Per-connection statement timeout |
| `CLEANUP_CRON` | string | No | `0 2 * * *` | Cron schedule for the cleanup job (2 AM daily) |
| `HEALTH_PORT` | string | No | `8082` | Health check server port |
| `LOG_LEVEL` | string | No | `info` | Log level |
| `LOG_FORMAT` | string | No | `text` | Log format |

The cleanup job deletes non-favorited Feed items 30 days after they were added and orphaned content after 90 days, in batches of 1000 rows. The retention periods and batch size are constants in `content/internal/jobs/cleanup_job.go`, not environment variables.

#### Example `.env` File

```bash
# Content API
PORT=8080
DB_HOST=localhost
DB_PORT=5432
DB_USER=cairn_content
DB_PASSWORD=change-me
DB_NAME=content_service
DB_SSLMODE=disable
VAULT_ADDR=http://localhost:8200
VAULT_TOKEN=dev-token
INGEST_RSS_SERVICE_URL=http://localhost:8085
EMAIL_INGEST_SERVICE_URL=http://localhost:8087
INTERNAL_API_KEY=change-me

# Content worker
CLEANUP_CRON="0 2 * * *"
HEALTH_PORT=8084
```

### RSS Fetcher Service Configuration

#### API (`cmd/ingest_rss`)

Uses only the shared variables above (default port `8081`); `INTERNAL_API_KEY` is required.

#### Worker (`cmd/ingest_rss_worker`)

The worker reads its own variables directly. Note `DB_SSL_MODE` here, versus `DB_SSLMODE` on the API. `INTERNAL_API_KEY` is not validated at startup, but outbox delivery to the Content Service is rejected without it.

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `DB_HOST` | string | No | `localhost` | PostgreSQL host |
| `DB_PORT` | int | No | `5434` | PostgreSQL port |
| `DB_USER` | string | No | `cairn_rss` | Database user |
| `DB_PASSWORD` | string | No | `cairn_rss_pass` | Database password |
| `DB_NAME` | string | No | `cairn_rss_fetcher` | Database name |
| `DB_SSL_MODE` | string | No | `disable` | SSL mode |
| `DB_MAX_OPEN_CONNS` | int | No | `10` | Maximum open connections |
| `DB_MAX_IDLE_CONNS` | int | No | `5` | Maximum idle connections |
| `DB_CONN_MAX_LIFETIME` | duration | No | `5m` | Maximum connection lifetime |
| `DB_CONN_MAX_IDLE_TIME` | duration | No | `2m` | Maximum connection idle time |
| `DB_STATEMENT_TIMEOUT` | duration | No | `30s` | Per-connection statement timeout |
| `CONTENT_SERVICE_URL` | string | No | `http://localhost:8080` | Content Service base URL |
| `CONTENT_SERVICE_TIMEOUT` | duration | No | `30s` | Content Service request timeout |
| `CONTENT_SERVICE_MAX_RETRIES` | int | No | `3` | Content Service request retries |
| `CONTENT_SERVICE_RETRY_DELAY` | duration | No | `1s` | Delay between retries |
| `INTERNAL_API_KEY` | string | No | - | Key sent to the Content Service |
| `FEED_WORKER_COUNT` | int | No | `5` | Feed fetch worker goroutines |
| `FEED_WORKER_QUEUE_SIZE` | int | No | `100` | Feed fetch queue size |
| `POLL_BATCH_SIZE` | int | No | `50` | Feeds claimed per poll |
| `POLL_INTERVAL` | duration | No | `1m` | Interval between polls |
| `POLL_WORKER_POOL_SIZE` | int | No | `5` | Poll scheduler worker pool size |
| `TIER_UPDATE_INTERVAL` | duration | No | `24h` | How often feed polling tiers are recalculated |
| `OUTBOX_CLEANUP_CRON` | string | No | `0 3 * * *` | Cron schedule for outbox cleanup (3 AM daily) |
| `FEED_ITEMS_CLEANUP_CRON` | string | No | `0 4 * * *` | Cron schedule for feed item cleanup (4 AM daily) |
| `HEALTH_PORT` | string | No | `8083` | Health check server port |
| `LOG_LEVEL` | string | No | `info` | Log level |
| `LOG_FORMAT` | string | No | `text` | Log format |

Item processing, content extraction, outbox delivery (retries, batch sizes, circuit breaker) and cleanup retention use the `Default*Config()` values in the fetcher code and are not configurable through the environment.

### Email Ingest Service Configuration

Both the API (`cmd/email_ingest`) and the worker (`cmd/email_ingest_worker`) load the same variables via `internal/config`.

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `PORT` | string | No | `8087` | HTTP server port |
| `DB_HOST` | string | No | `localhost` | PostgreSQL host |
| `DB_PORT` | string | No | `5432` | PostgreSQL port |
| `DB_USER` | string | No | `cairn_user_email` | Database user |
| `DB_PASSWORD` | string | Yes** | - | Database password |
| `DB_NAME` | string | No | `ingest_email` | Database name |
| `DB_SSL_MODE` | string | No | `disable` | SSL mode |
| `DB_STATEMENT_TIMEOUT` | duration | No | `30s` | Per-connection statement timeout |
| `EMAIL_DOMAIN` | string | No | `read.cairnapp.com` | Domain for per-user ingest addresses |
| `INGEST_API_KEY` | string | Yes | - | API key the Cloudflare email worker uses to authenticate |
| `VAULT_ADDR` | string | No | `http://localhost:8200` | Vault address |
| `VAULT_TOKEN` | string | No | - | Vault token (dev) |
| `VAULT_ROLE_ID` | string | No | - | AppRole role ID (production) |
| `VAULT_SECRET_ID` | string | No | - | AppRole secret ID (production) |
| `VAULT_AUTH_PATH` | string | No | `approle` | AppRole auth mount path |
| `JWT_PUBLIC_KEY_PATH` | string | No | `secret/data/jwt` | Vault path to the JWT public key. Deployments set `secret/data/jwt/public-key` |
| `JWT_PUBLIC_KEY_REFRESH_INTERVAL` | duration | No | `5m` | How often the public key is re-fetched |
| `CONTENT_SERVICE_URL` | string | No | `http://localhost:8083` | Content Service base URL (worker delivery) |
| `CONTENT_SERVICE_TIMEOUT` | duration | No | `30s` | Content Service request timeout |
| `EMAIL_PROCESS_WORKERS` | int | No | `3` | Email processing workers |
| `EMAIL_PROCESS_POLL_INTERVAL` | duration | No | `5s` | Email processing poll interval |
| `OUTBOX_WORKER_COUNT` | int | No | `3` | Outbox delivery workers |
| `OUTBOX_POLL_INTERVAL` | duration | No | `10s` | Outbox poll interval |
| `OUTBOX_MAX_RETRIES` | int | No | `6` | Maximum delivery attempts |
| `RAW_EMAIL_CLEANUP_CRON` | string | No | `0 5 * * *` | Cron schedule for raw email cleanup (5 AM daily) |
| `RAW_EMAIL_RETENTION_DAYS` | int | No | `7` | Days to keep raw emails |
| `OUTBOX_CLEANUP_CRON` | string | No | `0 6 * * *` | Cron schedule for outbox cleanup (6 AM daily) |
| `LOG_LEVEL` | string | No | `info` | Log level |
| `LOG_FORMAT` | string | No | `json` | Log format |
| `HEALTH_PORT` | string | No | `8090` | Worker health check port (worker only) |

The shutdown timeout is fixed at 30s. `RAW_EMAIL_RETENTION_DAYS` is the retention used by both the raw email and the outbox cleanup jobs.

---

## Database Configuration

Services connect with discrete `DB_*` variables (see the tables above), not a connection string. Each service owns its own logical database inside one PostgreSQL instance; the user and database names come from `POSTGRES_USER_*` / `POSTGRES_DB_*` in the compose `.env` and are created by `infrastructure/docker/scripts/init-databases.sh`.

### Connection Pool Settings

| Variable | Description |
|----------|-------------|
| `DB_MAX_OPEN_CONNS` | Max concurrent connections |
| `DB_MAX_IDLE_CONNS` | Max idle connections kept open |
| `DB_CONN_MAX_LIFETIME` | Max connection lifetime (duration) |

Sum the `DB_MAX_OPEN_CONNS` of every process and keep it below PostgreSQL's `max_connections`. The production compose file documents its budget in a comment above `cairn-db`.

---

## Environment-Specific Configs

### Development

Use `infrastructure/docker/dev/docker-compose.yml` with `infrastructure/docker/dev/.env.example`. Services run with `ENVIRONMENT=development`, `DB_SSLMODE=disable` and a Vault dev-mode root token.

### Production

Use `infrastructure/docker/prod/docker-compose.yml` with `infrastructure/docker/prod/.env.example`. Services authenticate to Vault with per-service AppRole credentials (`*_ROLE_ID` / `*_SECRET_ID`), run with `ENVIRONMENT=production`, and are reachable only through the Caddy reverse proxy. See [DEPLOYMENT.md](DEPLOYMENT.md).

---

## Security Configuration

### Database Security

- Use strong, unique passwords for each service database user (`openssl rand -base64 32`).
- The shared API loader defaults `DB_SSLMODE` to `require`. Set it to `disable` only when PostgreSQL is on a private network without TLS (as in the bundled compose files), and prefer `require` or stronger otherwise.

### Service-to-Service Keys

- `INTERNAL_API_KEY` authenticates the Fetcher worker to the Content Service. Generate a strong random value.
- `INGEST_API_KEY` (`EMAIL_INGEST_API_KEY` in compose) authenticates the Cloudflare email worker to the Email Ingest API.

---

## Validation & Testing

### Validate Configuration

```bash
# From infrastructure/docker/dev (or prod)
docker compose up -d

# Check logs for configuration errors
docker compose logs content-service | grep -i error
docker compose logs ingest-rss | grep -i error

# Test health endpoints (dev host ports; see infrastructure/docker/README.md)
curl http://localhost:8083/health/ready   # content-service
curl http://localhost:8085/health/ready   # ingest-rss
```

### Common Configuration Errors

**`invalid configuration: ... is required`**: a required variable is missing or empty (`INTERNAL_API_KEY`, `DB_PASSWORD` for the email service, `INGEST_API_KEY`, or Vault credentials for the Content and User services).

**Database connection failed**:
- Check `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- Check `DB_SSLMODE` (API) / `DB_SSL_MODE` (workers and Email Ingest) matches the server
- Verify the database is running and reachable

**Worker setting ignored**: the Content and Fetcher workers read `DB_SSL_MODE`; setting `DB_SSLMODE` on them has no effect.

---

## Configuration Best Practices

1. **Never commit secrets** to version control
2. **Use environment variables** for all sensitive data
3. **Use strong passwords** (min 32 characters, random)
4. **Enable SSL** where the database is not on a private network
5. **Test configuration changes** in a non-production environment first
6. **Rotate credentials** regularly

---

## Additional Resources

- [Deployment Guide](DEPLOYMENT.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)
- [Architecture Documentation](ARCHITECTURE.md)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
