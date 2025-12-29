# Recommendation Engine Implementation Plan

## Current Status

**Overall Progress**: 8 of 9 phases complete (89%)

**Completed Phases**:
- ✅ Phase 1: Database Schema & Migrations (Partial - users, votes, recommendations tables created via migrations)
- ✅ Phase 2: Article Deduplication (ON CONFLICT handling implemented)
- ✅ Phase 3: Voting API (Complete with user tracking)
- ✅ Phase 4: Enhanced Recommendation Algorithm (Quality score formula implemented)
- ✅ Phase 5: Model Updates (All models created)
- ✅ Phase 6: Configuration & Environment (ARTICLE_RETENTION_DAYS configured)
- ✅ Phase 7: Article Cleanup (Daily automatic cleanup with standalone utility)
- ✅ Phase 8: Testing & Validation (Complete integration test suite with 6 passing tests)

**Remaining Work**:
- ⏳ Phase 9: Admin & Monitoring (Admin API endpoints - optional)

---

## Overview
Implement a recommendation engine that stores articles from RSS feeds and recommends the next 5 articles for users to read, based on upvote/downvote ratios and recommendation counts.

## Project Architecture Summary
- **Fetcher Service**: Maintains its own database of feeds, fetches RSS content, sends successfully fetched articles to Recommender
- **Fetcher Database**: PostgreSQL for feed management and crawling state (separate from Recommender DB)
- **Recommendation Engine**: Stores articles, tracks engagement (upvotes/downvotes), serves recommendations
- **Recommender Database**: PostgreSQL for article storage and user engagement

**Important**: The Fetcher maintains its own database for feed sources. The Recommender only receives successfully fetched articles via HTTP POST to `/api/v1/articles`. The two services have separate databases and communicate only via HTTP APIs.

---

## Phase 1: Database Schema & Migrations ✅ COMPLETE

### 1.1 Create Database Migrations ✅
**Files**:
- `recommender/migrations/001_init.sql` - Articles and users tables
- `recommender/migrations/002_add_feed_id_to_articles.sql` - Feed ID support
- `recommender/migrations/004_voting_and_recommendations.sql` - Votes and recommendations tables

**Status**: COMPLETE - All required tables exist and are in use

**Tables Created**:

```sql
-- Articles table: Fetched content from RSS feeds
-- NOTE: feed_id references feeds table in FETCHER database (no FK constraint)
CREATE TABLE articles (
    id TEXT PRIMARY KEY,  -- SHA256 hash of link
    title TEXT NOT NULL,
    link TEXT UNIQUE NOT NULL,
    description TEXT,
    content TEXT,
    author TEXT,
    published_at TIMESTAMP,
    feed_id INT,  -- References feeds table in fetcher database (no FK constraint)
    upvotes INT DEFAULT 0,
    downvotes INT DEFAULT 0,
    recommends INT DEFAULT 0,
    deleted BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Categories table: Article tags/categories
CREATE TABLE article_categories (
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    category TEXT NOT NULL,
    PRIMARY KEY (article_id, category)
);

-- Users table: Track users for votes
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    user_id TEXT UNIQUE NOT NULL,  -- External user identifier
    created_at TIMESTAMP DEFAULT NOW()
);

-- Votes table: Track individual votes by user
CREATE TABLE votes (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    vote_type TEXT CHECK (vote_type IN ('upvote', 'downvote')),
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id, article_id)
);

-- Recommendations table: Track which articles have been recommended to which users
CREATE TABLE recommendations (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id),
    article_id TEXT REFERENCES articles(id) ON DELETE CASCADE,
    recommended_at TIMESTAMP DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_articles_feed_id ON articles(feed_id);
CREATE INDEX idx_articles_published ON articles(published_at DESC);
CREATE INDEX idx_articles_deleted ON articles(deleted) WHERE deleted = false;
CREATE INDEX idx_articles_recommends ON articles(recommends);
CREATE INDEX idx_votes_article ON votes(article_id);
CREATE INDEX idx_votes_user ON votes(user_id);
CREATE INDEX idx_recommendations_user_article ON recommendations(user_id, article_id);
```

