# Cairn Deployment Guide

This guide covers deploying all Cairn backend services to various environments, from local development to production servers.

## Table of Contents

- [Quick Start](#quick-start)
- [Prerequisites](#prerequisites)
- [Development Deployment](#development-deployment)
- [Production Deployment](#production-deployment)
- [Database Setup](#database-setup)
- [Health Checks](#health-checks)
- [Monitoring](#monitoring)
- [Backup & Recovery](#backup--recovery)
- [Scaling](#scaling)
- [Troubleshooting](#troubleshooting)

---

## Quick Start

### Self-Hosted Deployment (Simplest)

Run everything in two containers (app + PostgreSQL) — no Vault, no multi-container orchestration:

```bash
cd infrastructure/docker/selfhost
cp .env.example .env
# Edit .env and set DB_PASSWORD
docker compose up -d --build
curl http://localhost:8099/health/ready
```

All four backend services (users, content, RSS ingest, email ingest) and all background workers run in a single binary. JWT keys are auto-generated. See [infrastructure/docker/selfhost/README.md](../infrastructure/docker/selfhost/README.md) for full details.

### Development Deployment (Recommended)

Use the unified Docker Compose setup for all services:

```bash
cd infrastructure/docker/dev
cp .env.example .env
# Edit .env and set the passwords and keys
docker compose up --build
```

This starts all services with automatic database migrations and Vault initialization.

### Service Endpoints

After deployment, services are available at:

| Service | URL | Purpose |
|---------|-----|---------|
| User Service | http://localhost:8082 | Authentication, user management |
| Content Service | http://localhost:8083 | Article storage, Feed/Reads lists |
| Ingest RSS | http://localhost:8085 | RSS subscriptions and polling |
| Email Ingest | http://localhost:8089 | Newsletter ingestion |
| Vault UI | http://localhost:8200/ui | Secrets management (token: `VAULT_DEV_ROOT_TOKEN_ID` from `.env`) |
| PostgreSQL | localhost:5432 | Database (admin user `POSTGRES_ADMIN_USER`, default `cairn_admin`; per-service users and passwords from `.env`) |

---

## Prerequisites

### Required Software

- **Docker**: 20.10 or later
- **Docker Compose**: 2.0 or later (included with Docker Desktop)
- **Git**: For cloning the repository

### For Local Development

- **Go**: 1.21 or later
- **PostgreSQL**: 15 or later (if running database locally without Docker)
- **Make**: For using Makefile commands (optional)

---

## Development Deployment

### Unified Deployment (All Services)

The recommended approach for development is to use the unified Docker Compose setup:

```bash
cd infrastructure/docker/dev
cp .env.example .env
docker compose up --build
```

This starts:
1. **Vault** (dev mode) - Manages JWT keys
2. **PostgreSQL** (`cairn-db`) - Single instance with 4 databases (names come from `POSTGRES_DB_*` in `.env`; these are the `.env.example` values):
   - `cairn_users` - User accounts
   - `content_service` - Articles, user content, Feed/Reads routes
   - `rss_fetcher_service` - RSS feeds and subscriptions
   - `ingest_email` - Email senders and raw emails
3. **User Service** - Authentication and user management
4. **Content Service** (and worker) - Article storage and cleanup
5. **Ingest RSS** (and worker) - RSS feed polling and delivery
6. **Email Ingest** (and worker) - Newsletter ingestion

### Testing the Deployment

#### 1. Check Service Health

```bash
# All services
docker compose ps

# Individual health endpoints
curl http://localhost:8082/health/ready  # User Service
curl http://localhost:8083/health/ready  # Content Service
curl http://localhost:8085/health/ready  # Ingest RSS
curl http://localhost:8089/health/ready  # Email Ingest
```

### Individual Service Deployment

Each service can be deployed independently using its own Dockerfile and Makefile. All services require PostgreSQL and most require HashiCorp Vault for JWT authentication.

#### Shared Prerequisites

Before deploying any individual service, ensure:

1. **PostgreSQL** is running and accessible
2. **HashiCorp Vault** is running with JWT RSA keys stored (required by User Service, Read Content, and Email Ingest)

```bash
# Verify Vault has JWT keys
vault kv get secret/jwt/public-key
vault kv get secret/jwt/private-key  # Only needed by User Service
```

#### User Service

```bash
cd services/users

# Configure environment (no .env.example ships; a .env file is loaded if present)
# Set DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, VAULT_ADDR, VAULT_TOKEN

# Run database migrations
make migrate-up

# Option A: Run locally
make run

# Option B: Run with Docker
make docker-build
make docker-run
```

- **Port**: `PORT` (default 8080; the dev compose maps host port 8082)
- **Database**: `cairn_users`
- **Vault**: Required (reads RSA private + public keys for JWT signing/verification)
- **Health check**: `curl http://localhost:8080/health/ready` (use the port you configured)

#### Read Service (Content + Ingest RSS + Email Ingest)

The Read service consists of three microservices, each with an API server and background worker:

```bash
cd services/read

# Configure environment
cp .env.example .env
cp content/.env.example content/.env
cp fetcher/.env.example fetcher/.env
# Edit each .env: set database credentials, Vault config, and service URLs
# (the email service has no .env.example; see docs/CONFIGURATION.md)

# Build all services into services/read/bin/
make build-all

# Run services (each in a separate terminal)
./bin/content                # Content API (default port 8080)
./bin/content-worker         # Content background worker
./bin/ingest_rss             # RSS Ingest API (default port 8081)
./bin/ingest_rss_worker      # RSS Ingest background worker
./bin/email_ingest           # Email Ingest API (default port 8087)
./bin/email_ingest_worker    # Email Ingest background worker
```

Ports below are the code defaults (`PORT`); the dev compose maps them to host ports 8083, 8085 and 8089. See [CONFIGURATION.md](CONFIGURATION.md) for every variable.

- **Content Service**: Port 8080, Vault required (JWT public key)
- **Ingest RSS**: Port 8081, communicates with the Content Service via HTTP
- **Email Ingest**: Port 8087, Vault required (JWT public key)
- **Health checks**:
  ```bash
  curl http://localhost:8080/health/live  # Content Service
  curl http://localhost:8081/health/live  # Ingest RSS
  curl http://localhost:8087/health/live  # Email Ingest
  ```

#### Deployment Order

When deploying services individually, start them in this order:

1. **PostgreSQL** and **Vault** (infrastructure)
2. **User Service** (authentication provider, generates JWT tokens)
3. **Read Content Service** (core API)
4. **Read Ingest RSS** (feed ingestion, depends on step 3)
5. **Email Ingest Service** (depends on Content Service)

### Running Services Locally (Without Docker)

#### 1. Install Dependencies

```bash
# Install Go 1.21+
# See: https://golang.org/doc/install

# Install PostgreSQL 15+
# Ubuntu/Debian:
sudo apt-get install postgresql postgresql-contrib

# macOS:
brew install postgresql@15
```

#### 2. Set Up Databases

```bash
# Start PostgreSQL
sudo systemctl start postgresql  # Linux
brew services start postgresql@15  # macOS

# Create databases
sudo -u postgres psql <<EOF
CREATE USER cairn_user WITH PASSWORD 'cairn_dev';
CREATE DATABASE cairn_users OWNER cairn_user;
EOF
```

#### 3. Run Database Migrations

Each API service (User, Content, Ingest RSS, Email Ingest) runs its migrations automatically on startup; there is no setting to turn this off.

For manual migration management:

```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations for each service
cd services/users
make migrate-up

# Read services: make migrate-up-content / migrate-up-fetcher in services/read
# (see "Database migrations" in services/read/CLAUDE.md)
```

#### 4. Run Services

```bash
# Terminal 1: User Service
cd services/users
make run

# Terminal 2+: Read services (see "Read Service" above)
cd services/read
./bin/content
```

---

## Production Deployment

### Single-Server VPS Deployment

This is the recommended approach for production deployments on a single VPS (DigitalOcean, Linode, AWS EC2, etc.).

#### 1. Provision Server

**Recommended Specs**:
- Ubuntu 22.04 LTS
- 4 vCPUs
- 8GB RAM
- 50GB SSD
- SSH access

#### 2. Initial Server Setup

```bash
# SSH into server
ssh root@your-server-ip

# Update system
apt-get update && apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Docker Compose
apt-get install docker-compose-plugin -y

# Create cairn user
useradd -m -s /bin/bash cairn
usermod -aG docker cairn

# Switch to cairn user
su - cairn
```

#### 3. Deploy Application

```bash
# Clone repository
git clone https://github.com/andrew-craig/cairn-reader.git
cd cairn-reader

# Create production environment file
cd infrastructure/docker/prod
cp .env.example .env

# Edit .env
# - Change all default passwords and API keys
# - Set GITHUB_ORG and IMAGE_TAG for the published images
nano .env

# First start: vault-init initialises Vault, creates the JWT keys and writes the
# per-service AppRole credentials to /vault-keys/approle-credentials.env in the
# vault-keys volume. The services cannot authenticate to Vault until you add
# them to .env, so expect them to be unhealthy on this first start.
docker compose up -d

# Copy the AppRole credentials into .env
# (USER_SERVICE_ROLE_ID/SECRET_ID, CONTENT_SERVICE_*, EMAIL_INGEST_*)
docker compose exec vault-unseal cat /vault-keys/approle-credentials.env

# Restart so the services pick up the credentials
nano .env
docker compose up -d

# Verify services are running
docker compose ps
```

The vault-init script also writes the Vault unseal keys to `/vault-keys/UNSEAL_KEYS.txt` on first run. Copy them to secure storage and remove the file (see [infrastructure/docker/README.md](../infrastructure/docker/README.md)).

#### 4. Configure Firewall

```bash
# As root user
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP (redirects to HTTPS)
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# Note: All backend services are internal-only. The only published port is
# Caddy on 127.0.0.1:8099 (loopback), which the host nginx proxies to.
```

#### 5. Configure Domain and TLS

The production compose file runs a **Caddy** reverse proxy that routes API paths to the services and serves the web app. Caddy serves plain HTTP on container port 8080, published only on the host loopback as `127.0.0.1:8099`. **TLS is terminated by a reverse proxy on the host (for example nginx with Let's Encrypt/certbot)**, which must proxy your domain to `http://127.0.0.1:8099` and set `X-Forwarded-Proto: https`. The repository does not ship the host nginx configuration.

The routing configuration is in `infrastructure/docker/prod/Caddyfile`:

| Path | Routed To | Notes |
|------|-----------|-------|
| `/api/v1/auth/*` | User Service | Authentication |
| `/api/v1/user/*` | User Service | Account management |
| `/api/v1/content/*` | Content Service | Article storage |
| `/api/v1/source/email/*` | Email Ingest | Email ingestion |
| `/health/*` | User Service | Health checks |
| everything else | Web app | Static React SPA |

Internal endpoints (`/api/v1/internal/*`, `/api/v1/source/rss/*`) return 404 from the public proxy.

For local testing without a TLS proxy, call `http://127.0.0.1:8099` directly. (`DOMAIN` appears in `.env.example` but the Caddyfile does not read it.)

### Production Docker Compose Configuration

Use the production compose file at `infrastructure/docker/prod/docker-compose.yml` or use environment variables to override defaults.

**Key production settings** (already set in the compose file):

- Every long-running service has `restart: unless-stopped`, and the API and worker services have `/health/ready` healthchecks.
- Services run with `ENVIRONMENT=production` (User Service, Email Ingest) and authenticate to Vault with AppRole credentials.
- `JWT_ACCESS_TOKEN_EXPIRY=15m` and `JWT_REFRESH_TOKEN_EXPIRY=7d` for the User Service.
- Per-process `DB_MAX_OPEN_CONNS` values follow the connection budget documented above `cairn-db` in the compose file.
- The compose file sets no CPU or memory limits; add `deploy.resources.limits` to the services if you need them.

**Environment variables**: copy `infrastructure/docker/prod/.env.example` to `.env` and set strong values for the database passwords (`openssl rand -base64 32`), `INTERNAL_API_KEY`, `EMAIL_INGEST_API_KEY` and the AppRole credentials. See [CONFIGURATION.md](CONFIGURATION.md) for service variables.

---

## Database Setup

### Database Access

```bash
# Connect to PostgreSQL
# (POSTGRES_ADMIN_USER from .env; cairn_admin in .env.example)
docker compose exec cairn-db psql -U cairn_admin -d postgres

# List databases
docker compose exec cairn-db psql -U cairn_admin -d postgres -c '\l'

# Connect to specific database
docker compose exec cairn-db psql -U cairn_admin -d cairn_users
```

### Database SSL Mode

The compose files configure the services with `sslmode=disable` for PostgreSQL connections (the Content and Ingest RSS APIs would otherwise default to `require`). This is safe **within the Docker network** because:

- All containers run on the same private Docker bridge network
- Traffic never leaves the host machine
- PostgreSQL is not exposed on a public port

If you move the database outside the Docker network (e.g., managed PostgreSQL on AWS RDS, DigitalOcean Managed Databases, or a separate server), you **must** enable SSL:

```bash
# In each service's .env file, change:
DB_SSLMODE=disable

# To:
DB_SSLMODE=require    # Encrypt connection, trust server cert
# or
DB_SSLMODE=verify-full  # Encrypt and verify server certificate (recommended for production)
```

The `DB_SSLMODE` (or `DB_SSL_MODE` in the Email Ingest service) environment variable controls this setting in all services.

### Database Contents

**cairn_users:**
- users
- refresh_tokens
- email_verification_tokens

**content_service:**
- contents
- user_contents (each row in the `feed` or `reads` list)
- source_routes (per-user Feed/Reads choice per source)

**rss_fetcher_service:**
- feeds
- feed_subscriptions
- feed_items
- content_outbox

**ingest_email:**
- email_addresses, email_senders, raw_emails, content_outbox, api_keys

### Automatic Migrations

Each API service runs its migrations when it starts. There is no setting to disable this.

### Manual Migration Management

```bash
# User Service
cd services/users
make migrate-up              # Run migrations
make migrate-down            # Rollback
make migrate-version         # Check current version

# Read services (from services/read)
make migrate-up-content
make migrate-up-fetcher
```

### Common Database Commands

```sql
-- List all tables
\dt

-- Describe table structure
\d users

-- Check migration status
SELECT * FROM schema_migrations;

-- Check database size
SELECT pg_size_pretty(pg_database_size('cairn_users'));

-- List active connections
SELECT * FROM pg_stat_activity WHERE datname = 'cairn_users';

-- Vacuum and analyze
VACUUM ANALYZE;
```

---

## Health Checks

All services provide health check endpoints for monitoring and orchestration.

### Endpoints

**Liveness Check** (`/health/live`):
- Checks if the service process is running
- Returns 200 OK if alive
- Use for container/process restart policies

**Readiness Check** (`/health/ready`):
- Checks if service is ready to handle traffic
- Verifies database connection
- Returns 200 OK if ready, 503 if not
- Use for load balancer health checks

### Testing Health Checks

```bash
# User Service
curl http://localhost:8082/health/ready

# Content Service
curl http://localhost:8083/health/ready

# Ingest RSS
curl http://localhost:8085/health/ready
```

### Kubernetes Liveness/Readiness Probes

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health/ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

---

## Monitoring

### Logs

**View logs**:
```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f user-service
docker compose logs -f content-service
docker compose logs -f ingest-rss

# Last 100 lines
docker compose logs --tail=100 user-service

# Since timestamp
docker compose logs --since 2024-01-01T00:00:00 user-service
```

**Log levels**: DEBUG, INFO, WARN, ERROR

**Log format**: Structured logging (see [LOGGING_STRATEGY.md](LOGGING_STRATEGY.md))

### Metrics (Future Enhancement)

Planned Prometheus metrics endpoints:
- `GET /metrics` - Prometheus-format metrics

Metrics to be collected:
- HTTP request duration
- HTTP request count by status code
- Database query duration
- Feed polling success/failure rates
- Circuit breaker state

---

## Backup & Recovery

### Database Backups

#### Automated Backup Script

Create a backup script (for the self-hosted stack use the shipped `infrastructure/docker/selfhost/scripts/backup.sh`). Run it from `infrastructure/docker/prod`, with the admin credentials from `.env`:

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/cairn"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup all databases
docker compose exec -T cairn-db pg_dumpall -U cairn_admin | \
  gzip > $BACKUP_DIR/cairn_all_$DATE.sql.gz

# Or backup individually
docker compose exec -T cairn-db pg_dump -U cairn_admin cairn_users | \
  gzip > $BACKUP_DIR/cairn_users_$DATE.sql.gz

docker compose exec -T cairn-db pg_dump -U cairn_admin content_service | \
  gzip > $BACKUP_DIR/content_service_$DATE.sql.gz

docker compose exec -T cairn-db pg_dump -U cairn_admin rss_fetcher_service | \
  gzip > $BACKUP_DIR/rss_fetcher_service_$DATE.sql.gz

docker compose exec -T cairn-db pg_dump -U cairn_admin ingest_email | \
  gzip > $BACKUP_DIR/ingest_email_$DATE.sql.gz

# Keep only last 30 days of backups
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: $DATE"
```

#### Schedule with Cron

```bash
# Add to crontab
crontab -e

# Add line (run daily at 2 AM):
0 2 * * * /home/cairn/cairn-reader/scripts/backup.sh >> /var/log/cairn-backup.log 2>&1
```

### Restore from Backup

```bash
# Stop services
docker compose down

# Start only database
docker compose up -d cairn-db

# Wait for database to be ready
sleep 10

# Restore all databases
gunzip -c /var/backups/cairn/cairn_all_20240115_020000.sql.gz | \
  docker compose exec -T cairn-db psql -U cairn_admin -d postgres

# Or restore individual database
gunzip -c /var/backups/cairn/cairn_users_20240115_020000.sql.gz | \
  docker compose exec -T cairn-db psql -U cairn_admin -d cairn_users

# Start all services
docker compose up -d
```

---

## Single-Instance Topology

Cairn is currently designed and tested for **single-instance deployment** — one container per service behind a reverse proxy. This is the supported production topology.

### In-Memory Rate Limiter

The User Service (`pkg/middleware/rate_limit.go`) uses an in-memory token bucket rate limiter keyed by client IP. This works correctly only when there is **one running instance** of the service:

- With multiple instances, each instance holds its own independent counter. A client sending N requests round-robined across M instances will be allowed up to N×M requests before any single instance triggers the limit.
- A Redis-backed distributed rate limiter is planned for future multi-instance support.

**Scaling ceiling**: Run a single instance of each API service. Use vertical scaling (more CPU/memory) to handle increased traffic until Redis-based rate limiting is implemented.

### Services that require single-instance coordination

| Service | Reason |
|---------|--------|
| User Service | In-memory rate limiter (auth endpoints) |
| Ingest RSS Worker | In-memory job scheduler — multiple instances would cause duplicate feed fetches |
| Email Ingest Worker | In-memory job scheduler — multiple instances would cause duplicate email processing |

### Future Multi-Instance Support

To run multiple instances of the User Service safely:
1. Replace `pkg/middleware/rate_limit.go` with a Redis-backed implementation (e.g., `go-redis/redis_rate`)
2. Ensure a shared Redis instance is available to all service instances
3. Update the `RateLimit` and `RateLimitWithKey` middleware to use Redis instead of an in-memory map

---

## Scaling

### Horizontal Scaling (Multi-Instance)

> **Note**: See [Single-Instance Topology](#single-instance-topology) above. Horizontal scaling of the User Service requires replacing the in-memory rate limiter with a Redis-backed one first.

**Services that can be scaled horizontally (stateless, no rate limiter)**:
- Content Service API
- Ingest RSS Service API
- Email Ingest Service API

**Services that need coordination before scaling**:
- User Service API (in-memory rate limiter — replace with Redis first)
- Ingest RSS Worker (use distributed locks for multi-instance)
- Email Ingest Worker (use distributed locks for multi-instance)

#### Scaling with Docker Compose

```bash
# Scale Content Service to 3 instances (no rate limiter)
docker compose up -d --scale content-service=3

# Requires load balancer (nginx, HAProxy, etc.)
```

#### Load Balancer Configuration (Nginx)

```nginx
upstream user_service {
    least_conn;
    server user-service-1:8080;
    server user-service-2:8080;
    server user-service-3:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://user_service;
    }
}
```

### Vertical Scaling (Resource Limits)

Edit `docker-compose.yml` to increase resources:

```yaml
services:
  user-service:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G
        reservations:
          cpus: '1'
          memory: 1G
```

### Database Scaling

**Read Replicas**:
- Create PostgreSQL read replicas for read-heavy workloads
- Route read queries to replicas
- Write queries go to primary

**Connection Pooling**:
- Use PgBouncer for connection pooling
- Reduces connection overhead
- Improves performance under high load

---

## Troubleshooting

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed troubleshooting guide.

### Quick Checks

```bash
# Check if services are running
docker compose ps

# Check service logs
docker compose logs user-service
docker compose logs content-service
docker compose logs ingest-rss

# Check database connectivity
docker compose exec cairn-db psql -U cairn_admin -d postgres -c "SELECT 1;"

# Check disk space
df -h

# Check memory usage
free -h

# Check Docker disk usage
docker system df
```

### Common Issues

**Services won't start**:
- Port conflicts (change the host ports in the compose file)
- Docker not running
- Insufficient memory (increase Docker memory limit)

**Database connection errors**:
- Verify database is healthy: `docker compose ps cairn-db`
- Check database logs: `docker compose logs cairn-db`; migration logs are in the service logs
- Verify databases exist: `docker compose exec cairn-db psql -U cairn_admin -d postgres -l`

**Vault errors**:
- Check Vault logs: `docker compose logs vault`
- Verify keys were stored: `docker compose exec vault vault kv get secret/jwt/public-key`

### Reset Everything

```bash
# Nuclear option - removes all data
docker compose down -v
docker compose up --build
```

---

## Managing the Deployment

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f user-service
docker compose logs -f content-service
docker compose logs -f ingest-rss
```

### Restart Services

```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart user-service
```

### Stop Services

```bash
# Stop but keep data
docker compose down

# Stop and remove all data
docker compose down -v
```

### Rebuild After Code Changes

```bash
# Rebuild specific service
docker compose up --build user-service

# Rebuild all services
docker compose up --build
```

---

## Production Considerations

⚠️ **The default setup is for DEVELOPMENT only!**

For production:

### 1. Vault
- Use production Vault with proper storage backend
- Real authentication (not dev token)
- Enable TLS
- Set up unsealing keys

### 2. Database
- Use strong passwords (min 32 characters)
- Enable SSL/TLS when database is outside the Docker network (set `DB_SSLMODE=require` or `verify-full`; see [Database SSL Mode](#database-ssl-mode))
- Set up automated backups
- Consider managed PostgreSQL (AWS RDS, etc.)
- Regular maintenance (VACUUM, ANALYZE)

### 3. Secrets
- Use Docker secrets or Kubernetes secrets
- Never commit passwords to git
- Rotate credentials regularly
- Use environment-specific .env files

### 4. Networking
- Add reverse proxy (nginx/traefik)
- Use SSL certificates (Let's Encrypt)
- Configure firewall rules
- Rate limiting

### 5. Monitoring
- Add Prometheus metrics
- Configure log aggregation (ELK, Datadog)
- Set up alerting (PagerDuty, Opsgenie)
- Health check monitoring

### 6. Scaling
- Multiple service instances
- Load balancing
- Database replication
- Connection pooling (PgBouncer)
- CDN for static assets

---

## Email Ingest API Key Rotation

The Email Ingest Service authenticates the Cloudflare Email Worker via an `X-API-Key` header. Keys are stored as SHA-256 hashes in the `api_keys` table (raw key never persisted).

### Key Storage Schema

| Column | Notes |
|--------|-------|
| `key_name` | Human-readable identifier (e.g., `cloudflare-worker-prod`) |
| `key_hash` | SHA-256 of the raw key |
| `status` | `active` / `revoked` |
| `created_at` | When the key was created |
| `expires_at` | Optional expiry (NULL = no expiry) |
| `last_used_at` | Updated on each successful validation |
| `revoked_at` | When the key was revoked |
| `created_by`, `notes` | Provenance and purpose |

### Key Rotation Tool

The `manage_keys` CLI (`services/read/email/cmd/manage_keys/`) manages keys:

```bash
# Build the tool
cd services/read/email
go build -o bin/manage_keys ./cmd/manage_keys

# List all keys and their status
./bin/manage_keys list

# Create a new key (prints raw key — copy it now, it is not stored)
./bin/manage_keys create --name cloudflare-worker-prod --notes "Cloudflare Email Worker"

# Rotate: creates a new key, revokes the old one atomically
./bin/manage_keys rotate --name cloudflare-worker-prod

# Revoke a key immediately
./bin/manage_keys revoke --name cloudflare-worker-prod
```

The tool requires the same `DB_*` environment variables as the service (`DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, optionally `DB_SSL_MODE`). It is not included in the service container image, so run it from a checkout on a host that can reach the database. The dev compose publishes PostgreSQL on `localhost:5432`; the prod compose does not (uncomment the `ports` line on `cairn-db` temporarily, or tunnel to it).

### Rotation Procedure

1. **Run rotate**: `./bin/manage_keys rotate --name <key-name>`
2. **Copy the new raw key** printed to stdout — it is not stored anywhere.
3. **Update the Cloudflare Email Worker** with the new key.
4. The old key is revoked in the same database transaction as step 1, so there is no window where both are valid simultaneously. No service restart is needed; requests are validated against the `api_keys` table.

> **Note**: In the multi-container deployment the Email Ingest service validates requests only against the `api_keys` table. `INGEST_API_KEY` is still required at startup by the config loader, but is not used for validation. Only the self-hosted binary reads it, upserting it into `api_keys` under the name `selfhost-default`; rotate that key by changing `INGEST_API_KEY` and restarting.

---

## Security Checklist

- [ ] Change default database passwords
- [ ] Use strong, random passwords (min 32 characters)
- [ ] Enable firewall (ufw/iptables)
- [ ] Configure SSL/TLS with Let's Encrypt
- [ ] Disable SSH password authentication (use keys only)
- [ ] Keep system packages updated
- [ ] Regular database backups
- [ ] Monitor logs for suspicious activity
- [ ] Use non-root Docker user
- [ ] Limit resource usage with Docker resource constraints
- [ ] Enable HTTPS only in production
- [ ] Set secure headers (HSTS, CSP, etc.)

---

## Maintenance

### Regular Tasks

**Daily**:
- Check service health
- Review error logs

**Weekly**:
- Review disk usage
- Check backup integrity
- Review database performance

**Monthly**:
- Update system packages
- Review and optimize database indexes
- Test backup restoration
- Review security patches

### Updating Services

```bash
# Pull latest code
git pull origin main

# Rebuild and restart services
docker compose up -d --build

# Check logs for errors
docker compose logs -f
```

---

## Next Steps

After deployment:

1. **Integrate Mobile App**: Configure app to use deployed APIs
2. **Add Content**: Subscribe to RSS feeds, save articles, or forward newsletters to the email ingest address
3. **Monitor**: Check logs and health endpoints
4. **Customize**: Modify environment variables as needed
5. **Set up monitoring**: Add Prometheus and Grafana
6. **Configure backups**: Set up automated database backups

---

## Support

For detailed documentation:
- [infrastructure/docker/README.md](../infrastructure/docker/README.md) - Detailed deployment guide
- [CLAUDE.md](../CLAUDE.md) - Development and architecture guide
- [ARCHITECTURE.md](ARCHITECTURE.md) - System architecture
- [CONFIGURATION.md](CONFIGURATION.md) - Configuration reference
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Troubleshooting guide

For issues or questions:
- GitHub Issues: https://github.com/andrew-craig/cairn-reader/issues
