# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cairn is a read-it-later application consisting of:
- **Mobile App** (React Native/Expo): iOS and Android app for reading saved articles
- **Backend Services** (Go): Microservices for content discovery, storage, and user management
  - **Explore Service**: RSS feed fetching and content recommendation
  - **Read Service**: Article storage and user-specific metadata
  - **User Service**: Authentication and account management

## Quick Start - Running All Backend Services

The easiest way to run all backend services is using the centralized Docker Compose setup:

```bash
cd infrastructure/docker

# Copy and configure environment variables
cp .env.example .env
# Edit .env and set secure passwords

# Start all services (Vault, databases, and all backend services)
docker-compose up --build -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f

# Stop all services
docker-compose down
```

This starts:
- HashiCorp Vault (port 8200) with auto-initialization
- All PostgreSQL databases (ports 5432-5436)
- User Service (port 8082)
- Explore Recommender Service (port 8081)
- Explore Fetcher Service (port 8080)
- Content Service (port 8083)
- Ingest RSS Service (port 8085)
- Background workers (ports 8084, 8086)

See [infrastructure/docker/README.md](infrastructure/docker/README.md) for detailed documentation.

## Common Commands

### Mobile App (apps/mobile)
```bash
# Navigate to mobile app
cd apps/mobile

# Install dependencies
npm install

# Development
npm start                    # Start Expo dev server
npm run ios                  # Run on iOS simulator
npm run android              # Run on Android emulator

# Code quality
npm run lint                 # Run ESLint
npm run type-check           # TypeScript type checking (tsc --noEmit)
```

### Explore Service (services/explore)
```bash
cd services/explore

# Development (requires PostgreSQL running)
make run-fetcher             # Start explore_fetcher service (port 8080)
make run-recommender         # Start explore_recommender service (port 8081)
make run-cleanup             # Run article cleanup utility

# Build
make build                   # Build both binaries to bin/

# Testing
make test                    # Run all tests
go test ./...                # Alternative test command
go test -v ./fetcher/...     # Test specific package with verbose output

# Docker (recommended for development)
docker-compose up --build    # Start all services with databases
docker-compose up -d         # Start in detached mode
docker-compose logs -f       # Follow logs
docker-compose down          # Stop all services

# Or use make targets
make docker-up               # Start services
make docker-down             # Stop services
make docker-logs             # Show logs
make docker-restart          # Restart all services

# Code quality
make fmt                     # Format Go code (go fmt ./...)
make vet                     # Run go vet
make lint                    # Run fmt + vet
make tidy                    # Tidy go modules
```

### User Service (services/users)
```bash
cd services/users

# Build
make build                   # Build binaries to bin/

# Development
make run                     # Run user service
make dev                     # Run with live reload (requires air)

# Testing
make test                    # Run all tests
make test-coverage           # Generate coverage report (coverage.html)
go test ./...                # Run tests directly

# Database migrations
make migrate-up              # Run migrations
make migrate-down            # Rollback migrations
make migrate-version         # Check current migration version
make db-create               # Create database and user (requires psql)
make db-drop                 # Drop database

# Code quality
make fmt                     # Format code (go fmt ./...)
make vet                     # Run go vet
make lint                    # Run golangci-lint (requires installation)
make deps                    # Download and tidy dependencies

# Docker
make docker-build            # Build Docker image
make docker-run              # Run in Docker container
```

### Read Service (services/read)
```bash
cd services/read

# Development (requires Docker)
docker-compose up --build    # Start all services (Content + RSS Fetcher)
docker-compose up -d         # Start in detached mode
docker-compose logs -f       # Follow logs
docker-compose down          # Stop all services

# Build
make build                   # Build both services

# Testing
make test                    # Run all tests
make test-coverage           # Generate coverage report
make test-integration        # Run integration tests (requires PostgreSQL)

# Database migrations
make migrate-up              # Apply pending migrations
make migrate-down            # Rollback last migration
make migrate-status          # Check migration status
make migrate-create name=... # Create new migration

# Code quality
make fmt                     # Format code
make vet                     # Run go vet
make lint                    # Run linter
```

## Architecture