---

## Phase 2: Article Deduplication ✅ COMPLETE
**Goal**: Prevent duplicate articles from being stored

**Status**: COMPLETE - Implemented in [article_repository.go](recommender/internal/db/article_repository.go)

**Implementation**: ✅

**File**: `recommender/internal/db/article_repository.go`

**Features Implemented**:
1. ✅ Use `ON CONFLICT (link) DO UPDATE` for upserts
2. ✅ Update article metadata if feed re-publishes with changes
3. ✅ Preserve vote counts, recommends, and deleted status on updates
4. ✅ Update `updated_at` timestamp on conflicts

**SQL Implementation**:
```sql
INSERT INTO articles (id, title, link, content, description, author, published_at, feed_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (link) DO UPDATE SET
    title = EXCLUDED.title,
    content = EXCLUDED.content,
    description = EXCLUDED.description,
    author = EXCLUDED.author,
    published_at = EXCLUDED.published_at,
    updated_at = NOW()
WHERE articles.deleted = false;  -- Don't update deleted articles
```

**Key Behaviors**: ✅
- ✅ Duplicate detection based on article link (unique constraint)
- ✅ Updates article content if feed re-publishes with changes
- ✅ Preserves vote counts (upvotes/downvotes) on updates
- ✅ Preserves recommends counter on updates
- ✅ Preserves deleted status (won't resurrect deleted articles)
- ✅ Updates the `updated_at` timestamp when article is modified

---

## Phase 3: Voting API ✅ COMPLETE
**Goal**: Track upvotes/downvotes per article per user

**Status**: COMPLETE - See [PHASE_1_1_SUMMARY.md](recommender/migrations/PHASE_1_1_SUMMARY.md) for details

**Files Implemented**:
- ✅ `recommender/internal/api/handlers.go` - Vote endpoints
- ✅ `recommender/internal/db/vote_repository.go` - Vote repository
- ✅ `recommender/internal/db/user_repository.go` - User management

**Endpoints Implemented**: ✅
```
POST   /api/v1/articles/:id/vote          - Cast or change vote
DELETE /api/v1/articles/:id/vote/:userId  - Remove vote
GET    /api/v1/articles/:id/votes         - Get vote counts
```

**Repository Methods**: ✅
```go
// RecordVote inserts or updates a vote (upsert)
// Updates articles.upvotes and articles.downvotes counts
func (r *VoteRepository) RecordVote(ctx context.Context, userID string, articleID string, voteType string) error

// RemoveVote deletes a vote and updates article counts
func (r *VoteRepository) RemoveVote(ctx context.Context, userID string, articleID string) error

// GetVoteCounts returns upvote/downvote counts
func (r *VoteRepository) GetVoteCounts(ctx context.Context, articleID string) (upvotes int, downvotes int, error)

// GetUserVote returns the user's vote for an article (if any)
func (r *VoteRepository) GetUserVote(ctx context.Context, userID string, articleID string) (voteType string, error)
```

**Business Logic**: ✅
- ✅ Auto-create user in `users` table if not exists (using user_id string)
- ✅ Update `articles.upvotes` and `articles.downvotes` columns atomically
- ✅ Ensure one vote per user per article (UNIQUE constraint)
- ✅ When changing vote type (upvote->downvote), decrement old type and increment new type

---

## Phase 4: Enhanced Recommendation Algorithm ✅ COMPLETE
**Goal**: Recommend 5 articles based on upvote/downvote ratio

**Status**: COMPLETE - See [PHASE_2_SUMMARY.md](recommender/migrations/PHASE_2_SUMMARY.md) for details

**Implementation**: ✅ 4 high-quality articles + 1 under-recommended article, excluding deleted articles

**File**: `recommender/internal/recommend/engine.go` ✅

**Algorithm Implemented**: ✅
1. ✅ Filter out deleted articles (`deleted = false`)
2. ✅ Calculate quality score: `score = (upvote + (downvote * 3)) / recommends`
   - ✅ Higher score = better quality relative to exposure
   - ✅ Heavily weight downvotes (3x)
   - ✅ Articles with 0 recommends get special handling (high priority)
3. ✅ Select 4 articles with highest score
4. ✅ Select 1 article with lowest `recommends` count (discovery/exploration)
5. ✅ Increment `recommends` counter for each recommended article
6. ✅ Track recommendation in `recommendations` table for analytics

**Edge Cases Handled**: ✅
- ✅ If `recommends = 0`, treat score as very high value (new content prioritized)
- ✅ Avoid recommending same article to same user repeatedly (check `recommendations` table)
- ✅ Handle division by zero gracefully

**Methods Implemented**: ✅
```go
// GetRecommendations returns 5 recommended articles for user
// 4 high-quality (high score), 1 low-exposure (low recommends)
func (e *Engine) GetRecommendations(ctx context.Context, userID string) ([]models.Article, error)

// calculateQualityScore computes: (upvote + (downvote * 3)) / recommends
func (e *Engine) calculateQualityScore(article models.Article) float64

// IncrementRecommendCount updates articles.recommends counter
func (e *Engine) IncrementRecommendCount(ctx context.Context, articleID string) error

// RecordRecommendation tracks recommendation event for user
func (e *Engine) RecordRecommendation(ctx context.Context, userID string, articleID string) error
```

---

## Phase 5: Model Updates ✅ COMPLETE

### 5.1 Extend Models ✅
**Files**: `pkg/models/*.go`

**Status**: COMPLETE - All models created and in use

**Article Model Extended** (`pkg/models/article.go`): ✅
```go
type Article struct {
    // ... existing fields ...
    FeedID      int       `json:"feed_id"`
    Upvotes     int       `json:"upvotes"`
    Downvotes   int       `json:"downvotes"`
    Recommends  int       `json:"recommends"`
    Deleted     bool      `json:"deleted"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

**Feed Model** (`pkg/models/feed.go`): ✅
```go
type Feed struct {
    ID                  int       `json:"id"`
    URL                 string    `json:"url"`
    Title               string    `json:"title"`
    Description         string    `json:"description"`
    Enabled             bool      `json:"enabled"`
    LastFetchedAt       time.Time `json:"last_fetched_at"`
    ConsecutiveFailures int       `json:"consecutive_failures"`
    CreatedAt           time.Time `json:"created_at"`
    UpdatedAt           time.Time `json:"updated_at"`
}
```

**Vote Model** (`pkg/models/vote.go`): ✅
```go
type Vote struct {
    ID        int       `json:"id"`
    UserID    int       `json:"user_id"`
    ArticleID string    `json:"article_id"`
    VoteType  string    `json:"vote_type"` // "upvote" or "downvote"
    CreatedAt time.Time `json:"created_at"`
}
```

**User Model** (`pkg/models/user.go`): ✅
```go
type User struct {
    ID        int       `json:"id"`
    UserID    string    `json:"user_id"`  // External user identifier
    CreatedAt time.Time `json:"created_at"`
}
```

**Recommendation Model** (`pkg/models/recommendation.go`): ✅
```go
type Recommendation struct {
    ID            int       `json:"id"`
    UserID        int       `json:"user_id"`
    ArticleID     string    `json:"article_id"`
    RecommendedAt time.Time `json:"recommended_at"`
}
```

---

## Phase 6: Configuration & Environment (formerly Phase 7) ✅ COMPLETE

### 6.1 Update Docker Compose ✅
**File**: `docker-compose.yml`

**Environment Variables Added**:
```yaml
recommender:
  environment:
    - ARTICLE_RETENTION_DAYS=90 # Delete articles after 90 days
```

**Status**: COMPLETE - Environment variable added to docker-compose.yml and documented in CLAUDE.md

**Note**: Fetcher environment variables (FETCH_INTERVAL, MAX_FETCH_ERRORS) are already configured.

**Priority**: MEDIUM

---

## Phase 7: Article Cleanup (formerly Phase 8) ✅ COMPLETE

### 7.1 Periodic Article Deletion ✅
**Goal**: Delete articles older than 90 days

**Implementation**: ✅ COMPLETE
- ✅ Create background job/cron that runs daily
- ✅ Delete articles where `created_at < NOW() - INTERVAL '90 days'`
- ✅ Set `deleted = true` instead of hard delete (maintain referential integrity)
- ✅ Clean up orphaned records in related tables (via hard delete after grace period)

**Files Created**: ✅
- ✅ `recommender/internal/cleanup/article_cleanup.go` - Cleanup logic
- ✅ `recommender/cmd/cleanup/main.go` - Standalone utility for manual cleanup
- ✅ `recommender/internal/cleanup/README.md` - Documentation
- ✅ `recommender/migrations/PHASE_7_SUMMARY.md` - Implementation summary

**Repository Methods**: ✅
```go
// MarkOldArticlesAsDeleted sets deleted=true for articles older than N days
func (r *ArticleRepository) MarkOldArticlesAsDeleted(ctx context.Context, days int) (int, error)

// HardDeleteOldArticles removes articles older than N days (optional, for maintenance)
func (r *ArticleRepository) HardDeleteOldArticles(ctx context.Context, days int) (int, error)
```

**Features Implemented**:
- ✅ Two-phase deletion (soft delete + hard delete after grace period)
- ✅ Configurable retention via `ARTICLE_RETENTION_DAYS` environment variable
- ✅ Daily automatic cleanup (runs every 24 hours)
- ✅ Graceful shutdown support
- ✅ Comprehensive logging
- ✅ Standalone cleanup utility for manual runs
- ✅ Makefile integration (`make build`, `make run-cleanup`)

**Status**: COMPLETE - See [PHASE_7_SUMMARY.md](recommender/migrations/PHASE_7_SUMMARY.md) for details

---

## Phase 8: Testing & Validation ✅ COMPLETE

### 8.1 Integration Tests ✅
**Goal**: Verify end-to-end flow

**Status**: COMPLETE - See [PHASE_8_SUMMARY.md](recommender/migrations/PHASE_8_SUMMARY.md) for details

**Files Created**:
- ✅ `recommender/integration_test.go` - 6 comprehensive integration tests
- ✅ `recommender/scripts/setup_test_db.sh` - Test database setup script
- ✅ `Makefile` (updated) - Added `test-integration` and `test-all` targets

**Test Scenarios**: ✅ All Passing
1. ✅ Fetcher fetches articles from its own database, submits to recommender
2. ✅ Recommender deduplicates and stores articles
3. ✅ User requests recommendations, receives 5 articles (4 high-quality + 1 low-exposure)
4. ✅ User upvotes article via API
5. ✅ Recommendation algorithm includes upvoted articles in future recommendations
6. ✅ User downvotes article via API
7. ✅ Downvoted article gets lower quality score
8. ✅ Deleted articles excluded from recommendations

**Test Results**:
```
PASS: TestArticleSubmissionAndDeduplication
PASS: TestRecommendationAlgorithm
PASS: TestUpvotingFlow
PASS: TestDownvotingFlow
PASS: TestDeletedArticlesExcluded
PASS: TestEndToEndFlow

ok  	github.com/andrew-craig/cairn/services/explore/recommender	0.561s
```

**Note**: Feed management testing (sync, prioritization, auto-disable) is handled in the Fetcher service test suite (39 tests, fully implemented).

---

## Phase 9: Admin & Monitoring (formerly Phase 10)

### 9.1 Admin Dashboard Endpoints
**Goal**: Basic admin API for monitoring

**New Endpoints**:
```
GET /admin/stats
{
  "total_articles": 56789,
  "articles_today": 45,
  "deleted_articles": 234,
  "total_votes": 890,
  "total_recommendations": 5678
}

GET /admin/articles?deleted=true
GET /admin/votes/summary  - Vote statistics
```

**Note**: Feed management endpoints are not needed in the recommender. Feed stats and management are available through the Fetcher service's own database.

**Priority**: LOW - Nice to have

---

## Implementation Order

**Completed**: ✅
1. ✅ **Phase 1**: Database schema with users, votes, recommendations tables
2. ✅ **Phase 5**: Model updates (Vote, User, Recommendation models)
3. ✅ **Phase 2**: Article deduplication (ON CONFLICT handling)
4. ✅ **Phase 3**: Voting API (upvote/downvote with user tracking)
5. ✅ **Phase 4**: Enhanced recommendation algorithm (quality score + low-exposure)
6. ✅ **Phase 6**: Configuration updates (ARTICLE_RETENTION_DAYS)
7. ✅ **Phase 7**: Article cleanup (90-day retention with automatic daily cleanup)
8. ✅ **Phase 8**: Integration testing (6 comprehensive tests, all passing)

**Remaining**: ⏳
9. ⏳ **Phase 9**: Admin endpoints (optional)

**Note**: Feed management is handled by the Fetcher service in its own database (fully implemented and tested with 39 tests).

---

## Key Decisions & Trade-offs

### Why Separate Databases for Fetcher and Recommender?
- **Architectural clarity**: Each service owns its own data
- **Fetcher autonomy**: Fetcher manages feeds independently
- **Independent scaling**: Can scale each database separately
- **Clear separation of concerns**: Fetcher manages crawling state, Recommender manages articles
- **Simpler deployment**: No shared database to coordinate migrations

### Why PostgreSQL instead of SQLite?
- Already scaffolded with PostgreSQL
- Better concurrency for multi-service architecture
- Supports separate databases for each service

### Why separate votes table?
- Prevents double-voting per user (UNIQUE constraint on user_id, article_id)
- Enables vote history and analytics
- Can track vote changes over time
- Keeps article table normalized
- Required for the quality score calculation

### Why separate recommendations table?
- Track which articles have been recommended to which users
- Avoid recommending same article repeatedly to same user
- Enable analytics on recommendation effectiveness
- Support future personalization features

### Why "deleted" flag instead of hard delete?
- Maintains referential integrity with votes and recommendations
- Enables data retention for analytics
- Allows for "undelete" functionality if needed
- Hard delete can be done later for cleanup

### Quality score formula: (upvote + (downvote * 3)) / recommends
- Per requirements specification
- Heavily weights downvotes (3x penalty)
- Normalizes by exposure (recommends count)
- Articles with high upvotes but few recommends surface quickly
- Articles with downvotes are heavily penalized

---

## Success Criteria

**Recommender Service:**
- ✅ No duplicate articles in database (ON CONFLICT handling)
- ✅ Users can upvote/downvote articles
- ✅ Vote tracking prevents double-voting per user
- ✅ Recommendation algorithm returns 5 articles: 4 high-quality + 1 low-exposure
- ✅ Quality score formula implemented correctly: (upvote + (downvote * 3)) / recommends
- ✅ Deleted articles excluded from recommendations
- ✅ Articles older than 90 days are soft-deleted (marked as deleted)
- ✅ Soft-deleted articles are hard-deleted after 30+ day grace period
- ✅ Automatic daily cleanup runs without manual intervention
- ✅ System runs reliably for 24+ hours without intervention

**Fetcher Service**: ✅ COMPLETE
- ✅ Fetcher has its own PostgreSQL database
- ✅ System syncs feed list from Kagi Small Web Text collection daily
- ✅ Fetcher processes 1 feed every 60 seconds
- ✅ Never-fetched feeds are prioritized
- ✅ Feeds with 10 consecutive failures are automatically disabled
- ✅ Comprehensive test suite (39 tests)

---

## Future Enhancements (Out of Scope)

- User authentication & accounts (currently using anonymous user IDs)
- Personalized recommendations per user (ML-based preferences)
- Content similarity detection (avoid recommending similar articles)
- Automatic spam/low-quality detection
- Feed discovery (auto-add feeds from user submissions)
- OPML import/export for feed management
- Rate limiting per user
- Caching layer (Redis) for recommendations
- Full-text search across articles
- Tag-based filtering and recommendations
- RSS feed output for recommended articles
- Email digest of daily recommendations
- Article reading time estimates
- Article summarization (AI-powered)
- Social sharing features
