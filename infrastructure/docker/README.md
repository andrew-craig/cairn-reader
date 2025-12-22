# Cairn Docker Deployment

This directory contains the complete Docker setup for all Cairn backend services.

## Services

The Docker Compose configuration includes:

- **Vault**: HashiCorp Vault for secrets management (dev mode)
- **User Service**: Authentication and account management (port 8082)
- **Recommender Service**: Article storage and recommendations (port 8081)
- **Fetcher Service**: RSS feed fetching (port 8080)
- **PostgreSQL**: Single database instance with separate databases (port 5432)
  - `cairn_users` - User service database
  - `cairn_recommender` - Recommender service database
  - `cairn_fetcher` - Fetcher service database

## Architecture

```
┌─────────────┐
│   Vault     │ (Secrets Management)
│   :8200     │
└──────┬──────┘
       │
       ├──────────────────────────────┐
       │                              │
┌──────▼──────┐  ┌───────────┐  ┌────▼──────────┐
│User Service │  │  Fetcher  │  │  Recommender  │
│   :8082     │  │   :8080   │──│     :8081     │
└──────┬──────┘  └─────┬─────┘  └───────┬───────┘
       │               │                │
       └───────────────┼────────────────┘
                       │
              ┌────────▼─────────┐
              │   PostgreSQL     │
              │      :5432       │
              ├──────────────────┤
              │ cairn_users      │
              │ cairn_recommender│
              │ cairn_fetcher    │
              └──────────────────┘
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

- **Recommender Service**: http://localhost:8081
  - Health: http://localhost:8081/health
  - API: http://localhost:8081/api/v1/*

- **Fetcher Service**: http://localhost:8080
  - Health: http://localhost:8080/health
  - Manual fetch: http://localhost:8080/fetch

- **Vault UI**: http://localhost:8200/ui
  - Token: `dev-token` (dev mode only)

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

The PostgreSQL instance hosts three separate databases:

### cairn_users
User service database containing:
- `users` - User accounts
- `refresh_tokens` - JWT refresh token management

**Migrations**: `services/users/migrations/`

### cairn_recommender
Recommender service database containing:
- `users` - User activity tracking
- `articles` - RSS articles with metadata
- `user_articles` - User read status
- `votes` - Article voting (upvote/downvote)
- `recommendations` - Recommendation history

**Migrations**: `services/explore/recommender/migrations/`

### cairn_fetcher
Fetcher service database containing:
- `feeds` - RSS feed sources
- `fetch_history` - Fetch attempt tracking

**Migrations**: `services/explore/fetcher/migrations/`

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
- `DB_HOST=postgres`
- `DB_NAME=cairn_users`
- `DB_USER=cairn`
- `DB_PASSWORD=cairn_password`
- `VAULT_ADDR=http://vault:8200`
- `JWT_ACCESS_LIFETIME=15m`
- `JWT_REFRESH_LIFETIME=7d`

### Recommender Service
- `PORT=8081`
- `DB_HOST=postgres`
- `DB_NAME=cairn_recommender`
- `DB_USER=cairn`
- `DB_PASSWORD=cairn_password`
- `VAULT_ADDR=http://vault:8200`
- `ARTICLE_RETENTION_DAYS=90`

### Fetcher Service
- `PORT=8080`
- `DB_HOST=postgres`
- `DB_NAME=cairn_fetcher`
- `DB_USER=cairn`
- `DB_PASSWORD=cairn_password`
- `RECOMMENDER_URL=http://recommender:8081`
- `FETCH_INTERVAL=60` (seconds)
- `FETCH_TIMEOUT=30` (seconds)
- `MAX_FETCH_ERRORS=10`

## Viewing Logs

View logs for all services:
```bash
docker compose logs -f
```

View logs for a specific service:
```bash
docker compose logs -f user-service
docker compose logs -f recommender
docker compose logs -f fetcher
docker compose logs -f vault
```

## Testing the Setup

### 1. Check Service Health

```bash
# User Service
curl http://localhost:8082/health

# Recommender Service
curl http://localhost:8081/health

# Fetcher Service
curl http://localhost:8080/health
```

### 2. Trigger Manual Feed Fetch

```bash
curl -X POST http://localhost:8080/fetch
```

### 3. Get Recommendations

```bash
# Note: This endpoint requires JWT authentication
# You'll need to register a user and get a token first
curl http://localhost:8081/api/v1/recommendations/ \
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

- `postgres_data`: All PostgreSQL data (users, recommender, fetcher databases)
- `vault-keys`: Generated JWT keys (temporary, recreated on volume reset)

## Networking

All services communicate on the `cairn-network` bridge network.

Internal service communication uses service names as hostnames:
- `vault:8200`
- `postgres:5432`
- `recommender:8081`
- `user-service:8080`
- `fetcher:8080`

## Support

For issues or questions:
- Check service logs: `docker compose logs [service-name]`
- Review [CLAUDE.md](../../CLAUDE.md) for development guidance
- Check individual service READMEs in `services/*/README.md`
