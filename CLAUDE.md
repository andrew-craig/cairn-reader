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
cd infrastructure/docker/dev

# Copy and configure environment variables
cp .env.example .env
# Edit .env and set secure passwords

# Start all services (Vault, databases, and all backend services)
docker compose up --build -d

# Check service status
docker compose ps

# View logs
docker compose logs -f

# Stop all services
docker compose down
```

This starts:
- HashiCorp Vault (port 8200) with auto-initialization
- Consolidated PostgreSQL database (port 5432) with all service databases
- User Service (port 8082)
- Explore Recommender Service (port 8081)
- Explore Fetcher Service (port 8080)
- Content Service (port 8083)
- Ingest RSS Service (port 8085)
- Background workers (ports 8084, 8086)

See [infrastructure/docker/README.md](infrastructure/docker/README.md) for detailed documentation.

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
- Each service owns its own logical database (hosted in a single PostgreSQL instance)
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
GET  /api/v1/explore/user/{user_id}/votes           → Get user's voted articles (requires auth)
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
cd infrastructure/docker/dev
docker compose up --build

# View logs
docker compose logs -f

# Run tests
cd services/{service}
make test
```

### Database Migrations

Migrations are automatically run when PostgreSQL containers start. To reset databases:

```bash
cd infrastructure/docker/dev
docker compose down -v
docker compose up --build
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

The centralized Docker Compose setup ([infrastructure/docker/dev/docker-compose.yml](infrastructure/docker/dev/docker-compose.yml)) includes:
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
- Each service owns its own logical database (shared PostgreSQL instance, separate DBs)
- Use appropriate HTTP clients with timeouts and retry logic


## Documentation References

### Service-Specific Documentation
- **Mobile App**: [apps/mobile/CLAUDE.md](/apps/mobile/CLAUDE.md) - React Native app details
- **Explore Service**: [services/explore/CLAUDE.md](/services/explore/CLAUDE.md), [services/explore/README.md](/services/explore/README.md)
- **Read Service**: [services/read/CLAUDE.md](/services/read/CLAUDE.md), [services/read/README.md](/services/read/README.md)
- **User Service**: [services/users/CLAUDE.md](/services/users/CLAUDE.md), [services/users/README.md](/services/users/README.md)

### Project-Wide Documentation
- **Engineering Principles**: [docs/ENGINEERING_PRINCIPLES.md](/docs/ENGINEERING_PRINCIPLES.md) - Architectural principles, coding standards, testing philosophy
- **Main README**: [README.md](/README.md) - Project overview and getting started
- **Infrastructure**: [infrastructure/docker/README.md](/infrastructure/docker/README.md) - Docker Compose setup


## Task Tracking

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds