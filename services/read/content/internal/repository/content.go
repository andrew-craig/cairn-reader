package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
)

// ContentRepository defines the interface for content database operations
type ContentRepository interface {
	// Create creates a new content record
	Create(ctx context.Context, content *models.Content) error

	// CreateWithTx creates a new content record within a transaction
	CreateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error

	// GetByID retrieves a content record by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error)

	// GetByContentHashAndFeedID retrieves content by hash and feed ID (for RSS deduplication)
	GetByContentHashAndFeedID(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error)

	// GetByContentHashAndURL retrieves non-RSS content by hash and original URL (for email/manual deduplication)
	GetByContentHashAndURL(ctx context.Context, contentHash string, originalURL string) (*models.Content, error)

	// Update updates an existing content record
	Update(ctx context.Context, content *models.Content) error

	// UpdateWithTx updates an existing content record within a transaction
	UpdateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error

	// DeleteOrphaned deletes orphaned content older than the specified duration
	DeleteOrphaned(ctx context.Context, olderThan time.Duration) (int64, error)

	// List retrieves content records with pagination
	List(ctx context.Context, limit, offset int) ([]*models.Content, error)

	// GetByContentHashesAndFeedID retrieves multiple contents by their hashes and feed ID (for bulk deduplication)
	GetByContentHashesAndFeedID(ctx context.Context, contentHashes []string, feedID uuid.UUID) (map[string]*models.Content, error)

	// GetByIDs retrieves multiple content records by ID in a single query, returning a map keyed by ID.
	GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*models.Content, error)

	// BulkCreate creates multiple content records in a transaction
	BulkCreate(ctx context.Context, contents []*models.Content) error
}

// contentRepository implements ContentRepository
type contentRepository struct {
	db *sql.DB
}

// NewContentRepository creates a new ContentRepository
func NewContentRepository(db *sql.DB) ContentRepository {
	return &contentRepository{db: db}
}

// contentColumns is the full column list used by insert sites that need to
// return the true persisted row — including on the ON CONFLICT DO UPDATE
// no-op branch below, where the returned row belongs to whichever request
// (this one or a concurrent one) actually won the insert.
const contentColumns = `id, content_hash, cleaned_html, original_url, canonical_url,
	title, author, published_at, description, image_urls,
	source_type, source_feed_id, metadata, created_at, updated_at, orphaned_at`

// rssConflictTarget/nonRSSConflictTarget name the two partial UNIQUE indexes
// (idx_contents_rss_dedup, idx_contents_nonrss_dedup) content dedup relies
// on. RSS dedupes on (content_hash, source_feed_id); email/manual content
// has no feed, so it dedupes on (content_hash, original_url) instead — see
// migration 000004_unique_content_dedup.
const (
	rssConflictTarget    = `(content_hash, source_feed_id) WHERE source_type = 'rss'`
	nonRSSConflictTarget = `(content_hash, original_url) WHERE source_type != 'rss'`
)

func conflictTargetFor(sourceType string) string {
	if sourceType == models.SourceTypeRSS {
		return rssConflictTarget
	}
	return nonRSSConflictTarget
}

