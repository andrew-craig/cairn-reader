# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cairn Explore is an RSS feed fetcher and article recommendation system that collects long-form web content from the [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt) and recommends it to users. It consists of two microservices with separate databases that communicate via HTTP APIs:

1. **Fetcher Service** (port 8080): Maintains its own database of feeds, fetches RSS content, sends successfully fetched articles to recommender
2. **Recommender Service** (port 8081): Stores articles, manages user engagement (upvotes/downvotes), and serves personalized recommendations

**Architecture Principle**: Each service owns its own database. Services communicate via HTTP APIs only. The fetcher maintains feed sources and crawling state in its own PostgreSQL database, while the recommender manages articles and user engagement in a separate PostgreSQL database.

**Key Requirements**:
- Fetcher processes 1 feed per minute (prioritizes never-fetched and oldest feeds)
- Recommender serves 5 articles per request (4 high-quality + 1 under-exposed)
- Quality scoring: `(upvotes + (downvotes * 3)) / recommends`
- Automatic feed health tracking with auto-disable after 10 consecutive failures
- Daily feed list sync from Kagi Small Web collection
- 90-day article retention policy

## Common Commands

### Development
```bash
# Run services locally (requires PostgreSQL running)
make run-fetcher              # Start fetcher service
make run-recommender          # Start recommender service

# Build both binaries
make build                    # Outputs to bin/ directory

# Code quality
make fmt                      # Format Go code
make vet                      # Run go vet
make lint                     # Run fmt + vet
make test                     # Run all tests
make tidy                     # Tidy go modules
```

### Docker
```bash
# Start all services (recommended)
docker-compose up --build     # Build and start all services
docker-compose up -d          # Start in detached mode
docker-compose logs -f        # Follow logs
docker-compose down           # Stop all services

# Or use make targets
make docker-up                # Start services
make docker-down              # Stop services
make docker-logs              # Show logs
make docker-restart           # Restart all services
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./fetcher/internal/fetcher
go test ./recommender/internal/recommend

# Run tests with verbose output
go test -v ./...
```

## Architecture and Code Structure

### Service Communication Flow

```
Fetcher DB (PostgreSQL) <--SQL-- Fetcher (8080) --HTTP POST--> Recommender (8081) --SQL--> Recommender DB (PostgreSQL)
```

The fetcher manages its own feed sources in its own database, fetches RSS content, and sends successfully fetched articles to the recommender via `POST /explore/articles`. The recommender handles article deduplication, user engagement tracking, and serves recommendations from its own database.

### Directory Organization

```
fetcher/
├── cmd/fetcher/main.go                    # Service entry point
├── internal/
│   ├── fetcher/fetcher.go                 # RSS parsing logic (uses gofeed)
│   ├── client/recommender_client.go       # HTTP client for recommender API
│   ├── db/
│   │   ├── config.go                      # Database config
│   │   └── feed_repository.go             # Feed CRUD operations
│   └── sync/
│       └── feed_sync.go                   # Daily Kagi feed list sync
├── migrations/
│   └── 001_init_schema.sql                # Feeds table schema

recommender/
├── cmd/recommender/main.go                # Service entry point
├── internal/
│   ├── api/
│   │   ├── server.go                      # Route registration
│   │   ├── handlers.go                    # HTTP handlers
│   │   └── middleware.go                  # Request logging
│   ├── db/
│   │   ├── config.go                      # Database config
│   │   ├── article_repository.go          # Article CRUD
│   │   └── user_repository.go             # User tracking
│   └── recommend/
│       └── engine.go                      # Recommendation algorithm
├── migrations/
│   ├── 001_init.sql                       # Initial schema
│   └── 002_add_feed_id_to_articles.sql    # Add feed_id column

pkg/models/
├── article.go                             # Shared article model
└── feed.go                                # Shared feed model
```

### Key Architectural Patterns

1. **Repository Pattern**: Database access is isolated in `*_repository.go` files
2. **HTTP API Layer**: Handlers in `handlers.go` never directly import database code
3. **Shared Models**: Both services use models from `pkg/models/`
4. **Middleware Chain**: Request logging applied to all endpoints
5. **Graceful Shutdown**: Both services handle SIGINT/SIGTERM for clean shutdowns

## Database Schema

### Current Schema

**Fetcher Database** (fetcher_db):
See migration files in [fetcher/migrations/](fetcher/migrations/):
- [001_init_schema.sql](fetcher/migrations/001_init_schema.sql) - Feeds table and fetch history

