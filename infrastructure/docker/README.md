# Cairn Docker Deployment

This directory contains the complete Docker setup for all Cairn backend services, organized into separate `dev/` and `prod/` configurations.

## Directory Structure

```
infrastructure/docker/
├── dev/
│   ├── docker-compose.yml    # Development setup (builds from source)
│   ├── .env.example          # Development environment template
│   └── .env                  # Local dev environment (git-ignored)
├── prod/
│   ├── docker-compose.yml    # Production setup (pre-built images)
│   ├── Caddyfile             # Reverse proxy routing & TLS config
│   ├── .env.example          # Production environment template
│   └── .env                  # Local prod environment (git-ignored)
├── scripts/                  # Shared initialization scripts
│   ├── init-vault.sh         # Dev Vault setup
│   ├── init-vault-prod.sh    # Prod Vault setup (AppRole)
│   ├── init-databases.sh     # PostgreSQL database creation
│   └── init-postgres.sh      # Legacy PostgreSQL init
├── vault-config/             # Shared Vault configuration
│   ├── vault.hcl             # Vault server config (prod)
│   └── policies/             # Vault ACL policies
└── README.md
```

## Quick Start

**For local development (build from source):**
```bash
cd infrastructure/docker/dev
cp .env.example .env
# Edit .env with your settings
docker compose up --build -d
```

**For production/staging (use pre-built images):**
```bash
cd infrastructure/docker/prod
cp .env.example .env
# Edit .env with your settings
docker compose up -d
```

## Services

The Docker Compose configuration includes:

### Core Infrastructure
- **Caddy**: Reverse proxy with automatic Let's Encrypt TLS (ports 80, 443) — production only
- **Vault**: HashiCorp Vault for secrets management (dev mode, port 8200)

### Backend Services
In production, all services are internal-only (no host ports). Traffic is routed through Caddy.

- **User Service**: Authentication and account management (dev port 8082)
- **Explore Recommender**: Article storage and recommendations (dev port 8087)
- **Explore Fetcher**: RSS feed fetching for Explore (dev port 8088)
- **Content Service**: Article content storage for Read (dev port 8083)
- **Ingest RSS**: Feed subscription management (dev port 8085)

### Background Workers
- **Content Worker**: Article cleanup jobs (health port 8084)
- **Ingest RSS Worker**: Feed polling and processing (health port 8086)

### Database (PostgreSQL)
- `cairn-db` (port 5432) - Single PostgreSQL instance hosting all service databases as separate logical DBs

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
┌──────────────────┐           ┌────────────────────┐
│ Ingest RSS       │──────────▶│  Content Service   │
│   :8085          │           │      :8083         │
└────────┬─────────┘           └──────────┬─────────┘
         │                                │
         └───────────┬────────────────────┘
                     │
              ┌──────▼──────┐
              │  cairn-db   │  (5 logical databases)
              │   :5432     │
              └─────────────┘

Background Workers:
├── Content Worker (:8084) - Content cleanup
└── Ingest RSS Worker (:8086) - Feed polling
```

## Development Setup

### Prerequisites

- Docker Desktop (or Docker Engine + Docker Compose)
- At least 4GB RAM available for Docker

### Starting All Services

```bash
cd infrastructure/docker/dev
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

All services share a single PostgreSQL container (`cairn-db` on port 5432) but each service has its own **logical database** with a dedicated user. An init script (`scripts/init-databases.sh`) creates all databases and users on first startup.

| Logical Database | User | Service |
|---|---|---|
| `cairn_users` | `cairn_user` | User Service |
| `cairn_recommender` | `cairn_recommender` | Explore Recommender |
| `cairn_fetcher` | `cairn_fetcher` | Explore Fetcher |
| `content_service` | `cairn_content` | Content Service |
| `rss_fetcher_service` | `cairn_rss` | Ingest RSS |

Each service connects to the same host (`cairn-db:5432`) but authenticates with its own user and targets its own database. Schema isolation is fully maintained.

### Resetting Databases

To reset all databases and re-run migrations:

```bash
# From infrastructure/docker/dev or infrastructure/docker/prod
# Stop services and remove volumes
docker compose down -v

# Start fresh
docker compose up --build
```

## Environment Variables

All environment variables are configured in the respective `docker-compose.yml` files ([dev](dev/docker-compose.yml) | [prod](prod/docker-compose.yml)).

Key configurations:

### Vault (Development)
- `VAULT_DEV_ROOT_TOKEN_ID=dev-token`
- `VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200`

### PostgreSQL (consolidated)
- `POSTGRES_ADMIN_USER=cairn_admin` (superuser for init script)
- `POSTGRES_ADMIN_PASSWORD=...` (superuser password)
- Per-service credentials: `POSTGRES_USER_*`, `POSTGRES_PASSWORD_*`, `POSTGRES_DB_*`

### User Service
- `PORT=8080` (internal), exposed as 8082
- `DB_HOST=cairn-db`
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
- `DB_HOST=cairn-db`
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
- `DB_HOST=cairn-db`
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
- `DB_HOST=cairn-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_CONTENT}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_CONTENT}`
- `DB_NAME=${POSTGRES_DB_CONTENT}`
- `DB_SSL_MODE=disable`

### Content Worker (content-worker)
- `DB_HOST=cairn-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_CONTENT}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_CONTENT}`
- `DB_NAME=${POSTGRES_DB_CONTENT}`
- `DB_SSL_MODE=disable`
- `CLEANUP_CRON=0 2 * * *` (runs at 2 AM daily)
- `HEALTH_PORT=8084`

