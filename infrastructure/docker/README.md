# Cairn Docker Deployment

This directory contains the complete Docker setup for all Cairn backend services.

## Services

The Docker Compose configuration includes:

### Core Infrastructure
- **Vault**: HashiCorp Vault for secrets management (dev mode, port 8200)

### Backend Services
- **User Service**: Authentication and account management (port 8082)
- **Explore Recommender**: Article storage and recommendations (port 8087)
- **Explore Fetcher**: RSS feed fetching for Explore (port 8088)
- **Content Service**: Article content storage for Read (port 8083)
- **Ingest RSS**: Feed subscription management (port 8085)

### Background Workers
- **Content Worker**: Article cleanup jobs (health port 8084)
- **Ingest RSS Worker**: Feed polling and processing (health port 8086)

### Databases (PostgreSQL)
- `users-db` (port 5432) - User service database
- `recommender-db` (port 5433) - Explore Recommender database
- `fetcher-db` (port 5434) - Explore Fetcher database
- `content-db` (port 5435) - Content service database
- `rss-db` (port 5436) - Ingest RSS database

## Architecture

```
┌─────────────┐
│   Vault     │ (Secrets Management)
│   :8200     │
└──────┬──────┘
       │
       ├───────────────────────────────────────────┐
       │                                           │
┌──────▼────────┐  ┌────────────────┐  ┌──────────▼─────────┐
│ User Service  │  │ Explore Fetcher│  │ Explore Recommender│
│   :8082       │  │     :8088      │─▶│      :8087         │
└───────┬───────┘  └────────┬───────┘  └──────────┬─────────┘
        │                   │                     │
        │                   │                     │
   ┌────▼─────┐      ┌──────▼──────┐      ┌──────▼──────┐
   │users-db  │      │fetcher-db   │      │recommender-db│
   │  :5432   │      │   :5434     │      │    :5433     │
   └──────────┘      └─────────────┘      └──────────────┘

┌──────────────────┐           ┌────────────────────┐
│ Ingest RSS       │──────────▶│  Content Service   │
│   :8085          │           │      :8083         │
└────────┬─────────┘           └──────────┬─────────┘
         │                                │
         │                                │
    ┌────▼────────┐                ┌─────▼──────┐
    │ingest-rss-db│                │content-db  │
    │   :5436     │                │  :5435     │
    └─────────────┘                └────────────┘

Background Workers:
├── Content Worker (:8084) - Content cleanup
└── Ingest RSS Worker (:8086) - Feed polling
```

## Quick Start

### Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose)
- At least 4GB RAM available for Docker

### Starting All Services

From the project root:

```bash
cd infrastructure/docker
docker compose up --build
```

Or run in detached mode:

```bash
docker compose up --build -d
```

### Stopping Services

```bash
docker compose down
```

To also remove volumes (wipes all data):

```bash
docker compose down -v
```

## Service URLs

Once running, services are available at:

- **User Service**: http://localhost:8082
  - Health: http://localhost:8082/health
  - API: http://localhost:8082/auth/* and http://localhost:8082/users/*

- **Explore Recommender**: http://localhost:8087
  - Health: http://localhost:8087/health
  - API: http://localhost:8087/explore/*

- **Explore Fetcher**: http://localhost:8088
  - Health: http://localhost:8088/health
  - Manual fetch: http://localhost:8088/fetch

- **Content Service**: http://localhost:8083
  - Health: http://localhost:8083/health/live
  - API: http://localhost:8083/api/v1/users/{userID}/contents

- **Ingest RSS**: http://localhost:8085
  - Health: http://localhost:8085/health/live
  - API: http://localhost:8085/api/v1/users/{userID}/feeds

- **Content Worker**: http://localhost:8084/health/ready (health check only)

- **Ingest RSS Worker**: http://localhost:8086/health/ready (health check only)

- **Vault UI**: http://localhost:8200/ui
  - Token: Value from `.env` file (default: `dev-root-token` in dev mode)

## First-Time Setup

On first startup, the system automatically:

1. Creates all PostgreSQL databases
2. Runs database migrations
3. Starts Vault in dev mode
4. Generates RSA key pair for JWT signing
5. Stores keys in Vault
6. Starts all services

No manual initialization is required!

## JWT Authentication

JWT keys are automatically generated and stored in Vault on first startup:

- **Private Key**: Used by User Service to sign tokens
- **Public Key**: Used by Recommender Service to validate tokens

Keys are stored at:
- `secret/jwt/private-key`
- `secret/jwt/public-key`

## Database Structure

Each service has its own dedicated PostgreSQL database:

### users-db (cairn_users, port 5432)
User service database containing:
- `users` - User accounts
- `refresh_tokens` - JWT refresh token management

