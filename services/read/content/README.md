# Content Service

The Content Service is responsible for storing and serving article content with user-specific metadata.

Full service documentation (data model, routing, endpoints, env vars, workers) lives in
[../CLAUDE.md](/services/read/CLAUDE.md). Endpoint contracts are in [api/openapi.yaml](api/openapi.yaml).

## Responsibilities

- Store deduplicated article content (`contents`) and per-user metadata (`user_contents`: status, favorite, scroll position, list)
- Route each user's items into **Feed** or **Reads** (`source_routes`, `user_contents.list`); re-routing a source affects new items only
- Serve list, search and count queries for a user, authenticated with RS256 JWTs (public key fetched from Vault)
- Accept content from Ingest RSS and Email Ingest through `X-Internal-API-Key`-protected internal routes
- Proxy and aggregate subscription management across Ingest RSS and Email Ingest
- Run the background worker: delete non-favorited Feed items older than 30 days, then content orphaned for 90+ days

## Directory Structure

```
content/
├── cmd/
│   ├── content/main.go          # HTTP server (runs migrations on startup)
│   └── worker/main.go           # Background worker (cleanup cron, health server)
├── api/openapi.yaml             # OpenAPI spec
├── internal/
│   ├── api/                     # Router, handlers, middleware, DTOs
│   ├── config/                  # Configuration
│   ├── database/                # Connection pooling, migrations runner
│   ├── jobs/                    # Cleanup job (Feed expiry + orphaned content)
│   ├── models/                  # Data models
│   ├── processor/               # Readability, sanitization, hashing
│   ├── repository/              # Database layer
│   ├── service/                 # Business logic
│   └── testutil/                # Integration test helpers
├── migrations/                  # golang-migrate SQL files
├── selfhost/                    # Single-container selfhost entry point
├── Dockerfile                   # API server image
├── Dockerfile.worker            # Worker image
├── integration_test.go
└── .env.example                 # Example environment configuration
```

## Configuration

Environment variables (see `.env.example` and the Environment Variables section of [../CLAUDE.md](/services/read/CLAUDE.md)):

- **Server**: `PORT`
- **Database**: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `DB_SSLMODE`
- **Connection Pool**: `DB_MAX_OPEN_CONNS`, `DB_MAX_IDLE_CONNS`, `DB_CONN_MAX_LIFETIME`, `DB_CONN_MAX_IDLE_TIME`
- **JWT**: `VAULT_ADDR`, `VAULT_TOKEN` (or `VAULT_ROLE_ID`/`VAULT_SECRET_ID`), `VAULT_AUTH_PATH`, `JWT_PUBLIC_KEY_PATH`, `JWT_PUBLIC_KEY_REFRESH_INTERVAL`
- **Internal calls**: `INTERNAL_API_KEY` (required), `INGEST_RSS_SERVICE_URL`, `EMAIL_INGEST_SERVICE_URL`

## Building and Running

The full dev stack runs from `infrastructure/docker/dev/` (`docker compose up -d`); the Content Service is
exposed on http://localhost:8083 and the worker on its own container. To build locally from `services/read/`:

```bash
make build-content           # -> bin/content
make build-content-worker    # -> bin/content-worker
```

## Database Migrations

Migrations also run automatically when the server starts. From `services/read/`:

```bash
make migrate-up-content
make migrate-down-content
make migrate-create-content name=add_new_feature
make migrate-status-content
```

## Health Check

```bash
curl http://localhost:8083/health/live
curl http://localhost:8083/health/ready   # includes DB check
```

## URL Detection

`POST /api/v1/content/detect` (JWT required, 10 second timeout) classifies a URL as `feed`, `page` or `unknown`.
`POST /api/v1/content/discover-feed` looks for an RSS/Atom feed behind a page URL.

`POST /api/v1/content/user/{user_id}` routes a submission by `type`:
- **feed**: subscribes the user via Ingest RSS; an optional `list` (`feed`|`reads`) sets the feed's route
- **page**: extracts the article and adds it to Reads (saved links always go to Reads)

If `type` is omitted the service detects it. Request and response shapes are in [api/openapi.yaml](api/openapi.yaml).

The mobile and web "Add" flow calls `/detect` first and submits with the detected type hint
(see `apps/mobile/src/components/AddLinkModal.tsx`).

## API Endpoints

Summary only; paths, parameters and schemas are in [api/openapi.yaml](api/openapi.yaml), and the route
table with auth requirements is in [../CLAUDE.md](/services/read/CLAUDE.md).

- **JWT (`Authorization: Bearer`)**: `/api/v1/content/detect`, `/discover-feed`, `/api/v1/content/user/bulk`,
  and everything under `/api/v1/content/user/{user_id}` (list, count, search, get, patch, delete, subscriptions)
- **Internal (`X-Internal-API-Key`)**: `/api/v1/content` create/get/update, `/bulk`, `/check-duplicate`,
  and `/api/v1/internal/content/user/bulk`
- **Unauthenticated**: `/health/live`, `/health/ready`, `/`

## Development

### Dependencies

- Go 1.21+
- PostgreSQL 15+
- golang-migrate (for migrations)

### Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test package
go test -v ./internal/api/handlers/
go test -v ./internal/api/handlers/ -run TestUserContent

# Integration tests (build tag; needs PostgreSQL, TEST_DB_* env vars)
go test -tags=integration ./...
```

### Testing Protected Endpoints

When testing user-specific endpoints, you need a valid JWT token. Here's how:

1. **Get a token from User Service**:
   ```bash
   curl -X POST http://localhost:8082/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{"email": "user@example.com", "password": "password"}'
   ```

2. **Use token in request header**:
   ```bash
   curl -X GET http://localhost:8083/api/v1/content/user/{user_id} \
     -H "Authorization: Bearer <token>"
   ```

3. **Test authorization failure** (accessing other user's content):
   ```bash
   curl -X GET http://localhost:8083/api/v1/content/user/other-user-id \
     -H "Authorization: Bearer <token>"
   # Returns 403 Forbidden
   ```

### Testing Without JWT

Only the health endpoints are unauthenticated (`/detect` and `/discover-feed` need a JWT):

```bash
curl http://localhost:8083/health/ready
```

## License

Proprietary
