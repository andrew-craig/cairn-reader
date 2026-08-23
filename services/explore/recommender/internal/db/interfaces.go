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

	// RecordShown atomically records that an article was shown to a user
	// and, the first time for that (user, article) pair, increments
	// articles.recommends. Both writes happen in one transaction, so a
	// failure can never leave the recommendations row and the recommends
	// counter permanently disagreeing — either both land or neither does,
	// and a retry is always safe. Returns recorded=true only when this
	// call performed both writes; recorded=false when the pair was
	// already recorded (ON CONFLICT DO NOTHING short-circuits before the
	// counter is touched). Returns apperrors.ErrArticleNotFound if the
	// article does not exist.
	RecordShown(ctx context.Context, userID string, articleID string) (recorded bool, err error)

	// Search returns non-deleted articles whose title, description, or author
	// contain q (case-insensitive), paginated with limit/offset.
	Search(ctx context.Context, q string, limit, offset int) ([]models.Article, error)

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

	// GetUserVoteStats returns aggregate upvote/downvote counts for a user without fetching rows
	GetUserVoteStats(ctx context.Context, userID string) (upvotes int, downvotes int, err error)
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
