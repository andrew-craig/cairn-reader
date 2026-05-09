# Cairn Backend Architecture

This document provides a detailed overview of Cairn's architecture, design decisions, data flows, and technical implementation across all backend services.

## Table of Contents

- [System Overview](#system-overview)
- [User Service](#user-service)
- [Explore Service](#explore-service)
- [Read Service](#read-service)
- [Security Considerations](#security-considerations)
- [Performance Considerations](#performance-considerations)
- [Monitoring & Observability](#monitoring--observability)

## System Overview

Cairn is a microservices-based read-it-later application backend designed for scalability, reliability, and maintainability. The system consists of multiple independent services that communicate via REST APIs.

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Client Layer                        │
│                    (Mobile App / Web App)                   │
└────────────────────────────┬────────────────────────────────┘
                             │
            ┌────────────────┼────────────────┐
            │                │                │
    ┌───────▼──────┐  ┌──────▼───────┐  ┌─────▼────────┐
    │ User Service │  │   Explore    │  │     Read     │
    │    :8082     │  │   Service    │  │   Service    │
    │              │  │  :8080/:8081 │  │  :8083/:8084 │
    │  - Auth      │  │              │  │              │
    │  - Users     │  │  - RSS Feeds │  │  - Content   │
    │  - JWT       │  │  - Recommend │  │  - Articles  │
    └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
           │                 │                 │
    ┌──────▼────┐     ┌──────▼──────┐   ┌─────▼──────┐
    │PostgreSQL │     │ PostgreSQL  │   │ PostgreSQL │
    │  (users)  │     │ (explore)   │   │   (read)   │
    └───────────┘     └─────────────┘   └────────────┘
```

### Design Principles

1. **Service Isolation**: Each service owns its data; no direct database access across services
2. **REST-Only Communication**: Services communicate exclusively via HTTP REST APIs
3. **Eventual Consistency**: Acceptable for content delivery; outbox pattern ensures reliability
4. **Idempotency**: Operations designed to be safely retried
5. **Fail-Safe Defaults**: Graceful degradation when dependencies are unavailable
6. **Single Responsibility**: Each service has a clear, focused purpose

---

## User Service

The User Service is responsible for managing user access to the Cairn platform, including user registration, authentication, and account management. It provides stateless JWT-based authentication with secure key management through HashiCorp Vault.

### Service Purpose and Responsibilities

**Core Responsibilities**:
- User account creation and management (email/password or mobile device ID)
- Stateless JWT authentication with RS256 signing
- Refresh token management with automatic rotation
- Mobile device authentication via Expo device ID
- Account upgrade from device-only to email/password
- Authorization middleware ensuring users can only access their own data
- Secure secrets management with HashiCorp Vault

**Technology Stack**:
- Go 1.21+
- Gin HTTP framework
- PostgreSQL for user data and refresh tokens
- HashiCorp Vault for key management
- JWT with RS256 (2048-bit RSA keys)
- Bcrypt for password hashing (cost factor 12+)

### Authentication Flow

#### Registration and Login Flow

```
┌─────────────────────────────────────────────────────────┐
│              Email/Password Registration                 │
└─────────────────────────────────────────────────────────┘

Client                    User Service              Database
  │                            │                        │
  ├──POST /auth/register──────>│                        │
  │  {email, password}         │                        │
  │                            ├──Hash password─────────>│
  │                            ├──Create user record────>│
  │                            ├──Generate JWT tokens───>│
  │<───{access_token,──────────┤                        │
  │     refresh_token}         │                        │


┌─────────────────────────────────────────────────────────┐
│              Mobile Device Registration                  │
└─────────────────────────────────────────────────────────┘

Client                    User Service              Database
  │                            │                        │
  ├──POST /auth/register/mobile>│                       │
  │  {expo_device_id}          │                        │
  │                            ├──Create user record────>│
  │                            ├──Generate JWT tokens───>│
  │<───{access_token,──────────┤                        │
  │     refresh_token}         │                        │
```

#### Token Refresh Flow

```
Client                    User Service              Database
  │                            │                        │
  ├──POST /auth/refresh───────>│                        │
  │  {refresh_token}           │                        │
  │                            ├──Hash & lookup token──>│
  │                            ├──Validate expiry───────>│
  │                            ├──Rotate refresh token─>│
  │                            ├──Generate new access───>│
  │<───{access_token,──────────┤                        │
  │     refresh_token}         │                        │
```

### JWT Token Management

**Access Tokens**:
- Short-lived tokens (default: 15 minutes)
- Signed with RS256 (2048-bit RSA private key)
- Contains user_id claim for authorization
- Stateless validation using public key
- Standard claims: issuer, audience, subject, expiry

**Refresh Tokens**:
- Long-lived tokens (default: 7 days)
- Stored as SHA-256 hash in database
- Supports automatic rotation on refresh
- Tracks device info and IP address for security
- Token family tracking for reuse detection
- Cascade deletion when user is deleted

**Token Claims Structure**:
```json
{
  "user_id": "uuid-v4",
  "iss": "cairn-user-service",
  "aud": ["cairn-api"],
  "sub": "uuid-v4",
  "iat": 1234567890,
  "exp": 1234567890,
  "nbf": 1234567890
}
```

### HashiCorp Vault Integration

**Purpose**: Secure storage and management of cryptographic keys and secrets

**Vault Storage**:
- RSA private key (2048-bit) for JWT signing
- RSA public key for JWT validation
- Database credentials (optional, for production)
- Support for key rotation

**Vault Authentication**:
- Token authentication for development
- AppRole authentication for production
- Automatic token renewal
- Health checks for Vault connectivity

**Key Rotation Support**:
- Background key rotation manager
- Configurable rotation interval
- Atomic key updates with rollback on failure
- Callback support for notifying dependent services

### API Endpoints

**Authentication Endpoints** (Rate Limited):
```
POST /auth/register           # Create account with email/password
POST /auth/register/mobile    # Create mobile-only account (Expo device ID)
POST /auth/login              # Login with email/password
POST /auth/login/mobile       # Login with Expo device ID
POST /auth/refresh            # Exchange refresh token for new access token
POST /auth/logout             # Revoke specific refresh token
POST /auth/logout-all         # Revoke all refresh tokens for user (requires auth)
```

**User Management Endpoints** (All require JWT authentication):
```
GET    /users/:id             # Get user profile
PATCH  /users/:id             # Update user profile
POST   /users/:id/upgrade     # Add email/password to mobile-only account
DELETE /users/:id             # Delete account and all data
```

**Health Endpoints**:
```
GET /health                   # Liveness check (is process running?)
GET /ready                    # Readiness check (DB + Vault connectivity)
```

### Database Schema

#### Users Table

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE,                    -- NULL for mobile-only
    password_hash VARCHAR(255),                    -- NULL for mobile-only
    expo_device_id VARCHAR(255) UNIQUE,            -- NULL for email-only
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_login_at TIMESTAMP WITH TIME ZONE,

    -- Account type constraints
    CONSTRAINT check_account_type CHECK (
        (email IS NOT NULL AND password_hash IS NOT NULL) OR
        (expo_device_id IS NOT NULL)
    ),
    CONSTRAINT check_email_with_password CHECK (
        (email IS NULL AND password_hash IS NULL) OR
        (email IS NOT NULL AND password_hash IS NOT NULL)
    )
);

CREATE INDEX idx_users_email ON users(email) WHERE email IS NOT NULL;
CREATE INDEX idx_users_expo_device_id ON users(expo_device_id) WHERE expo_device_id IS NOT NULL;
```

**Key Fields**:
- `id`: Primary UUID identifier
- `email`: Email address (NULL for mobile-only accounts)
- `password_hash`: Bcrypt hash (NULL for mobile-only accounts)
- `expo_device_id`: Expo Application Installation ID (NULL for email-only accounts)
- `last_login_at`: Timestamp of last successful login

#### Refresh Tokens Table

```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL UNIQUE,       -- SHA-256 hash
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    device_info TEXT,                               -- User agent or device info
    ip_address VARCHAR(45),                         -- IPv4 or IPv6
    token_family UUID                               -- For rotation tracking
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens(token_hash);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_token_family ON refresh_tokens(token_family) WHERE token_family IS NOT NULL;
```

**Key Fields**:
- `token_hash`: SHA-256 hash of the refresh token (not stored in plaintext)
- `token_family`: UUID to track token rotation chains for reuse detection
- `device_info`: User agent or device information for security tracking
- `ip_address`: IP address from which token was created

### Security Considerations

**Password Security**:
- Bcrypt hashing with cost factor 12+ (configurable)
- Automatic cost increase as hardware improves
- Passwords never stored in plaintext or logged

**Token Security**:
- RS256 asymmetric signing (more secure than HS256)
- 2048-bit RSA keys stored in Vault
- Refresh tokens hashed before database storage (SHA-256)
- Token rotation on refresh to limit exposure
- Token family tracking to detect reuse attacks

**API Security**:
- Rate limiting on authentication endpoints (10 requests/minute per IP)
- HTTPS required in production
- CORS middleware with configurable origins
- Security headers (CSP, X-Frame-Options, X-Content-Type-Options)
- Recovery middleware to prevent panic crashes

**Authorization**:
- JWT middleware validates all protected endpoints
- Authorization checks in service layer ensure users can only access their own data
- User ID extracted from validated JWT claims (not request parameters)

**Audit Trail**:
- Refresh tokens track device info and IP address
- `last_login_at` timestamp for login monitoring
- `last_used_at` timestamp for refresh token usage

**Status**: Fully implemented. See [services/users/README.md](../services/users/README.md) for deployment details.

---

## Explore Service

The Explore Service is a dual-microservice system for discovering and recommending RSS content to users. It consists of two independent services (Fetcher and Recommender) with separate databases that communicate exclusively via REST APIs.

### Service Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                     Explore Service                           │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌────────────────────┐              ┌───────────────────┐  │
│  │  Fetcher Service   │   HTTP POST  │ Recommender Svc   │  │
│  │     :8080          │──────────────>│     :8081         │  │
│  │                    │   Articles    │                   │  │
│  │  - Feed Mgmt       │              │  - Article Store  │  │
│  │  - RSS Polling     │              │  - Voting         │  │
│  │  - Feed Sync       │              │  - Recommendations│  │
│  └─────────┬──────────┘              └─────────┬─────────┘  │
│            │                                   │             │
│     ┌──────▼────────┐                  ┌──────▼──────────┐  │
│     │ Fetcher DB    │                  │ Recommender DB  │  │
│     │  (feeds,      │                  │  (articles,     │  │
│     │   history)    │                  │   votes, users) │  │
│     └───────────────┘                  └─────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

**Key Architectural Principle**: Each service owns its own database. Services communicate only via HTTP APIs, never direct database access.

### Fetcher Service

**Purpose**: Manage RSS feed sources and fetch content for users

**Core Responsibilities**:
- Maintain feed sources in dedicated PostgreSQL database
- Fetch RSS/Atom feeds at controlled rate (1 feed per minute)
- Parse feed items and extract metadata
- Filter new articles since last successful fetch
- Submit new articles to Recommender via HTTP POST
- Auto-disable feeds after 10 consecutive failures
- Daily sync from Kagi Small Web collection
- Track fetch history for monitoring

**Technology Stack**:
- Go 1.23+
- PostgreSQL (fetcher_db)
- gofeed for RSS/Atom parsing
- 30-second HTTP timeout for fetches
- 10-second timeout for API calls

**Feed Management**:
- Daily sync from [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt)
- Prioritizes never-fetched feeds, then oldest
- Fetches 1 feed every 60 seconds (configurable)
- Auto-disables feeds after 10 consecutive failures
- Only successfully fetched articles sent to recommender

**Endpoints**:
```
GET  /health              # Health check (liveness)
POST /fetch               # Manually trigger feed fetch
```

### Recommender Service

**Purpose**: Store articles and provide personalized recommendations

**Core Responsibilities**:
- Receive articles from Fetcher via HTTP POST
- Store articles with SHA256 hash IDs (deduplication)
- Manage user engagement (upvotes/downvotes)
- Serve personalized recommendations (5 articles per request)
- Track articles read per user
- Track recommendation history to avoid repeats
- Auto-create users on first interaction

**Technology Stack**:
- Go 1.23+
- PostgreSQL (cairn_db)
- JWT authentication (validates tokens from User Service)
- Quality-based recommendation algorithm

**Endpoints**:
```
GET  /health                               # Health check
POST /explore/articles                     # Submit articles (from fetcher)
GET  /explore/recommendation              # Get 5 recommendations for authenticated user (requires auth)
POST /explore/articles/read                # Mark article as read (requires auth)
POST /explore/articles/{articleID}/vote    # Vote on article (requires auth)
DELETE /explore/articles/{articleID}/vote  # Remove vote (requires auth)
GET  /explore/articles/{articleID}/votes   # Get vote counts (requires auth)
```

### Recommendation Algorithm

**Algorithm**: 4 high-quality articles + 1 low-exposure article (exploration)

**Quality Score Formula**:
```
quality_score = (upvotes - (downvotes × 3)) / recommends

Where:
- upvotes: Number of positive votes
- downvotes: Number of negative votes (weighted 3x)
- recommends: Number of times recommended to any user
```

**Selection Process**:
1. Get articles eligible for recommendation (not deleted, not already recommended to user)
2. Calculate quality score for each article
3. Select 4 articles with highest quality scores
4. Select 1 article with lowest recommends count (exploration)
5. Track recommendations to avoid repeats
6. Increment recommends counter for each returned article

**Special Cases**:
- Articles with 0 recommends get very high default score (1000.0)
- Articles with 0 recommends and upvotes get infinite score (prioritized)
- If < 5 eligible articles, return what's available

**Benefits**:
- Balances quality (user votes) with freshness (low exposure)
- Downvotes heavily penalize quality (3x multiplier)
- New articles get surfaced quickly
- Prevents recommendation repetition per user

### Database Schema

#### Fetcher Database (fetcher_db)

**Feeds Table**:
```sql
CREATE TABLE feeds (
    id SERIAL PRIMARY KEY,
    url TEXT UNIQUE NOT NULL,
    title TEXT,
    description TEXT,
    last_fetched_at TIMESTAMP,
    consecutive_failures INT DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_feeds_last_fetched ON feeds(last_fetched_at NULLS FIRST) WHERE enabled = true;
CREATE INDEX idx_feeds_enabled ON feeds(enabled);
```

**Fetch History Table**:
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

CREATE INDEX idx_fetch_history_feed_id ON fetch_history(feed_id);
CREATE INDEX idx_fetch_history_created_at ON fetch_history(created_at DESC);
```

#### Recommender Database (cairn_db)

**Users Table**:
```sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,  -- External user ID from User Service
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Articles Table**:
```sql
CREATE TABLE articles (
    id VARCHAR(255) PRIMARY KEY,      -- SHA256 hash of article link
    title TEXT NOT NULL,
    link TEXT NOT NULL,
    description TEXT,
    content TEXT,
    author VARCHAR(255),
    published TIMESTAMP NOT NULL,
    feed_url TEXT NOT NULL,
    feed_title VARCHAR(255),
    categories TEXT[],                -- Array of category tags
    upvotes INT DEFAULT 0,
    downvotes INT DEFAULT 0,
    recommends INT DEFAULT 0,         -- Counter for recommendation tracking
    deleted BOOLEAN DEFAULT FALSE,
    feed_id INT,                      -- Reference to fetcher DB (no FK constraint)
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP
);

CREATE INDEX idx_articles_published ON articles(published DESC);
CREATE INDEX idx_articles_feed_url ON articles(feed_url);
CREATE INDEX idx_articles_categories ON articles USING GIN(categories);
CREATE INDEX idx_articles_recommends ON articles(recommends);
CREATE INDEX idx_articles_quality_score ON articles(upvotes, downvotes, recommends) WHERE deleted = false;
```

**User Articles Table** (tracks read status):
```sql
CREATE TABLE user_articles (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    article_id VARCHAR(255) REFERENCES articles(id) ON DELETE CASCADE,
    read BOOLEAN DEFAULT FALSE,
    read_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, article_id)
);

CREATE INDEX idx_user_articles_user_id ON user_articles(user_id);
CREATE INDEX idx_user_articles_read ON user_articles(read);
```

**Votes Table**:
```sql
CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    article_id VARCHAR(255) REFERENCES articles(id) ON DELETE CASCADE,
    vote_type TEXT CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)
);

CREATE INDEX idx_votes_article ON votes(article_id);
CREATE INDEX idx_votes_user ON votes(user_id);
CREATE INDEX idx_votes_vote_type ON votes(vote_type);
```

**Recommendations Table** (tracks recommendation history):
```sql
CREATE TABLE recommendations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    article_id VARCHAR(255) REFERENCES articles(id) ON DELETE CASCADE,
    recommended_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_recommendations_user_article ON recommendations(user_id, article_id);
CREATE INDEX idx_recommendations_article ON recommendations(article_id);
```

### Communication Patterns

#### Fetcher → Recommender Communication

**Pattern**: HTTP POST with JSON payload

**Flow**:
```
Fetcher Service                 Recommender Service
      │                                │
      ├──1. Fetch RSS feed─────────────┤
      ├──2. Parse feed items───────────┤
      ├──3. Filter new articles────────┤
      │                                │
      ├──POST /explore/articles────────>│
      │  {articles: [...]}             │
      │                                ├──4. Store articles in DB
      │                                ├──5. Deduplicate by hash
      │<───{status: "success"}─────────┤
      │                                │
      ├──6. Record fetch history───────┤
      └──7. Update last_fetched_at─────┘
```

**Request Format**:
```json
{
  "articles": [
    {
      "id": "sha256-hash",
      "title": "Article Title",
      "link": "https://example.com/article",
      "description": "Article description",
      "content": "Full article content",
      "author": "Author Name",
      "published": "2024-01-01T00:00:00Z",
      "feed_url": "https://example.com/feed.xml",
      "feed_title": "Feed Title",
      "categories": ["tech", "programming"]
    }
  ]
}
```

**Error Handling**:
- 30-second timeout for HTTP requests
- 10-second timeout for API calls
- Failed submissions marked in fetch history
- Consecutive failures tracked per feed
- Auto-disable after 10 consecutive failures

### Article ID Generation

**Method**: SHA256 hash of article link

**Benefits**:
- Ensures uniqueness across all feeds
- Enables automatic deduplication
- Deterministic ID generation
- No need for centralized ID allocation

**Implementation**:
```go
hash := sha256.Sum256([]byte(articleLink))
articleID := hex.EncodeToString(hash[:])
```

### Feed Polling Strategy

**Objective**: Gentle polling to avoid overwhelming feed sources

**Strategy**:
- Fetch 1 feed every 60 seconds (configurable via `FETCH_INTERVAL`)
- Prioritize never-fetched feeds first
- Then fetch oldest by `last_fetched_at`
- Only fetch from enabled feeds
- Filter articles: only send items published after `last_fetched_at`
- First fetch of a feed sends all available articles

**Failure Handling**:
- Track consecutive failures per feed
- Auto-disable after 10 consecutive failures (configurable via `MAX_FETCH_ERRORS`)
- Failed feeds can be manually re-enabled
- Each successful fetch resets consecutive failure counter

**Feed Sources**:
- Daily sync from Kagi Small Web collection
- URL: `https://raw.githubusercontent.com/kagisearch/smallweb/main/smallweb.txt`
- Runs on startup and every 24 hours
- New feeds automatically added (existing feeds unchanged)

### Voting System

**Supported Actions**:
- `upvote`: Positive vote (increases quality score)
- `downvote`: Negative vote (decreases quality score 3x)
- Remove vote: Delete existing vote

**Vote Mechanics**:
- One vote per user per article (enforced by unique constraint)
- Changing vote type replaces previous vote
- Vote counts materialized in articles table for performance
- Votes tracked in separate votes table for audit trail

**Vote Impact on Recommendations**:
- Upvotes increase article quality score
- Downvotes heavily penalize quality (3x multiplier)
- Quality score influences selection of 4 high-quality articles
- Low-exposure article (exploration) not affected by votes

### Security Considerations

**Authentication**:
- All user-facing endpoints require JWT authentication
- JWT tokens validated using User Service public key
- User ID extracted from validated JWT claims
- No query parameter authentication (prevents tampering)

**Input Validation**:
- Article batch size limited to 10MB
- Simple request size limited to 1KB
- Vote type must be 'upvote' or 'downvote'
- Article IDs validated before database operations

**Database Security**:
- No raw SQL queries (parameterized queries only)
- Prepared statements prevent SQL injection
- Foreign key constraints enforce referential integrity
- Check constraints enforce valid enum values

**Rate Limiting**:
- Fetcher: 1 feed per minute (protects feed sources)
- Recommender: No rate limiting (relies on User Service auth)

**Status**: Fully operational. See [services/explore/README.md](../services/explore/README.md) for deployment details.

---

## Read Service

The Read Service is a microservices-based read-it-later backend system designed for scalability, reliability, and maintainability. It consists of two primary services that communicate exclusively via REST APIs.

### Read Service Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Client Layer                         │
│                    (Mobile App / Web App)                    │
└────────────────────────────┬────────────────────────────────┘
                             │
                   ┌─────────┴─────────┐
                   │                   │
         ┌─────────▼──────────┐  ┌────▼──────────────────┐
         │  Content Service   │  │  RSS Fetcher Service  │
         │                    │  │                        │
         │  HTTP API (8080)   │◄─┤  HTTP API (8081)      │
         │                    │  │                        │
         │  - Content CRUD    │  │  - Subscription Mgmt   │
         │  - User Lists      │  │  - Feed Polling        │
         │  - Search          │  │  - Content Extraction  │
         │  - Metadata Mgmt   │  │  - Outbox Delivery     │
         └────────┬───────────┘  └────────┬───────────────┘
                  │                       │
         ┌────────▼───────────┐  ┌───────▼────────────────┐
         │  PostgreSQL DB     │  │  PostgreSQL DB         │
         │  (content_service) │  │  (rss_fetcher_service) │
         │                    │  │                         │
         │  - contents        │  │  - feeds                │
         │  - user_contents   │  │  - feed_subscriptions   │
         └────────────────────┘  │  - feed_items           │
                                 │  - content_outbox       │
                                 └─────────────────────────┘
```

### Content Service

**Purpose**: Store and serve article content with user-specific metadata

**Core Responsibilities**:
- Store cleaned and sanitized article content
- Manage user-content relationships (one content → many users)
- Provide CRUD operations for content
- Implement full-text search across user's content
- Support bulk operations for RSS Fetcher integration
- Handle content deduplication

**Technology Stack**:
- Go 1.21+
- Chi HTTP router
- PostgreSQL with full-text search (GIN indexes)
- go-readability for content extraction
- bluemonday for HTML sanitization

**Endpoints**:
```
POST   /api/v1/contents                              # Create content
PUT    /api/v1/contents/:id                          # Update content
GET    /api/v1/contents/:id                          # Get content
GET    /api/v1/users/:user_id/contents               # List user's contents
POST   /api/v1/users/:user_id/contents               # Add content to user
PATCH  /api/v1/users/:user_id/contents/:content_id   # Update user-content metadata
DELETE /api/v1/users/:user_id/contents/:content_id   # Remove from user's list
GET    /api/v1/users/:user_id/contents/search        # Search user's contents
POST   /api/v1/contents/bulk                         # Bulk create/update
POST   /api/v1/contents/check-duplicates             # Check for duplicates
```

### RSS Fetcher Service

**Purpose**: Manage RSS feed subscriptions and deliver content to users

**Core Responsibilities**:
- Subscribe/unsubscribe users to RSS feeds
- Poll feeds using intelligent tiered strategy
- Extract full article content from feed items
- Detect content updates via HTTP caching headers
- Deliver content to Content Service via outbox pattern
- Auto-disable failing feeds
- Manage feed polling tiers based on activity

**Technology Stack**:
- Go 1.21+
- Chi HTTP router
- PostgreSQL for feed metadata and outbox
- gofeed for RSS/Atom parsing
- robfig/cron for job scheduling
- sony/gobreaker for circuit breaker

**Endpoints**:
```
POST   /api/v1/users/:user_id/feeds/subscribe   # Subscribe to feed
DELETE /api/v1/users/:user_id/feeds/:feed_id    # Unsubscribe
GET    /api/v1/users/:user_id/feeds             # List subscriptions
PATCH  /api/v1/feeds/:feed_id/enable            # Re-enable disabled feed
```

**Background Workers**:
1. **Feed Polling Worker**: Continuously polls active feeds
2. **Content Extraction Worker**: Extracts full content from feed items
3. **Outbox Delivery Worker**: Delivers content to Content Service
4. **Tier Management Job**: Daily job to adjust feed tiers
5. **Cleanup Jobs**: Remove old outbox entries and feed items

### Read Service Data Models

#### Contents Table

```sql
CREATE TABLE contents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    author TEXT,
    source_url TEXT NOT NULL,
    canonical_url TEXT,
    cleaned_html TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    published_at TIMESTAMP WITH TIME ZONE,
    source_feed_id UUID,  -- NULL for manually added content
    orphaned_at TIMESTAMP WITH TIME ZONE,  -- Set when last user removes it
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(content_hash, source_feed_id)
);

CREATE INDEX idx_contents_orphaned_at ON contents(orphaned_at) WHERE orphaned_at IS NOT NULL;
CREATE INDEX idx_contents_source_feed_id ON contents(source_feed_id);
```

**Key Fields**:
- `content_hash`: SHA-256 hash of cleaned_html for deduplication
- `canonical_url`: Normalized URL (tracking params removed)
- `orphaned_at`: Timestamp when last user removed content (for cleanup)
- `source_feed_id`: NULL for manually saved content, UUID for RSS content

#### User Contents Table

```sql
CREATE TABLE user_contents (
    user_id UUID NOT NULL,
    content_id UUID NOT NULL REFERENCES contents(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'reading', 'completed', 'archived')),
    scroll_position INTEGER DEFAULT 0,
    is_favorite BOOLEAN DEFAULT FALSE,
    added_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_accessed_at TIMESTAMP WITH TIME ZONE,

    PRIMARY KEY (user_id, content_id)
);

CREATE INDEX idx_user_contents_user_status ON user_contents(user_id, status);
CREATE INDEX idx_user_contents_user_favorite ON user_contents(user_id, is_favorite) WHERE is_favorite = TRUE;
CREATE INDEX idx_user_contents_added_at ON user_contents(user_id, added_at DESC);
```

**Triggers**:
- **After INSERT**: Clear `orphaned_at` on contents when user adds it
- **After DELETE**: Set `orphaned_at` on contents when last user removes it

#### Feeds Table

```sql
CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_url TEXT NOT NULL UNIQUE,
    title TEXT,
    description TEXT,
    site_url TEXT,
    polling_tier TEXT NOT NULL DEFAULT 'active' CHECK (polling_tier IN ('active', 'moderate', 'quiet')),
    status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'error')),
    last_fetched_at TIMESTAMP WITH TIME ZONE,
    last_published_at TIMESTAMP WITH TIME ZONE,
    next_poll_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    consecutive_error_days INTEGER DEFAULT 0,
    last_error_at TIMESTAMP WITH TIME ZONE,
    last_error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_feeds_next_poll ON feeds(next_poll_at) WHERE status = 'active';
CREATE INDEX idx_feeds_polling_tier ON feeds(polling_tier);
```

**Polling Tiers**:
- `active`: Last published within 7 days → poll every 1 hour
- `moderate`: Last published within 30 days → poll every 6 hours
- `quiet`: Last published over 30 days ago → poll every 24 hours

#### Feed Subscriptions Table

```sql
CREATE TABLE feed_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    subscribed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(user_id, feed_id)
);

CREATE INDEX idx_feed_subscriptions_user ON feed_subscriptions(user_id);
CREATE INDEX idx_feed_subscriptions_feed ON feed_subscriptions(feed_id);
```

**Trigger**: Enforce 100 feed limit per user

#### Feed Items Table

```sql
CREATE TABLE feed_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_id UUID NOT NULL REFERENCES feeds(id) ON DELETE CASCADE,
    item_guid TEXT NOT NULL,
    title TEXT NOT NULL,
    author TEXT,
    source_url TEXT NOT NULL,
    description TEXT,
    published_at TIMESTAMP WITH TIME ZONE,
    content_hash TEXT,
    http_last_modified TEXT,  -- HTTP Last-Modified header
    http_etag TEXT,            -- HTTP ETag header
    last_checked_at TIMESTAMP WITH TIME ZONE,
    processing_status TEXT NOT NULL DEFAULT 'pending' CHECK (processing_status IN ('pending', 'processing', 'completed', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    UNIQUE(feed_id, item_guid)
);

CREATE INDEX idx_feed_items_processing_status ON feed_items(processing_status) WHERE processing_status = 'pending';
CREATE INDEX idx_feed_items_feed_id ON feed_items(feed_id);
```

#### Content Outbox Table

```sql
CREATE TABLE content_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    feed_item_id UUID NOT NULL REFERENCES feed_items(id),
    content_payload JSONB NOT NULL,  -- Full content ready for Content Service API
    user_ids UUID[] NOT NULL,        -- Array of user IDs to deliver to
    delivery_status TEXT NOT NULL DEFAULT 'pending' CHECK (delivery_status IN ('pending', 'sending', 'delivered', 'failed')),
    content_service_id UUID,         -- ID returned from Content Service
    retry_count INTEGER DEFAULT 0,
    next_retry_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    delivered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_content_outbox_delivery_status ON content_outbox(delivery_status, next_retry_at) WHERE delivery_status IN ('pending', 'failed');
```

### Communication Patterns

#### RSS Fetcher → Content Service

**Pattern**: REST API calls with outbox pattern for reliability

**Flow**:
1. RSS Fetcher extracts content from feed item
2. RSS Fetcher writes to local `content_outbox` table
3. Outbox worker picks up pending entries
4. Worker calls Content Service REST API
5. On success: Mark as delivered, store content ID
6. On failure: Increment retry counter, schedule next retry

**Benefits**:
- Decouples content extraction from delivery
- Survives Content Service downtime
- Enables retry with exponential backoff
- Provides audit trail

**Retry Schedule**:
```
Retry 1: 1 minute
Retry 2: 5 minutes
Retry 3: 15 minutes
Retry 4: 1 hour
Retry 5: 4 hours
Retry 6: 12 hours
After 6 retries: Mark as failed
```

#### Circuit Breaker Pattern

**Purpose**: Prevent overwhelming a failing Content Service

**Implementation**: Using `sony/gobreaker`

**States**:
- **Closed**: Normal operation, requests flow through
- **Open**: After 5 consecutive failures, reject requests immediately
- **Half-Open**: After 30 seconds, allow test request

**Benefits**:
- Prevents cascade failures
- Allows failing service to recover
- Provides fast-fail behavior

### Content Processing Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│                    Content Processing                        │
└─────────────────────────────────────────────────────────────┘

1. Receive HTML or URL
   │
   ├─ If URL provided → Fetch HTML (30s timeout)
   └─ If HTML provided → Use directly
   │
2. Apply Readability Parsing (go-readability)
   │ ├─ Extract title, author, published_at
   │ ├─ Extract main content
   │ └─ Remove ads, navigation, footers
   │
3. Apply HTML Sanitization (bluemonday)
   │ ├─ Remove <script> tags
   │ ├─ Remove <iframe> tags
   │ ├─ Allow safe HTML tags only
   │ └─ Remove dangerous attributes (onclick, etc.)
   │
4. Generate Content Hash (SHA-256)
   │ └─ Hash of cleaned_html for deduplication
   │
5. Canonicalize URL
   │ ├─ Normalize scheme (https)
   │ ├─ Lowercase host
   │ └─ Remove tracking parameters (utm_*, fbclid, etc.)
   │
6. Validate Size (max 5MB)
   │
7. Check for Duplicates (content_hash + source_feed_id)
   │ ├─ If exists → Return existing content
   │ └─ If new → Store in database
   │
8. Store in Database
   └─ Return content ID
```

### Feed Polling Strategy

**Objective**: Reduce load on inactive feeds while keeping active feeds fresh

**Tier Assignment Logic**:

```
IF last_published_at > NOW() - INTERVAL '7 days'
    THEN tier = 'active' (poll every 1 hour)
ELSE IF last_published_at > NOW() - INTERVAL '30 days'
    THEN tier = 'moderate' (poll every 6 hours)
ELSE
    THEN tier = 'quiet' (poll every 24 hours)
```

**Tier Management**:
- Daily background job evaluates all feeds
- Updates `polling_tier` based on `last_published_at`
- Updates `next_poll_at` based on new tier
- New content promotes feed to 'active' tier immediately

### Reliability Patterns

#### 1. Outbox Pattern

**Purpose**: Ensure content delivery even if Content Service is temporarily unavailable

**Implementation**:
- RSS Fetcher writes to local `content_outbox` table (ACID transaction)
- Separate worker processes outbox entries
- Worker retries failed deliveries with exponential backoff
- Provides at-least-once delivery guarantee

**Benefits**:
- Decouples content extraction from delivery
- No content loss during Content Service downtime
- Audit trail of all deliveries

#### 2. Circuit Breaker

**Purpose**: Prevent overwhelming a failing Content Service

**Implementation**: Using `sony/gobreaker`

**Configuration**:
- Open after 5 consecutive failures
- Half-open after 30 seconds
- Test with single request before closing

**Benefits**:
- Fast-fail when service is down
- Prevents resource exhaustion
- Automatic recovery detection

#### 3. Idempotency

**Content Service**:
- Duplicate check on `content_hash + source_feed_id`
- Returns existing content if duplicate
- Safe to retry creation

**RSS Fetcher**:
- Feed items deduplicated by `feed_id + item_guid`
- Outbox entries retried safely

#### 4. Graceful Degradation

**Strategies**:
- Fetch timeout → Use RSS description instead of full article
- Content Service unavailable → Queue in outbox for later delivery
- Feed fetch failure → Increment error counter, retry next poll

---

## Security Considerations

### HTML Sanitization (Read Service)

**Library**: bluemonday (industry-standard Go HTML sanitizer)

**Allowed Tags**:
- Text: `<p>`, `<h1>`-`<h6>`, `<br>`, `<hr>`
- Formatting: `<strong>`, `<em>`, `<u>`, `<s>`
- Lists: `<ul>`, `<ol>`, `<li>`
- Links: `<a>` (with `href` attribute only)
- Media: `<img>` (with `src`, `alt` attributes only)
- Code: `<code>`, `<pre>`
- Tables: `<table>`, `<tr>`, `<td>`, `<th>`

**Blocked/Removed**:
- `<script>` tags and JavaScript
- `<iframe>` and embedded content
- Event handlers (`onclick`, `onerror`, etc.)
- `<form>` elements
- `<style>` tags (inline styles allowed on safe attributes)

### Input Validation

**Content Service**:
- Content size limit: 5MB
- URL format validation
- UUID validation for IDs
- Status enum validation
- Scroll position >= 0

**RSS Fetcher Service**:
- Feed URL format validation
- 100 feed limit per user (enforced by trigger)
- UUID validation for IDs

### Database Security

- No raw SQL queries (use parameterized queries)
- Prepared statements prevent SQL injection
- Foreign key constraints enforce referential integrity
- Check constraints enforce valid enum values

### User Service Security

**Password Security**:
- Algorithm: bcrypt with cost factor 12 (minimum for production)
- Minimum password length: 8 characters
- Passwords and hashes are never logged

**JWT Security**:
- Algorithm: RS256 (asymmetric — private key signs, public key validates)
- Access token lifetime: 15 minutes
- Refresh token lifetime: 7 days
- Private key stored in HashiCorp Vault; public key distributed to other services via Vault

**Refresh Token Security**:
- Tokens are SHA-256 hashed before database storage (raw token never persisted)
- Rotated on each use — each refresh issues a new token
- Tracks `device_info` and `ip_address` for anomaly detection
- All tokens for a user can be revoked on suspected compromise

**Authorization Middleware**:
- Extracts `user_id` from validated JWT claims
- Compares against `user_id` in URL path — returns 403 if mismatch
- Ensures users can only access their own resources

**Rate Limiting**:
- Auth endpoints (`/auth/login`, `/auth/register`): 10 requests/minute per IP
- User endpoints: 60 requests/minute per IP
- Prevents brute-force attacks on authentication

**Mobile Device Authentication**:
- Expo device ID treated as a credential (HTTPS-only transmission)
- App reinstall invalidates device ID; users must re-register
- Mobile-only accounts can upgrade to email/password; after upgrade, device ID login is permanently disabled to enforce recoverable credentials

**Security Headers** (applied globally):
- `Content-Security-Policy`, `X-Frame-Options`, `X-Content-Type-Options`, `HSTS`

### Explore Service Security

**JWT Authentication** (Recommender service only):
- Retrieves JWT public key from HashiCorp Vault on startup
- Validates RS256-signed tokens on every protected request
- Extracts `user_id` from JWT claims for authorization

**Endpoint Access Control**:

| Endpoint | Auth Required |
|---|---|
| `GET /health/live`, `GET /health/ready` | No |
| `POST /api/v1/explore/article` | No |
| `GET /api/v1/explore/recommendation` | Yes |
| `GET /api/v1/explore/user/votes` | Yes |
| `POST/DELETE /api/v1/explore/article/:id/vote` | Yes |
| `POST /api/v1/explore/article/:id/read` | Yes |

**Input Validation**:
- Feed URL format validated before storage
- 100 feed limit per user enforced by database trigger
- UUID validation for all resource IDs

**RSS Feed Fetching**:
- Fetch timeout enforced to prevent hanging connections
- Feed error counter tracks unreachable feeds; repeated failures reduce polling priority
- External feed content is sanitized before storage (via Read Service HTML sanitization pipeline)

---

## Performance Considerations

### Connection Pooling

All services use connection pooling:
```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(5 * time.Minute)
```

### Pagination

- Cursor-based pagination for user content lists (20 items/page)
- Offset-based pagination for feed subscriptions (50 items/page)

### Caching

Future enhancement:
- Cache feed metadata (1 hour TTL)
- Cache user content counts
- HTTP caching headers for API responses

### Batch Processing

- Bulk content creation (max 100 items)
- Batch feed polling (10 feeds at a time)
- Batch outbox delivery (concurrent workers)

---

## Monitoring & Observability

### Health Checks

All services expose:
- `GET /health/live` - Liveness check (is process running?)
- `GET /health/ready` - Readiness check (can handle traffic? DB connected?)

### Logging

Structured logging includes:
- Request ID for tracing
- HTTP method, path, status code
- Response time
- User ID (when available)
- Error stack traces

See [LOGGING_STRATEGY.md](LOGGING_STRATEGY.md) for detailed logging standards.

### Metrics (Future)

Planned Prometheus metrics:
- HTTP request duration (histogram)
- HTTP request count by status code
- Database query duration
- Feed polling success/failure rates
- Outbox queue depth
- Circuit breaker state changes

---

## Scalability Considerations

### Horizontal Scaling

**Services that can be scaled horizontally**:
- User Service API (stateless)
- Explore Service APIs (stateless)
- Read Service - Content Service API (stateless)
- Read Service - RSS Fetcher Service API (stateless)

**Services that need coordination**:
- RSS Fetcher Worker (use distributed locks)

### Database Scaling

**Current**: Single PostgreSQL instance per service

**Future**:
- Read replicas for read-heavy workloads
- Connection pooler (PgBouncer) for connection management
- Partitioning for large tables (contents, feed_items)

### Worker Scaling

**Current**: Single worker instance per service

**Future**:
- Multiple worker instances with job queue (Redis/RabbitMQ)
- Distributed job scheduling
- Worker-specific scaling based on queue depth

---

## Future Enhancements

### Authentication & Authorization

- JWT-based authentication (User Service - implemented)
- API key authentication for service-to-service
- Fine-grained authorization policies

### Caching Layer

- Redis for feed metadata caching
- API response caching with ETags
- User content count caching

### Message Queue

- Replace outbox polling with message queue (Redis Streams, RabbitMQ)
- Event-driven architecture for real-time updates

### Observability

- Distributed tracing (OpenTelemetry)
- Prometheus metrics
- Grafana dashboards
- Alert rules for critical failures

### Content Features

- Thumbnail extraction and storage
- PDF support
- Video/podcast content
- Multi-language support with language detection
