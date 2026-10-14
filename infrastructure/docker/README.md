# Cairn Docker Deployment

This directory contains the complete Docker setup for all Cairn backend services, organized into separate `dev/` and `prod/` configurations. The single-binary self-hosted stack lives in `selfhost/` and is documented in [selfhost/README.md](selfhost/README.md); the rest of this file covers `dev/` and `prod/` only.

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
│   ├── unseal-vault.sh       # Prod Vault unseal watchdog
│   └── init-databases.sh     # PostgreSQL database creation
├── vault-config/             # Shared Vault configuration
│   ├── vault.hcl             # Vault server config (prod)
│   └── policies/             # Vault ACL policies
├── vault-init/               # Custom image used by the vault-init and vault-unseal containers
│   └── Dockerfile            # hashicorp/vault + openssl/grep, runs as non-root
├── selfhost/                 # Single-binary stack (see selfhost/README.md)
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
- **Caddy**: Reverse proxy that routes API paths and serves the web app, plain HTTP on `127.0.0.1:8099` (TLS is terminated by a host reverse proxy; see [DEPLOYMENT.md](../../docs/DEPLOYMENT.md)) — production only
- **Vault**: HashiCorp Vault for secrets management (dev mode with port 8200 in dev; file-storage server, internal only, in prod)
- **vault-init**: One-shot container that populates Vault with the JWT keys (and, in prod, initialises Vault and creates AppRoles)
- **vault-unseal**: Prod-only watchdog that unseals Vault after every restart

### Backend Services
In production, all services are internal-only (no host ports). Traffic is routed through Caddy.

- **User Service**: Authentication and account management (dev port 8082)
- **Content Service**: Article content storage for Read (dev port 8083)
- **Ingest RSS**: Feed subscription management (dev port 8085)
- **Email Ingest**: Newsletter ingestion API (dev port 8089)
- **Web**: Static web app served by Caddy (prod only, `cairn-web` image)

### Background Workers
- **Content Worker**: Article cleanup jobs (health port 8084)
- **Ingest RSS Worker**: Feed polling and processing (health port 8086)
- **Email Ingest Worker**: Email processing and delivery (health port 8090)

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
┌──────▼────────┐
│ User Service  │
│   :8082       │
└───────┬───────┘
        │
┌──────────────────┐           ┌────────────────────┐
│ Ingest RSS       │──────────▶│  Content Service   │
│   :8085          │           │      :8083         │
└────────┬─────────┘           └──────────┬─────────┘
         │                                │
         └───────────┬────────────────────┘
                     │
              ┌──────▼──────┐
              │  cairn-db   │  (4 logical databases)
              │   :5432     │
              └─────────────┘

Email Ingest (:8089) uses the same Vault and cairn-db pattern, with its own database.

Background Workers:
├── Content Worker (:8084) - Content cleanup
├── Ingest RSS Worker (:8086) - Feed polling
└── Email Ingest Worker (:8090) - Email processing
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
  - Health: http://localhost:8082/health/ready
  - API: http://localhost:8082/api/v1/auth/* and http://localhost:8082/api/v1/user/*

- **Content Service**: http://localhost:8083
  - Health: http://localhost:8083/health/ready
  - API: http://localhost:8083/api/v1/content/*

- **Ingest RSS**: http://localhost:8085
  - Health: http://localhost:8085/health/ready
  - API: http://localhost:8085/api/v1/source/rss/* (service-to-service; not exposed by the prod proxy)

- **Email Ingest**: http://localhost:8089
  - Health: http://localhost:8089/health/ready
  - API: http://localhost:8089/api/v1/source/email/*

- **Content Worker**: http://localhost:8084/health/ready (health check only)

- **Ingest RSS Worker**: http://localhost:8086/health/ready (health check only)

- **Email Ingest Worker**: http://localhost:8090/health/ready (health check only)

- **Vault UI**: http://localhost:8200/ui
  - Token: `VAULT_DEV_ROOT_TOKEN_ID` from your `.env` file

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
- **Public Key**: Used by the other services to validate tokens

Keys are stored at:
- `secret/jwt/private-key`
- `secret/jwt/public-key`

## Database Structure

All services share a single PostgreSQL container (`cairn-db` on port 5432) but each service has its own **logical database** with a dedicated user. An init script (`scripts/init-databases.sh`) creates all databases and users on first startup.

| Logical Database | User | Service |
|---|---|---|
| `cairn_users` | `cairn_user` | User Service |
| `content_service` | `cairn_content` | Content Service |
| `rss_fetcher_service` | `cairn_rss` | Ingest RSS |
| `ingest_email` | `cairn_email` | Email Ingest |

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
- `VAULT_DEV_ROOT_TOKEN_ID` (set in `.env`)
- `VAULT_DEV_LISTEN_ADDRESS=0.0.0.0:8200`

### PostgreSQL (consolidated)
- `POSTGRES_ADMIN_USER=cairn_admin` (superuser for init script)
- `POSTGRES_ADMIN_PASSWORD=...` (superuser password)
- Per-service credentials: `POSTGRES_USER_*`, `POSTGRES_PASSWORD_*`, `POSTGRES_DB_*`

### Services and workers

