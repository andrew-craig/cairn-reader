package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// contentRow builds a sqlmock result row matching contentColumns' order, for
// tests exercising Create/CreateWithTx/BulkCreate's ON CONFLICT ... DO
// UPDATE ... RETURNING path (which always returns the full persisted row).
func contentRow(id uuid.UUID, contentHash, cleanedHTML, originalURL string, canonicalURL interface{},
	title string, author, publishedAt, description, imageURLs interface{},
	sourceType string, sourceFeedID, metadata interface{}, createdAt, updatedAt time.Time, orphanedAt interface{}) *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "content_hash", "cleaned_html", "original_url", "canonical_url",
		"title", "author", "published_at", "description", "image_urls",
		"source_type", "source_feed_id", "metadata", "created_at", "updated_at", "orphaned_at",
	}).AddRow(id, contentHash, cleanedHTML, originalURL, canonicalURL,
		title, author, publishedAt, description, imageURLs,
		sourceType, sourceFeedID, metadata, createdAt, updatedAt, orphanedAt)
}

func TestContentRepository_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()
	feedID := uuid.New()
	now := time.Now()

	content := &models.Content{
		ID:           contentID,
		ContentHash:  "hash123",
		CleanedHTML:  "<p>Test content</p>",
		OriginalURL:  "https://example.com/article",
		Title:        "Test Article",
		SourceType:   "rss",
		SourceFeedID: &feedID,
	}

	mock.ExpectQuery(`INSERT INTO contents`).
		WithArgs(
			sqlmock.AnyArg(), // id
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
			sqlmock.AnyArg(), // created_at
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnRows(contentRow(contentID, "hash123", "<p>Test content</p>", "https://example.com/article", nil,
			"Test Article", nil, nil, nil, nil, "rss", &feedID, nil, now, now, nil))

	err = repo.Create(ctx, content)
	assert.NoError(t, err)
	assert.NotNil(t, content.CreatedAt)
	assert.NotNil(t, content.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_Create_GeneratesUUID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	now := time.Now()
	newID := uuid.New()

	content := &models.Content{
		ContentHash: "hash123",
		CleanedHTML: "<p>Test content</p>",
		OriginalURL: "https://example.com/article",
		Title:       "Test Article",
		SourceType:  "web",
	}

	mock.ExpectQuery(`INSERT INTO contents`).
		WillReturnRows(contentRow(newID, "hash123", "<p>Test content</p>", "https://example.com/article", nil,
			"Test Article", nil, nil, nil, nil, "web", nil, nil, now, now, nil))

	err = repo.Create(ctx, content)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, content.ID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_GetByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()
	feedID := uuid.New()
	now := time.Now()
	author := "John Doe"
	description := "Test description"
	canonicalURL := "https://example.com/canonical"

	mock.ExpectQuery(`SELECT (.+) FROM contents WHERE id = \$1`).
		WithArgs(contentID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "content_hash", "cleaned_html", "original_url", "canonical_url",
			"title", "author", "published_at", "description", "image_urls",
			"source_type", "source_feed_id", "metadata", "created_at", "updated_at", "orphaned_at",
		}).AddRow(
			contentID, "hash123", "<p>Test</p>", "https://example.com", canonicalURL,
			"Test Article", author, now, description, pq.StringArray{"image1.jpg"},
			"rss", feedID, models.JSONB{"key": "value"}, now, now, nil,
		))

	result, err := repo.GetByID(ctx, contentID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, contentID, result.ID)
	assert.Equal(t, "hash123", result.ContentHash)
	assert.Equal(t, "Test Article", result.Title)
	assert.Equal(t, &author, result.Author)
	assert.Equal(t, &description, result.Description)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM contents WHERE id = \$1`).
		WithArgs(contentID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByID(ctx, contentID)
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_GetByContentHashAndFeedID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()
	feedID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM contents WHERE content_hash = \$1 AND source_feed_id = \$2 AND source_type = 'rss'`).
		WithArgs("hash123", feedID).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "content_hash", "cleaned_html", "original_url", "canonical_url",
			"title", "author", "published_at", "description", "image_urls",
			"source_type", "source_feed_id", "metadata", "created_at", "updated_at", "orphaned_at",
		}).AddRow(
			contentID, "hash123", "<p>Test</p>", "https://example.com", nil,
			"Test", nil, nil, nil, pq.StringArray{},
			"rss", feedID, nil, now, now, nil,
		))

	result, err := repo.GetByContentHashAndFeedID(ctx, "hash123", feedID)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "hash123", result.ContentHash)
	assert.Equal(t, &feedID, result.SourceFeedID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_GetByContentHashAndFeedID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	feedID := uuid.New()

	mock.ExpectQuery(`SELECT (.+) FROM contents WHERE content_hash = \$1 AND source_feed_id = \$2`).
		WithArgs("nonexistent", feedID).
		WillReturnError(sql.ErrNoRows)

	result, err := repo.GetByContentHashAndFeedID(ctx, "nonexistent", feedID)
	assert.NoError(t, err)
	assert.Nil(t, result) // Not found is not an error for deduplication check
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_Update_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()
	now := time.Now()

	content := &models.Content{
		ID:          contentID,
		ContentHash: "newhash",
		CleanedHTML: "<p>Updated</p>",
		OriginalURL: "https://example.com",
		Title:       "Updated Title",
		SourceType:  "rss",
	}

	mock.ExpectQuery(`UPDATE contents SET`).
		WithArgs(
			contentID,
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
			sqlmock.AnyArg(), // updated_at
		).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

	err = repo.Update(ctx, content)
	assert.NoError(t, err)
	assert.NotNil(t, content.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_Update_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	content := &models.Content{
		ID:          uuid.New(),
		ContentHash: "hash",
		CleanedHTML: "<p>Test</p>",
		OriginalURL: "https://example.com",
		Title:       "Test",
		SourceType:  "web",
	}

	mock.ExpectQuery(`UPDATE contents SET`).
		WillReturnError(sql.ErrNoRows)

	err = repo.Update(ctx, content)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_DeleteOrphaned_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	olderThan := 90 * 24 * time.Hour

	mock.ExpectExec(`DELETE FROM contents WHERE orphaned_at IS NOT NULL AND orphaned_at < \$1`).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 5))

	deleted, err := repo.DeleteOrphaned(ctx, olderThan)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), deleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_List_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	now := time.Now()

	mock.ExpectQuery(`SELECT (.+) FROM contents ORDER BY created_at DESC LIMIT \$1 OFFSET \$2`).
		WithArgs(10, 0).
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "content_hash", "cleaned_html", "original_url", "canonical_url",
			"title", "author", "published_at", "description", "image_urls",
			"source_type", "source_feed_id", "metadata", "created_at", "updated_at", "orphaned_at",
		}).
			AddRow(uuid.New(), "hash1", "<p>1</p>", "https://example.com/1", nil, "Title 1", nil, nil, nil, pq.StringArray{}, "web", nil, nil, now, now, nil).
			AddRow(uuid.New(), "hash2", "<p>2</p>", "https://example.com/2", nil, "Title 2", nil, nil, nil, pq.StringArray{}, "web", nil, nil, now, now, nil))

	results, err := repo.List(ctx, 10, 0)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Title 1", results[0].Title)
	assert.Equal(t, "Title 2", results[1].Title)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestContentRepository_GetByContentHashesAndFeedID_Success tests the bulk hash lookup
