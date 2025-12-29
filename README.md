# Cairn

A modern read-it-later application with a React Native mobile app and Go-based backend services for content discovery, storage, and user management.

## Overview

Cairn is a monorepo containing:
- **Mobile App**: React Native application for iOS and Android
- **Backend Services**: Go microservices for content management and discovery
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

- RSS feed discovery and automatic content delivery
- Content recommendation engine
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
│   ├── explore/             # RSS Fetcher & Recommendation Engine
│   │   ├── fetcher/         # RSS feed fetching service
│   │   ├── recommender/     # Content recommendation service
│   │   ├── pkg/             # Shared packages
│   │   └── README.md        # Explore service documentation
│   │
│   ├── read/                # Content Storage & RSS Feed Management
│   │   ├── content/         # Content Service (storage, search, user metadata)
│   │   ├── fetcher/         # RSS Fetcher Service (feed subscriptions, polling)
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
cd infrastructure/docker
docker compose up --build
```

✅ **All backend services are validated and working!** See [DOCKER_VALIDATION_SUCCESS.md](DOCKER_VALIDATION_SUCCESS.md) for validation report.

For comprehensive deployment instructions, see [DEPLOYMENT.md](DEPLOYMENT.md).

#### Running Individual Services

See the documentation for each service:
- [Explore Service](services/explore/README.md) - RSS fetching and recommendations
- [Read Service](services/read/README.md) - Content storage and RSS feed management
- [User Service](services/users/README.md) - Authentication and user management

## Services Documentation

### Explore Service
The Explore service consists of two microservices:
- **Fetcher**: Discovers and fetches content from RSS feeds
- **Recommender**: Stores content and implements recommendation algorithms

Features:
- Automatic RSS feed polling
- Content deduplication
- Recommendation engine for suggesting articles
- PostgreSQL storage

See the [Explore Service README](services/explore/README.md) for detailed documentation.

### Read Service
The Read service is a comprehensive article storage and RSS feed management system consisting of two microservices:

**Content Service** (port 8080):
- Cleaned HTML content extraction using go-readability
- HTML sanitization with bluemonday
- Content deduplication by hash and feed ID
- User-specific metadata (read status, scroll position, favorites, notes)
- Full-text search with PostgreSQL GIN index
- Cursor-based pagination
- Orphaned content cleanup (90-day retention)

**RSS Fetcher Service** (port 8081):
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

### Reading List (Home)
- Displays all unread articles
- Pull to refresh
- Tap to view article details
- Heart icon to toggle favorites

### Explore
- Discover new content from RSS feeds
- Browse recommended articles
- Subscribe to feeds

### Read
- Access saved articles
- Track reading progress

### Favorites
- Shows all favorited articles
- Same interactions as Reading List

### Archive
- Displays all read articles
- Helps keep your reading list clean

### Settings
- App information
- Account management
- Clear all articles option

### Add Article
- Add new articles by URL
- Paste from clipboard
- Manual title entry

### Article Detail
- View article information
- Open in browser
- Mark as read/unread
- Toggle favorite status
- Share article
- Delete article

## Data Model

### Article Interface

```typescript
interface Article {
  id: string;
  url: string;
  title: string;
  description?: string;
  imageUrl?: string;
  author?: string;
  publishedDate?: string;
  readingTime?: number;
  tags: string[];
  isRead: boolean;
  isFavorite: boolean;
  addedAt: number;
  readAt?: number;
  notes?: string;
}
```

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
┌───▼────────┐  ┌──────────────┐  ┌───────▼──────┐
│   User     │  │   Explore    │  │     Read     │
│  Service   │  │   Service    │  │   Service    │
│            │  │              │  │              │
│  - Auth    │  │  - Fetcher   │  │  - Content   │
│  - JWT     │  │  - Recommender│  │  - Storage   │
└─────┬──────┘  └──────┬───────┘  └──────┬───────┘
      │                │                  │
      │                │                  │
      └────────────────┼──────────────────┘
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

## Future Enhancements

### Mobile App
- [ ] Offline reading mode
- [ ] Reading progress sync
- [ ] Enhanced article reader with adjustable fonts
- [ ] Dark mode scheduling
- [ ] Export/import data

### Backend Services
- [x] Full-text search across articles (implemented in Read Service)
- [x] RSS feed management and polling (implemented in Read Service)
- [ ] Article summarization with AI
- [ ] Email digest notifications
- [ ] Image hosting and optimization
- [ ] Reading analytics and statistics
- [ ] Social features (sharing, comments)
- [ ] Recommendation engine (Read Service)
- [ ] Import/export functionality (Read Service)
- [ ] GraphQL API option (Read Service)

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
# Explore service
cd services/explore
go test ./...

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
- [Explore Service Setup](services/explore/README.md)
- [Read Service Migrations](services/read/README.md)

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
