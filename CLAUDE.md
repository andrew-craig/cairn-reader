# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

> 📖 **For detailed service-specific guidance:**
> - [Mobile App](/apps/mobile/CLAUDE.md) - React Native/Expo mobile application
> - [Explore Service](/services/explore/CLAUDE.md) - RSS feed fetching and content recommendation
> - [Read Service](/services/read/CLAUDE.md) - Article storage and RSS feed management
> - [User Service](/services/users/CLAUDE.md) - Authentication and user management

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

> 📖 **For detailed mobile app documentation, see [apps/mobile/CLAUDE.md](/apps/mobile/CLAUDE.md)**

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

> 📖 **For detailed Explore service documentation, see [services/explore/CLAUDE.md](/services/explore/CLAUDE.md)**

```bash
cd services/explore

# Development (requires PostgreSQL running)
make run-fetcher             # Start explore_fetcher service (port 8080)
make run-recommender         # Start explore_recommender service (port 8081)

# Build and test
make build                   # Build both binaries to bin/
make test                    # Run all tests

# Docker (recommended for development)
docker-compose up --build    # Start all services with databases

# Code quality
make fmt                     # Format Go code
make lint                    # Run fmt + vet
```

### User Service (services/users)

> 📖 **For detailed User service documentation, see [services/users/CLAUDE.md](/services/users/CLAUDE.md)**

```bash
cd services/users

# Build and run
make build                   # Build binaries to bin/
make run                     # Run user service
make dev                     # Run with live reload (requires air)

# Testing
make test                    # Run all tests
make test-coverage           # Generate coverage report

# Database migrations
make migrate-up              # Run migrations
make migrate-down            # Rollback migrations

# Code quality
make fmt                     # Format code
make lint                    # Run golangci-lint
```

### Read Service (services/read)

> 📖 **For detailed Read service documentation, see [services/read/CLAUDE.md](/services/read/CLAUDE.md)**

```bash
cd services/read

# Development (requires Docker)
docker-compose up --build    # Start all services (Content + RSS Fetcher)

# Build and test
make build                   # Build both services
make test                    # Run all tests
make test-integration        # Run integration tests

# Database migrations
make migrate-up              # Apply pending migrations
make migrate-status          # Check migration status

# Code quality
make fmt                     # Format code
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

**Key Principles**:
- Each service owns its own database
- Services communicate only via HTTP REST APIs
- JWT-based authentication (stateless)
- HashiCorp Vault for secrets management

### Service Overview

#### Mobile App (port: Expo dev server)
- React Native with Expo for iOS and Android
- Supports light/dark mode with theme system
- Uses AsyncStorage for local persistence
- Service layer for backend API communication

See [apps/mobile/CLAUDE.md](/apps/mobile/CLAUDE.md) for details.

#### User Service (port 8082)
- JWT-based authentication with RS256 signing
- Refresh token management with automatic rotation
- Mobile device authentication via Expo device ID
- Account upgrade from device-only to email/password

See [services/users/CLAUDE.md](/services/users/CLAUDE.md) for details.

#### Explore Service (ports 8080, 8081)
Two microservices with separate databases:
- **Fetcher** (8080): RSS feed fetching from Kagi Small Web
- **Recommender** (8081): Article recommendations with voting

See [services/explore/CLAUDE.md](/services/explore/CLAUDE.md) for details.

#### Read Service (ports 8083, 8085)
Two microservices with separate databases:
- **Content Service** (8083): Article storage with readability extraction
- **Ingest RSS** (8085): User feed subscriptions with tiered polling

See [services/read/CLAUDE.md](/services/read/CLAUDE.md) for details.

## API Endpoints Summary

> **Note:** All services use `/health/live` (liveness) and `/health/ready` (readiness) for health checks.

### User Service (port 8082)
```
POST /api/v1/auth/register               → Create account
POST /api/v1/auth/login                  → Login
POST /api/v1/auth/refresh                → Refresh token
GET  /api/v1/user/{user_id}              → Get user profile (requires auth)
```

### Explore Service
**Fetcher (port 8080)**:
```
POST /api/v1/explore/feed/fetch          → Trigger feed fetch
GET  /api/v1/explore/feed/stats          → Get feed statistics
```

**Recommender (port 8081)**:
```
GET  /api/v1/explore/recommendation/{user_id}       → Get recommendations (requires auth)
POST /api/v1/explore/article/{id}/vote              → Vote on article (requires auth)
POST /api/v1/explore/article/{id}/read              → Mark as read (requires auth)
```

### Read Service
**Content Service (port 8083)**:
```
POST /api/v1/content/detect                         → Detect URL type (feed/page)
POST /api/v1/content/user/{user_id}                 → Add URL to user's list
GET  /api/v1/content/user/{user_id}                 → List user's contents
GET  /api/v1/content/user/{user_id}/search          → Full-text search
```

**Ingest RSS (port 8085)**:
```
POST /api/v1/source/rss/user/{user_id}/subscription → Subscribe to feed
GET  /api/v1/source/rss/user/{user_id}/subscription → List subscriptions
```

For complete API documentation, see:
- [services/explore/api/openapi.yaml](/services/explore/api/openapi.yaml)
- [services/users/api/openapi.yaml](/services/users/api/openapi.yaml)
- [services/read/api/openapi.yaml](/services/read/api/openapi.yaml)

## Testing and Development Workflow

> 📖 **For comprehensive testing standards and patterns, see [Engineering Principles - Testing Philosophy](/docs/ENGINEERING_PRINCIPLES.md#testing-philosophy)**

### Testing Mobile App Changes
```bash
cd apps/mobile
npm start                    # Start Expo dev server
npm run ios                  # Run on iOS simulator
npm run android              # Run on Android emulator
npm run type-check           # TypeScript validation
npm run lint                 # ESLint
```

### Testing Backend Changes
```bash
# Start all services
cd infrastructure/docker
docker-compose up --build