**Migrations**: `services/users/migrations/`

### explore-recommender-db (cairn_recommender, port 5433)
Recommender service database containing:
- `users` - User activity tracking
- `articles` - RSS articles with metadata
- `user_articles` - User read status
- `votes` - Article voting (upvote/downvote)
- `recommendations` - Recommendation history

**Migrations**: `services/explore/recommender/migrations/`

### explore-fetcher-db (cairn_fetcher, port 5434)
Fetcher service database containing:
- `feeds` - RSS feed sources
- `fetch_history` - Fetch attempt tracking

**Migrations**: `services/explore/fetcher/migrations/`

### content-db (content_service, port 5435)
Content service database containing:
- `contents` - Cleaned article content
- `user_contents` - User-specific metadata

**Migrations**: Managed by Content Service

### ingest-rss-db (ingest_rss, port 5436)
Ingest RSS service database containing:
- `feeds` - RSS feed metadata
- `user_feeds` - User subscriptions
- `feed_items` - Pending feed items
- `outbox` - Content delivery queue

**Migrations**: Managed by Ingest RSS Service

### Resetting Databases

To reset all databases and re-run migrations:

```bash
# Stop services and remove volumes
docker compose down -v

# Start fresh
docker compose up --build
```

## Environment Variables

All environment variables are configured in [docker-compose.yml](docker-compose.yml).

Key configurations:

### Vault (Development)
- `VAULT_DEV_ROOT_TOKEN_ID=dev-token`
- `VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200`

### PostgreSQL
- `POSTGRES_USER=cairn`
- `POSTGRES_PASSWORD=cairn_password`
- `POSTGRES_DB=postgres` (default, not used by services)

### User Service
- `PORT=8080` (internal), exposed as 8082
- `DB_HOST=users-db`
- `DB_PORT=5432`
- `DB_NAME=${POSTGRES_DB_USERS}`
- `DB_USER=${POSTGRES_USER_USERS}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_USERS}`
- `VAULT_ADDR=http://vault:8200`
- `VAULT_TOKEN=${VAULT_DEV_ROOT_TOKEN_ID}`
- `JWT_ACCESS_LIFETIME=15m`
- `JWT_REFRESH_LIFETIME=7d`
- `SERVER_ENVIRONMENT=development`

### Explore Recommender (explore-recommender)
- `PORT=8081` (internal), exposed as 8087
- `DB_HOST=explore-recommender-db`
- `DB_PORT=5432`
- `DB_NAME=${POSTGRES_DB_RECOMMENDER}`
- `DB_USER=${POSTGRES_USER_RECOMMENDER}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_RECOMMENDER}`
- `DB_SSLMODE=disable`
- `VAULT_ADDR=http://vault:8200`
- `VAULT_TOKEN=${VAULT_DEV_ROOT_TOKEN_ID}`
- `JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key`
- `ARTICLE_RETENTION_DAYS=90`

### Explore Fetcher (explore-fetcher)
- `PORT=8080` (internal), exposed as 8088
- `DB_HOST=explore-fetcher-db`
- `DB_PORT=5432`
- `DB_NAME=${POSTGRES_DB_FETCHER}`
- `DB_USER=${POSTGRES_USER_FETCHER}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_FETCHER}`
- `DB_SSLMODE=disable`
- `RECOMMENDER_URL=http://explore-recommender:8081`
- `FETCH_INTERVAL=60` (seconds)
- `FETCH_TIMEOUT=30` (seconds)
- `MAX_FETCH_ERRORS=10`
- `KAGI_FEED_URL=https://raw.githubusercontent.com/kagisearch/smallweb/main/smallweb.txt`

### Content Service (content-service)
- `PORT=8080` (internal), exposed as 8083
- `DB_HOST=content-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_CONTENT}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_CONTENT}`
- `DB_NAME=${POSTGRES_DB_CONTENT}`
- `DB_SSL_MODE=disable`

### Content Worker (content-worker)
- `DB_HOST=content-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_CONTENT}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_CONTENT}`
- `DB_NAME=${POSTGRES_DB_CONTENT}`
- `DB_SSL_MODE=disable`
- `CLEANUP_CRON=0 2 * * *` (runs at 2 AM daily)
- `HEALTH_PORT=8084`

### Ingest RSS Service (ingest-rss)
- `PORT=8081` (internal), exposed as 8085
- `DB_HOST=ingest-rss-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_RSS}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_RSS}`
- `DB_NAME=${POSTGRES_DB_RSS}`
- `DB_SSL_MODE=disable`