**Tables**:
- **feeds**: RSS feed sources managed by fetcher (id SERIAL, url TEXT UNIQUE, enabled BOOLEAN, last_fetched_at TIMESTAMP, consecutive_failures INT)
- **fetch_history**: Track each fetch attempt for monitoring

**Recommender Database** (cairn_db):
See migration files in [recommender/migrations/](recommender/migrations/):
- [001_init.sql](recommender/migrations/001_init.sql) - Initial schema
- [002_add_feed_id_to_articles.sql](recommender/migrations/002_add_feed_id_to_articles.sql) - Add feed_id column to articles

**Tables**:
- **users**: User accounts (id VARCHAR, auto-created on first interaction)
- **articles**: RSS articles with SHA256 hash IDs, includes categories as TEXT[], optional feed_id reference (no FK constraint - references fetcher DB)
- **user_articles**: Tracks read status per user (composite PK: user_id, article_id)

**Important**:
- Article IDs are SHA256 hashes of the article link, ensuring uniqueness across feeds
- The `feed_id` column in articles references the feeds table in the **fetcher database** (different database), so it has no foreign key constraint

### Running Migrations

Migrations are automatically run when the PostgreSQL containers are first created. For existing databases:

```bash
# Reset both databases to run all migrations fresh
docker-compose down
docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data
docker-compose up --build
```

See migration README files for detailed instructions:
- [fetcher/migrations/README.md](fetcher/migrations/README.md) - Fetcher database migrations
- [recommender/migrations/README.md](recommender/migrations/README.md) - Recommender database migrations

### Planned Schema (see implementation plans)

The [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) includes planned tables for:
- **votes**: Upvote/downvote tracking per user (UNIQUE constraint on user_id, article_id)
- **recommendations**: Track which articles recommended to which users
- **article_categories**: Normalized category relationships (for future use)

**Schema changes still needed**:
```sql
-- Add to articles table (voting and recommendation tracking)
ALTER TABLE articles ADD COLUMN upvotes INT DEFAULT 0;
ALTER TABLE articles ADD COLUMN downvotes INT DEFAULT 0;
ALTER TABLE articles ADD COLUMN recommends INT DEFAULT 0;
ALTER TABLE articles ADD COLUMN deleted BOOLEAN DEFAULT false;
```

## Current Implementation Status

### Completed (Basic MVP)
- ✅ PostgreSQL database with Docker Compose orchestration
- ✅ Fetcher service with RSS parsing (gofeed library)
- ✅ Batch article submission via HTTP POST
- ✅ Recommender REST API with health checks
- ✅ Article storage with deduplication (ON CONFLICT handling)
- ✅ Basic recommendation algorithm (recency + content length + randomization)
- ✅ User tracking with read status
- ✅ Request logging middleware
- ✅ **Feed ID column** (migration 002_add_feed_id_to_articles.sql)
  - Article model with optional feed_id reference (no FK constraint)

### In Progress / Next Steps (Per requirements.md)

#### Fetcher Requirements ✅ COMPLETE
- ✅ **Feed database management** (Requirement 1)
  - ✅ Created fetcher's own PostgreSQL database
  - ✅ Store feed list in fetcher database
  - ✅ Daily sync from Kagi Small Web Text collection
  - ✅ Feed repository in fetcher service
  - ✅ Comprehensive test suite (39 tests)
- ✅ **One-feed-per-minute fetching** (Requirement 2)
  - ✅ Fetches 1 feed per minute
  - ✅ Prioritizes never-fetched and oldest feeds
  - ✅ Reports success/failure to database
  - ✅ Auto-disables after 10 consecutive failures

#### Recommender Requirements (See [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md))
- 🔄 **Article database enhancements** (Requirement 1)
  - ✅ Feed schema and feed_id column added
  - ❌ Add upvotes, downvotes, recommends, deleted columns
  - ✅ Endpoint for fetcher to submit articles (working)
- ❌ **Article cleanup** (Requirement 2)
  - Delete articles older than 90 days
  - Background job or cron
- ❌ **Enhanced recommendation algorithm** (Requirement 3)
  - Currently: basic scoring (recency + length + random)
  - Need: `(upvotes + (downvotes * 3)) / recommends` quality score
  - Serve 4 high-quality + 1 low-exposure article
  - Track recommendation counts
- ❌ **Voting system** (Requirement 4)
  - POST /explore/articles/:id/vote endpoint
  - Track votes per user (prevent double-voting)
  - Update upvotes/downvotes counters

