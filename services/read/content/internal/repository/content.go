package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/models"
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

// Create creates a new content record
func (r *contentRepository) Create(ctx context.Context, content *models.Content) error {
	query := `
		INSERT INTO contents (
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING id, created_at, updated_at
	`

	// Generate UUID if not provided
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now()
	content.CreatedAt = now
	content.UpdatedAt = now

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
		content.SourceType,
		content.SourceFeedID,
		content.Metadata,
		content.CreatedAt,
		content.UpdatedAt,
	).Scan(&content.ID, &content.CreatedAt, &content.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create content: %w", err)
	}

	return nil
}

// CreateWithTx creates a new content record within a transaction
func (r *contentRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	query := `
		INSERT INTO contents (
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING id, created_at, updated_at
	`

	// Generate UUID if not provided
	if content.ID == uuid.Nil {
		content.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now()
	content.CreatedAt = now
	content.UpdatedAt = now

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
		content.SourceType,
		content.SourceFeedID,
		content.Metadata,
		content.CreatedAt,
		content.UpdatedAt,
	).Scan(&content.ID, &content.CreatedAt, &content.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create content with transaction: %w", err)
	}

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
	defer rows.Close()

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
	defer rows.Close()

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

// BulkCreate creates multiple content records in a transaction
func (r *contentRepository) BulkCreate(ctx context.Context, contents []*models.Content) error {
	if len(contents) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Prepare the insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO contents (
			id, content_hash, cleaned_html, original_url, canonical_url,
			title, author, published_at, description, image_urls,
			source_type, source_feed_id, metadata, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
		)
		RETURNING id, created_at, updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	now := time.Now()

	// Insert each content
	for _, content := range contents {
		// Generate UUID if not provided
		if content.ID == uuid.Nil {
			content.ID = uuid.New()
		}

		// Set timestamps
		content.CreatedAt = now
		content.UpdatedAt = now

		err := stmt.QueryRowContext(
			ctx,
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
		).Scan(&content.ID, &content.CreatedAt, &content.UpdatedAt)

		if err != nil {
			return fmt.Errorf("failed to insert content in bulk: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit bulk create transaction: %w", err)
	}

	return nil
}
