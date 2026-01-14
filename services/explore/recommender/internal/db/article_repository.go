// Package db provides database access for the recommender service.
// It implements the repository pattern for articles, users, and votes.
package db

import (
	"context"
	"fmt"
	"log/slog"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// articleRepository handles article database operations
type articleRepository struct {
	db             *pgxpool.Pool
	userRepository UserRepositoryInterface
}

// NewArticleRepository creates a new article repository
func NewArticleRepository(db *pgxpool.Pool) ArticleRepositoryInterface {
	return &articleRepository{db: db}
}

// SetUserRepository sets the user repository (for dependency injection after creation)
func (r *articleRepository) SetUserRepository(userRepo UserRepositoryInterface) {
	r.userRepository = userRepo
}

// Create inserts a new article into the database
// Implements Phase 2 deduplication: ON CONFLICT (link) DO UPDATE
// Preserves vote counts, recommends, and deleted status on updates
func (r *articleRepository) Create(ctx context.Context, article models.Article) error {
	query := `
		INSERT INTO articles (
			id, title, link, description, content, author, published,
			feed_url, feed_title, categories, feed_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (link) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			author = EXCLUDED.author,
			published = EXCLUDED.published,
			updated_at = NOW()
		WHERE articles.deleted = false
	`

	_, err := r.db.Exec(ctx, query,
		article.ID,
		article.Title,
		article.Link,
		article.Description,
		article.Content,
		article.Author,
		article.Published,
		article.FeedURL,
		article.FeedTitle,
		article.Categories,
		article.FeedID,
	)

	if err != nil {
		return fmt.Errorf("failed to create article: %w", err)
	}

	return nil
}

// CreateBatch inserts multiple articles into the database
// Implements Phase 2 deduplication: ON CONFLICT (link) DO UPDATE
// Preserves vote counts, recommends, and deleted status on updates
func (r *articleRepository) CreateBatch(ctx context.Context, articles []models.Article) error {
	if len(articles) == 0 {
		return nil
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && err != pgx.ErrTxClosed {
			slog.Error("failed to rollback transaction", slog.Any("error", err))
		}
	}()

	query := `
		INSERT INTO articles (
			id, title, link, description, content, author, published,
			feed_url, feed_title, categories, feed_id, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		ON CONFLICT (link) DO UPDATE SET
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			content = EXCLUDED.content,
			author = EXCLUDED.author,
			published = EXCLUDED.published,
			updated_at = NOW()
		WHERE articles.deleted = false
	`

	for _, article := range articles {
		_, err := tx.Exec(ctx, query,
			article.ID,
			article.Title,
			article.Link,
			article.Description,
			article.Content,
			article.Author,
			article.Published,
			article.FeedURL,
			article.FeedTitle,
			article.Categories,
			article.FeedID,
		)
		if err != nil {
			return fmt.Errorf("failed to insert article %s: %w", article.ID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetByID retrieves an article by its ID
func (r *articleRepository) GetByID(ctx context.Context, id string) (*models.Article, error) {
	query := `
		SELECT id, title, link, description, content, author, published, feed_url, feed_title, categories, feed_id,
		       upvotes, downvotes, recommends, deleted, created_at, updated_at
		FROM articles
		WHERE id = $1
	`

	var article models.Article

	err := r.db.QueryRow(ctx, query, id).Scan(
		&article.ID,
		&article.Title,
		&article.Link,
		&article.Description,
		&article.Content,
		&article.Author,
		&article.Published,
		&article.FeedURL,
		&article.FeedTitle,
		&article.Categories,
		&article.FeedID,
		&article.Upvotes,
		&article.Downvotes,
		&article.Recommends,
		&article.Deleted,
		&article.CreatedAt,
		&article.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, apperrors.ErrArticleNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get article: %w", err)
	}

	return &article, nil
}

// GetRecent retrieves the most recent articles
func (r *articleRepository) GetRecent(ctx context.Context, limit int) ([]models.Article, error) {
	query := `
		SELECT id, title, link, description, content, author, published, feed_url, feed_title, categories, feed_id,
		       upvotes, downvotes, recommends, deleted, created_at, updated_at
		FROM articles
		ORDER BY published DESC
		LIMIT $1
	`

	rows, err := r.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent articles: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// GetUnreadForUser retrieves unread articles for a user
func (r *articleRepository) GetUnreadForUser(ctx context.Context, userID string, limit int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.link, a.description, a.content, a.author, a.published,
		       a.feed_url, a.feed_title, a.categories, a.feed_id,
		       a.upvotes, a.downvotes, a.recommends, a.deleted, a.created_at, a.updated_at
		FROM articles a
		LEFT JOIN user_articles ua ON a.id = ua.article_id AND ua.user_id = $1
		WHERE ua.article_id IS NULL OR ua.read = false
		ORDER BY a.published DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get unread articles: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// GetForRecommendation retrieves articles suitable for recommendation
// Excludes deleted articles and articles already recommended to the user
func (r *articleRepository) GetForRecommendation(ctx context.Context, userID string, limit int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.link, a.description, a.content, a.author, a.published,
		       a.feed_url, a.feed_title, a.categories, a.feed_id,
		       a.upvotes, a.downvotes, a.recommends, a.deleted, a.created_at, a.updated_at
		FROM articles a
		LEFT JOIN recommendations r ON a.id = r.article_id AND r.user_id = $1
		WHERE a.deleted = false
		  AND r.id IS NULL
		ORDER BY a.published DESC
		LIMIT $2
	`

	slog.Info("getting articles for recommendation",
		slog.String("user_id", userID),
		slog.Int("limit", limit))

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get articles for recommendation: %w", err)
	}
	defer rows.Close()

	articles, err := r.scanArticles(rows)
	if err != nil {
		return nil, err
	}

	slog.Info("returning articles for recommendation",
		slog.Int("count", len(articles)),
		slog.String("user_id", userID))
	return articles, nil
}

// GetLowExposureArticles retrieves articles with the lowest recommend counts
// Used for exploration/discovery in recommendation algorithm
func (r *articleRepository) GetLowExposureArticles(ctx context.Context, userID string, limit int) ([]models.Article, error) {
	query := `
		SELECT a.id, a.title, a.link, a.description, a.content, a.author, a.published,
		       a.feed_url, a.feed_title, a.categories, a.feed_id,
		       a.upvotes, a.downvotes, a.recommends, a.deleted, a.created_at, a.updated_at
		FROM articles a
		LEFT JOIN recommendations r ON a.id = r.article_id AND r.user_id = $1
		WHERE a.deleted = false
		  AND r.id IS NULL
		ORDER BY a.recommends ASC, a.published DESC
		LIMIT $2
	`

	rows, err := r.db.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get low exposure articles: %w", err)
	}
	defer rows.Close()

	return r.scanArticles(rows)
}

// IncrementRecommendCount increments the recommends counter for an article
func (r *articleRepository) IncrementRecommendCount(ctx context.Context, articleID string) error {
	query := `
		UPDATE articles
		SET recommends = recommends + 1
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, articleID)
	if err != nil {
		return fmt.Errorf("failed to increment recommend count: %w", err)
	}

	if result.RowsAffected() == 0 {
		return apperrors.ErrArticleNotFound
	}

	return nil
}

// RecordRecommendation tracks that an article was recommended to a user
// Uses ON CONFLICT DO NOTHING to handle duplicate recommendations gracefully
func (r *articleRepository) RecordRecommendation(ctx context.Context, userID string, articleID string) error {
	query := `
		INSERT INTO recommendations (user_id, article_id, recommended_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, article_id) DO NOTHING
	`

	_, err := r.db.Exec(ctx, query, userID, articleID)
	if err != nil {
		return fmt.Errorf("failed to record recommendation: %w", err)
	}

	return nil
}

// MarkOldArticlesAsDeleted sets deleted=true for articles older than N days
// Returns the number of articles marked as deleted
func (r *articleRepository) MarkOldArticlesAsDeleted(ctx context.Context, days int) (int, error) {
	query := `
		UPDATE articles
		SET deleted = true, updated_at = NOW()
		WHERE created_at < NOW() - INTERVAL '1 day' * $1
		  AND deleted = false
	`

	result, err := r.db.Exec(ctx, query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to mark old articles as deleted: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// HardDeleteOldArticles permanently removes articles older than N days
// This is for maintenance and should be used with caution
// Returns the number of articles deleted
func (r *articleRepository) HardDeleteOldArticles(ctx context.Context, days int) (int, error) {
	query := `
		DELETE FROM articles
		WHERE created_at < NOW() - INTERVAL '1 day' * $1
		  AND deleted = true
	`

	result, err := r.db.Exec(ctx, query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to hard delete old articles: %w", err)
	}

	return int(result.RowsAffected()), nil
}

// scanArticles is a helper to scan multiple article rows
func (r *articleRepository) scanArticles(rows pgx.Rows) ([]models.Article, error) {
	articles := make([]models.Article, 0)

	for rows.Next() {
		var article models.Article

		err := rows.Scan(
			&article.ID,
			&article.Title,
			&article.Link,
			&article.Description,
			&article.Content,
			&article.Author,
			&article.Published,
			&article.FeedURL,
			&article.FeedTitle,
			&article.Categories,
			&article.FeedID,
			&article.Upvotes,
			&article.Downvotes,
			&article.Recommends,
			&article.Deleted,
			&article.CreatedAt,
			&article.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan article: %w", err)
		}

		articles = append(articles, article)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating articles: %w", err)
	}

	return articles, nil
}