### Implementation Timeline
**Fetcher**: ✅ COMPLETE - All core functionality implemented and tested
**Recommender**: See detailed implementation plan:
- [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) - 10 phases with voting and enhanced recommendations

## API Endpoints

### Fetcher Service (port 8080)
**Current:**
```
GET  /health        → {"status":"healthy","service":"fetcher"}
POST /fetch         → Manually trigger fetch
```

### Recommender Service (port 8081)
**Current:**
```
GET  /health                               → Health check
POST /explore/articles                     → Submit articles (from fetcher)
GET  /explore/recommendations/{userID}     → Get 5 recommendations
POST /explore/articles/read                → Mark article as read
     Body: {"article_id": "..."} (user_id from JWT)

# Voting System
POST   /explore/articles/:id/vote          → Upvote/downvote article (requires auth)
       Body: {"vote_type": "upvote|downvote"} (user_id from JWT)
DELETE /explore/articles/:id/vote          → Remove vote (requires auth)
GET    /explore/articles/:id/votes         → Get vote counts (requires auth)
```

**Planned (per implementation plans):**
```
# Feed Management (for fetcher)
GET  /explore/feeds/next               → Get next feed to fetch (prioritized)
POST /explore/feeds/fetch-result       → Report fetch success/failure
POST /explore/feeds/import             → Import/update feeds from Kagi list
GET  /explore/feeds                    → List all feeds (admin)

# Admin & Monitoring
GET  /admin/stats                     → System statistics
GET  /admin/feeds                     → Feed management dashboard
GET  /admin/articles?deleted=true     → View deleted articles
```

## Code Conventions

### Error Handling
- Return errors up the call stack; let handlers format HTTP responses
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Log errors before returning HTTP 500 responses

### Database Operations
- Use context.Context for all database calls (enables cancellation)
- Prefer batch operations over loops (e.g., batch insert articles)
- Use `ON CONFLICT` for upsert semantics instead of SELECT-then-INSERT

### HTTP Clients
- Always set reasonable timeouts (fetcher uses 30s for HTTP, 10s for recommender API)
- Include context in HTTP requests for cancellation support
- Check response status codes explicitly

### Recommendation Algorithm
**Current Implementation** in [recommender/internal/recommend/engine.go](recommender/internal/recommend/engine.go):
- Filters out already-read articles
- Scores based on: recency (10pts/7days, 5pts/30days), content length, title quality
- Adds randomization to avoid monotony
- Returns top 5 scored articles

**Required Implementation** (per [requirements.md](requirements.md)):
- Quality score formula: `(upvotes + (downvotes * 3)) / recommends`
- Return 5 articles: 4 with highest quality score + 1 with lowest recommends count
- Filter out deleted articles
- Increment recommends counter for each article returned
- Track recommendations per user to avoid repeats

## Environment Variables

### Fetcher
**Current:**
```bash
PORT=8080                      # HTTP server port
RECOMMENDER_URL=...            # URL to recommender service
FETCH_INTERVAL=60              # 60 seconds = 1 feed per minute
FETCH_TIMEOUT=30               # 30 seconds timeout per feed
MAX_FETCH_ERRORS=10            # Disable feed after N consecutive failures
DB_HOST=fetcher_db             # Fetcher's own PostgreSQL database
DB_PORT=5432
DB_USER=fetcher
DB_PASSWORD=fetcher_password
DB_NAME=fetcher_db
KAGI_FEED_URL=...              # URL to Kagi Small Web feed list
```

### Recommender
**Current:**
```bash
PORT=8081                      # HTTP server port
DB_HOST=postgres               # PostgreSQL host
DB_PORT=5432                   # PostgreSQL port
DB_USER=cairn                  # Database user
DB_PASSWORD=cairn_password     # Database password
DB_NAME=cairn_db               # Database name
ARTICLE_RETENTION_DAYS=90      # Delete articles after 90 days
```

## Dependencies

Key third-party packages:
- **github.com/mmcdole/gofeed** (v1.3.0): RSS/Atom feed parser
- **github.com/lib/pq** (v1.10.9): PostgreSQL driver

## Development Workflow

1. **Making changes to shared models**: Edit `pkg/models/*.go` - affects both services
2. **Adding new endpoints**: Update `recommender/internal/api/handlers.go` and `server.go`
3. **Database changes**:
   - Fetcher database: Create migration in `fetcher/migrations/` (use sequential numbering)
   - Recommender database: Create migration in `recommender/migrations/` (use sequential numbering)