// contentDedupKey mirrors the arbiter a row will conflict on, so rows
// returned by an insert's RETURNING clause can be matched back to the
// content struct(s) that requested them.
func contentDedupKey(c *models.Content) string {
	if c.SourceType == models.SourceTypeRSS {
		feedID := ""
		if c.SourceFeedID != nil {
			feedID = c.SourceFeedID.String()
		}
		return "rss\x00" + c.ContentHash + "\x00" + feedID
	}
	return "other\x00" + c.ContentHash + "\x00" + c.OriginalURL
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanContent(s rowScanner) (*models.Content, error) {
	c := &models.Content{}
	err := s.Scan(
		&c.ID,
		&c.ContentHash,
		&c.CleanedHTML,
		&c.OriginalURL,
		&c.CanonicalURL,
		&c.Title,
		&c.Author,
		&c.PublishedAt,
		&c.Description,
		&c.ImageURLs,
		&c.SourceType,
		&c.SourceFeedID,
		&c.Metadata,
		&c.CreatedAt,
		&c.UpdatedAt,
		&c.OrphanedAt,
	)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// Create creates a new content record. On a dedup conflict (RSS:
// content_hash+source_feed_id, non-RSS: content_hash+original_url) it
// resolves to the already-persisted row instead of erroring, so a
// concurrent duplicate delivery is idempotent at the DB level.
func (r *contentRepository) Create(ctx context.Context, content *models.Content) error {
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}
	now := time.Now()
	content.CreatedAt = now
	content.UpdatedAt = now

	query := `
		INSERT INTO contents (
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT ` + conflictTargetFor(content.SourceType) + `
		DO UPDATE SET updated_at = contents.updated_at
		RETURNING ` + contentColumns

	persisted, err := scanContent(r.db.QueryRowContext(
		ctx, query,
		content.ID,
		content.ContentHash,
		content.CleanedHTML,
		content.OriginalURL,
		content.CanonicalURL,
		content.Title,
		content.Author,
		content.PublishedAt,
		content.Description,
		content.ImageURLs,
		content.SourceType,
		content.SourceFeedID,
		content.Metadata,
		content.CreatedAt,
		content.UpdatedAt,
	))
	if err != nil {
		return fmt.Errorf("failed to create content: %w", err)
	}

	*content = *persisted
	return nil
}

// CreateWithTx creates a new content record within a transaction. See
// Create for the dedup-conflict resolution behavior.
func (r *contentRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}
	now := time.Now()
	content.CreatedAt = now
	content.UpdatedAt = now

	query := `
		INSERT INTO contents (
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		ON CONFLICT ` + conflictTargetFor(content.SourceType) + `
		DO UPDATE SET updated_at = contents.updated_at
		RETURNING ` + contentColumns

	persisted, err := scanContent(tx.QueryRowContext(
		ctx, query,
		content.ID,
		content.ContentHash,
		content.CleanedHTML,
		content.OriginalURL,
		content.CanonicalURL,
		content.Title,
		content.Author,
		content.PublishedAt,
		content.Description,
		content.ImageURLs,
		content.SourceType,
		content.SourceFeedID,
		content.Metadata,
		content.CreatedAt,
		content.UpdatedAt,
	))
	if err != nil {
		return fmt.Errorf("failed to create content with transaction: %w", err)
	}

	*content = *persisted
	return nil
}

// GetByID retrieves a content record by ID
func (r *contentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	query := `
		SELECT
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at, orphaned_at
		FROM contents
		WHERE id = $1
	`

	content := &models.Content{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&content.ID,
		&content.ContentHash,
		&content.CleanedHTML,
		&content.OriginalURL,
		&content.CanonicalURL,
		&content.Title,
		&content.Author,
		&content.PublishedAt,
		&content.Description,
		&content.ImageURLs,
		&content.SourceType,
		&content.SourceFeedID,
		&content.Metadata,
		&content.CreatedAt,
		&content.UpdatedAt,
		&content.OrphanedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("content not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get content: %w", err)
	}

	return content, nil
}

// GetByContentHashAndFeedID retrieves content by hash and feed ID (for RSS deduplication)
func (r *contentRepository) GetByContentHashAndFeedID(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	query := `
		SELECT
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at, orphaned_at
		FROM contents
		WHERE content_hash = $1 AND source_feed_id = $2 AND source_type = 'rss'
	`

	content := &models.Content{}
	err := r.db.QueryRowContext(ctx, query, contentHash, feedID).Scan(
		&content.ID,
		&content.ContentHash,
		&content.CleanedHTML,
		&content.OriginalURL,
		&content.CanonicalURL,
		&content.Title,
		&content.Author,
		&content.PublishedAt,
		&content.Description,
		&content.ImageURLs,
		&content.SourceType,
		&content.SourceFeedID,
		&content.Metadata,
		&content.CreatedAt,
		&content.UpdatedAt,
		&content.OrphanedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for deduplication check
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get content by hash and feed ID: %w", err)
	}

	return content, nil
}

// GetByContentHashAndURL retrieves non-RSS content by hash and original URL
// (for email/manual deduplication, where there is no feed to key off)
func (r *contentRepository) GetByContentHashAndURL(ctx context.Context, contentHash string, originalURL string) (*models.Content, error) {
	query := `
		SELECT ` + contentColumns + `
		FROM contents
		WHERE content_hash = $1 AND original_url = $2 AND source_type != 'rss'
	`

	persisted, err := scanContent(r.db.QueryRowContext(ctx, query, contentHash, originalURL))
	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for deduplication check
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get content by hash and URL: %w", err)
	}

	return persisted, nil
}