> 📖 **For detailed architectural principles and rationale, see [Engineering Principles - Core Architectural Principles](/docs/ENGINEERING_PRINCIPLES.md#core-architectural-principles)**

### System Architecture
Cairn follows a microservices architecture where services communicate via REST APIs:

```
Mobile App (React Native) → REST APIs → Backend Services
                                        ├── User Service (Auth)
                                        ├── Explore Service (RSS)
                                        └── Read Service (Storage)
                                               ↓
                                        PostgreSQL
```

### Explore Service Architecture
The Explore service consists of two microservices with **separate databases**:

```
Fetcher DB ← Explore Fetcher (8080) → HTTP POST → Explore Recommender (8081) → Recommender DB
```

**Key architectural principle**: Each service owns its own database. Services communicate only via HTTP APIs.

- **Explore Fetcher** (explore_fetcher): Manages feed sources in its own PostgreSQL database, fetches RSS content (1 feed/minute), sends successfully fetched articles to recommender via HTTP POST
- **Explore Recommender** (explore_recommender): Stores articles in its own PostgreSQL database, manages user engagement (upvotes/downvotes), serves personalized recommendations

### Mobile App Structure
```
apps/mobile/
├── src/
│   ├── components/      # Reusable UI components
│   │   └── common/      # Shared components (Button, ArticleCard, EmptyState)
│   ├── constants/       # App constants (theme, colors)
│   ├── contexts/        # React contexts
│   ├── navigation/      # Navigation configuration
│   │   ├── RootNavigator.tsx    # Stack navigator
│   │   └── TabNavigator.tsx     # Bottom tabs (Explore, Read, Settings)
│   ├── screens/         # App screens (Home, Explore, Read, Favorites, Archive, Settings, etc.)
│   ├── services/        # Services (Storage via AsyncStorage, API clients)
│   ├── types/           # TypeScript type definitions
│   │   ├── article.ts   # Article interface
│   │   └── navigation.ts # Navigation types
│   └── utils/           # Utility functions
├── assets/              # Images, fonts, assets
├── App.tsx              # Entry point
├── app.json             # Expo configuration
└── package.json         # Dependencies and scripts
```

**Mobile App Patterns**:
- **Navigation**: React Navigation with Stack and Bottom Tabs
- **State Management**: React hooks and Context API
- **Storage**: AsyncStorage for local persistence
- **Styling**: React Native core components with theme system (supports dark mode)

### Backend Service Structure
Each Go service follows a consistent structure:

```
services/{service}/
├── cmd/{service}/main.go     # Application entrypoint
├── internal/                 # Private application code
│   ├── api/                  # HTTP handlers, routes, middleware
│   ├── db/                   # Database repositories, config
│   └── {domain}/             # Domain-specific logic
├── pkg/                      # Public shared libraries
├── migrations/               # Database migrations (numbered SQL files)
├── Dockerfile                # Multi-stage Docker build
├── Makefile                  # Build and development commands
└── README.md                 # Service documentation
```

**Backend Patterns**:
- **Repository Pattern**: Database access isolated in `*_repository.go` files
- **HTTP API Layer**: Handlers never directly import database code
- **Shared Models**: Services use models from `pkg/models/`
- **Middleware Chain**: Request logging, authentication, authorization
- **Graceful Shutdown**: Services handle SIGINT/SIGTERM

## Data Models

### Mobile App - Article Interface
```typescript
interface Article {
  id: string;              // Unique identifier
  url: string;             // Source URL
  title: string;
  description?: string;
  imageUrl?: string;
  author?: string;
  publishedDate?: string;
  readingTime?: number;    // Minutes
  tags: string[];
  isRead: boolean;
  isFavorite: boolean;
  addedAt: number;         // Timestamp
  readAt?: number;         // Timestamp
  notes?: string;
}
```

### Backend - Database Schema

**Explore Service - Fetcher Database** (`fetcher_db`):
- `feeds`: RSS feed sources (id, url, enabled, last_fetched_at, consecutive_failures)
- `fetch_history`: Track each fetch attempt for monitoring

**Explore Service - Recommender Database** (`cairn_db`):
- `users`: User accounts (id VARCHAR, auto-created on first interaction)
- `articles`: RSS articles with SHA256 hash IDs, includes categories as TEXT[], optional feed_id reference
- `user_articles`: Tracks read status per user (composite PK: user_id, article_id)

**User Service Database** (`cairn_users`):
- `users`: User accounts with email/password or device ID authentication
- `refresh_tokens`: JWT refresh token management

**Read Service - Content Database** (`content_service`):
- `contents`: Cleaned article content with readability extraction and HTML sanitization
- `user_contents`: User-specific metadata (status, favorites, scroll position, notes)
- Supports content deduplication by hash and feed ID
- Full-text search with PostgreSQL GIN index

**Read Service - Ingest RSS Database** (`ingest_rss`):
- `feeds`: RSS feed metadata and polling configuration
- `user_feeds`: User subscriptions to feeds (max 100 per user)
- `feed_items`: Pending feed items for extraction
- `outbox`: Reliable content delivery queue using outbox pattern
- Tiered polling strategy (hourly/6-hourly/daily)

## Key Implementation Details

### Explore Service - Feed Management
**Implementation Status**: ✅ COMPLETE

- Feeds stored in fetcher's own PostgreSQL database
- Daily sync from [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt)
- Fetches 1 feed every 60 seconds (prioritizes never-fetched feeds, then oldest)
- Auto-disables feeds after 10 consecutive failures
- Only successfully fetched articles sent to recommender via HTTP POST

See: `services/explore/fetcher/internal/sync/feed_sync.go` and `services/explore/fetcher/internal/fetcher/fetcher.go`

### Explore Service - Recommendation Algorithm
**Current Implementation**: Basic scoring with recency, content length, title quality, and randomization

**Required Implementation** (per requirements):
- Quality score formula: `(upvotes + (downvotes * 3)) / recommends`
- Return 5 articles: 4 with highest quality score + 1 with lowest recommends count
- Filter out deleted articles
- Increment recommends counter for each returned article
- Track recommendations per user to avoid repeats

See: `services/explore/RECOMMENDER_PLAN.md` for detailed implementation plan

### User Service - Authentication
- JWT-based stateless authentication with RS256 signing
- Refresh token management with automatic rotation
- Mobile device authentication via Expo device ID
- Account upgrade from device-only to email/password
- Secure secrets management with HashiCorp Vault
- Authorization middleware ensures users can only access their own data

### Read Service - Content Management
**Implementation Status**: ✅ COMPLETE

The Read service consists of two microservices:

**Content Service** (port 8080):
- Stores cleaned article content using go-readability extraction
- HTML sanitization with bluemonday
- Content deduplication by hash + feed ID
- User-specific metadata (status, favorites, scroll position)
- Full-text search with PostgreSQL GIN index
- Cursor-based pagination (20 items/page)
- Content size limit (5MB)
- Orphaned content cleanup (90-day retention)

**Ingest RSS Service** (ingest_rss, port 8081):
- User feed subscriptions (100 feed limit per user)
- Tiered polling strategy: hourly (active), 6-hourly (moderate), daily (quiet)
- Auto-disable feeds after 7 consecutive error days
- Content extraction and processing
- Outbox pattern for reliable content delivery
- Circuit breaker for Content Service calls
- Update detection via ETag/Last-Modified headers

See: `services/read/README.md` for detailed documentation

### Mobile App - Theme System
- Supports light and dark modes (follows system preference)
- Color schemes defined in `src/constants/theme.ts`
- Automatically applies theme across all components

## API Endpoints

> **Note:** All services have been migrated to API v1 with standardized endpoints. Health checks now use `/health/live` (liveness) and `/health/ready` (readiness) for all services.

### Explore Service - Explore Fetcher (explore_fetcher, port 8080)
```
# Health
GET  /health/live                        → Liveness check
GET  /health/ready                       → Readiness check (includes DB)

# Feed Management
POST /api/v1/explore/feed/fetch          → Manually trigger feed fetch
POST /api/v1/explore/feed/sync           → Sync feeds from Kagi Small Web
GET  /api/v1/explore/feed/stats          → Get feed statistics
```

### Explore Service - Explore Recommender (explore_recommender, port 8081)
```
# Health
GET  /health/live                                   → Liveness check
GET  /health/ready                                  → Readiness check (includes DB)

# Article Management
POST /api/v1/explore/article                        → Submit article (from fetcher)
     Body: {"id": "...", "link": "...", "title": "...", "description": "...", ...}

# Recommendations
GET  /api/v1/explore/recommendation/{user_id}       → Get 5 recommendations (requires auth)

# User Interactions
POST /api/v1/explore/article/{article_id}/read      → Mark article as read (requires auth)
     user_id extracted from JWT

POST /api/v1/explore/article/{article_id}/vote      → Vote on article (requires auth)
     Body: {"vote_type": "upvote|downvote"} (user_id from JWT)

DELETE /api/v1/explore/article/{article_id}/vote    → Remove vote (requires auth)

GET  /api/v1/explore/article/{article_id}/vote      → Get vote counts (requires auth)
```

### User Service (port 8082)
```
# Health
GET  /health/live                            → Liveness check
GET  /health/ready                           → Readiness check (DB + Vault connectivity)

# Authentication
POST /api/v1/auth/register                   → Create account with email/password
POST /api/v1/auth/register/mobile            → Create mobile-only account (Expo device ID)
POST /api/v1/auth/login                      → Login and return tokens
POST /api/v1/auth/login/mobile               → Authenticate with device ID
POST /api/v1/auth/refresh                    → Exchange refresh token for new access token
POST /api/v1/auth/logout                     → Revoke specific refresh token
POST /api/v1/auth/logout-all                 → Revoke all refresh tokens

# User Management
GET    /api/v1/user/{user_id}                → Get user profile (authenticated)
PATCH  /api/v1/user/{user_id}                → Update user profile (authenticated)
POST   /api/v1/user/{user_id}/upgrade        → Add email/password to mobile-only account
DELETE /api/v1/user/{user_id}                → Delete account (authenticated)
```

### Read Service - Content Service (port 8083)
```
# Health
GET  /health/live                                           → Liveness check
GET  /health/ready                                          → Readiness check (includes DB)

# Content Management (Direct)
POST   /api/v1/content                                      → Create content from HTML/URL
       Body: {"html": "...", "source_url": "...", "title": "...", ...}

GET    /api/v1/content/{content_id}                         → Get content by ID

PUT    /api/v1/content/{content_id}                         → Update existing content

POST   /api/v1/content/bulk                                 → Bulk create/update contents
       Body: [{"html": "...", "source_url": "...", ...}, ...]

POST   /api/v1/content/check-duplicate                      → Check for duplicate content
       Body: {"items": [{"content_hash": "...", "source_feed_id": "..."}, ...]}

# User Content Management
POST   /api/v1/content/user/{user_id}                       → Add content to user's list
       Body: {"url": "...", "html": "...", "source_type": "rss|manual|web"}

GET    /api/v1/content/user/{user_id}                       → List user's contents (paginated)
       Query params: ?status=..., ?is_favorite=true, ?limit=20, ?offset=0

GET    /api/v1/content/user/{user_id}/search                → Full-text search user's contents
       Query params: ?q=golang, ?limit=20, ?offset=0

PATCH  /api/v1/content/user/{user_id}/{content_id}          → Update user-content metadata
       Body: {"status": "reading|completed|archived", "is_favorite": true, "scroll_position": 0.5, "notes": "..."}

DELETE /api/v1/content/user/{user_id}/{content_id}          → Remove content from user's list

POST   /api/v1/content/user/bulk                            → Bulk add contents to multiple users
       Body: [{"user_id": "...", "url": "...", "title": "...", ...}, ...]
```

### Read Service - Ingest RSS Service (port 8085)
```
# Health
GET  /health/live                                                    → Liveness check
GET  /health/ready                                                   → Readiness check (includes DB)

# Feed Subscriptions
POST   /api/v1/source/rss/user/{user_id}/subscription               → Subscribe to feed
       Body: {"feed_url": "https://..."}

DELETE /api/v1/source/rss/user/{user_id}/subscription/{feed_id}     → Unsubscribe from feed

GET    /api/v1/source/rss/user/{user_id}/subscription               → List user's subscriptions

# Feed Management
GET    /api/v1/source/rss/feed                                      → List all feeds (admin)

GET    /api/v1/source/rss/feed/{feed_id}                            → Get feed details

PATCH  /api/v1/source/rss/feed/{feed_id}                            → Update feed (enable/disable)
       Body: {"enabled": true|false}

POST   /api/v1/source/rss/feed/{feed_id}/refresh                    → Manually trigger feed refresh
```

### OpenAPI/Swagger Documentation

Formal API specifications are available in OpenAPI 3.0 format:

- **Explore Service**: `services/explore/api/openapi.yaml`
- **User Service**: `services/users/api/openapi.yaml`
- **Read Service**: `services/read/api/openapi.yaml`

**Viewing the API Documentation:**

Using Swagger UI (requires Docker):
```bash
# Explore Service API
docker run -p 8082:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/explore/api:/api swaggerapi/swagger-ui

# User Service API
docker run -p 8083:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/users/api:/api swaggerapi/swagger-ui

# Read Service API
docker run -p 8084:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/read/api:/api swaggerapi/swagger-ui
```

Then visit:
- Explore Service: http://localhost:8082
- User Service: http://localhost:8083
- Read Service: http://localhost:8084

**Validating OpenAPI Specs:**
```bash
# Validate Explore Service spec
npx @apidevtools/swagger-cli validate services/explore/api/openapi.yaml

# Validate User Service spec
npx @apidevtools/swagger-cli validate services/users/api/openapi.yaml

# Validate Read Service spec
npx @apidevtools/swagger-cli validate services/read/api/openapi.yaml
```

## Testing and Development Workflow

> 📖 **For comprehensive testing standards and patterns, see [Engineering Principles - Testing Philosophy](/docs/ENGINEERING_PRINCIPLES.md#testing-philosophy)**

### Testing Mobile App Changes
1. Start Expo dev server: `cd apps/mobile && npm start`
2. Open in iOS simulator: Press `i` in terminal or `npm run ios`
3. Open in Android emulator: Press `a` in terminal or `npm run android`
4. Test on physical device: Scan QR code with Expo Go app
5. Type checking: `npm run type-check`
6. Linting: `npm run lint`

### Testing Backend Changes
1. Start services: `cd infrastructure/docker && docker-compose up --build`
2. View logs: `docker-compose logs -f`
3. Trigger fetch: `curl -X POST http://localhost:8080/api/v1/explore/feed/fetch`
4. Check recommendations: `curl -H "Authorization: Bearer <JWT>" http://localhost:8081/api/v1/explore/recommendation/user123`
5. Mark as read: `curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/api/v1/explore/article/{article_id}/read`
6. Vote on article: `curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/api/v1/explore/article/{article_id}/vote -d '{"vote_type":"upvote"}'`
7. Run tests: `make test` or `go test ./...`

### Database Migrations
Migrations are automatically run when PostgreSQL containers are first created. To reset databases:

```bash
cd services/explore
docker-compose down
docker volume rm cairn-explore_postgres_data cairn-explore_fetcher_postgres_data
docker-compose up --build
```

For user service migrations:
```bash
cd services/users
make migrate-up              # Run migrations
make migrate-down            # Rollback
make migrate-version         # Check current version
```

### Adding New Features

> 📖 **For detailed implementation patterns, see [Engineering Principles - Common Patterns](/docs/ENGINEERING_PRINCIPLES.md#common-patterns)**

**Adding a new mobile screen**:
1. Create screen component in `apps/mobile/src/screens/`
2. Add screen to navigation in `src/navigation/RootNavigator.tsx` or `TabNavigator.tsx`
3. Update navigation types in `src/types/navigation.ts`
4. Add any new shared components to `src/components/common/`

**Adding a new backend endpoint**:
1. Add handler function in `internal/api/handlers.go`
2. Register route in `internal/api/server.go`
3. Update repository if database access needed in `internal/db/*_repository.go`
4. Add tests for new functionality

**Adding a database migration**:
1. Create numbered SQL file in `migrations/` (e.g., `003_add_new_table.sql`)
2. Update migration logic in `cmd/{service}/main.go`
3. Reset database to test migration (see above)

## Code Conventions

> 📖 **For comprehensive coding standards and style guides, see [Engineering Principles - Development Standards](/docs/ENGINEERING_PRINCIPLES.md#development-standards)**

### Go Code Style
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Return errors up the call stack; let handlers format HTTP responses
- Use `context.Context` for all database calls (enables cancellation)
- Prefer batch operations over loops (e.g., batch insert articles)
- Use `ON CONFLICT` for upsert semantics
- Always set timeouts on HTTP clients (fetcher uses 30s for HTTP, 10s for API calls)

### TypeScript/React Native Code Style
- Use functional components with hooks
- Define types for all component props
- Use TypeScript interfaces for data models
- Keep styles close to components or use theme system
- Handle loading and error states in all screens

## Environment Variables

### Explore Service - Explore Fetcher (explore_fetcher)
```bash
PORT=8080                      # HTTP server port
RECOMMENDER_URL=...            # URL to recommender service
FETCH_INTERVAL=60              # 60 seconds = 1 feed per minute
FETCH_TIMEOUT=30               # 30 seconds timeout per feed
MAX_FETCH_ERRORS=10            # Disable feed after N consecutive failures
DB_HOST=fetcher_db             # PostgreSQL host
DB_PORT=5432
DB_USER=fetcher
DB_PASSWORD=fetcher_password
DB_NAME=fetcher_db
KAGI_FEED_URL=...              # URL to Kagi Small Web feed list
```

### Explore Service - Explore Recommender (explore_recommender)
```bash
PORT=8081                      # HTTP server port
DB_HOST=postgres               # PostgreSQL host
DB_PORT=5432
DB_USER=cairn
DB_PASSWORD=cairn_password
DB_NAME=cairn_db
DB_SSLMODE=disable             # SSL mode for local dev
ARTICLE_RETENTION_DAYS=90      # Delete articles after 90 days
# Vault configuration (REQUIRED for JWT authentication)
VAULT_ADDR=http://localhost:8200       # HashiCorp Vault address
VAULT_TOKEN=...                        # Vault token
JWT_PUBLIC_KEY_PATH=secret/data/jwt/public-key  # Path to JWT public key in Vault
```

### User Service
```bash
PORT=8080                      # HTTP server port
DB_HOST=localhost              # PostgreSQL host
DB_PORT=5432
DB_USER=cairn_user
DB_PASSWORD=...                # Database password
DB_NAME=cairn_users
# Vault configuration (REQUIRED for JWT key generation and storage)
VAULT_ADDR=http://localhost:8200      # HashiCorp Vault address
VAULT_TOKEN=...                       # Vault token
JWT_ACCESS_LIFETIME=15m        # Access token lifetime
JWT_REFRESH_LIFETIME=7d        # Refresh token lifetime
SERVER_ENVIRONMENT=development # Environment mode
```

### Read Service - Content Service
```bash
DATABASE_URL=postgres://...    # PostgreSQL connection string
SERVER_PORT=8080               # HTTP server port
LOG_LEVEL=info                 # Logging level
MAX_CONTENT_SIZE=5242880       # 5MB content size limit
ORPHANED_CONTENT_DAYS=90       # Days before deleting orphaned content
```

### Read Service - Ingest RSS Service (ingest_rss)
```bash
DATABASE_URL=postgres://...    # PostgreSQL connection string
SERVER_PORT=8081               # HTTP server port
CONTENT_SERVICE_URL=...        # URL to Content Service
LOG_LEVEL=info                 # Logging level
MAX_FEEDS_PER_USER=100         # Maximum feeds per user
FEED_ERROR_THRESHOLD=7         # Days of errors before disabling feed
POLL_INTERVAL_TIER1=1h         # Active feeds poll interval
POLL_INTERVAL_TIER2=6h         # Moderate feeds poll interval
POLL_INTERVAL_TIER3=24h        # Quiet feeds poll interval
```

## Important Implementation Notes

### HashiCorp Vault Dependency

**CRITICAL**: The User Service and Explore Recommender Service require HashiCorp Vault for JWT authentication.

**What Vault is used for:**
- User Service: Generates and stores RS256 JWT signing keys
- Explore Recommender Service: Retrieves JWT public key for token verification
- All services: Secret management in production

**Development Setup:**

The centralized Docker Compose setup ([infrastructure/docker/docker-compose.yml](infrastructure/docker/docker-compose.yml)) includes:
1. Vault container running in dev mode (port 8200)
2. Automated `vault-init` service that generates RSA keys and stores them in Vault
3. All services configured to use the shared Vault instance

**Running services individually:**

If you need to run services outside Docker Compose:

1. Start Vault in dev mode:
   ```bash
   docker run -d --name vault -p 8200:8200 \
     -e VAULT_DEV_ROOT_TOKEN_ID=dev-root-token \
     hashicorp/vault:latest server -dev
   ```

2. Initialize Vault with JWT keys (see `infrastructure/docker/scripts/init-vault.sh`)

3. Configure services with Vault environment variables

**Production:** Use a properly configured Vault cluster with persistent storage, TLS, and proper authentication.

### Explore Service - Database Separation
The Explore service uses **two separate PostgreSQL databases**:
- Explore Fetcher (explore_fetcher) maintains its own database for feed management and crawling state
- Explore Recommender (explore_recommender) has its own database for articles and user engagement
- Services communicate only via HTTP APIs, never direct database access
- This enables independent scaling, clearer separation of concerns, and simpler deployment

### Explore Service - Article IDs
- Article IDs are SHA256 hashes of the article link
- Ensures uniqueness across feeds and enables deduplication
- The `feed_id` column in articles references the feeds table in the **fetcher database** (different database), so it has no foreign key constraint

### User Service - Security
- Passwords hashed using bcrypt with cost factor 12+
- JWT tokens signed with RS256 (2048-bit RSA keys)
- Refresh tokens hashed before database storage
- All secrets managed through HashiCorp Vault in production
- HTTPS required in production

### Mobile App - Data Persistence
- Uses AsyncStorage for local data persistence
- All article data stored locally (no backend integration yet)
- Article model defined in `src/types/article.ts`
- Storage service in `src/services/storage.ts`

## Documentation References

- **Engineering Principles**: `/docs/ENGINEERING_PRINCIPLES.md` - Architectural principles, coding standards, testing philosophy, and code review guidelines
- **Main README**: `/README.md` - Project overview and getting started
- **Explore Service**: `/services/explore/README.md` - RSS fetcher and recommender
- **Explore Service Plan**: `/services/explore/RECOMMENDER_PLAN.md` - Implementation roadmap
- **Explore Service Requirements**: `/services/explore/requirements.md` - Detailed requirements
- **User Service**: `/services/users/README.md` - Authentication and user management
- **Read Service**: `/services/read/README.md` - Content storage and RSS feed management
- **Read Service Requirements**: `/services/read/requirements.md` - Detailed specifications
- **Read Service Implementation**: `/services/read/IMPLEMENTATION_PLAN.md` - Implementation details
- **Mobile App Data Model**: See TypeScript interfaces in `/apps/mobile/src/types/`

## Next Steps and Planned Work

### Explore Service
- ✅ Explore Fetcher (explore_fetcher): COMPLETE (all features implemented and tested)
- 🔄 Explore Recommender (explore_recommender): See `/services/explore/RECOMMENDER_PLAN.md` for detailed plan
  - Implement voting system (upvote/downvote API)
  - Enhanced recommendation algorithm with quality scoring
  - Article cleanup (90-day retention)
  - Admin dashboard endpoints

### Read Service
- ✅ Content Service: COMPLETE (content storage, user metadata, search)
- ✅ Ingest RSS (ingest_rss): COMPLETE (feed subscriptions, tiered polling, outbox pattern)
- Integration with mobile app pending
- Planned features: recommendation engine, import/export, GraphQL API

### User Service
- ✅ Core authentication features implemented
- Integration with mobile app pending
- Additional features: rate limiting, analytics

### Mobile App
- Currently uses local AsyncStorage
- Backend integration pending
- Planned features: offline reading, sync, enhanced reader