4. **Fetcher modifications**: Core logic in `fetcher/internal/fetcher/fetcher.go`

### Adding a New Repository

1. Create `recommender/internal/db/new_repository.go`
2. Define repository struct with `*sql.DB` field
3. Implement methods using prepared statements or inline queries
4. Initialize in `main.go` and pass to handlers via dependency injection

### Testing Changes

**Current Functionality:**
1. Start services: `docker-compose up --build`
2. Trigger fetch: `curl -X POST http://localhost:8080/fetch`
3. Check recommendations: `curl -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/recommendations/user123`
4. Mark as read: `curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/read -d '{"article_id":"abc..."}'`
5. View logs: `docker-compose logs -f`

**Voting (requires authentication):**
```bash
# Upvote/downvote
curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/abc123/vote \
  -d '{"vote_type": "upvote"}'
curl -X DELETE -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/abc123/vote
curl -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/abc123/votes

# Admin
curl http://localhost:8081/admin/stats
curl http://localhost:8081/admin/feeds?enabled=false
```

## Feed Management ✅ COMPLETE

**Implementation Status**: All feed management functionality is fully implemented and tested.

**Current Implementation**:
1. **Feed Source Management**: ✅
   - Feeds stored in **fetcher's own PostgreSQL database**
   - Daily sync from [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt)
   - Feed repository in fetcher service manages all feed operations
   - See: [fetcher/internal/sync/feed_sync.go](fetcher/internal/sync/feed_sync.go)

2. **One-Feed-Per-Minute Fetching**: ✅
   - Fetches 1 feed every 60 seconds (not all feeds at once)
   - Prioritizes: never-fetched feeds first, then oldest last_fetched_at
   - Only fetches enabled feeds
   - Tracks fetch success/failure in fetcher database
   - See: [fetcher/internal/fetcher/fetcher.go](fetcher/internal/fetcher/fetcher.go)

3. **Feed Health Tracking**: ✅
   - Tracks consecutive_failures counter in fetcher database
   - Auto-disables feed after 10 consecutive failures
   - Resets counter to 0 on successful fetch
   - See: [fetcher/internal/db/feed_repository.go](fetcher/internal/db/feed_repository.go)

**Architecture Principle**: Fetcher maintains its own database for feed management. Only successfully fetched articles are sent to the Recommender via HTTP POST.

## Recommendation Algorithm Details

