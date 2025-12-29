# Deployment Guide

This guide covers deploying Cairn Backend to various environments, from local development to production servers.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Local Development](#local-development)
- [Docker Compose Deployment](#docker-compose-deployment)
- [Production Deployment](#production-deployment)
- [Database Setup](#database-setup)
- [Environment Configuration](#environment-configuration)
- [Health Checks](#health-checks)
- [Monitoring](#monitoring)
- [Backup & Recovery](#backup--recovery)
- [Scaling](#scaling)

## Prerequisites

### Required Software

- **Docker**: 20.10 or later
- **Docker Compose**: 2.0 or later (included with Docker Desktop)
- **Git**: For cloning the repository

### For Local Development

- **Go**: 1.21 or later
- **PostgreSQL**: 15 or later (if running database locally)
- **Make**: For using Makefile commands (optional)

### System Requirements

**Minimum**:
- CPU: 2 cores
- RAM: 4GB
- Disk: 20GB
- OS: Linux, macOS, or Windows with WSL2

**Recommended for Production**:
- CPU: 4 cores
- RAM: 8GB
- Disk: 50GB SSD
- OS: Ubuntu 22.04 LTS or similar

## Local Development

### Option 1: Run Services Locally (Without Docker)

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
CREATE DATABASE content_service;
CREATE DATABASE rss_fetcher_service;
CREATE USER cairn WITH PASSWORD 'cairn_dev';
GRANT ALL PRIVILEGES ON DATABASE content_service TO cairn;
GRANT ALL PRIVILEGES ON DATABASE rss_fetcher_service TO cairn;
EOF
```

#### 3. Run Database Migrations

```bash
# Install migrate CLI
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations for Content Service
migrate -path services/content-service/migrations \
  -database "postgres://cairn:cairn_dev@localhost:5432/content_service?sslmode=disable" \
  up

# Run migrations for RSS Fetcher Service
migrate -path services/rss-fetcher-service/migrations \
  -database "postgres://cairn:cairn_dev@localhost:5433/rss_fetcher_service?sslmode=disable" \
  up
```

#### 4. Configure Environment Variables

```bash
# Content Service
cd services/content-service
cp .env.example .env
# Edit .env with your configuration

# RSS Fetcher Service
cd services/rss-fetcher-service
cp .env.example .env
# Edit .env with your configuration
```

#### 5. Run Services

```bash
# Terminal 1: Run Content Service
cd services/content-service
go run ./cmd/server

# Terminal 2: Run RSS Fetcher Service
cd services/rss-fetcher-service
go run ./cmd/server

# Terminal 3: Run RSS Fetcher Workers
cd services/rss-fetcher-service
go run ./cmd/worker
```

### Option 2: Run Services with Docker Compose

See [Docker Compose Deployment](#docker-compose-deployment) below.

## Docker Compose Deployment

Docker Compose is the recommended deployment method for single-server setups, including development, staging, and small production environments.

### Quick Start

```bash
# Clone repository
git clone https://github.com/andrew-craig/cairn-read.git
cd cairn-read

# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Check service health
curl http://localhost:8080/health/ready  # Content Service
curl http://localhost:8081/health/ready  # RSS Fetcher Service
```

### Services Started

The `docker-compose.yml` file starts:

1. **postgres-content**: PostgreSQL database for Content Service (port 5432)
2. **postgres-fetcher**: PostgreSQL database for RSS Fetcher Service (port 5433)
3. **content-service**: Content Service API (port 8080)
4. **rss-fetcher-service**: RSS Fetcher Service API (port 8081)
5. **rss-fetcher-worker**: Background workers for feed polling and content delivery

### Useful Docker Compose Commands

```bash
# Start services in background
docker-compose up -d

# Start services with build (after code changes)
docker-compose up -d --build

# View logs
docker-compose logs -f

# View logs for specific service
docker-compose logs -f content-service

# Stop services
docker-compose stop

# Stop and remove containers
docker-compose down

# Stop and remove containers + volumes (WARNING: deletes data)
docker-compose down -v

# Restart a specific service
docker-compose restart content-service

# Execute command in running container
docker-compose exec content-service sh

# View running containers
docker-compose ps

# View resource usage
docker stats
```

### Accessing Services

**APIs**:
- Content Service: http://localhost:8080
- RSS Fetcher Service: http://localhost:8081

**Databases**:
```bash
# Content Service database
docker-compose exec postgres-content psql -U cairn -d content_service

# RSS Fetcher database
docker-compose exec postgres-fetcher psql -U cairn -d rss_fetcher_service
```

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
git clone https://github.com/andrew-craig/cairn-read.git
cd cairn-read

# Create production environment file
cp docker-compose.yml docker-compose.prod.yml

# Edit docker-compose.prod.yml
# - Change default passwords
# - Set production environment variables
# - Add restart policies
# - Configure resource limits

# Start services
docker-compose -f docker-compose.prod.yml up -d

# Verify services are running
docker-compose -f docker-compose.prod.yml ps
```

#### 4. Configure Firewall

```bash
# As root user
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable

# Note: Services run on 8080/8081 internally
# Use a reverse proxy (nginx) to expose on 80/443
```

#### 5. Set Up Reverse Proxy (Nginx)

```bash
# Install nginx
sudo apt-get install nginx -y

# Create nginx config
sudo nano /etc/nginx/sites-available/cairn

# Add configuration:
```

```nginx
upstream content_service {
    server localhost:8080;
}

upstream rss_fetcher_service {
    server localhost:8081;
}

server {
    listen 80;
    server_name api.yourdomain.com;

    # Content Service
    location /api/v1/contents {
        proxy_pass http://content_service;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api/v1/users {
        proxy_pass http://content_service;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # RSS Fetcher Service
    location /api/v1/feeds {
        proxy_pass http://rss_fetcher_service;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # Health checks
    location /health {
        proxy_pass http://content_service;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/cairn /etc/nginx/sites-enabled/

# Test configuration
sudo nginx -t

# Reload nginx
sudo systemctl reload nginx
```

#### 6. Configure SSL with Let's Encrypt

```bash
# Install certbot
sudo apt-get install certbot python3-certbot-nginx -y

# Obtain SSL certificate
sudo certbot --nginx -d api.yourdomain.com

# Auto-renewal is configured automatically
# Test renewal:
sudo certbot renew --dry-run
```

### Production Docker Compose Configuration

Create `docker-compose.prod.yml`:

```yaml
version: '3.8'

services:
  postgres-content:
    image: postgres:15-alpine
    container_name: postgres-content
    environment:
      POSTGRES_USER: cairn
      POSTGRES_PASSWORD: ${POSTGRES_CONTENT_PASSWORD}  # Set in .env
      POSTGRES_DB: content_service
    volumes:
      - postgres-content-data:/var/lib/postgresql/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cairn"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

  postgres-fetcher:
    image: postgres:15-alpine
    container_name: postgres-fetcher
    environment:
      POSTGRES_USER: cairn
      POSTGRES_PASSWORD: ${POSTGRES_FETCHER_PASSWORD}  # Set in .env
      POSTGRES_DB: rss_fetcher_service
    volumes:
      - postgres-fetcher-data:/var/lib/postgresql/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U cairn"]
      interval: 10s
      timeout: 5s
      retries: 5
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

  content-service:
    build:
      context: ./services/content-service
      dockerfile: Dockerfile
    container_name: content-service
    environment:
      DATABASE_URL: postgres://cairn:${POSTGRES_CONTENT_PASSWORD}@postgres-content:5432/content_service?sslmode=disable
      SERVER_PORT: 8080
      LOG_LEVEL: info
    ports:
      - "8080:8080"
    depends_on:
      postgres-content:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health/ready"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G

  rss-fetcher-service:
    build:
      context: ./services/rss-fetcher-service
      dockerfile: Dockerfile
    container_name: rss-fetcher-service
    environment:
      DATABASE_URL: postgres://cairn:${POSTGRES_FETCHER_PASSWORD}@postgres-fetcher:5432/rss_fetcher_service?sslmode=disable
      SERVER_PORT: 8081
      CONTENT_SERVICE_URL: http://content-service:8080
      LOG_LEVEL: info
    ports:
      - "8081:8081"
    depends_on:
      postgres-fetcher:
        condition: service_healthy
      content-service:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health/ready"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
    deploy:
      resources:
        limits:
          cpus: '1'
          memory: 1G

  rss-fetcher-worker:
    build:
      context: ./services/rss-fetcher-service
      dockerfile: Dockerfile
      target: worker
    container_name: rss-fetcher-worker
    environment:
      DATABASE_URL: postgres://cairn:${POSTGRES_FETCHER_PASSWORD}@postgres-fetcher:5432/rss_fetcher_service?sslmode=disable
      CONTENT_SERVICE_URL: http://content-service:8080
      LOG_LEVEL: info
    depends_on:
      postgres-fetcher:
        condition: service_healthy
      content-service:
        condition: service_healthy
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 2G

volumes:
  postgres-content-data:
  postgres-fetcher-data:
```

Create `.env` file:

```bash
POSTGRES_CONTENT_PASSWORD=<strong-random-password>
POSTGRES_FETCHER_PASSWORD=<strong-random-password>
```

## Database Setup

### Automatic Migrations

Migrations run automatically when services start. The application uses `golang-migrate` to manage database schema.

### Manual Migration Management

```bash
# Check current migration version
make migrate-status

# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Create new migration
make migrate-create name=add_new_feature

# This creates:
# services/content-service/migrations/000XXX_add_new_feature.up.sql
# services/content-service/migrations/000XXX_add_new_feature.down.sql
```

### Migration Files Location

- Content Service: `services/content-service/migrations/`
- RSS Fetcher Service: `services/rss-fetcher-service/migrations/`

### Database Access

```bash
# Access Content Service database
docker-compose exec postgres-content psql -U cairn -d content_service

# Access RSS Fetcher database
docker-compose exec postgres-fetcher psql -U cairn -d rss_fetcher_service

# Run SQL file
docker-compose exec -T postgres-content psql -U cairn -d content_service < backup.sql
```

### Common Database Commands

```sql
-- List all tables
\dt

-- Describe table structure
\d contents

-- Check migration status
SELECT * FROM schema_migrations;

-- Check database size
SELECT pg_size_pretty(pg_database_size('content_service'));

-- List active connections
SELECT * FROM pg_stat_activity WHERE datname = 'content_service';

-- Vacuum and analyze
VACUUM ANALYZE;
```

## Environment Configuration

See [READ_SERVICE_CONFIGURATION.md](READ_SERVICE_CONFIGURATION.md) for complete configuration reference.

### Essential Environment Variables

**Content Service** (`.env` or docker-compose environment):

```bash
DATABASE_URL=postgres://user:pass@host:5432/content_service?sslmode=disable
SERVER_PORT=8080
LOG_LEVEL=info
MAX_CONTENT_SIZE_MB=5
```

**RSS Fetcher Service**:

```bash
DATABASE_URL=postgres://user:pass@host:5432/rss_fetcher_service?sslmode=disable
SERVER_PORT=8081
CONTENT_SERVICE_URL=http://content-service:8080
LOG_LEVEL=info
WORKER_CONCURRENCY=5
FEED_FETCH_TIMEOUT_SECONDS=30
```

## Health Checks

Both services provide health check endpoints for monitoring and orchestration.

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
# Content Service
curl http://localhost:8080/health/live
curl http://localhost:8080/health/ready

# RSS Fetcher Service
curl http://localhost:8081/health/live
curl http://localhost:8081/health/ready
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

## Monitoring

### Logs

**View logs**:
```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f content-service

# Last 100 lines
docker-compose logs --tail=100 content-service

# Since timestamp
docker-compose logs --since 2025-01-01T00:00:00 content-service
```

**Log levels**: DEBUG, INFO, WARN, ERROR

**Log format**: JSON structured logging (future)

### Metrics (Future Enhancement)

Planned Prometheus metrics endpoints:
- `GET /metrics` - Prometheus-format metrics

Metrics to be collected:
- HTTP request duration
- HTTP request count by status code
- Database query duration
- Feed polling success/failure rates
- Outbox queue depth
- Circuit breaker state

## Backup & Recovery

### Database Backups

#### Automated Backup Script

Create `scripts/backup.sh`:

```bash
#!/bin/bash
BACKUP_DIR="/var/backups/cairn"
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup Content Service database
docker-compose exec -T postgres-content pg_dump -U cairn content_service | \
  gzip > $BACKUP_DIR/content_service_$DATE.sql.gz

# Backup RSS Fetcher database
docker-compose exec -T postgres-fetcher pg_dump -U cairn rss_fetcher_service | \
  gzip > $BACKUP_DIR/rss_fetcher_service_$DATE.sql.gz

# Keep only last 30 days of backups
find $BACKUP_DIR -name "*.sql.gz" -mtime +30 -delete

echo "Backup completed: $DATE"
```

#### Schedule with Cron

```bash
# Add to crontab
crontab -e

# Add line (run daily at 2 AM):
0 2 * * * /home/cairn/cairn-read/scripts/backup.sh >> /var/log/cairn-backup.log 2>&1
```

### Restore from Backup

```bash
# Stop services
docker-compose down

# Start only databases
docker-compose up -d postgres-content postgres-fetcher

# Wait for databases to be ready
sleep 10

# Restore Content Service database
gunzip -c /var/backups/cairn/content_service_20250115_020000.sql.gz | \
  docker-compose exec -T postgres-content psql -U cairn -d content_service

# Restore RSS Fetcher database
gunzip -c /var/backups/cairn/rss_fetcher_service_20250115_020000.sql.gz | \
  docker-compose exec -T postgres-fetcher psql -U cairn -d rss_fetcher_service

# Start all services
docker-compose up -d
```

## Scaling

### Horizontal Scaling (Multi-Instance)

**Services that can be scaled horizontally**:
- Content Service API (stateless)
- RSS Fetcher Service API (stateless)

**Services that need coordination**:
- RSS Fetcher Worker (use distributed locks)

#### Scaling with Docker Compose

```bash
# Scale Content Service to 3 instances
docker-compose up -d --scale content-service=3

# Requires load balancer (nginx, HAProxy, etc.)
```

#### Load Balancer Configuration (Nginx)

```nginx
upstream content_service {
    least_conn;
    server content-service-1:8080;
    server content-service-2:8080;
    server content-service-3:8080;
}

server {
    listen 80;
    location / {
        proxy_pass http://content_service;
    }
}
```

### Vertical Scaling (Resource Limits)

Edit `docker-compose.yml` to increase resources:

```yaml
services:
  content-service:
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

## Troubleshooting

See [READ_SERVICE_TROUBLESHOOTING.md](READ_SERVICE_TROUBLESHOOTING.md) for detailed troubleshooting guide.

### Quick Checks

```bash
# Check if services are running
docker-compose ps

# Check service logs
docker-compose logs content-service

# Check database connectivity
docker-compose exec postgres-content psql -U cairn -d content_service -c "SELECT 1;"

# Check disk space
df -h

# Check memory usage
free -h

# Check Docker disk usage
docker system df
```

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
docker-compose up -d --build

# Check logs for errors
docker-compose logs -f
```

## Support

For issues or questions:
- GitHub Issues: https://github.com/andrew-craig/cairn-read/issues
- Documentation: `/docs` directory
- Architecture: [READ_SERVICE_ARCHITECTURE.md](READ_SERVICE_ARCHITECTURE.md)
- Configuration: [READ_SERVICE_CONFIGURATION.md](READ_SERVICE_CONFIGURATION.md)
- Troubleshooting: [READ_SERVICE_TROUBLESHOOTING.md](READ_SERVICE_TROUBLESHOOTING.md)
