# CLAUDE.md - Explore Service

This file provides guidance to Claude Code (claude.ai/code) when working with the Explore service.

> 📖 **For project-wide context and conventions, see [/CLAUDE.md](/CLAUDE.md) and [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)**

## Service Overview

The Explore service provides RSS feed discovery and content recommendation for the Cairn read-it-later application. It consists of two independent microservices with **separate databases**:

### Architecture: Two Services, Two Databases

```
Fetcher DB ← Explore Fetcher (8080) → HTTP POST → Explore Recommender (8081) → Recommender DB
```

**Critical Architectural Principle**: Each service owns its own database. Services communicate ONLY via HTTP APIs.

### 1. Explore Fetcher (explore_fetcher)
- **Port**: 8080
- **Database**: `fetcher_db` (PostgreSQL)
- **Purpose**: Discovers and fetches RSS feed content
- **Key Responsibilities**:
  - Maintains feed list synchronized from [Kagi Small Web](https://github.com/kagisearch/smallweb)
  - Fetches 1 feed every 60 seconds
  - Prioritizes never-fetched feeds, then oldest
  - Auto-disables feeds after 10 consecutive failures
  - Sends successfully fetched articles to Recommender via HTTP POST

### 2. Explore Recommender (explore_recommender)
- **Port**: 8081
- **Database**: `cairn_db` (PostgreSQL)
- **Purpose**: Stores articles and recommends content to users
- **Key Responsibilities**:
  - Receives articles from Fetcher via HTTP API
  - Deduplicates articles by link
  - Tracks user engagement (upvotes/downvotes)
  - Implements recommendation algorithm
  - Serves personalized recommendations

## Directory Structure

```
services/explore/
├── fetcher/                    # Fetcher service
│   ├── cmd/explore_fetcher/
│   │   └── main.go            # Fetcher entrypoint (also wires up HTTP handlers: health, stats, triggers)
│   ├── internal/
│   │   ├── client/            # HTTP client for recommender API
│   │   ├── db/                # Database repositories
│   │   ├── fetcher/           # Core RSS fetching logic
│   │   └── sync/              # Feed sync from Kagi
│   ├── migrations/            # Fetcher database migrations
│   └── Dockerfile
├── recommender/                # Recommender service
│   ├── cmd/
│   │   ├── explore_recommender/
│   │   │   └── main.go        # Recommender entrypoint
│   │   └── explore_cleanup/
│   │       └── main.go        # Article cleanup utility
│   ├── internal/
│   │   ├── api/               # HTTP handlers (articles, votes, recommendations)
│   │   ├── cleanup/           # Article retention cleanup
│   │   ├── db/                # Database repositories
│   │   └── recommend/         # Recommendation algorithm
│   ├── migrations/            # Recommender database migrations
│   └── Dockerfile
├── api/
│   └── openapi.yaml           # OpenAPI 3.0 specification
├── Makefile                   # Build and development commands
├── README.md                  # Service documentation
├── AGENTS.md                  # This file (CLAUDE.md symlinks to it)
└── CLAUDE.md                  # Symlink to AGENTS.md
```

## Quick Start

### Running All Services (Recommended)

The easiest way to run all Cairn backend services (including Explore) is using the centralized Docker Compose:

```bash
# From repository root
cd infrastructure/docker/dev

# Start all services (includes Vault, databases, all microservices)
docker compose up --build -d

# Check service status
docker compose ps

# View logs
docker compose logs -f explore-fetcher explore-recommender

# Stop services
docker compose down
```

### Running Explore Services Locally

There is no standalone Docker Compose file for the Explore service — `cairn-db` (Postgres) and Vault are shared, centralized infrastructure. For development focused on the Explore service, start the shared infrastructure via the centralized compose, then run the Go binaries natively so you get fast rebuild/restart cycles:

```bash
# Start shared infra (Postgres, Vault) via the centralized compose
cd infrastructure/docker/dev
docker compose up -d cairn-db vault vault-init

# From services/explore, run the services natively (see Makefile Commands below)
cd services/explore
make run-fetcher
make run-recommender
```

### Makefile Commands

```bash
# Build
make build                   # Build both fetcher and recommender binaries
make build-fetcher           # Build only fetcher
make build-recommender       # Build only recommender

# Run (requires databases running)
make run-fetcher             # Run fetcher service (port 8080)
make run-recommender         # Run recommender service (port 8081)
make run-cleanup             # Run article cleanup utility

# Testing
make test                    # Run all tests
make test-integration        # Run integration tests (requires test DB)
make test-all                # Run unit + integration tests

# Docker
make docker-up               # Start services
make docker-down             # Stop services
make docker-logs             # Show logs
make docker-restart          # Restart all services

# Code quality
make fmt                     # Format Go code
make vet                     # Run go vet
make lint                    # Run fmt + vet
make tidy                    # Tidy go modules
```

## API Endpoints

### Explore Fetcher (port 8080)

#### Health Checks
```bash
# Liveness check
curl http://localhost:8080/health/live

# Readiness check (includes DB connectivity)
curl http://localhost:8080/health/ready
```

#### Feed Management (⚠️ Internal only — requires `X-Internal-API-Key` header)
```bash
# Manually trigger feed fetch
curl -X POST -H "X-Internal-API-Key: $INTERNAL_API_KEY" http://localhost:8080/api/v1/explore/feed/fetch

# Sync feeds from Kagi Small Web
curl -X POST -H "X-Internal-API-Key: $INTERNAL_API_KEY" http://localhost:8080/api/v1/explore/feed/sync

# Get feed statistics
curl -H "X-Internal-API-Key: $INTERNAL_API_KEY" http://localhost:8080/api/v1/explore/feed/stats
```

### Explore Recommender (port 8081)

#### Health Checks
```bash
# Liveness check
curl http://localhost:8081/health/live

# Readiness check (includes DB connectivity)
curl http://localhost:8081/health/ready
```

#### Article Management (⚠️ Internal only — requires `X-Internal-API-Key` header)
```bash
# Submit articles (from fetcher) — note the request wraps articles in an
# "articles" array, and fields are "published"/"feed_url"/"feed_title"/
# "categories" (not "published_at"/"feed_id"); article "id" is a content hash,
# not a hash of the link (see Article ID Generation below)
curl -X POST http://localhost:8081/api/v1/explore/article \
  -H "Content-Type: application/json" \
  -H "X-Internal-API-Key: $INTERNAL_API_KEY" \
  -d '{
    "articles": [{
      "id": "...",
      "link": "https://example.com/article",
      "title": "Article Title",
      "description": "Article description",
      "content": "Full content...",
      "author": "Author Name",
      "published": "2025-01-08T10:00:00Z",
      "feed_url": "https://example.com/feed.xml",
      "feed_title": "Feed Title"
    }]
  }'
```

#### Recommendations (requires authentication)
```bash
# Get a page of recommended articles for the authenticated user (user
# identified by JWT), ranked by quality score. Paginate with ?offset=N
# (page size is fixed at 10, ranked from a pool of the top 100 eligible
# articles). This is a pure read — it does not affect recommends counts.
curl -H "Authorization: Bearer <JWT>" \
  "http://localhost:8081/api/v1/explore/recommendation?offset=0"

# Full-text search over articles
curl -H "Authorization: Bearer <JWT>" \
  "http://localhost:8081/api/v1/explore/search?q=keyword"

# Record that a batch of articles was shown to the user — this is what
# actually increments articles.recommends and writes recommendations rows
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"article_ids": ["..."]}' \
  http://localhost:8081/api/v1/explore/shown
```

#### User Interactions (requires authentication)
```bash
# Mark article as read
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/read

# Vote on article (upvote or downvote)
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"vote_type":"upvote"}' \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote

# Remove vote
curl -X DELETE \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote

# Get vote counts
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote

# Get all articles user has voted on (with pagination)
curl -H "Authorization: Bearer <JWT>" \
  "http://localhost:8081/api/v1/explore/user/votes?limit=20&offset=0"

# Get aggregate vote counts for the authenticated user
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/user/vote-stats
```

## Data Models

### Database Schema - Fetcher Database (fetcher_db)

**Feeds Table** (`feeds`):
```sql
CREATE TABLE feeds (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    description TEXT,
    enabled BOOLEAN DEFAULT true,
    last_fetched_at TIMESTAMP,
    consecutive_failures INT DEFAULT 0,
    etag TEXT,                        -- HTTP ETag from last fetch (conditional GET)
    last_modified TEXT,               -- HTTP Last-Modified from last fetch
    fetch_lease_expires_at TIMESTAMP WITH TIME ZONE,  -- Atomic-claim lease; set by GetNextFeed, cleared by UpdateFetchResult
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Fetch History Table** (`fetch_history`):
```sql
CREATE TABLE fetch_history (
    id SERIAL PRIMARY KEY,
    feed_id INT REFERENCES feeds(id) ON DELETE CASCADE,
    fetch_started_at TIMESTAMP NOT NULL,
    fetch_completed_at TIMESTAMP,
    success BOOLEAN,
    articles_found INT DEFAULT 0,
    articles_sent INT DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Database Schema - Recommender Database (cairn_db)

**Articles Table** (`articles`):
```sql
CREATE TABLE articles (
    id VARCHAR(255) PRIMARY KEY,      -- SHA256 hash of cleaned article content (not the link)
    title TEXT NOT NULL,
    link TEXT UNIQUE NOT NULL,
    description TEXT,
    content TEXT,
    author VARCHAR(255),
    published TIMESTAMP NOT NULL,
    feed_url TEXT NOT NULL,
    feed_title VARCHAR(255),
    categories TEXT[],
    feed_id INT,                      -- References feeds in fetcher DB (no FK)
    upvotes INT DEFAULT 0,
    downvotes INT DEFAULT 0,
    recommends INT DEFAULT 0,
    deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP
);
```

**Users Table** (`users`):
```sql
CREATE TABLE users (
    id TEXT PRIMARY KEY,              -- External user ID from User Service, used directly
    created_at TIMESTAMP DEFAULT NOW()
);
```

**User Articles Table** (`user_articles`, tracks read status):
```sql
CREATE TABLE user_articles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id TEXT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_id)
);
```

**Votes Table** (`votes`):
```sql
CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id TEXT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    vote_type TEXT CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)       -- One vote per user per article
);
```

**Recommendations Table** (`recommendations`):
```sql
CREATE TABLE recommendations (
    id SERIAL PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    article_id TEXT NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    recommended_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)       -- One tracked recommendation per user/article
);
```

**Article Categories Table** (`article_categories`):
```sql
CREATE TABLE article_categories (
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    PRIMARY KEY (article_id, category)
);
```
Note: this table exists in migrations but is not currently read from or written to by application code — categories are stored on `articles.categories` (a `TEXT[]` column) instead.

## Key Implementation Details

### Fetcher Service - Feed Management

**Feed Sync** (`fetcher/internal/sync/feed_sync.go`):
- Daily sync from Kagi Small Web Text collection
- Adds new feeds, preserves existing feed state
- Does not delete removed feeds (preserves history)

**Feed Fetching** (`fetcher/internal/fetcher/fetcher.go`):
- Fetches 1 feed every 60 seconds
- Prioritizes never-fetched feeds first
- Then selects oldest `last_fetched_at`
- Auto-disables feeds after 10 consecutive failures
- Only sends successfully fetched articles to recommender

**Feed Selection Priority**:
1. Enabled feeds only
2. Never-fetched feeds (NULL `last_fetched_at`)
3. Oldest fetched feeds

**Failure Handling**:
- Increments `consecutive_failures` on fetch errors
- Resets `consecutive_failures` to 0 on success
- Sets `enabled = false` when `consecutive_failures >= 10`

### Recommender Service - Article Management

**Article Deduplication** (`recommender/internal/db/article_repository.go`):
```sql
INSERT INTO articles (id, title, link, ...)
VALUES ($1, $2, $3, ...)
ON CONFLICT (link) DO UPDATE SET
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    updated_at = NOW()
WHERE articles.deleted = false;
```

**Key Behaviors**:
- Duplicate detection based on article link (UNIQUE constraint)
- Updates article metadata if feed re-publishes with changes
- Preserves vote counts (upvotes/downvotes) on updates
- Preserves deleted status (won't resurrect deleted articles)

### Recommendation Algorithm

**Algorithm** (`recommender/internal/recommend/engine.go`):

`GetRecommendations` is a pure read — it does not mutate `recommends` counts or
write `recommendations` rows. It returns one offset-paginated page:

1. **Fetch eligible articles**: not deleted, not already recommended to this
   user (`GetForRecommendation`), up to a pool of the top 100 candidates
2. **Calculate quality score** for each: `(upvotes - (downvotes * 3)) / recommends`
   - Higher score = better quality relative to exposure
   - Heavily weights downvotes (3x penalty)
   - Articles with `recommends == 0` score `+Inf` if they have upvotes, else `1000.0`
3. **Sort the pool** by quality score descending (ties broken by article ID descending, for stable pagination)
4. **Slice out a page of 10** starting at the caller's `offset` query param (default 0)

Tracking is driven separately by the client: `POST /api/v1/explore/shown` is
the sole writer of `recommendations` rows and the sole driver of the
`articles.recommends` counter, called once articles have actually scrolled
into view.

**Edge Cases Handled**:
- If `recommends = 0`, treat score as very high/infinite (new content prioritized)
- Avoid recommending same article to same user repeatedly (filtered at the eligibility query)
- Handle division by zero gracefully
- `offset` beyond the pool size returns an empty page rather than erroring

**Why This Formula?**
- Downvotes heavily penalized (3x weight) to surface quality content
- Normalizes by exposure (recommends count)
- New articles with high upvotes surface quickly
- Mix of exploitation (high quality) and exploration (low exposure) — the split is emergent from the ranking rather than a fixed "N quality + 1 discovery" slot

### Voting System

**Vote Tracking** (`recommender/internal/db/vote_repository.go`):
- Auto-creates user in `users` table if not exists
- Updates `articles.upvotes` and `articles.downvotes` atomically
- Enforces one vote per user per article (UNIQUE constraint)
- When changing vote type (upvote→downvote), updates both counters

**Vote API** (`recommender/internal/api/handlers.go`):
- `POST /api/v1/explore/article/:id/vote` - Cast or change vote
- `DELETE /api/v1/explore/article/:id/vote` - Remove vote
- `GET /api/v1/explore/article/:id/vote` - Get vote counts

### Article Cleanup

**Retention Policy** (`recommender/internal/cleanup/article_cleanup.go`):
- Two-phase deletion:
  1. **Soft delete**: Mark as `deleted = true` after 90 days
  2. **Hard delete**: Remove from database after 30+ day grace period
- Configurable via `ARTICLE_RETENTION_DAYS` environment variable
- Runs automatically every 24 hours
- Graceful shutdown support

**Manual Cleanup**:
```bash
# Run cleanup utility
make run-cleanup

# Or run directly
go run recommender/cmd/explore_cleanup/main.go
```

**Why Two-Phase Deletion?**
- Maintains referential integrity with votes and recommendations
- Enables data retention for analytics
- Allows for "undelete" functionality if needed
- Hard delete performed later for disk space cleanup

### Authentication

**JWT Authentication** (Recommender only):
- Recommender service requires JWT authentication for user-specific endpoints
- Uses HashiCorp Vault for JWT public key retrieval
- Middleware: `pkg/auth/middleware.go` (shared package, not local to the recommender)
- Extracts `user_id` from JWT claims

**Internal API key authentication** (both services):
- The fetcher's manual trigger endpoints and the recommender's article-submission
  endpoint require the `X-Internal-API-Key` header (shared `INTERNAL_API_KEY` secret)
- Middleware: `pkg/auth/internal_auth.go` (`RequireInternalAPIKey`, shared package)

**Public Endpoints**:
- Health checks (`/health/live`, `/health/ready`)

**Internal-only Endpoints** (require `X-Internal-API-Key`):
- Article submission (`POST /api/v1/explore/article`) — recommender
- Trigger feed fetch (`POST /api/v1/explore/feed/fetch`) — fetcher
- Trigger feed sync (`POST /api/v1/explore/feed/sync`) — fetcher
- Feed statistics (`GET /api/v1/explore/feed/stats`) — fetcher

**Protected Endpoints** (require JWT):
- Recommendations (`GET /api/v1/explore/recommendation`)
- Search (`GET /api/v1/explore/search`)
- Mark shown (`POST /api/v1/explore/shown`)
- User voted articles (`GET /api/v1/explore/user/votes`)
- User vote stats (`GET /api/v1/explore/user/vote-stats`)
- Voting (`POST/DELETE /api/v1/explore/article/:id/vote`)
- Read tracking (`POST /api/v1/explore/article/:id/read`)

## Testing and Development Workflow

### Running Tests

```bash
# Unit tests
make test

# Integration tests (requires test database)
make test-integration

# All tests
make test-all

# Verbose output
go test -v ./...

# Test specific package
go test -v ./fetcher/internal/fetcher/...
go test -v ./recommender/internal/recommend/...
```

### Integration Test Setup

```bash
# Setup test database (one time)
cd recommender
./scripts/setup_test_db.sh

# Run integration tests
make test-integration
```

### Testing Local Changes

**1. Start services:**
```bash
# Use the centralized Docker Compose (there is no standalone compose file
# for the Explore service — see "Running Explore Services Locally" above
# for running the Go binaries natively instead)
cd infrastructure/docker/dev
docker compose up --build
```

**2. Trigger feed fetch:**
```bash
curl -X POST http://localhost:8080/api/v1/explore/feed/fetch
```

**3. Check feed stats:**
```bash
curl http://localhost:8080/api/v1/explore/feed/stats
```

**4. Get recommendations (requires JWT from User Service):**
```bash
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/recommendation
```

**5. Vote on article:**
```bash
curl -X POST \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"vote_type":"upvote"}' \
  http://localhost:8081/api/v1/explore/article/{article_id}/vote
```

**6. View logs:**
```bash
docker compose logs -f explore-fetcher explore-recommender
```

### Database Operations

**Reset databases:**
```bash
# The fetcher and recommender databases are two logical databases inside the
# single consolidated `cairn-db` Postgres container (infrastructure/docker/dev),
# not separate Postgres containers, so resetting means dropping that container's
# volume — `-v` removes it along with the rest of the stack's volumes.
cd infrastructure/docker/dev
docker compose down -v
docker compose up --build
```

**Access database directly:**
```bash
# Run from infrastructure/docker/dev. Both databases live in the same
# consolidated `cairn-db` Postgres container; connect with the per-service
# credentials from .env (see .env.example for the POSTGRES_*_FETCHER /
# POSTGRES_*_RECOMMENDER variable names). `docker compose exec` resolves the
# service name for you, unlike `docker exec` with a guessed container name.
# Fetcher database
docker compose exec cairn-db psql -U cairn_fetcher -d cairn_fetcher

# Recommender database
docker compose exec cairn-db psql -U cairn_recommender -d cairn_recommender
```

**View migrations:**
```bash
# Fetcher migrations
ls -la fetcher/migrations/

# Recommender migrations
ls -la recommender/migrations/
```

## Code Conventions

### Go Code Style

**Error Handling**:
```go
// Use fmt.Errorf with %w for error wrapping
if err != nil {
    return fmt.Errorf("failed to fetch feed %s: %w", feedURL, err)
}

// Return errors up the call stack; let handlers format HTTP responses
func (r *Repository) GetArticle(ctx context.Context, id string) (*models.Article, error) {
    // ... database logic ...
    if err != nil {
        return nil, fmt.Errorf("get article: %w", err)
    }
    return article, nil
}
```

**Database Operations**:
```go
// Always use context.Context for database calls
func (r *ArticleRepository) GetArticle(ctx context.Context, id string) (*models.Article, error) {
    // Implementation
}

// Use ON CONFLICT for upserts instead of SELECT-then-INSERT
_, err := tx.ExecContext(ctx, `
    INSERT INTO articles (id, title, link, ...)
    VALUES ($1, $2, $3, ...)
    ON CONFLICT (link) DO UPDATE SET
        title = EXCLUDED.title,
        updated_at = NOW()
`)

// Prefer batch operations over loops
// Good: Batch insert
articles := []models.Article{...}
err := repo.BulkInsert(ctx, articles)

// Avoid: Loop insert (multiple DB round-trips)
for _, article := range articles {
    repo.Insert(ctx, article)
}
```

**HTTP Clients**:
```go
// Always set timeouts on HTTP clients
client := &http.Client{
    Timeout: 30 * time.Second,
}

// Include context in HTTP requests for cancellation support
req, err := http.NewRequestWithContext(ctx, "POST", url, body)
```

### Repository Pattern

**Structure**:
```
internal/db/
├── config.go              # Database connection config
├── article_repository.go  # Article data access
├── feed_repository.go     # Feed data access
├── vote_repository.go     # Vote data access
└── user_repository.go     # User data access
```

**Pattern**:
```go
type ArticleRepository struct {
    db *sql.DB
}

func NewArticleRepository(db *sql.DB) *ArticleRepository {
    return &ArticleRepository{db: db}
}

func (r *ArticleRepository) GetByID(ctx context.Context, id string) (*models.Article, error) {
    var article models.Article
    err := r.db.QueryRowContext(ctx,
        "SELECT id, title, link FROM articles WHERE id = $1",
        id,
    ).Scan(&article.ID, &article.Title, &article.Link)

    if err != nil {
        return nil, fmt.Errorf("get article by id: %w", err)
    }
    return &article, nil
}
```

### API Handler Pattern

**Structure**:
```go
func (s *Server) handleGetRecommendations(w http.ResponseWriter, r *http.Request) {
    // 1. Extract parameters
    userID := chi.URLParam(r, "user_id")

    // 2. Validate input
    if userID == "" {
        http.Error(w, "user_id required", http.StatusBadRequest)
        return
    }

    // 3. Call repository/service
    articles, err := s.engine.GetRecommendations(r.Context(), userID)
    if err != nil {
        log.Printf("Error getting recommendations: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // 4. Return JSON response
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(articles); err != nil {
        log.Printf("Error encoding response: %v", err)
    }
}
```

## Environment Variables

### Explore Fetcher (explore_fetcher)
```bash
PORT=8080                          # HTTP server port
RECOMMENDER_URL=http://localhost:8081  # URL to recommender service
FETCH_INTERVAL=60                  # Seconds between fetches (1 feed/minute)
DB_HOST=cairn-db                   # PostgreSQL host (consolidated Postgres container; "localhost" if unset)
DB_PORT=5432                       # PostgreSQL port
DB_USER=fetcher                    # Database user
DB_PASSWORD=fetcher_password       # Database password
DB_NAME=fetcher_db                 # Database name
FEED_LIST_PATH=/app/feeds/feeds.txt   # Mount your own list here to override the default
FEED_LIST_URL=https://raw.githubusercontent.com/cairn-app/cairn-reader/main/services/explore/feeds/default-feeds.txt
INTERNAL_API_KEY=...                  # Required; validates X-Internal-API-Key on manual triggers, and sent to the recommender's article-submission endpoint
```
Note: the 30-second per-fetch HTTP timeout and the 10-consecutive-failure auto-disable
threshold are hardcoded (`pkg/rss/fetch` and `feed_repository.go` respectively), not
configurable via `FETCH_TIMEOUT`/`MAX_FETCH_ERRORS` env vars — those variables are not
read anywhere in the codebase.

### Explore Recommender (explore_recommender)
```bash
PORT=8081                          # HTTP server port
DB_HOST=cairn-db                   # PostgreSQL host (consolidated Postgres container; "localhost" if unset)
DB_PORT=5432                       # PostgreSQL port
DB_USER=cairn                      # Database user
DB_PASSWORD=cairn_password         # Database password
DB_NAME=cairn_db                   # Database name
DB_SSLMODE=disable                 # SSL mode (disable for local dev)
ARTICLE_RETENTION_DAYS=90          # Delete articles after N days

# Vault configuration (REQUIRED for JWT authentication)
VAULT_ADDR=http://localhost:8200              # HashiCorp Vault address
VAULT_TOKEN=dev-root-token                    # Vault token
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key  # Path to JWT public key

INTERNAL_API_KEY=...                          # Required; validates X-Internal-API-Key on article submission
```

## Common Development Tasks

### Adding a New Endpoint

**1. Update handler:**
```go
// recommender/internal/api/handlers.go
func (s *Server) handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

**2. Register route:**
```go
// recommender/internal/api/server.go
r.Get("/api/v1/explore/new-endpoint", s.handleNewEndpoint)
```

**3. Add repository method if needed:**
```go
// recommender/internal/db/*_repository.go
func (r *Repository) NewMethod(ctx context.Context) error {
    // Implementation
}
```

**4. Update OpenAPI spec:**
```yaml
# api/openapi.yaml
paths:
  /api/v1/explore/new-endpoint:
    get:
      summary: Description
      responses:
        '200':
          description: Success
```

**5. Add tests:**
```go
// recommender/internal/api/handlers_test.go
func TestNewEndpoint(t *testing.T) {
    // Test implementation
}
```

### Adding a Database Migration

Migrations use [golang-migrate](https://github.com/golang-migrate/migrate) and are
auto-discovered from each service's `migrations/` directory (embedded via
`migrations/embed.go`) — there is no list to update in `main.go`.

**For Fetcher Database:**

**1. Create migration files** (golang-migrate up/down pair, 6-digit sequence prefix):
```sql
-- fetcher/migrations/000003_add_new_column.up.sql
ALTER TABLE feeds ADD COLUMN new_column TEXT;

-- fetcher/migrations/000003_add_new_column.down.sql
ALTER TABLE feeds DROP COLUMN new_column;
```

**2. Test migration:**
```bash
cd infrastructure/docker/dev
docker compose down -v
docker compose up --build
```

**For Recommender Database:**

Follow the same pattern in `recommender/migrations/`.

### Debugging

**View service logs:**
```bash
# All logs
docker compose logs -f

# Fetcher only
docker compose logs -f explore-fetcher

# Recommender only
docker compose logs -f explore-recommender

# Tail last 100 lines
docker compose logs --tail=100 explore-fetcher
```

**Check database state:**
```bash
# Run from infrastructure/docker/dev

# Fetcher database - view feeds
docker compose exec cairn-db psql -U cairn_fetcher -d cairn_fetcher \
  -c "SELECT id, url, enabled, last_fetched_at, consecutive_failures FROM feeds ORDER BY last_fetched_at LIMIT 10;"

# Fetcher database - view fetch history
docker compose exec cairn-db psql -U cairn_fetcher -d cairn_fetcher \
  -c "SELECT * FROM fetch_history ORDER BY fetched_at DESC LIMIT 10;"

# Recommender database - view articles
docker compose exec cairn-db psql -U cairn_recommender -d cairn_recommender \
  -c "SELECT id, title, upvotes, downvotes, recommends, deleted FROM articles LIMIT 10;"

# Recommender database - view votes
docker compose exec cairn-db psql -U cairn_recommender -d cairn_recommender \
  -c "SELECT v.*, a.title FROM votes v JOIN articles a ON v.article_id = a.id LIMIT 10;"
```

**Test individual components:**
```bash
# Test fetcher without running full service
cd fetcher
go test -v ./internal/fetcher/...

# Test recommender algorithm
cd recommender
go test -v ./internal/recommend/...

# Test with race detection
go test -race ./...
```

## Important Notes

### HashiCorp Vault Dependency

**CRITICAL**: The Recommender service requires HashiCorp Vault for JWT authentication.

**What Vault is used for:**
- Retrieves JWT public key for token verification
- All user-specific endpoints require valid JWT

**Development Setup:**
- Centralized Docker Compose ([infrastructure/docker/dev](/infrastructure/docker/dev)) includes Vault container
- Automated `vault-init` service generates RSA keys
- All services configured to use shared Vault instance

**Running without Docker Compose:**
```bash
# Start Vault in dev mode
docker run -d --name vault -p 8200:8200 \
  -e VAULT_DEV_ROOT_TOKEN_ID=dev-root-token \
  hashicorp/vault:latest server -dev

# Initialize Vault with JWT keys
# See: infrastructure/docker/scripts/init-vault.sh
```

**Production**: Use properly configured Vault cluster with persistent storage, TLS, and proper authentication.

### Article ID Generation

**Article IDs are SHA256 hashes of the cleaned article content, not the link**
(`pkg/rss/hash.ContentHash`, called from the fetcher's `convertToArticle`).
Deduplication in the recommender is still keyed on `link` (`ON CONFLICT (link)`),
independent of this ID:

```go
// pkg/rss/hash/hash.go
func ContentHash(html []byte) string {
    normalized := bytes.TrimSpace(html)
    sum := sha256.Sum256(normalized)
    return hex.EncodeToString(sum[:])
}
```

**Why content hashes?**
- Deterministic and reproducible
- Shared with the Read service so both pipelines derive consistent content-hash IDs
- No need for auto-incrementing IDs or UUIDs
- Note: this changed from an earlier `SHA256(link)` scheme (see fetcher migration `000002_add_etag_columns`, which documents the switch)

### Feed ID References

The `feed_id` column in the `articles` table references the feeds table in the **fetcher database** (different database), so it has **no foreign key constraint**.

This is by design:
- Fetcher owns feed data in its database
- Recommender stores `feed_id` as metadata only
- Services remain loosely coupled
- No cross-database foreign key constraints

### Why Separate Databases?

**Architectural Benefits**:
1. **Clear ownership**: Each service owns its own data
2. **Independent scaling**: Can scale databases separately
3. **Simpler deployment**: No shared database coordination
4. **Loose coupling**: Services communicate only via HTTP
5. **Autonomy**: Fetcher manages crawling state independently

### Docker Multi-Stage Builds

Both services use multi-stage Dockerfiles:
1. **Build stage**: `golang:1.23-alpine` - compiles binary
2. **Runtime stage**: `alpine:latest` - minimal image with binary only

This reduces final image size significantly (~20MB vs 300MB+).

## Current Implementation Status

### Explore Fetcher: ✅ COMPLETE
- ✅ Feed synchronization from Kagi Small Web (daily)
- ✅ Rate-limited fetching (1 feed/60 seconds)
- ✅ Priority-based feed selection (never-fetched first, then oldest)
- ✅ Automatic feed disabling (10 consecutive failures)
- ✅ HTTP client for recommender API
- ✅ Feed health tracking
- ✅ Comprehensive test suite (39 tests)

### Explore Recommender: ✅ MOSTLY COMPLETE
- ✅ Article storage with deduplication (ON CONFLICT handling)
- ✅ Voting system (upvote/downvote with user tracking)
- ✅ Enhanced recommendation algorithm (quality score formula)
- ✅ Article cleanup (90-day retention with automatic daily cleanup)
- ✅ JWT authentication with Vault integration
- ✅ Integration test suite (6 tests, all passing)
- ⏳ Admin endpoints (optional, low priority)

### Implementation Progress

**Completed Phases** (8 of 9):
1. ✅ Phase 1: Database Schema & Migrations
2. ✅ Phase 2: Article Deduplication
3. ✅ Phase 3: Voting API
4. ✅ Phase 4: Enhanced Recommendation Algorithm
5. ✅ Phase 5: Model Updates
6. ✅ Phase 6: Configuration & Environment
7. ✅ Phase 7: Article Cleanup
8. ✅ Phase 8: Testing & Validation

**Remaining Work**:
- ⏳ Phase 9: Admin & Monitoring (optional)
  - Admin dashboard endpoints (`GET /admin/stats`, `GET /admin/articles`)
  - Monitoring metrics

## Next Steps

### High Priority
None - All core features are complete and tested.

### Medium Priority
- Admin dashboard endpoints for system monitoring
- Performance metrics and observability
- Caching layer (Redis) for recommendations

### Low Priority
- Full-text search across articles
- Content categorization and tagging
- ML-based personalized recommendations
- Email digest of recommendations

## Architectural Decisions & Rationale

### Why One Feed Per Minute?
- Per requirements: "once per minute, identify the feed with longest time since update"
- Spreads load evenly (no spike when fetching all feeds at once)
- For ~1440 feeds, each feed fetched ~once per day
- Prioritization ensures never-fetched feeds processed first

### Why Separate Votes Table?
- Prevents double-voting per user (UNIQUE constraint on user_id, article_id)
- Enables vote history and analytics
- Can track vote changes over time
- Keeps articles table normalized
- Required for quality score calculation

### Why Quality Score Formula: (upvotes - (downvotes * 3)) / recommends?
- Heavily weights downvotes (3x penalty) to filter low-quality content
- Normalizes by exposure (recommends count)
- New articles with high upvotes surface quickly
- Mix of exploitation (high quality) and exploration (low exposure)
- Prevents filter bubbles by including under-exposed content

### Why Track Failures in Fetcher Database?
- Fetcher owns feed management and crawling state
- Enables autonomous decisions about which feeds to fetch
- Simpler architecture: no need to query recommender for feed metadata
- Recommender only receives successfully fetched articles

## Documentation References

- **Main Project CLAUDE.md**: [/CLAUDE.md](/CLAUDE.md) - Project-wide context
- **Engineering Principles**: [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md) - Standards and conventions
- **Service README**: [README.md](README.md) - Detailed service documentation
- **OpenAPI Spec**: [api/openapi.yaml](api/openapi.yaml) - Formal API specification
- **Fetcher Migrations**: [fetcher/migrations/README.md](fetcher/migrations/README.md)
- **Recommender Migrations**: [recommender/migrations/README.md](recommender/migrations/README.md)

## Inspiration and References

This project is inspired by **Mat Duggan's blog post "Making RSS More Fun"** which proposes:
- Aggregating content from the "small web" (Kagi Small Web collection)
- Using engagement metrics (upvotes/downvotes) to surface quality content
- Smart fetching strategies to spread load and prioritize active feeds
- Community-driven content curation through voting

The implementation follows these principles while adding:
- Microservices architecture with separate databases
- JWT-based authentication and authorization
- Automatic content cleanup and retention policies
- Comprehensive testing and monitoring

## Getting Help

- **Check service logs**: `docker compose logs -f`
- **Review test cases**: Look at `*_test.go` files for usage examples
- **Consult OpenAPI spec**: [api/openapi.yaml](api/openapi.yaml) for API reference
- **Check main docs**: [/CLAUDE.md](/CLAUDE.md) and [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)
- **Database issues**: Check migration files and README files in `migrations/` directories
