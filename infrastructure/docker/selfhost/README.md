# Cairn Self-Hosted

Run all Cairn backend services in a single container alongside PostgreSQL. No Vault, no multi-container orchestration — just two containers.

## Quick Start

```bash
cd infrastructure/docker/selfhost

# 1. Configure
cp .env.example .env
# Edit .env and set DB_PASSWORD to something secure

# 2. Start
docker compose up -d --build

# 3. Verify
curl http://localhost:8099/health/ready
```

## What's Included

The single `cairn` container runs:

- **User Service** — registration, login, JWT authentication
- **Content Service** — article storage and management
- **RSS Ingest** — feed subscriptions, polling, and content extraction
- **Email Ingest** — email-to-article pipeline
- **Explore Fetcher** — RSS feed discovery
- **Explore Recommender** — article recommendations

All background workers (feed polling, outbox delivery, cleanup jobs) run as goroutines within the same process.

## Architecture

```
┌─────────────────┐     ┌──────────────────┐
│  cairn (:8099)  │────▶│  cairn-db (:5432) │
│                 │     │  PostgreSQL       │
│  All 6 services │     │  6 logical DBs    │
│  All workers    │     └──────────────────┘
│  Single binary  │
└─────────────────┘
```

JWT keys are auto-generated on first start and persisted in the `cairn_data` volume.

## Configuration

All settings go in `.env`. Only `DB_PASSWORD` is required. See `.env.example` for the full list.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_PASSWORD` | *(required)* | PostgreSQL password |
| `DB_USER` | `cairn` | PostgreSQL user |
| `PORT` | `8099` | HTTP port |
| `EMAIL_DOMAIN` | `read.cairnapp.com` | Domain for email ingestion |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `ARTICLE_RETENTION_DAYS` | `90` | Explore article retention |

### PostgreSQL tuning

The `cairn-db` container ships with lightweight defaults suited to a small
host (e.g. a 1 GB VPS or Raspberry Pi). Override any of these via `.env`:

| Variable | Default | Description |
|----------|---------|-------------|
| `POSTGRES_MAX_CONNECTIONS` | `20` | Max concurrent backend connections |
| `POSTGRES_SHARED_BUFFERS` | `16MB` | Shared buffer cache |
| `POSTGRES_WORK_MEM` | `512kB` | Per-operation sort/hash memory |
| `POSTGRES_MAINTENANCE_WORK_MEM` | `16MB` | Memory for VACUUM / index builds |
| `POSTGRES_EFFECTIVE_CACHE_SIZE` | `128MB` | Planner hint for OS page cache |
| `POSTGRES_WAL_BUFFERS` | `512kB` | WAL write buffer |
| `POSTGRES_MIN_WAL_SIZE` | `32MB` | WAL recycling floor |
| `POSTGRES_MAX_WAL_SIZE` | `256MB` | WAL recycling ceiling |
| `POSTGRES_SHM_SIZE` | `64mb` | Container `/dev/shm` size |
| `DB_MAX_CONNS` | `3` | Per-service client pool cap |
| `DB_MIN_CONNS` | `1` | Per-service idle pool floor |

The six services share one Postgres instance, so the sum of `DB_MAX_CONNS`
across services (default `6 × 3 = 18`) must stay under
`POSTGRES_MAX_CONNECTIONS`. If you raise one, raise the other.

## Data & Backups

Two Docker volumes store persistent data:

- `cairn_db_data` — PostgreSQL data
- `cairn_data` — JWT keys

**Backup the database:**

```bash
docker compose exec cairn-db pg_dumpall -U cairn > backup.sql
```

**Restore:**

```bash
docker compose exec -i cairn-db psql -U cairn -d postgres < backup.sql
```

## TLS / HTTPS

The selfhost binary listens on plain HTTP (port 8099 by default). **You must place a TLS-terminating reverse proxy in front of it** before exposing it to the internet.

Each internal service enforces HTTPS by inspecting the `X-Forwarded-Proto` header — the same mechanism used in the multi-container production deployment (where Caddy sets that header). The selfhost binary sets `X-Forwarded-Proto: https` in its middleware layer to satisfy this check, mirroring what a real proxy would do. If your proxy already sets the header, the binary's default is skipped (`if` guard on the header value).

**Example nginx config (minimal):**

```nginx
server {
    listen 443 ssl;
    server_name cairn.example.com;

    ssl_certificate     /etc/ssl/certs/cairn.crt;
    ssl_certificate_key /etc/ssl/private/cairn.key;

    location / {
        proxy_pass http://localhost:8099;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

**Example Caddy config (automatic TLS via Let's Encrypt):**

```caddyfile
cairn.example.com {
    reverse_proxy localhost:8099
}
```

> **Why not terminate TLS inside the binary?** TLS certificate lifecycle (renewal, rotation, ACME) is a solved problem for dedicated tools like Caddy and nginx. Keeping the binary plain-HTTP preserves operator flexibility and matches how every other service in this project is deployed.

## API Endpoints

All services share port 8099:

| Prefix | Service |
|--------|---------|
| `/api/v1/auth/*` | User Service |
| `/api/v1/user/*` | User Service |
| `/api/v1/content/*` | Content Service |
| `/api/v1/internal/content/*` | Content Service |
| `/api/v1/source/rss/*` | RSS Ingest |
| `/api/v1/source/email/*` | Email Ingest |
| `/api/v1/explore/article*` | Explore Recommender |
| `/api/v1/explore/user/*` | Explore Recommender |
| `/api/v1/explore/feed/*` | Explore Fetcher |
| `/health/live` | Liveness check |
| `/health/ready` | Readiness check (includes DB) |

## Upgrading

```bash
docker compose pull   # if using published images
# or
docker compose up -d --build  # if building from source
```

Database migrations run automatically on startup.

## Troubleshooting

**Check logs:**

```bash
docker compose logs -f cairn
```

**Check health:**

```bash
curl -s http://localhost:8099/health/ready | jq
```

**Reset everything:**

```bash
docker compose down -v  # removes volumes — all data lost
docker compose up -d --build
```
