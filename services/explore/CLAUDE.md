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
│   ├── cmd/fetcher/
│   │   └── main.go            # Fetcher entrypoint
│   ├── internal/
│   │   ├── api/               # HTTP handlers (health, stats, triggers)
│   │   ├── client/            # HTTP client for recommender API
│   │   ├── db/                # Database repositories
│   │   ├── fetcher/           # Core RSS fetching logic
│   │   └── sync/              # Feed sync from Kagi
│   ├── migrations/            # Fetcher database migrations
│   └── Dockerfile
├── recommender/                # Recommender service
│   ├── cmd/
│   │   ├── recommender/
│   │   │   └── main.go        # Recommender entrypoint
│   │   └── cleanup/
│   │       └── main.go        # Article cleanup utility
│   ├── internal/
│   │   ├── api/               # HTTP handlers (articles, votes, recommendations)
│   │   ├── auth/              # JWT authentication middleware
│   │   ├── cleanup/           # Article retention cleanup
│   │   ├── db/                # Database repositories
│   │   └── recommend/         # Recommendation algorithm
│   ├── migrations/            # Recommender database migrations
│   └── Dockerfile
├── pkg/
│   └── models/                # Shared data models (Article, Feed, Vote, etc.)
├── api/
│   └── openapi.yaml           # OpenAPI 3.0 specification
├── docker-compose.yml         # Local development setup
├── Makefile                   # Build and development commands
├── README.md                  # Service documentation
├── RECOMMENDER_PLAN.md        # Implementation roadmap
└── CLAUDE.md                  # This file
```

## Quick Start

### Running All Services (Recommended)

The easiest way to run all Cairn backend services (including Explore) is using the centralized Docker Compose:

```bash
# From repository root
cd infrastructure/docker

# Start all services (includes Vault, databases, all microservices)
docker-compose up --build -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f explore_fetcher explore_recommender

# Stop services
docker-compose down
```

### Running Explore Services Locally

For development focused on the Explore service:

```bash
cd services/explore

# Start both services with databases
docker-compose up --build

# Or start in detached mode
docker-compose up --build -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
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

#### Feed Management
```bash
# Manually trigger feed fetch
curl -X POST http://localhost:8080/api/v1/explore/feed/fetch

# Sync feeds from Kagi Small Web
curl -X POST http://localhost:8080/api/v1/explore/feed/sync

# Get feed statistics
curl http://localhost:8080/api/v1/explore/feed/stats
```

### Explore Recommender (port 8081)

#### Health Checks
```bash
# Liveness check
curl http://localhost:8081/health/live

# Readiness check (includes DB connectivity)
curl http://localhost:8081/health/ready
```

#### Article Management
```bash
# Submit article (from fetcher)
curl -X POST http://localhost:8081/api/v1/explore/article \
  -H "Content-Type: application/json" \
  -d '{
    "id": "...",
    "link": "https://example.com/article",
    "title": "Article Title",
    "description": "Article description",
    "content": "Full content...",
    "author": "Author Name",
    "published_at": "2025-01-08T10:00:00Z",
    "feed_id": 123
  }'
```