### Ingest RSS Service (ingest-rss)
- `PORT=8081` (internal), exposed as 8085
- `DB_HOST=cairn-db`
- `DB_PORT=5432`
- `DB_USER=${POSTGRES_USER_RSS}`
- `DB_PASSWORD=${POSTGRES_PASSWORD_RSS}`
- `DB_NAME=${POSTGRES_DB_RSS}`
- `DB_SSL_MODE=disable`

### Ingest RSS Worker (ingest-rss-worker)
- `DB_HOST=cairn-db`
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
docker compose ps cairn-db
```

Ensure the init script ran successfully:
```bash
docker compose logs cairn-db
```

Check that all databases were created:
```bash
docker compose exec cairn-db psql -U cairn_admin -l
```

### Port conflicts

If ports are already in use, you can modify them in the respective `docker-compose.yml`:

```yaml
ports:
  - "NEW_PORT:INTERNAL_PORT"
```

## Production Deployment

The production deployment uses `prod/docker-compose.yml` with significant security improvements over the development setup.

### Key Security Features

| Feature | Development | Production |
|---------|-------------|------------|
| Vault Authentication | Root token | AppRole per service |
| Vault Port | Exposed (8200) | Internal only |
| Token Permissions | Unlimited | Scoped policies |
| Unseal Keys | 1 key | 5 keys (3 threshold) |

### First-Time Production Setup

```bash
cd infrastructure/docker/prod

# 1. Copy and configure environment
cp .env.example .env
# Edit .env with secure database passwords

# 2. Start services (first run initializes Vault)
docker compose up -d

# 3. Wait for vault-init to complete
docker compose logs -f vault-init

# 4. Retrieve AppRole credentials from the vault-keys volume
docker compose exec vault cat /vault-keys/approle-credentials.env

# 5. Update .env with the AppRole credentials
# USER_SERVICE_ROLE_ID=<from step 4>
# USER_SERVICE_SECRET_ID=<from step 4>
# EXPLORE_RECOMMENDER_ROLE_ID=<from step 4>
# EXPLORE_RECOMMENDER_SECRET_ID=<from step 4>

# 6. Restart services to use AppRole auth
docker compose up -d

# 7. IMPORTANT: Secure the unseal keys
docker compose exec vault cat /vault-keys/UNSEAL_KEYS.txt
# Copy these keys to a secure location and delete the file:
docker compose exec vault rm /vault-keys/UNSEAL_KEYS.txt
```

### Vault Security Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Vault (internal only)                    │
│  ┌─────────────────┐    ┌─────────────────────────────────────┐ │
│  │   AppRole Auth  │    │           Secrets (KV v2)           │ │
│  │                 │    │  secret/data/jwt/private-key        │ │
│  │  user-service   │───▶│  secret/data/jwt/public-key         │ │
│  │  (read priv+pub)│    │                                     │ │
│  │                 │    │                                     │ │
│  │  explore-recomm │───▶│  (read public key only)             │ │
│  │  (read pub only)│    │                                     │ │
│  └─────────────────┘    └─────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Service Permissions:**
- **user-service**: Can read both private and public JWT keys (signs tokens)
- **explore-recommender**: Can only read the public JWT key (verifies tokens)

### Vault Unseal After Restart

If Vault restarts, it will be sealed. To unseal:

```bash
# Connect to the vault container (from infrastructure/docker/prod)
docker compose exec vault sh

# Unseal with 3 of the 5 keys (from your secure storage)
vault operator unseal <key1>
vault operator unseal <key2>
vault operator unseal <key3>
```

### Additional Production Recommendations

1. **Secure database passwords**:
   - Use strong, unique passwords for each database
   - Store in environment variables or secrets manager
   - Never commit passwords to version control

2. **Enable HTTPS** (external traffic):
   - Add nginx or traefik reverse proxy
   - Use SSL certificates (Let's Encrypt)
   - Internal Docker network traffic can remain HTTP

3. **Set up monitoring**:
   - Add Prometheus metrics
   - Configure log aggregation
   - Set up health check monitoring

4. **Database backups**:
   - Configure automated backups
   - Test restore procedures
   - Consider managed PostgreSQL

5. **Rotate AppRole Secret IDs periodically**:
   ```bash
   # Generate new secret ID for a service (from infrastructure/docker/prod)
   docker compose exec vault vault write -f auth/approle/role/user-service/secret-id
   # Update .env and restart the service
   ```

## Scripts

### init-vault.sh (Development)

Automatically runs on first startup in development mode to:
- Generate RSA key pair (2048-bit)
- Store keys in Vault
- Verify key storage

Located at: `scripts/init-vault.sh`

### init-vault-prod.sh (Production)

Runs on first production startup to securely initialize Vault:
- Initialize Vault with 5 unseal keys (3 threshold)
- Enable KV v2 secrets engine
- Generate and store RSA JWT keys
- Enable AppRole authentication
- Create scoped policies for each service
- Create AppRoles with limited permissions
- Output AppRole credentials for service configuration

**Security features:**
- Root token is NOT stored or logged
- Each service gets its own AppRole with minimal permissions
- Unseal keys are written to a file that should be retrieved and deleted

Located at: `scripts/init-vault-prod.sh`

### init-databases.sh

Automatically runs on PostgreSQL first startup to:
- Create all five service databases with dedicated users
- Each service gets its own logical database within the single PostgreSQL instance

Located at: `scripts/init-databases.sh`

## Volumes

Persistent data is stored in Docker volumes:

- `cairn_db_data`: Consolidated PostgreSQL data (all service databases)
- `vault-keys`: Generated JWT keys (temporary, recreated on volume reset)

## Networking

All services communicate on the `cairn-network` bridge network.

Internal service communication uses service names as hostnames:
- `vault:8200`
- `cairn-db:5432` (all services connect here, each to its own logical database)
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
