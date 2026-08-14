// Package db provides database access for the recommender service.
// It implements the repository pattern for articles, users, and votes.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	apperrors "github.com/cairn-app/cairn-reader/pkg/errors"
	"github.com/cairn-app/cairn-reader/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// CreateBatch inserts multiple articles in a single multi-row INSERT.
// Implements Phase 2 deduplication: ON CONFLICT (link) DO UPDATE
// Preserves vote counts, recommends, and deleted status on updates
//
// Article ids are content hashes (SHA256 of cleaned content), so syndicated
// or re-published content can arrive under a different link but the same
// id as an article already stored. Postgres allows only one ON CONFLICT
// arbiter per statement, and this table enforces uniqueness on both id
// (primary key) and link, so an id collision would otherwise raise an
// unhandled primary-key violation and roll back the whole batch. Articles
// whose id already exists under a *different* link — in the database or
// earlier in this same batch — are dropped before the INSERT runs: the
// first-seen link for that content stays canonical and the rest of the
// batch is still written. An article whose id exists under the *same*
// link is kept (it's an ordinary re-fetch, not a collision) and still
// flows through ON CONFLICT (link) DO UPDATE to refresh its metadata.
// Articles with an empty link, or a link repeated later in the batch, are
// dropped for the same reason (an empty/duplicate link makes two source
// rows target the same ON CONFLICT (link) row, which Postgres rejects
// outright).
//
// A concurrent writer can insert one of these ids (under a different link)
// between dedupeForInsert's check and the INSERT below, so on a PK conflict
// we re-filter against the now-current table state and retry, bounded to
// maxCreateBatchAttempts rounds of concurrent collisions.
func (r *articleRepository) CreateBatch(ctx context.Context, articles []models.Article) error {
	const maxCreateBatchAttempts = 3

	articles, err := r.dedupeForInsert(ctx, articles)
	if err != nil {
		return err
	}

	var lastErr error
	for attempt := 0; attempt < maxCreateBatchAttempts; attempt++ {
		if len(articles) == 0 {
			return nil
		}

		lastErr = r.execBatchInsert(ctx, articles)
		if lastErr == nil {
			return nil
		}
		if !isArticlesPKConflict(lastErr) {
			return lastErr
		}

		articles, err = r.dedupeForInsert(ctx, articles)
		if err != nil {
			return err
		}
	}

	return lastErr
}

// execBatchInsert runs the multi-row upsert for an already-deduplicated
// batch (see dedupeForInsert).
func (r *articleRepository) execBatchInsert(ctx context.Context, articles []models.Article) error {
	// Build argument list and placeholder rows for a single INSERT statement.
	const colsPerRow = 11
	valueRows := make([]string, len(articles))
	args := make([]interface{}, 0, len(articles)*colsPerRow)

	for i, article := range articles {
		base := i * colsPerRow
		valueRows[i] = fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,NOW(),NOW())",
			base+1, base+2, base+3, base+4, base+5,
			base+6, base+7, base+8, base+9, base+10, base+11,
		)
		args = append(args,
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
	}

	query := `INSERT INTO articles (
		id, title, link, description, content, author, published,
		feed_url, feed_title, categories, feed_id, created_at, updated_at
	) VALUES ` + strings.Join(valueRows, ", ") + `
	ON CONFLICT (link) DO UPDATE SET
		title = EXCLUDED.title,
		description = EXCLUDED.description,
		content = EXCLUDED.content,
		author = EXCLUDED.author,
		published = EXCLUDED.published,
		updated_at = NOW()
	WHERE articles.deleted = false`

	if _, err := r.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to batch insert articles: %w", err)
	}

	return nil
}

// isArticlesPKConflict reports whether err is a unique-violation on the
// articles primary key (id), as opposed to the link constraint that
// ON CONFLICT (link) already handles.
func isArticlesPKConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "articles_pkey"
}

// dedupeForInsert collapses a batch to the rows CreateBatch's single
// multi-row INSERT can safely handle: at most one row per id and per link,
// and none whose id already exists in the database under a different link
// (see CreateBatch).
func (r *articleRepository) dedupeForInsert(ctx context.Context, articles []models.Article) ([]models.Article, error) {
	if len(articles) == 0 {
		return nil, nil
	}

	seenIDs := make(map[string]struct{}, len(articles))
	seenLinks := make(map[string]struct{}, len(articles))
	deduped := make([]models.Article, 0, len(articles))

	for _, article := range articles {
		if article.Link == "" {
			continue
		}
		if _, ok := seenIDs[article.ID]; ok {
			continue
		}
		if _, ok := seenLinks[article.Link]; ok {
			continue
		}
		seenIDs[article.ID] = struct{}{}
		seenLinks[article.Link] = struct{}{}
		deduped = append(deduped, article)
	}
	if len(deduped) == 0 {
		return nil, nil
	}

	ids := make([]string, len(deduped))
	for i, article := range deduped {
		ids[i] = article.ID
	}

	rows, err := r.db.Query(ctx, `SELECT id, link FROM articles WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing article ids: %w", err)
	}
	defer rows.Close()

	existingLinks := make(map[string]string, len(ids))
	for rows.Next() {
		var id, link string
		if err := rows.Scan(&id, &link); err != nil {
			return nil, fmt.Errorf("failed to scan existing article id: %w", err)
		}
		existingLinks[id] = link
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating existing article ids: %w", err)
	}

	insertable := deduped[:0]
	for _, article := range deduped {
		if existingLink, ok := existingLinks[article.ID]; ok && existingLink != article.Link {
			continue
		}
		insertable = append(insertable, article)
	}

	return insertable, nil
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

// Search returns non-deleted articles whose title, description, or author
// match the query string (case-insensitive ILIKE), ordered by published DESC.
func (r *articleRepository) Search(ctx context.Context, q string, limit, offset int) ([]models.Article, error) {
	query := `
		SELECT id, title, link, description, content, author, published, feed_url, feed_title, categories, feed_id,
		       upvotes, downvotes, recommends, deleted, created_at, updated_at
		FROM articles
		WHERE deleted = false
		  AND (title ILIKE $1 OR description ILIKE $1 OR author ILIKE $1)
		ORDER BY published DESC
		LIMIT $2 OFFSET $3
	`

	pattern := "%" + q + "%"
	rows, err := r.db.Query(ctx, query, pattern, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search articles: %w", err)
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

// RecordRecommendation tracks that an article was shown to a user.
// Uses ON CONFLICT DO NOTHING so duplicate (user, article) pairs are
// no-ops. Returns inserted=true only when a new row was written.
func (r *articleRepository) RecordRecommendation(ctx context.Context, userID string, articleID string) (bool, error) {
	query := `
		INSERT INTO recommendations (user_id, article_id, recommended_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, article_id) DO NOTHING
	`

	result, err := r.db.Exec(ctx, query, userID, articleID)
	if err != nil {
		return false, fmt.Errorf("failed to record recommendation: %w", err)
	}

	return result.RowsAffected() == 1, nil
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