### Current Algorithm
The scoring formula in [recommender/internal/recommend/engine.go](recommender/internal/recommend/engine.go#L50-L80):

```
score = recency_score + length_score + title_score + random_factor

Where:
- recency_score: 10 (last 7 days) | 5 (last 30 days) | 0 (older)
- length_score: min(len(content)/1000, 5) [0-5 points]
- title_score: min(len(title)/10, 3) [0-3 points]
- random_factor: rand.Float64() * 5 [0-5 points]
```

This balances recency, content depth, and randomization to provide diverse recommendations.

### Required Algorithm (per [requirements.md](requirements.md))

**Quality Score Formula:**
```
quality_score = (upvotes + (downvotes * 3)) / recommends
```

**Selection Strategy:**
1. Filter out deleted articles (`deleted = false`)
2. Calculate quality score for all eligible articles
3. Handle division by zero: if `recommends = 0`, treat as infinite/very high score
4. Select **4 articles** with highest quality score
5. Select **1 article** with lowest `recommends` count (exploration/discovery)
6. Return these 5 articles
7. Increment `recommends` counter for each returned article
8. Track in `recommendations` table (avoid showing same article to same user repeatedly)

**Key Design Decisions:**
- Downvotes heavily penalized (3x weight) to surface quality content
- Mix of exploitation (high quality) and exploration (low exposure)
- Prevents filter bubbles by including under-exposed content
- Normalizes by exposure (recommends count) to give new content a chance

See [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) Phase 4.3 for implementation details.

## Docker Multi-Stage Builds

Both services use multi-stage Dockerfiles:
1. **Build stage**: `golang:1.23-alpine` - compiles binary
2. **Runtime stage**: `alpine:latest` - minimal image with binary only

This reduces final image size significantly (~20MB vs 300MB+).

## Database Migrations

Currently manual: migrations run on first connection in `main.go` for each service.

**To add a new migration**:

For Fetcher database:
1. Create `00X_new_feature.sql` in `fetcher/migrations/`
2. Update migration logic in `fetcher/cmd/fetcher/main.go` to run new file

For Recommender database:
1. Create `00X_new_feature.sql` in `recommender/migrations/`
2. Update migration logic in `recommender/cmd/recommender/main.go` to run new file

Consider using a migration tool (e.g., golang-migrate) for production

## Health Checks

Both services expose `/health` endpoints:
- Used by Docker Compose health checks
- Returns JSON: `{"status":"healthy","service":"fetcher|recommender"}`
- Recommender checks database connectivity before reporting healthy

## Project Requirements & Implementation Plans

### Core Requirements
See [requirements.md](requirements.md) for full specification. Key requirements:

**Fetcher:**
1. Maintain source list from Kagi Small Web collection (daily sync)
2. Fetch 1 feed per minute (prioritize never-fetched, then oldest)
3. Auto-disable feeds after 10 consecutive failures

**Recommender:**
1. Store articles with metadata (upvotes, downvotes, recommends, deleted)
2. Delete articles after 90 days
3. Serve 5 recommendations using quality score algorithm
4. Provide upvote/downvote API

### Implementation Plans

**Fetcher**: ✅ COMPLETE
  - ✅ Separate PostgreSQL database for fetcher
  - ✅ Database schema for feeds table and fetch history
  - ✅ Daily feed sync from Kagi Small Web collection
  - ✅ One-feed-per-minute fetching logic
  - ✅ Article filtering
  - ✅ Feed health tracking in fetcher database
  - ✅ Comprehensive test suite (39 tests)

**Recommender**: See [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) for detailed implementation plan:
  - Enhanced database schema (votes, recommendations tables)
  - Voting system API
  - Enhanced recommendation algorithm with quality scoring
  - Article cleanup (90-day retention)
  - Admin dashboard endpoints

### Key Architectural Decisions

**Why One Feed Per Minute?**
- Requirements specify: "once per minute, identify the feed with the longest time since an update"
- Spreads load evenly across time (no spike when fetching all feeds at once)
- For ~1440 feeds, each feed gets fetched approximately once per day
- Prioritization ensures never-fetched feeds are processed first

**Why Separate Votes Table?**
- Prevents double-voting per user (UNIQUE constraint on user_id, article_id)
- Enables vote history and analytics
- Keeps articles table normalized
- Required for quality score calculation

**Why Separate Databases for Fetcher and Recommender?**
- **Architectural clarity**: Each service owns its own data
- **Fetcher autonomy**: Doesn't depend on Recommender's database schema
- **Independent scaling**: Can scale fetcher database separately
- **Clear separation of concerns**: Fetcher manages crawling state, Recommender manages articles
- **Simpler deployment**: No shared database to coordinate migrations

**Why Track Failures in Fetcher Database?**
- Fetcher owns feed management and crawling state
- Enables fetcher to make autonomous decisions about which feeds to fetch
- Simpler architecture: no need to query recommender for feed metadata
- Recommender only receives successfully fetched articles

**Why Quality Score Formula: (upvotes + (downvotes * 3)) / recommends?**
- Per requirements specification
- Heavily weights downvotes (3x penalty) to filter low-quality content
- Normalizes by exposure (recommends count)
- New articles with high upvotes surface quickly
- Articles with downvotes are heavily penalized

**Why "deleted" Flag vs Hard Delete?**
- Maintains referential integrity with votes and recommendations tables
- Enables data retention for analytics
- Allows for "undelete" functionality if needed
- Hard delete can be done later for cleanup

### Next Steps
All fetcher implementation is complete. Remaining work is in the Recommender service:
1. Implement database migrations for votes, recommendations tables (RECOMMENDER_PLAN Phase 1)
2. Implement article deduplication with ON CONFLICT handling (RECOMMENDER_PLAN Phase 2)
3. Implement voting system (RECOMMENDER_PLAN Phase 3)
4. Enhance recommendation algorithm with quality scoring (RECOMMENDER_PLAN Phase 4)
5. Add article cleanup job (RECOMMENDER_PLAN Phase 8)

See [RECOMMENDER_PLAN.md](RECOMMENDER_PLAN.md) for detailed implementation steps.

## Inspiration and References

This project is inspired by Mat Duggan's blog post "Making RSS More Fun" which proposes:
- Aggregating content from the "small web" (Kagi Small Web collection)
- Using engagement metrics (upvotes/downvotes) to surface quality content
- Smart fetching strategies to spread load and prioritize active feeds
- Community-driven content curation through voting
