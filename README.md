# Cairn

A modern read-it-later application with a React Native mobile app and Go-based backend services for content ingestion, storage, and user management.

## Overview

Cairn is a monorepo containing:
- **Mobile App**: React Native application for iOS and Android
- **Backend Services**: Go microservices for content management and ingestion
- **Infrastructure**: Docker configuration and deployment scripts

### Mobile App Features

- 📚 Save articles for later reading
- ⭐ Mark articles as favorites
- ✅ Track read/unread status
- 🗂️ Organize articles in archive
- 🔗 Open articles in browser
- 📤 Share articles
- 🌓 Dark mode support
- 💾 Local storage with AsyncStorage

### Backend Features

- RSS feed subscriptions and automatic content delivery
- Feed and Reads lists, with the destination chosen per source
- User authentication with JWT
- Article storage with cleaned HTML content
- Multi-user content deduplication

## Tech Stack

### Mobile App
- **Framework**: React Native with Expo
- **Navigation**: React Navigation (Stack & Bottom Tabs)
- **Language**: TypeScript
- **Storage**: AsyncStorage
- **UI**: React Native core components
- **Icons**: Expo Vector Icons (Ionicons)

### Backend Services
- **Language**: Go 1.23+
- **Database**: PostgreSQL
- **Authentication**: JWT with RS256 signing
- **Secrets Management**: HashiCorp Vault
- **Deployment**: Docker & Docker Compose

## Project Structure

```
cairn/
├── apps/
│   └── mobile/              # React Native mobile app
│       ├── src/
│       │   ├── components/  # Reusable UI components
│       │   ├── screens/     # App screens
│       │   ├── navigation/  # Navigation configuration
│       │   ├── services/    # Services (Storage, API, etc.)
│       │   ├── types/       # TypeScript type definitions
│       │   ├── utils/       # Utility functions
│       │   └── constants/   # App constants (theme, colors, etc.)
│       ├── assets/          # Images, fonts, and other assets
│       ├── App.tsx          # Entry point
│       └── package.json
│
├── services/
│   ├── read/                # Content Storage & RSS Feed Management
│   │   ├── content/         # Content Service (storage, search, user metadata)
│   │   ├── fetcher/         # Ingest RSS (ingest_rss) - feed subscriptions, polling
│   │   ├── api/             # OpenAPI specifications
│   │   └── README.md        # Read service documentation
│   │
│   └── users/               # User Management & Authentication
│       ├── cmd/             # Application entrypoints
│       ├── internal/        # Private application code
│       ├── pkg/             # Public shared libraries
│       ├── migrations/      # Database migrations
│       └── README.md        # User service documentation
│
├── infrastructure/
│   └── docker/              # Docker configurations
│
├── packages/                # Shared packages across services
│
└── scripts/                 # Build and deployment scripts
```

## Getting Started

### Prerequisites

- Node.js (v16 or later)
- npm or yarn
- Expo CLI (optional, but recommended)
- Go 1.23+ (for backend services)
- Docker & Docker Compose (for running backend services)
- PostgreSQL 14+ (if running services locally without Docker)

### Environment Configuration

Before running backend services with Docker, you need to configure environment variables for sensitive data like database passwords and API tokens.

#### Setting Up Environment Files

1. **Infrastructure Services**:
   ```bash
   cd infrastructure/docker/dev
   cp .env.example .env
   # Edit .env with your configuration
   ```

2. **Read Service** (when running standalone):
   ```bash
   cd services/read
   cp .env.example .env
   # Edit .env with your configuration
   ```

#### Environment File Structure

Each `.env.example` file contains template values. Copy it to `.env` and update with your actual values:

- **NEVER commit `.env` files** to version control (already in `.gitignore`)
- Use strong, unique passwords for production
- Keep development passwords simple for local testing
- Update tokens and secrets for each environment

Example `.env` file structure:
```bash
# PostgreSQL Configuration
POSTGRES_USER=your_username
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_DB=database_name

# Vault Configuration (infrastructure only)
VAULT_DEV_ROOT_TOKEN_ID=your_vault_token_here
```

### Mobile App Development

1. Navigate to the mobile app directory:

```bash
cd apps/mobile
```

2. Install dependencies:

```bash
npm install
```

3. Start the development server:

```bash
npm start
```

4. Run on iOS simulator:

```bash
npm run ios
```

5. Run on Android emulator:

```bash
npm run android
```

#### Using Expo Go

1. Install Expo Go on your iOS or Android device
2. Run `npm start` to start the development server
3. Scan the QR code with:
   - iOS: Camera app
   - Android: Expo Go app

### Backend Services Development

#### Running All Services with Docker (Recommended)

```bash
cd infrastructure/docker/dev
docker compose up --build
```

✅ **All backend services are validated and working!** See [DOCKER_VALIDATION_SUCCESS.md](DOCKER_VALIDATION_SUCCESS.md) for validation report.

For comprehensive deployment instructions, see [DEPLOYMENT.md](DEPLOYMENT.md).