# View logs
docker-compose logs -f

# Run tests
cd services/{service}
make test
```

### Database Migrations

Migrations are automatically run when PostgreSQL containers start. To reset databases:

```bash
cd services/{service}
docker-compose down
docker volume rm {volume-name}
docker-compose up --build
```

Or use migration commands:
```bash
make migrate-up              # Apply migrations
make migrate-down            # Rollback
make migrate-status          # Check status
```

## Code Conventions

> 📖 **For comprehensive coding standards and style guides, see [Engineering Principles - Development Standards](/docs/ENGINEERING_PRINCIPLES.md#development-standards)**

### Go Code Style
- Use `fmt.Errorf("context: %w", err)` for error wrapping
- Return errors up the call stack; let handlers format HTTP responses
- Use `context.Context` for all database calls
- Prefer batch operations over loops
- Use `ON CONFLICT` for upsert semantics
- Always set timeouts on HTTP clients

### TypeScript/React Native Code Style
- Use functional components with hooks
- Define types for all component props
- Use TypeScript interfaces for data models
- Keep styles close to components or use theme system
- Handle loading and error states in all screens

## Important Cross-Service Notes

### HashiCorp Vault Dependency

**CRITICAL**: User Service and Explore Recommender require HashiCorp Vault for JWT authentication.

**What Vault is used for:**
- User Service: Generates and stores RS256 JWT signing keys
- Explore Recommender: Retrieves JWT public key for token verification
- All services: Secret management in production

**Development Setup:**

The centralized Docker Compose setup ([infrastructure/docker/docker-compose.yml](infrastructure/docker/docker-compose.yml)) includes:
1. Vault container in dev mode (port 8200)
2. Automated `vault-init` service that generates RSA keys
3. All services configured to use shared Vault instance

**Production:** Use properly configured Vault cluster with persistent storage, TLS, and proper authentication.

### Authentication Flow

1. **User Service** generates JWT tokens signed with RSA private key (from Vault)
2. **Other services** validate JWT tokens using RSA public key (from Vault)
3. All user-specific endpoints require valid JWT in `Authorization: Bearer <token>` header
4. Services extract `user_id` from JWT claims for authorization

### Service Communication

- **ALWAYS** use REST APIs for inter-service communication
- **NEVER** access another service's database directly
- Each service owns its own database
- Use appropriate HTTP clients with timeouts and retry logic

### Database Schema Notes

**User Service** (`cairn_users`):
- `users`: User accounts (email/password or device ID)
- `refresh_tokens`: JWT refresh token management

**Explore Fetcher** (`fetcher_db`):
- `feeds`: RSS feed sources with health tracking

**Explore Recommender** (`cairn_db`):
- `articles`: SHA256-hashed article IDs for deduplication
- `votes`: User voting (upvote/downvote)
- `user_articles`: Read status tracking

**Content Service** (`content_service`):
- `contents`: Cleaned article content with deduplication
- `user_contents`: User-specific metadata (status, favorites, scroll position)

**Ingest RSS** (`ingest_rss`):
- `feeds`: RSS feed metadata with tiered polling
- `user_feeds`: User subscriptions (100 feed limit)
- `outbox`: Reliable content delivery queue

## Documentation References

### Service-Specific Documentation
- **Mobile App**: [apps/mobile/CLAUDE.md](/apps/mobile/CLAUDE.md) - React Native app details
- **Explore Service**: [services/explore/CLAUDE.md](/services/explore/CLAUDE.md) - RSS fetching and recommendations
- **Read Service**: [services/read/CLAUDE.md](/services/read/CLAUDE.md) - Content storage and feed management
- **User Service**: [services/users/CLAUDE.md](/services/users/CLAUDE.md) - Authentication and user management

### Project-Wide Documentation
- **Engineering Principles**: [docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md) - Architectural principles, coding standards, testing philosophy
- **Main README**: [README.md](/README.md) - Project overview and getting started
- **Infrastructure**: [infrastructure/docker/README.md](/infrastructure/docker/README.md) - Docker Compose setup

### Service READMEs
- [services/explore/README.md](/services/explore/README.md) - Explore service details
- [services/users/README.md](/services/users/README.md) - User service details
- [services/read/README.md](/services/read/README.md) - Read service details

## Implementation Status

### Mobile App
- ✅ Core UI components and navigation
- ✅ Theme system (light/dark mode)
- ✅ Local storage with AsyncStorage
- 🔄 Backend integration pending

### User Service
- ✅ JWT authentication (RS256)
- ✅ Refresh token rotation
- ✅ Mobile device authentication
- ✅ Account upgrade flow

### Explore Service
- ✅ Fetcher: RSS feed fetching from Kagi Small Web
- ✅ Recommender: Voting system and recommendation algorithm
- ✅ Article cleanup (90-day retention)

### Read Service
- ✅ Content Service: Content storage with readability extraction
- ✅ Ingest RSS: Feed subscriptions with tiered polling
- ✅ Outbox pattern for reliable delivery
- ✅ Content update detection

## Getting Help

For service-specific questions:
- **Mobile App**: See [apps/mobile/CLAUDE.md](/apps/mobile/CLAUDE.md)
- **Explore Service**: See [services/explore/CLAUDE.md](/services/explore/CLAUDE.md)
- **Read Service**: See [services/read/CLAUDE.md](/services/read/CLAUDE.md)
- **User Service**: See [services/users/CLAUDE.md](/services/users/CLAUDE.md)

For project-wide issues:
- Check logs: `docker-compose logs -f <service-name>`
- Review tests for usage examples
- Consult OpenAPI specs for API details
- See [docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md) for conventions