### Ingest RSS Worker (ingest-rss-worker)
- `DB_HOST=ingest-rss-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_RSS}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_RSS}`
- `DB_NAME=${POSTGRES_DB_RSS}`
- `DB_SSL_MODE=disable`
- `CONTENT_SERVICE_URL=http://content-service:8080`
- `HEALTH_PORT=8086`
- `OUTBOX_CLEANUP_CRON=0 3 * * *` (runs at 3 AM daily)
- `FEED_ITEMS_CLEANUP_CRON=0 4 * * *` (runs at 4 AM daily)

## Viewing Logs

View logs for all services:
```bash
docker compose logs -f
```

View logs for a specific service:
```bash
docker compose logs -f user-service
docker compose logs -f explore-recommender
docker compose logs -f explore-fetcher
docker compose logs -f vault
```

## Testing the Setup

### 1. Check Service Health

```bash
# User Service
curl http://localhost:8082/health

# Explore Recommender
curl http://localhost:8087/health

# Explore Fetcher
curl http://localhost:8088/health

# Content Service
curl http://localhost:8083/health/live

# Ingest RSS Service
curl http://localhost:8085/health/live
```

### 2. Trigger Manual Feed Fetch

```bash
curl -X POST http://localhost:8088/fetch
```

### 3. Get Recommendations

```bash
# Note: This endpoint requires JWT authentication
# You'll need to register a user and get a token first
curl http://localhost:8087/api/v1/recommendations/ \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### 4. Register a Test User

```bash
curl -X POST http://localhost:8082/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "TestPassword123!"
  }'
```

## Troubleshooting

### Services fail to start

Check dependencies are healthy:
```bash
docker compose ps
```

All services should show `healthy` status.

### "Failed to connect to Vault"

Ensure Vault is running:
```bash
docker compose logs vault
docker compose logs vault-init
```

Vault should be initialized and keys stored before other services start.

### Database connection errors

Check database health:
```bash
docker compose ps postgres
```

Ensure migrations ran successfully:
```bash
docker compose logs postgres
```

Check that all three databases were created:
```bash
docker compose exec postgres psql -U cairn -l
```

### Port conflicts

If ports are already in use, you can modify them in [docker-compose.yml](docker-compose.yml):

```yaml
ports:
  - "NEW_PORT:INTERNAL_PORT"
```

## Production Deployment

⚠️ **This setup is for DEVELOPMENT only!**

For production deployment:

1. **Use production Vault** (not dev mode):
   - Deploy Vault with proper storage backend
   - Use real authentication (not dev token)
   - Enable TLS/SSL
   - Use unsealing keys

2. **Secure database passwords**:
   - Use strong passwords
   - Store in environment variables or secrets manager
   - Don't commit passwords to version control

3. **Enable HTTPS**:
   - Add reverse proxy (nginx/traefik)
   - Use SSL certificates (Let's Encrypt)

4. **Use environment-specific configs**:
   - Separate `.env` files for prod/staging/dev
   - Use Docker secrets or Kubernetes secrets

5. **Set up monitoring**:
   - Add Prometheus metrics
   - Configure log aggregation
   - Set up health check monitoring

6. **Database backups**:
   - Configure automated backups
   - Test restore procedures
   - Use managed PostgreSQL if possible

## Scripts

### init-vault.sh

Automatically runs on first startup to:
- Generate RSA key pair (2048-bit)
- Store keys in Vault
- Verify key storage

Located at: `infrastructure/docker/scripts/init-vault.sh`

### init-postgres.sh

Automatically runs on PostgreSQL first startup to:
- Create three separate databases (cairn_users, cairn_recommender, cairn_fetcher)
- Run all migrations for each service

Located at: `infrastructure/docker/scripts/init-postgres.sh`

## Volumes

Persistent data is stored in Docker volumes:

- `users_db_data`: User service PostgreSQL data
- `explore_recommender_db_data`: Explore recommender PostgreSQL data
- `explore_fetcher_db_data`: Explore fetcher PostgreSQL data
- `content_db_data`: Content service PostgreSQL data
- `ingest_rss_db_data`: Ingest RSS service PostgreSQL data
- `vault-keys`: Generated JWT keys (temporary, recreated on volume reset)

## Networking

All services communicate on the `cairn-network` bridge network.

Internal service communication uses service names as hostnames:
- `vault:8200`
- `users-db:5432`
- `explore-recommender-db:5432`
- `explore-fetcher-db:5432`
- `content-db:5432`
- `ingest-rss-db:5432`
- `user-service:8080`
- `explore-recommender:8081`
- `explore-fetcher:8080`
- `content-service:8080`
- `ingest-rss:8081`

## Support

For issues or questions:
- Check service logs: `docker compose logs [service-name]`
- Review [CLAUDE.md](../../CLAUDE.md) for development guidance
- Check individual service READMEs in `services/*/README.md`
