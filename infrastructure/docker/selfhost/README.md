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
curl http://localhost:8080/health/ready
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
│  cairn (:8080)  │────▶│  cairn-db (:5432) │
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
| `PORT` | `8080` | HTTP port |
| `EMAIL_DOMAIN` | `read.cairnapp.com` | Domain for email ingestion |
| `JWT_ACCESS_EXPIRY` | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `ARTICLE_RETENTION_DAYS` | `90` | Explore article retention |

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

## API Endpoints

All services share port 8080:

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
curl -s http://localhost:8080/health/ready | jq
```

**Reset everything:**

```bash
docker compose down -v  # removes volumes — all data lost
docker compose up -d --build
```
