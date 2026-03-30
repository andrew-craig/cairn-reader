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

### Development Deployment (Recommended)

Use the unified Docker Compose setup for all services:

```bash
cd infrastructure/docker
docker compose up --build
```

This starts all services with automatic database migrations and Vault initialization.

### Service Endpoints

After deployment, services are available at:

| Service | URL | Purpose |
|---------|-----|---------|
| User Service | http://localhost:8082 | Authentication, user management |
| Recommender | http://localhost:8081 | Article recommendations, voting |
| Fetcher | http://localhost:8080 | RSS feed fetching |
| Vault UI | http://localhost:8200/ui | Secrets management (dev token: `dev-token`) |
| PostgreSQL | localhost:5432 | Database (user: `cairn`, password: `cairn_password`) |

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
cd infrastructure/docker
docker compose up --build
```

This starts:
1. **Vault** (dev mode) - Manages JWT keys
2. **PostgreSQL** - Single instance with 3 databases:
   - `cairn_users` - User accounts
   - `cairn_recommender` - Articles, votes
   - `cairn_fetcher` - RSS feeds
3. **User Service** - Authentication and user management
4. **Recommender Service** - Article recommendations
5. **Fetcher Service** - RSS feed polling

### Testing the Deployment

#### 1. Check Service Health

```bash
# All services
docker compose ps

# Individual health endpoints
curl http://localhost:8082/health  # User Service
curl http://localhost:8081/health  # Recommender
curl http://localhost:8080/health  # Fetcher
```

#### 2. Trigger Manual Feed Fetch

```bash
curl -X POST http://localhost:8080/fetch
```

Expected response:
```json
{"status":"fetch triggered"}
```

#### 3. Check Feed Statistics

```bash
# Check how many feeds are loaded
docker compose exec postgres psql -U cairn -d cairn_fetcher -c \
  "SELECT COUNT(*) as total_feeds FROM feeds;"

# Check how many articles have been fetched
docker compose exec postgres psql -U cairn -d cairn_recommender -c \
  "SELECT COUNT(*) as total_articles FROM articles;"
```

#### 4. Monitor Fetcher Logs

```bash
docker compose logs -f fetcher
```

You should see feeds being fetched every 60 seconds (1 feed per minute).

### Individual Service Deployment (Read Service Example)

> **TODO**: Add deployment instructions for individual services (User Service, Explore Service)

For the Read Service, you can deploy it independently:

```bash
cd services/read
docker compose up -d
```

This starts:
- Content Service (port 8080)
- RSS Fetcher Service (port 8081)
- RSS Fetcher Worker (background)
- PostgreSQL databases (content_service, rss_fetcher_service)

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
CREATE DATABASE cairn_users;
CREATE DATABASE cairn_recommender;
CREATE DATABASE cairn_fetcher;
CREATE USER cairn WITH PASSWORD 'cairn_dev';
GRANT ALL PRIVILEGES ON DATABASE cairn_users TO cairn;
GRANT ALL PRIVILEGES ON DATABASE cairn_recommender TO cairn;
GRANT ALL PRIVILEGES ON DATABASE cairn_fetcher TO cairn;
EOF
```

#### 3. Run Database Migrations

Migrations run automatically on service startup when `ENABLE_AUTO_MIGRATIONS=true`.

For manual migration management:

```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations for each service
cd services/users
make migrate-up

cd services/explore
# (migrations run automatically via docker-compose)
```

#### 4. Run Services

```bash
# Terminal 1: User Service
cd services/users
make run

# Terminal 2: Fetcher Service
cd services/explore
make run-fetcher

# Terminal 3: Recommender Service
cd services/explore
make run-recommender
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
git clone https://github.com/cairn-app/cairn-reader.git
cd cairn

# Create production environment file
cd infrastructure/docker/prod
cp .env.example .env

# Edit .env
# - Change default passwords
# - Set LOG_LEVEL=info
# - Disable auto-migrations (run manually)
nano .env

# Start services
docker compose up -d

# Verify services are running
docker compose ps
```

#### 4. Configure Firewall

```bash
# As root user
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP (redirects to HTTPS)
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 443/udp   # HTTP/3 (QUIC)
sudo ufw enable

# Note: All backend services are internal-only (no host ports exposed).
# Caddy reverse proxy handles TLS and routes traffic to services.
```

#### 5. Configure Domain and TLS

The production Docker Compose includes a **Caddy** reverse proxy that automatically obtains and renews TLS certificates via Let's Encrypt. No separate nginx or certbot installation is needed.

Set your domain in `.env`:

```bash
# Set to your production domain
DOMAIN=api.yourdomain.com
```

Caddy will:
- Automatically obtain a Let's Encrypt TLS certificate on first request
- Redirect HTTP to HTTPS
- Renew certificates automatically before expiry
- Support HTTP/3 (QUIC) out of the box

The routing configuration is in `infrastructure/docker/prod/Caddyfile`:

| Path | Routed To | Notes |
|------|-----------|-------|
| `/api/v1/auth/*` | User Service | Authentication |
| `/api/v1/user/*` | User Service | Account management |
| `/api/v1/explore/*` | Explore Recommender | Recommendations & voting |
| `/api/v1/content/*` | Content Service | Article storage |
| `/api/v1/source/email/*` | Email Ingest | Email ingestion |
| `/health/*` | User Service | Health checks |