// NOTE: This test is skipped because it requires PostgreSQL-specific array functionality
// that doesn't work with sqlmock. Use integration tests with testcontainers for this.
func TestContentRepository_GetByContentHashesAndFeedID_Success(t *testing.T) {
	t.Skip("PostgreSQL array functionality requires integration testing with real database")
}

func TestContentRepository_GetByContentHashesAndFeedID_EmptyInput(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	results, err := repo.GetByContentHashesAndFeedID(ctx, []string{}, uuid.New())
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestContentRepository_BulkCreate_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contents := []*models.Content{
		{
			ContentHash: "hash1",
			CleanedHTML: "<p>1</p>",
			OriginalURL: "https://example.com/1",
			Title:       "Title 1",
			SourceType:  "rss",
		},
		{
			ContentHash: "hash2",
			CleanedHTML: "<p>2</p>",
			OriginalURL: "https://example.com/2",
			Title:       "Title 2",
			SourceType:  "rss",
		},
	}

	now := time.Now()
	mock.ExpectQuery(`INSERT INTO contents`).
		WillReturnRows(contentRow(uuid.New(), "hash1", "<p>1</p>", "https://example.com/1", nil,
			"Title 1", nil, nil, nil, nil, "rss", nil, nil, now, now, nil).
			AddRow(uuid.New(), "hash2", "<p>2</p>", "https://example.com/2", nil,
				"Title 2", nil, nil, nil, nil, "rss", nil, nil, now, now, nil))

	err = repo.BulkCreate(ctx, contents)
	assert.NoError(t, err)
	for _, c := range contents {
		assert.NotEqual(t, uuid.Nil, c.ID)
		assert.NotNil(t, c.CreatedAt)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_BulkCreate_EmptyInput(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	err = repo.BulkCreate(ctx, []*models.Content{})
	assert.NoError(t, err)
}

func TestContentRepository_BulkCreate_ExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contents := []*models.Content{
		{
			ContentHash: "hash1",
			CleanedHTML: "<p>1</p>",
			OriginalURL: "https://example.com/1",
			Title:       "Title 1",
			SourceType:  "rss",
		},
	}

	mock.ExpectQuery(`INSERT INTO contents`).
		WillReturnError(sql.ErrConnDone)

	err = repo.BulkCreate(ctx, contents)
	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_CreateWithTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()
	now := time.Now()

	content := &models.Content{
		ID:          contentID,
		ContentHash: "hash123",
		CleanedHTML: "<p>Test</p>",
		OriginalURL: "https://example.com",
		Title:       "Test",
		SourceType:  "web",
	}

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)

	mock.ExpectQuery(`INSERT INTO contents`).
		WillReturnRows(contentRow(contentID, "hash123", "<p>Test</p>", "https://example.com", nil,
			"Test", nil, nil, nil, nil, "web", nil, nil, now, now, nil))

	err = repo.CreateWithTx(ctx, tx, content)
	assert.NoError(t, err)
	assert.NotNil(t, content.CreatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestContentRepository_UpdateWithTx_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewContentRepository(db)
	ctx := context.Background()

	contentID := uuid.New()
	now := time.Now()

	content := &models.Content{
		ID:          contentID,
		ContentHash: "newhash",
		CleanedHTML: "<p>Updated</p>",
		OriginalURL: "https://example.com",
		Title:       "Updated",
		SourceType:  "rss",
	}

	mock.ExpectBegin()
	tx, err := db.Begin()
	require.NoError(t, err)

	mock.ExpectQuery(`UPDATE contents SET`).
		WillReturnRows(sqlmock.NewRows([]string{"updated_at"}).AddRow(now))

	err = repo.UpdateWithTx(ctx, tx, content)
	assert.NoError(t, err)
	assert.NotNil(t, content.UpdatedAt)
	assert.NoError(t, mock.ExpectationsWereMet())
}