// Update updates an existing content record
func (r *contentRepository) Update(ctx context.Context, content *models.Content) error {
	query := `
		UPDATE contents
		SET
			content_hash = $2,
			cleaned_html = $3,
			original_url = $4,
			canonical_url = $5,
			title = $6,
			author = $7,
			published_at = $8,
			description = $9,
			image_urls = $10,
			metadata = $11,
			updated_at = $12
		WHERE id = $1
		RETURNING updated_at
	`

	// Update timestamp
	content.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(
		ctx, query,
		content.ID,
		content.ContentHash,
		content.CleanedHTML,
		content.OriginalURL,
		content.CanonicalURL,
		content.Title,
		content.Author,
		content.PublishedAt,
		content.Description,
		content.ImageURLs,
		content.Metadata,
		content.UpdatedAt,
	).Scan(&content.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("content not found for update: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to update content: %w", err)
	}

	return nil
}

// UpdateWithTx updates an existing content record within a transaction
func (r *contentRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	query := `
		UPDATE contents
		SET
			content_hash = $2,
			cleaned_html = $3,
			original_url = $4,
			canonical_url = $5,
			title = $6,
			author = $7,
			published_at = $8,
			description = $9,
			image_urls = $10,
			metadata = $11,
			updated_at = $12
		WHERE id = $1
		RETURNING updated_at
	`

	// Update timestamp
	content.UpdatedAt = time.Now()

	err := tx.QueryRowContext(
		ctx, query,
		content.ID,
		content.ContentHash,
		content.CleanedHTML,
		content.OriginalURL,
		content.CanonicalURL,
		content.Title,
		content.Author,
		content.PublishedAt,
		content.Description,
		content.ImageURLs,
		content.Metadata,
		content.UpdatedAt,
	).Scan(&content.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("content not found for update: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to update content with transaction: %w", err)
	}

	return nil
}