Per-service variables (`PORT`, `DB_*`, `VAULT_*`, `INTERNAL_API_KEY`, `HEALTH_PORT`, cron schedules and so on) are set in the compose files linked above. See [CONFIGURATION.md](../../docs/CONFIGURATION.md) for what each variable does and its default.

## Viewing Logs

View logs for all services:
```bash
docker compose logs -f
```

View logs for a specific service:
```bash
docker compose logs -f user-service
docker compose logs -f vault
```

## Testing the Setup

### 1. Check Service Health

```bash
# User Service
curl http://localhost:8082/health/ready

# Content Service
curl http://localhost:8083/health/ready

# Ingest RSS Service
curl http://localhost:8085/health/ready
```

### 2. Register a Test User

```bash
curl -X POST http://localhost:8082/api/v1/auth/register \
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
docker compose exec vault-unseal cat /vault-keys/approle-credentials.env

# 5. Update .env with the AppRole credentials
# USER_SERVICE_ROLE_ID=<from step 4>
# USER_SERVICE_SECRET_ID=<from step 4>
# CONTENT_SERVICE_ROLE_ID=<from step 4>
# CONTENT_SERVICE_SECRET_ID=<from step 4>
# EMAIL_INGEST_ROLE_ID=<from step 4>
# EMAIL_INGEST_SECRET_ID=<from step 4>

# 6. Restart services to use AppRole auth
docker compose up -d

# 7. IMPORTANT: Secure the unseal keys
docker compose exec vault-unseal cat /vault-keys/UNSEAL_KEYS.txt
# Copy these keys to a secure location.
```

The `vault-unseal` watchdog reads `/vault-keys/UNSEAL_KEYS.txt` (mounted read-only) to unseal Vault after every restart, so deleting that file means Vault must be unsealed by hand (see below) each time it restarts.

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
│  │  content-service│───▶│  (read public key only)             │ │
│  │  email-ingest   │    │                                     │ │
│  │  (read pub only)│    │                                     │ │
│  └─────────────────┘    └─────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

**Service Permissions:**
- **user-service**: Can read both private and public JWT keys (signs tokens)
- **content-service**, **email-ingest**: Can only read the public JWT key (verify tokens)

### Vault Unseal After Restart

If Vault restarts, the `vault-unseal` watchdog unseals it automatically as long as `/vault-keys/UNSEAL_KEYS.txt` still exists. If you removed that file, unseal manually:

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
   - Caddy serves plain HTTP on `127.0.0.1:8099`; terminate TLS with a host reverse proxy (see [DEPLOYMENT.md](../../docs/DEPLOYMENT.md))
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
   The root token is not stored (see `init-vault-prod.sh`), so this command needs a `VAULT_TOKEN` with permission to write to that path.

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

### vault-init image (`vault-init/Dockerfile`)

Both `init-vault.sh` and `init-vault-prod.sh` need `openssl` (to generate the
RSA key pair for JWT signing) and GNU `grep` (to parse the JSON output of
`vault operator init`). Rather than `apk add`-ing those packages from the
container's entrypoint — which forces the container to run as root and makes
startup depend on Alpine package mirrors — we extend `hashicorp/vault` with a
small custom image that:

- Pre-installs `openssl` and `grep` at build time
- Pre-creates `/vault-keys` owned by the `vault` user
- Drops to the unprivileged `vault` user as the default `USER`

The `vault-init` services in `dev/docker-compose.yml`, `prod/docker-compose.yml`,
and the integration test compose all build this image via a `build:` directive.

> **Upgrading an existing prod deployment:** the `vault-keys` volume created by
> the previous root-owned image is owned by root, which the new non-root
> container cannot write to. Either recreate the volume (`docker compose down
> -v` — destructive, requires re-init) or `chown -R vault:vault` the volume
> contents from a temporary root container before bringing the stack back up.

### unseal-vault.sh (Production)

Run by the `vault-unseal` container. Waits for Vault, unseals it with keys 1-3 from `/vault-keys/UNSEAL_KEYS.txt`, then checks seal status every 30 seconds and re-unseals if needed.

Located at: `scripts/unseal-vault.sh`

### init-databases.sh

Automatically runs on PostgreSQL first startup to:
- Create all four service databases with dedicated users
- Each service gets its own logical database within the single PostgreSQL instance

Located at: `scripts/init-databases.sh`

## Volumes

Persistent data is stored in Docker volumes:

- `cairn_db_data`: Consolidated PostgreSQL data (all service databases)
- `vault-keys`: Init output shared between `vault-init` and `vault-unseal` (dev: init data; prod: unseal keys and AppRole credentials). The JWT keys themselves live in Vault
- `vault-data` (prod): Vault file storage
- `caddy_data`, `caddy_config` (prod): Caddy runtime state

## Networking

All services communicate on the `cairn-network` bridge network.

Internal service communication uses service names as hostnames:
- `vault:8200`
- `cairn-db:5432` (all services connect here, each to its own logical database)
- `user-service:8080`
- `content-service:8080`
- `ingest-rss:8081`
- `email-ingest:8087`
- `web:8080` (prod)

## Support

For issues or questions:
- Check service logs: `docker compose logs [service-name]`
- Review [CLAUDE.md](../../CLAUDE.md) for development guidance
- Check individual service READMEs in `services/*/README.md`