#### Recommendations (requires authentication)
```bash
# Get 5 recommendations for user
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8081/api/v1/explore/recommendation/user123
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
  "http://localhost:8081/api/v1/explore/user/{user_id}/votes?limit=20&offset=0"
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
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Fetch History Table** (`fetch_history`):
```sql
CREATE TABLE fetch_history (
    id SERIAL PRIMARY KEY,
    feed_id INT REFERENCES feeds(id) ON DELETE CASCADE,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    articles_found INT DEFAULT 0,
    fetched_at TIMESTAMP DEFAULT NOW()
);
```

### Database Schema - Recommender Database (cairn_db)

**Articles Table** (`articles`):
```sql
CREATE TABLE articles (
    id TEXT PRIMARY KEY,              -- SHA256 hash of link
    title TEXT NOT NULL,
    link TEXT UNIQUE NOT NULL,
    description TEXT,
    content TEXT,
    author TEXT,
    published_at TIMESTAMP,
    feed_id INT,                      -- References feeds in fetcher DB (no FK)
    upvotes INT DEFAULT 0,
    downvotes INT DEFAULT 0,
    recommends INT DEFAULT 0,
    deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

**Users Table** (`users`):
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,     -- External user identifier
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Votes Table** (`votes`):
```sql
CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    vote_type TEXT CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)       -- One vote per user per article
);
```

**Recommendations Table** (`recommendations`):
```sql
CREATE TABLE recommendations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    recommended_at TIMESTAMP DEFAULT NOW()
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

1. **Filter out deleted articles** (`deleted = false`)
2. **Calculate quality score**: `(upvotes + (downvotes * 3)) / recommends`
   - Higher score = better quality relative to exposure
   - Heavily weights downvotes (3x penalty)
   - Articles with 0 recommends get special handling (high priority)
3. **Select 4 articles** with highest quality score
4. **Select 1 article** with lowest `recommends` count (discovery)
5. **Increment `recommends`** counter for each recommended article
6. **Track recommendation** in `recommendations` table

**Edge Cases Handled**:
- If `recommends = 0`, treat score as very high (new content prioritized)
- Avoid recommending same article to same user repeatedly
- Handle division by zero gracefully

**Why This Formula?**
- Per requirements specification
- Downvotes heavily penalized (3x weight) to surface quality content
- Normalizes by exposure (recommends count)
- New articles with high upvotes surface quickly
- Mix of exploitation (high quality) and exploration (low exposure)

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
go run recommender/cmd/cleanup/main.go
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
- Middleware: `recommender/internal/auth/middleware.go`
- Extracts `user_id` from JWT claims

**Public Endpoints**:
- Health checks (`/health/live`, `/health/ready`)
- Article submission (`POST /api/v1/explore/article`)

**Protected Endpoints** (require JWT):
- Recommendations (`GET /api/v1/explore/recommendation/:user_id`)
- User voted articles (`GET /api/v1/explore/user/:user_id/votes`)
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
# Option A: Use centralized Docker Compose (recommended)
cd infrastructure/docker
docker-compose up --build

# Option B: Use local Docker Compose
cd services/explore
docker-compose up --build
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
  http://localhost:8081/api/v1/explore/recommendation/user123
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
docker-compose logs -f explore_fetcher explore_recommender
```

### Database Operations

**Reset databases:**
```bash
# Local Docker Compose
cd services/explore
docker-compose down
docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data
docker-compose up --build

# Centralized Docker Compose
cd infrastructure/docker
docker-compose down
docker volume rm docker_postgres_data docker_fetcher_postgres_data
docker-compose up --build
```

**Access database directly:**
```bash
# Fetcher database
docker exec -it fetcher_db psql -U fetcher -d fetcher_db

# Recommender database
docker exec -it postgres psql -U cairn -d cairn_db
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
FETCH_TIMEOUT=30                   # Timeout per feed fetch (seconds)
MAX_FETCH_ERRORS=10                # Disable feed after N consecutive failures
DB_HOST=fetcher_db                 # PostgreSQL host
DB_PORT=5432                       # PostgreSQL port
DB_USER=fetcher                    # Database user
DB_PASSWORD=fetcher_password       # Database password
DB_NAME=fetcher_db                 # Database name
KAGI_FEED_URL=https://github.com/kagisearch/smallweb/raw/main/smallweb.txt
```

### Explore Recommender (explore_recommender)
```bash
PORT=8081                          # HTTP server port
DB_HOST=postgres                   # PostgreSQL host
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

**For Fetcher Database:**

**1. Create migration file:**
```sql
-- fetcher/migrations/004_add_new_column.sql
ALTER TABLE feeds ADD COLUMN new_column TEXT;
```

**2. Update migration logic:**
```go
// fetcher/cmd/fetcher/main.go
migrations := []string{
    "001_init.sql",
    "002_add_feed_history.sql",
    "003_feed_indexes.sql",
    "004_add_new_column.sql",  // Add new migration
}
```

**3. Test migration:**
```bash
docker-compose down
docker volume rm cairn-explore_fetcher_postgres_data
docker-compose up --build
```

**For Recommender Database:**

Follow the same pattern but update `recommender/cmd/recommender/main.go` and `recommender/migrations/`.

### Debugging

**View service logs:**
```bash
# All logs
docker-compose logs -f

# Fetcher only
docker-compose logs -f explore_fetcher

# Recommender only
docker-compose logs -f explore_recommender

# Tail last 100 lines
docker-compose logs --tail=100 explore_fetcher
```

**Check database state:**
```bash
# Fetcher database - view feeds
docker exec -it fetcher_db psql -U fetcher -d fetcher_db \
  -c "SELECT id, url, enabled, last_fetched_at, consecutive_failures FROM feeds ORDER BY last_fetched_at LIMIT 10;"

# Fetcher database - view fetch history
docker exec -it fetcher_db psql -U fetcher -d fetcher_db \
  -c "SELECT * FROM fetch_history ORDER BY fetched_at DESC LIMIT 10;"

# Recommender database - view articles
docker exec -it postgres psql -U cairn -d cairn_db \
  -c "SELECT id, title, upvotes, downvotes, recommends, deleted FROM articles LIMIT 10;"

# Recommender database - view votes
docker exec -it postgres psql -U cairn -d cairn_db \
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
- Centralized Docker Compose ([infrastructure/docker](/infrastructure/docker)) includes Vault container
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

**Article IDs are SHA256 hashes of the article link**:

```go
import (
    "crypto/sha256"
    "encoding/hex"
)

func GenerateArticleID(link string) string {
    hash := sha256.Sum256([]byte(link))
    return hex.EncodeToString(hash[:])
}
```

**Why SHA256 hashes?**
- Ensures uniqueness across feeds
- Enables deduplication (same link = same ID)
- Deterministic and reproducible
- No need for auto-incrementing IDs or UUIDs

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

See [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) for detailed progress.

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

### Why Quality Score Formula: (upvotes + (downvotes * 3)) / recommends?
- Per requirements specification
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
- **Implementation Plan**: [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) - Roadmap and progress tracking
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

- **Check service logs**: `docker-compose logs -f`
- **Review test cases**: Look at `*_test.go` files for usage examples
- **Consult OpenAPI spec**: [api/openapi.yaml](api/openapi.yaml) for API reference
- **Read implementation plan**: [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) for feature status
- **Check main docs**: [/CLAUDE.md](/CLAUDE.md) and [/docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md)
- **Database issues**: Check migration files and README files in `migrations/` directories