Internal endpoints (`/api/v1/internal/*`, `/api/v1/explore/feed/*`, `/api/v1/source/rss/*`) are blocked from public access.

For local testing without TLS, set `DOMAIN=localhost` in `.env`.

### Production Docker Compose Configuration

Use the production compose file at `infrastructure/docker/prod/docker-compose.yml` or use environment variables to override defaults.

**Key production settings**:

```yaml
services:
  postgres:
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

  user-service:
    restart: unless-stopped
    environment:
      LOG_LEVEL: info
      ENABLE_AUTO_MIGRATIONS: false  # Run manually
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8082/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  # Similar for other services...
```

**Environment variables** (`.env.production`):

```bash
# Strong passwords (generate with: openssl rand -base64 32)
POSTGRES_PASSWORD=<strong-random-password>
VAULT_TOKEN=<strong-random-token>

# Production settings
LOG_LEVEL=info
ENABLE_AUTO_MIGRATIONS=false

# Service-specific
JWT_ACCESS_LIFETIME=15m
JWT_REFRESH_LIFETIME=7d
```

---

## Database Setup

### Database Access

```bash
# Connect to PostgreSQL
docker compose exec postgres psql -U cairn

# List databases
docker compose exec postgres psql -U cairn -c '\l'

# Connect to specific database
docker compose exec postgres psql -U cairn -d cairn_users
```

### Database Contents

**cairn_users:**
- users
- refresh_tokens

**cairn_recommender:**
- users (activity tracking)
- articles
- user_articles (read status)
- votes
- recommendations

**cairn_fetcher:**
- feeds
- fetch_history

### Automatic Migrations

Migrations run automatically when services start (if `ENABLE_AUTO_MIGRATIONS=true`).

### Manual Migration Management

```bash
# User Service
cd services/users
make migrate-up              # Run migrations
make migrate-down            # Rollback
make migrate-version         # Check current version

# For other services, migrations run via init scripts
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

**Liveness Check** (`/health/live` or `/health`):
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
curl http://localhost:8082/health

# Recommender Service
curl http://localhost:8081/health

# Fetcher Service
curl http://localhost:8080/health
```

### Kubernetes Liveness/Readiness Probes

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8082
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /health
    port: 8082
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
docker compose logs -f recommender
docker compose logs -f fetcher

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

Create `scripts/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/cairn"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup all databases
docker compose exec -T postgres pg_dumpall -U cairn | \
  gzip > $BACKUP_DIR/cairn_all_$DATE.sql.gz

# Or backup individually
docker compose exec -T postgres pg_dump -U cairn cairn_users | \
  gzip > $BACKUP_DIR/cairn_users_$DATE.sql.gz

docker compose exec -T postgres pg_dump -U cairn cairn_recommender | \
  gzip > $BACKUP_DIR/cairn_recommender_$DATE.sql.gz

docker compose exec -T postgres pg_dump -U cairn cairn_fetcher | \
  gzip > $BACKUP_DIR/cairn_fetcher_$DATE.sql.gz

# Keep only last 30 days of backups
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: $DATE"
```

#### Schedule with Cron

```bash
# Add to crontab
crontab -e

# Add line (run daily at 2 AM):
0 2 * * * /home/cairn/cairn/scripts/backup.sh >> /var/log/cairn-backup.log 2>&1
```

### Restore from Backup

```bash
# Stop services
docker compose down

# Start only database
docker compose up -d postgres

# Wait for database to be ready
sleep 10

# Restore all databases
gunzip -c /var/backups/cairn/cairn_all_20240115_020000.sql.gz | \
  docker compose exec -T postgres psql -U cairn

# Or restore individual database
gunzip -c /var/backups/cairn/cairn_users_20240115_020000.sql.gz | \
  docker compose exec -T postgres psql -U cairn -d cairn_users

# Start all services
docker compose up -d
```

---

## Scaling

### Horizontal Scaling (Multi-Instance)

**Services that can be scaled horizontally**:
- User Service API (stateless)
- Recommender Service API (stateless)
- Fetcher Service API (stateless)

**Services that need coordination**:
- Fetcher Worker (use distributed locks for multi-instance)

#### Scaling with Docker Compose

```bash
# Scale User Service to 3 instances
docker compose up -d --scale user-service=3

# Requires load balancer (nginx, HAProxy, etc.)
```

#### Load Balancer Configuration (Nginx)

```nginx
upstream user_service {
    least_conn;
    server user-service-1:8082;
    server user-service-2:8082;
    server user-service-3:8082;
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
docker compose logs recommender
docker compose logs fetcher

# Check database connectivity
docker compose exec postgres psql -U cairn -c "SELECT 1;"

# Check disk space
df -h

# Check memory usage
free -h

# Check Docker disk usage
docker system df
```

### Common Issues

**Services won't start**:
- Port conflicts (change ports in docker-compose.yml)
- Docker not running
- Insufficient memory (increase Docker memory limit)

**Database connection errors**:
- Verify database is healthy: `docker compose ps postgres`
- Check migration logs: `docker compose logs postgres`
- Verify databases exist: `docker compose exec postgres psql -U cairn -l`

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
docker compose logs -f recommender
docker compose logs -f fetcher
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
- Enable SSL/TLS
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
2. **Add Content**: RSS feeds will auto-sync from Kagi Small Web
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
- GitHub Issues: https://github.com/cairn-app/cairn-reader/issues