// DeleteOrphaned deletes orphaned content older than the specified duration
func (r *contentRepository) DeleteOrphaned(ctx context.Context, olderThan time.Duration) (int64, error) {
	query := `
		DELETE FROM contents
		WHERE orphaned_at IS NOT NULL
		AND orphaned_at < $1
	`

	cutoffTime := time.Now().Add(-olderThan)
	result, err := r.db.ExecContext(ctx, query, cutoffTime)
	if err != nil {
		return 0, fmt.Errorf("failed to delete orphaned content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// List retrieves content records with pagination
func (r *contentRepository) List(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	query := `
		SELECT
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at, orphaned_at
		FROM contents
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list contents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var contents []*models.Content
	for rows.Next() {
		content := &models.Content{}
		err := rows.Scan(
			&content.ID,
			&content.ContentHash,
			&content.CleanedHTML,
			&content.OriginalURL,
			&content.CanonicalURL,
			&content.Title,
			&content.Author,
			&content.PublishedAt,
			&content.Description,
			&content.ImageURLs,
			&content.SourceType,
			&content.SourceFeedID,
			&content.Metadata,
			&content.CreatedAt,
			&content.UpdatedAt,
			&content.OrphanedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content: %w", err)
		}
		contents = append(contents, content)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating content rows: %w", err)
	}

	return contents, nil
}

// GetByContentHashesAndFeedID retrieves multiple contents by their hashes and feed ID (for bulk deduplication)
func (r *contentRepository) GetByContentHashesAndFeedID(ctx context.Context, contentHashes []string, feedID uuid.UUID) (map[string]*models.Content, error) {
	if len(contentHashes) == 0 {
		return make(map[string]*models.Content), nil
	}

	query := `
		SELECT
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at, orphaned_at
		FROM contents
		WHERE content_hash = ANY($1) AND source_feed_id = $2 AND source_type = 'rss'
	`

	rows, err := r.db.QueryContext(ctx, query, contentHashes, feedID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents by hashes and feed ID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[string]*models.Content)
	for rows.Next() {
		content := &models.Content{}
		err := rows.Scan(
			&content.ID,
			&content.ContentHash,
			&content.CleanedHTML,
			&content.OriginalURL,
			&content.CanonicalURL,
			&content.Title,
			&content.Author,
			&content.PublishedAt,
			&content.Description,
			&content.ImageURLs,
			&content.SourceType,
			&content.SourceFeedID,
			&content.Metadata,
			&content.CreatedAt,
			&content.UpdatedAt,
			&content.OrphanedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content: %w", err)
		}
		result[content.ContentHash] = content
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating content rows: %w", err)
	}

	return result, nil
}

// GetByIDs retrieves multiple content records by ID in a single query.
func (r *contentRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*models.Content, error) {
	if len(ids) == 0 {
		return make(map[uuid.UUID]*models.Content), nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at, orphaned_at
		FROM contents
		WHERE id IN (%s)
	`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get contents by IDs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make(map[uuid.UUID]*models.Content, len(ids))
	for rows.Next() {
		content := &models.Content{}
		err := rows.Scan(
			&content.ID,
			&content.ContentHash,
			&content.CleanedHTML,
			&content.OriginalURL,
			&content.CanonicalURL,
			&content.Title,
			&content.Author,
			&content.PublishedAt,
			&content.Description,
			&content.ImageURLs,
			&content.SourceType,
			&content.SourceFeedID,
			&content.Metadata,
			&content.CreatedAt,
			&content.UpdatedAt,
			&content.OrphanedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan content: %w", err)
		}
		result[content.ID] = content
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating content rows: %w", err)
	}

	return result, nil
}

// BulkCreate creates multiple content records. Rows are grouped by dedup
// arbiter (RSS vs. non-RSS — see conflictTargetFor) since a single INSERT
// can only declare one ON CONFLICT target, then each group is inserted with
// ON CONFLICT DO UPDATE so a row colliding with an existing one (or with
// another concurrent request) resolves to that row's real persisted state
// instead of PK-colliding or silently going missing.
func (r *contentRepository) BulkCreate(ctx context.Context, contents []*models.Content) error {
	if len(contents) == 0 {
		return nil
	}

	var rssItems, otherItems []*models.Content
	for _, content := range contents {
		if content.SourceType == models.SourceTypeRSS {
			rssItems = append(rssItems, content)
		} else {
			otherItems = append(otherItems, content)
		}
	}

	if len(rssItems) > 0 {
		if err := r.bulkInsertGroup(ctx, rssItems, rssConflictTarget); err != nil {
			return err
		}
	}
	if len(otherItems) > 0 {
		if err := r.bulkInsertGroup(ctx, otherItems, nonRSSConflictTarget); err != nil {
			return err
		}
	}

	return nil
}

// bulkInsertGroup inserts a batch of contents that all share the same ON
// CONFLICT arbiter. Postgres rejects a multi-row INSERT ... ON CONFLICT DO
// UPDATE that proposes the same conflict key twice in one statement, so
// duplicates within the batch (e.g. a bulk request that redundantly repeats
// an item) are collapsed to a single VALUES row before inserting, then the
// resolved row is fanned back out to every content struct that shares the key.
func (r *contentRepository) bulkInsertGroup(ctx context.Context, group []*models.Content, conflict string) error {
	now := time.Now()

	const colsPerRow = 15
	keys := make([]string, len(group))
	seen := make(map[string]bool, len(group))
	var toInsert []*models.Content

	for i, content := range group {
		if content.ID == uuid.Nil {
			content.ID = uuid.New()
		}
		content.CreatedAt = now
		content.UpdatedAt = now

		key := contentDedupKey(content)
		keys[i] = key
		if !seen[key] {
			seen[key] = true
			toInsert = append(toInsert, content)
		}
	}

	placeholders := make([]string, len(toInsert))
	args := make([]interface{}, 0, len(toInsert)*colsPerRow)
	for i, content := range toInsert {
		base := i * colsPerRow
		placeholders[i] = fmt.Sprintf(
			"($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			base+1, base+2, base+3, base+4, base+5,
			base+6, base+7, base+8, base+9, base+10,
			base+11, base+12, base+13, base+14, base+15,
		)
		args = append(args,
			content.ID,
			content.ContentHash,
			content.CleanedHTML,
			content.OriginalURL,
			content.CanonicalURL,
			content.Title,
			content.Author,
			content.PublishedAt,
			content.Description,
			content.ImageURLs,
			content.SourceType,
			content.SourceFeedID,
			content.Metadata,
			content.CreatedAt,
			content.UpdatedAt,
		)
	}

	query := `INSERT INTO contents (
		id, content_hash, cleaned_html, original_url, canonical_url,
		title, author, published_at, description, image_urls,
		source_type, source_feed_id, metadata, created_at, updated_at
	) VALUES ` + strings.Join(placeholders, ", ") + `
	ON CONFLICT ` + conflict + `
	DO UPDATE SET updated_at = contents.updated_at
	RETURNING ` + contentColumns

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to bulk create contents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	byKey := make(map[string]*models.Content, len(toInsert))
	for rows.Next() {
		persisted, err := scanContent(rows)
		if err != nil {
			return fmt.Errorf("failed to scan bulk-created content: %w", err)
		}
		byKey[contentDedupKey(persisted)] = persisted
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating bulk-created rows: %w", err)
	}

	for i, content := range group {
		persisted, ok := byKey[keys[i]]
		if !ok {
			return fmt.Errorf("bulk create: no result returned for content_hash %s", content.ContentHash)
		}
		*content = *persisted
	}

	return nil
}