#### Running Individual Services

See the documentation for each service:
- [Read Service](services/read/README.md) - Content storage and RSS feed management
- [User Service](services/users/README.md) - Authentication and user management

## Services Documentation

### Read Service
The Read service is a comprehensive article storage and RSS feed management system consisting of two microservices:

**Content Service** (port 8080):
- Cleaned HTML content extraction using go-readability
- HTML sanitization with bluemonday
- Content deduplication by hash and feed ID
- User-specific metadata (read status, scroll position, favorites, notes)
- Full-text search with PostgreSQL GIN index
- Cursor-based pagination
- Feed vs Reads routing: a per-source destination list (`source_routes`, `user_contents.list`); new items only
- Feed retention (non-favorited Feed items deleted after 30 days) and orphaned content cleanup (90 days)

**Ingest RSS Service** (ingest_rss, port 8081):
- User feed subscriptions (100 feed limit per user)
- Tiered polling strategy (hourly/6-hourly/daily)
- Content extraction and processing
- Outbox pattern for reliable content delivery
- Circuit breaker for fault tolerance
- Auto-disable feeds after 7 consecutive error days
- Update detection via ETag/Last-Modified headers

See the [Read Service README](services/read/README.md) for detailed documentation.

### User Service
The User service handles authentication and account management.

Features:
- User registration with email/password or mobile device ID
- Stateless JWT authentication with RS256 signing
- Refresh token management with automatic rotation
- Account upgrade from device-only to email/password
- Secure secrets management with HashiCorp Vault

See the [User Service README](services/users/README.md) for detailed documentation.

## Mobile App Screens

Navigation lives in `apps/mobile/src/navigation`: a root stack (`RootNavigator`) wrapping a bottom tab navigator (`TabNavigator`) with three tabs, **Feed | Reads | You**. Screens are in `apps/mobile/src/screens`.

### Tabs
- **Feed** (`FeedScreen`): skim list of items from sources routed to Feed. Online-only; non-favorited items expire after 30 days.
- **Reads** (`ReadsScreen`): items from sources routed to Reads, plus directly saved links. Offline-readable, with search and reading progress.
- **You** (`YouScreen`): profile hub with subscription and bookmark counts and links to the screens below.

### Stack screens
- **ArticleDetail** (`ReadArticleDetailScreen`): the reader, shared by Feed and Reads. Favorite and Next actions; "Save to Reads" for Feed items, Archive for Reads items.
- **AddArticle** (`AddArticleScreen`): save an article by URL.
- **Bookmarks** (`BookmarksScreen`): favorited articles.
- **RSS** and **Newsletters** (`RssScreen`, `NewslettersScreen`): subscriptions, with a per-source Feed/Reads toggle.
- **Account** (`AccountScreen`) and **About** (`AboutScreen`): account settings and app/server versions.
- **Login** (`LoginScreen`): shown instead of the stack when signed out.

## Data Model

The `Article` type, along with the auth and Read service API types, is shared by the mobile and web clients and defined in `apps/shared/src/types` (`article.ts`, `auth.ts`, `read.ts`, exported from `@cairn/shared`).

## Theme

The app supports both light and dark modes, automatically following the system preference. Color schemes are defined in `src/constants/theme.ts`.

## Architecture

Cairn follows a microservices architecture:

```
┌─────────────────┐
│   Mobile App    │
│ (React Native)  │
└────────┬────────┘
         │
         │ REST APIs
         │
    ┌────┴─────────────────────────────────┐
    │                                      │
    │                                      │
┌───▼────────┐                          ┌───────▼──────┐
│   User     │                          │  Read Service│
│  Service   │                          │              │
│            │                          │  - Content   │
│  - Auth    │                          │  - ingest_rss│
│  - JWT     │                          │              │
└─────┬──────┘                          └──────┬───────┘
      │                                        │
      └────────────────────┬───────────────────┘
                           │
                      ┌────▼─────┐
                      │PostgreSQL│
                      └──────────┘
```

### Service Communication
- All services expose REST APIs
- Services are independently deployable
- Shared data models in PostgreSQL
- JWT-based authentication across services

## Development

### Mobile App Development

#### Type Checking

```bash
cd apps/mobile
npm run type-check
```

#### Linting

```bash
cd apps/mobile
npm run lint
```

#### Building for Production

##### iOS

```bash
cd apps/mobile
expo build:ios
```

##### Android

```bash
cd apps/mobile
expo build:android
```

### Backend Services Development

#### Running Tests

```bash
# User service
cd services/users
go test ./...

# Read service
cd services/read
make test                    # Run all tests
make test-coverage           # Generate coverage report
make test-integration        # Run integration tests
```

#### Running with Live Reload

```bash
# Install air for live reloading
go install github.com/cosmtrek/air@latest

# Run with air in each service directory
cd services/users
air
```

#### Database Migrations

See individual service READMEs for migration instructions:
- [User Service Migrations](services/users/README.md)
- [Read Service Migrations](services/read/README.md)

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
