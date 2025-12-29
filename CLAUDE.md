# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Cairn is a read-it-later application consisting of:
- **Mobile App** (React Native/Expo): iOS and Android app for reading saved articles
- **Backend Services** (Go): Microservices for content discovery, storage, and user management
  - **Explore Service**: RSS feed fetching and content recommendation
  - **Read Service**: Article storage and user-specific metadata (planned)
  - **User Service**: Authentication and account management

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
make run-fetcher             # Start fetcher service (port 8080)
make run-recommender         # Start recommender service (port 8081)
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
Fetcher DB ← Fetcher Service (8080) → HTTP POST → Recommender Service (8081) → Recommender DB
```

**Key architectural principle**: Each service owns its own database. Services communicate only via HTTP APIs.

- **Fetcher Service**: Manages feed sources in its own PostgreSQL database, fetches RSS content (1 feed/minute), sends successfully fetched articles to recommender via HTTP POST
- **Recommender Service**: Stores articles in its own PostgreSQL database, manages user engagement (upvotes/downvotes), serves personalized recommendations

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

### Mobile App - Theme System
- Supports light and dark modes (follows system preference)
- Color schemes defined in `src/constants/theme.ts`
- Automatically applies theme across all components

## API Endpoints

### Explore Service - Fetcher (port 8080)
```
GET  /health              → Health check
POST /fetch               → Manually trigger feed fetch
```

### Explore Service - Recommender (port 8081)
```
GET  /health                               → Health check
POST /explore/articles                     → Submit articles (from fetcher)
GET  /explore/recommendations/{userID}     → Get 5 recommendations (requires auth)
POST /explore/articles/read                → Mark article as read (requires auth)
     Body: {"article_id": "..."} (user_id from JWT)
POST /explore/articles/{articleID}/vote    → Vote on article (requires auth)
     Body: {"vote_type": "upvote|downvote"} (user_id from JWT)
DELETE /explore/articles/{articleID}/vote  → Remove vote (requires auth)
GET  /explore/articles/{articleID}/votes   → Get vote counts (requires auth)
```

### User Service (port 8080)
```
# Authentication
POST /auth/register                    → Create account with email/password
POST /auth/register/mobile             → Create mobile-only account (Expo device ID)
POST /auth/login                       → Login and return tokens
POST /auth/login/mobile                → Authenticate with device ID
POST /auth/refresh                     → Exchange refresh token for new access token
POST /auth/logout                      → Revoke specific refresh token
POST /auth/logout-all                  → Revoke all refresh tokens

# User Management
GET    /users/{id}                     → Get user profile (authenticated)
PATCH  /users/{id}                     → Update user profile (authenticated)
POST   /users/{id}/upgrade             → Add email/password to mobile-only account
DELETE /users/{id}                     → Delete account (authenticated)

# Health
GET /health                            → Basic health check
GET /ready                             → Readiness check (DB + Vault connectivity)
```

### OpenAPI/Swagger Documentation

Formal API specifications are available in OpenAPI 3.0 format:

- **Explore Service**: `services/explore/api/openapi.yaml`
- **User Service**: `services/users/api/openapi.yaml`

**Viewing the API Documentation:**

Using Swagger UI (requires Docker):
```bash
# Explore Service API
docker run -p 8082:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/explore/api:/api swaggerapi/swagger-ui

# User Service API
docker run -p 8083:8080 -e SWAGGER_JSON=/api/openapi.yaml \
  -v $(pwd)/services/users/api:/api swaggerapi/swagger-ui
```

Then visit:
- Explore Service: http://localhost:8082
- User Service: http://localhost:8083

**Validating OpenAPI Specs:**
```bash
# Validate Explore Service spec
npx @apidevtools/swagger-cli validate services/explore/api/openapi.yaml

# Validate User Service spec
npx @apidevtools/swagger-cli validate services/users/api/openapi.yaml
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
1. Start services: `cd services/explore && docker-compose up --build`
2. View logs: `docker-compose logs -f`
3. Trigger fetch: `curl -X POST http://localhost:8080/fetch`
4. Check recommendations: `curl -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/recommendations/user123`
5. Mark as read: `curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/read -d '{"article_id":"..."}'`
6. Vote on article: `curl -X POST -H "Authorization: Bearer <JWT>" http://localhost:8081/explore/articles/{id}/vote -d '{"vote_type":"upvote"}'`
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

### Explore Service - Fetcher
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

### Explore Service - Recommender
```bash
PORT=8081                      # HTTP server port
DB_HOST=postgres               # PostgreSQL host
DB_PORT=5432
DB_USER=cairn
DB_PASSWORD=cairn_password
DB_NAME=cairn_db
ARTICLE_RETENTION_DAYS=90      # Delete articles after 90 days
```

### User Service
```bash
PORT=8080                      # HTTP server port
DB_HOST=localhost              # PostgreSQL host
DB_PORT=5432
DB_USER=cairn_user
DB_PASSWORD=...                # Database password
DB_NAME=cairn_users
VAULT_ADDR=...                 # HashiCorp Vault address
VAULT_TOKEN=...                # Vault token
JWT_ACCESS_LIFETIME=15m        # Access token lifetime
JWT_REFRESH_LIFETIME=7d        # Refresh token lifetime
```

## Important Implementation Notes

### Explore Service - Database Separation
The Explore service uses **two separate PostgreSQL databases**:
- Fetcher maintains its own database for feed management and crawling state
- Recommender has its own database for articles and user engagement
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
- **Read Service**: `/services/read/requirements.md` - Content storage requirements
- **Mobile App Data Model**: See TypeScript interfaces in `/apps/mobile/src/types/`

## Next Steps and Planned Work

### Explore Service
- ✅ Fetcher service: COMPLETE (all features implemented and tested)
- 🔄 Recommender service: See `/services/explore/RECOMMENDER_PLAN.md` for detailed plan
  - Implement voting system (upvote/downvote API)
  - Enhanced recommendation algorithm with quality scoring
  - Article cleanup (90-day retention)
  - Admin dashboard endpoints

### Read Service
- Not yet implemented (see `/services/read/requirements.md` for specifications)
- Will provide content storage and user-specific metadata
- Integration with mobile app for article saving and reading

### User Service
- ✅ Core authentication features implemented
- Integration with mobile app pending
- Additional features: rate limiting, analytics

### Mobile App
- Currently uses local AsyncStorage
- Backend integration pending
- Planned features: offline reading, sync, enhanced reader
