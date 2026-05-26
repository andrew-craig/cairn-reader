package db

import (
	"context"

	"github.com/cairn-app/cairn-reader/pkg/models"
)

// ArticleRepositoryInterface defines the contract for article database operations
type ArticleRepositoryInterface interface {
	// Create inserts a new article into the database
	// Implements Phase 2 deduplication: ON CONFLICT (link) DO UPDATE
	Create(ctx context.Context, article models.Article) error

	// CreateBatch inserts multiple articles into the database
	CreateBatch(ctx context.Context, articles []models.Article) error

	// GetByID retrieves an article by its ID
	GetByID(ctx context.Context, id string) (*models.Article, error)

	// GetRecent retrieves the most recent articles
	GetRecent(ctx context.Context, limit int) ([]models.Article, error)

	// GetUnreadForUser retrieves unread articles for a user
	GetUnreadForUser(ctx context.Context, userID string, limit int) ([]models.Article, error)

	// GetForRecommendation retrieves articles suitable for recommendation
	// Excludes deleted articles and articles already recommended to the user
	GetForRecommendation(ctx context.Context, userID string, limit int) ([]models.Article, error)

	// GetLowExposureArticles retrieves articles with the lowest recommend counts
	// Used for exploration/discovery in recommendation algorithm
	GetLowExposureArticles(ctx context.Context, userID string, limit int) ([]models.Article, error)

	// IncrementRecommendCount increments the recommends counter for an article
	IncrementRecommendCount(ctx context.Context, articleID string) error

	// RecordRecommendation tracks that an article was shown to a user.
	// Returns inserted=true when a new row was written, and inserted=false
	// when the (user, article) pair already existed (ON CONFLICT DO NOTHING).
	// Callers use this to decide whether downstream side effects — such as
	// incrementing articles.recommends — should fire for this call.
	RecordRecommendation(ctx context.Context, userID string, articleID string) (inserted bool, err error)

	// MarkOldArticlesAsDeleted sets deleted=true for articles older than N days
	MarkOldArticlesAsDeleted(ctx context.Context, days int) (int, error)

	// HardDeleteOldArticles permanently removes articles older than N days
	HardDeleteOldArticles(ctx context.Context, days int) (int, error)

	// SetUserRepository sets the user repository (for dependency injection)
	SetUserRepository(userRepo UserRepositoryInterface)
}

// VotedArticle represents an article with the user's vote type
type VotedArticle struct {
	Article  models.Article `json:"article"`
	VoteType string         `json:"vote_type"`
}

// VoteRepositoryInterface defines the contract for vote database operations
type VoteRepositoryInterface interface {
	// RecordVote inserts or updates a vote (upsert)
	// Updates articles.upvotes and articles.downvotes counts atomically
	RecordVote(ctx context.Context, userID string, articleID string, voteType string) error

	// RemoveVote deletes a vote and updates article counts
	RemoveVote(ctx context.Context, userID string, articleID string) error

	// GetVoteCounts returns upvote/downvote counts for an article
	GetVoteCounts(ctx context.Context, articleID string) (upvotes int, downvotes int, err error)

	// GetUserVote returns the user's vote for an article (if any)
	// Returns empty string if user hasn't voted
	GetUserVote(ctx context.Context, userID string, articleID string) (voteType string, err error)

	// GetUserVotedArticles returns all articles a user has voted on with their vote types
	GetUserVotedArticles(ctx context.Context, userID string, limit int, offset int) ([]VotedArticle, error)
}

// UserRepositoryInterface defines the contract for user database operations
type UserRepositoryInterface interface {
	// EnsureUserExists creates a user if they don't already exist
	// This implements the auto-create user behavior
	EnsureUserExists(ctx context.Context, userID string) error

	// MarkArticleAsRead marks an article as read for a user
	MarkArticleAsRead(ctx context.Context, userID, articleID string) error

	// GetReadArticleIDs returns the IDs of articles a user has read
	GetReadArticleIDs(ctx context.Context, userID string) ([]string, error)
}

// Ensure concrete implementations satisfy interfaces at compile time
var (
	_ ArticleRepositoryInterface = (*articleRepository)(nil)
	_ VoteRepositoryInterface    = (*voteRepository)(nil)
	_ UserRepositoryInterface    = (*userRepository)(nil)
)
