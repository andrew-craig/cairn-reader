package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
)

// UserContentRepository defines the interface for user-content database operations
type UserContentRepository interface {
	// Create creates a new user-content relationship
	Create(ctx context.Context, userContent *models.UserContent) error

	// CreateWithTx creates a new user-content relationship within a transaction
	CreateWithTx(ctx context.Context, tx *sql.Tx, userContent *models.UserContent) error

	// GetByID retrieves a user-content record by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.UserContent, error)

	// GetByUserAndContent retrieves a user-content record by user ID and content ID
	GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.UserContent, error)

	// ListByUser retrieves all content for a user with pagination
	ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.UserContent, error)

	// ListByUserWithFilter retrieves content for a user with filtering and pagination
	ListByUserWithFilter(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit, offset int) ([]*models.UserContent, error)

	// ListByUserWithCursor retrieves content for a user with optional filtering and keyset (cursor) pagination.
	// cursorTime and cursorID, when non-nil, restrict results to items strictly before that position.
	// Callers should request limit+1 rows to determine whether a next page exists.
	ListByUserWithCursor(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]*models.UserContent, error)

	// Update updates an existing user-content record
	Update(ctx context.Context, userContent *models.UserContent) error

	// UpdateWithTx updates an existing user-content record within a transaction
	UpdateWithTx(ctx context.Context, tx *sql.Tx, userContent *models.UserContent) error

	// UpdateMetadata updates only the metadata fields (status, scroll_position, is_favorite)
	UpdateMetadata(ctx context.Context, id uuid.UUID, status *string, scrollPosition *int, isFavorite *bool) error

	// Delete deletes a user-content relationship
	Delete(ctx context.Context, userID, contentID uuid.UUID) error

	// DeleteWithTx deletes a user-content relationship within a transaction
	DeleteWithTx(ctx context.Context, tx *sql.Tx, userID, contentID uuid.UUID) error

	// CountByUser counts the total number of content items for a user
	CountByUser(ctx context.Context, userID uuid.UUID) (int64, error)

	// Search searches user's content by title and author
	Search(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]*models.UserContent, error)

	// SearchWithCursor searches user's content using full-text search with keyset (cursor) pagination.
	// cursorTime and cursorID, when non-nil, restrict results to items strictly before that position.
	// Callers should request limit+1 rows to determine whether a next page exists.
	SearchWithCursor(ctx context.Context, userID uuid.UUID, query string, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]*models.UserContent, error)

	// BulkCreate creates multiple user-content relationships in a transaction
	BulkCreate(ctx context.Context, userContents []*models.UserContent) error
}

// userContentRepository implements UserContentRepository
type userContentRepository struct {
	db *sql.DB
}

// NewUserContentRepository creates a new UserContentRepository
func NewUserContentRepository(db *sql.DB) UserContentRepository {
	return &userContentRepository{db: db}
}

// Create creates a new user-content relationship
func (r *userContentRepository) Create(ctx context.Context, userContent *models.UserContent) error {
	query := `
		INSERT INTO user_contents (
			id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		RETURNING id, added_at, updated_at
	`

	// Generate UUID if not provided
	if userContent.ID == uuid.Nil {
		userContent.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now()
	userContent.AddedAt = now
	userContent.UpdatedAt = now

	err := r.db.QueryRowContext(
		ctx, query,
		userContent.ID,
		userContent.UserID,
		userContent.ContentID,
		userContent.Status,
		userContent.ScrollPosition,
		userContent.IsFavorite,
		userContent.AddedAt,
		userContent.UpdatedAt,
	).Scan(&userContent.ID, &userContent.AddedAt, &userContent.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user-content: %w", err)
	}

	return nil
}

// CreateWithTx creates a new user-content relationship within a transaction
func (r *userContentRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, userContent *models.UserContent) error {
	query := `
		INSERT INTO user_contents (
			id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		RETURNING id, added_at, updated_at
	`

	// Generate UUID if not provided
	if userContent.ID == uuid.Nil {
		userContent.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now()
	userContent.AddedAt = now
	userContent.UpdatedAt = now

	err := tx.QueryRowContext(
		ctx, query,
		userContent.ID,
		userContent.UserID,
		userContent.ContentID,
		userContent.Status,
		userContent.ScrollPosition,
		userContent.IsFavorite,
		userContent.AddedAt,
		userContent.UpdatedAt,
	).Scan(&userContent.ID, &userContent.AddedAt, &userContent.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user-content with transaction: %w", err)
	}

	return nil
}

// GetByID retrieves a user-content record by ID
func (r *userContentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.UserContent, error) {
	query := `
		SELECT id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		FROM user_contents
		WHERE id = $1
	`

	userContent := &models.UserContent{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&userContent.ID,
		&userContent.UserID,
		&userContent.ContentID,
		&userContent.Status,
		&userContent.ScrollPosition,
		&userContent.IsFavorite,
		&userContent.AddedAt,
		&userContent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("user-content not found: %w", err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user-content: %w", err)
	}

	return userContent, nil
}

// GetByUserAndContent retrieves a user-content record by user ID and content ID
func (r *userContentRepository) GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.UserContent, error) {
	query := `
		SELECT id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		FROM user_contents
		WHERE user_id = $1 AND content_id = $2
	`

	userContent := &models.UserContent{}
	err := r.db.QueryRowContext(ctx, query, userID, contentID).Scan(
		&userContent.ID,
		&userContent.UserID,
		&userContent.ContentID,
		&userContent.Status,
		&userContent.ScrollPosition,
		&userContent.IsFavorite,
		&userContent.AddedAt,
		&userContent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Not found is not an error for this query
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user-content by user and content: %w", err)
	}

	return userContent, nil
}

// ListByUser retrieves all content for a user with pagination
func (r *userContentRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.UserContent, error) {
	query := `
		SELECT id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		FROM user_contents
		WHERE user_id = $1
		ORDER BY added_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user contents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var userContents []*models.UserContent
	for rows.Next() {
		userContent := &models.UserContent{}
		err := rows.Scan(
			&userContent.ID,
			&userContent.UserID,
			&userContent.ContentID,
			&userContent.Status,
			&userContent.ScrollPosition,
			&userContent.IsFavorite,
			&userContent.AddedAt,
			&userContent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user-content: %w", err)
		}
		userContents = append(userContents, userContent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user-content rows: %w", err)
	}

	return userContents, nil
}

// ListByUserWithFilter retrieves content for a user with filtering and pagination
func (r *userContentRepository) ListByUserWithFilter(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit, offset int) ([]*models.UserContent, error) {
	query := `
		SELECT id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		FROM user_contents
		WHERE user_id = $1
	`

	args := []interface{}{userID}
	argPos := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	if isFavorite != nil {
		query += fmt.Sprintf(" AND is_favorite = $%d", argPos)
		args = append(args, *isFavorite)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY added_at DESC LIMIT $%d OFFSET $%d", argPos, argPos+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list user contents with filter: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var userContents []*models.UserContent
	for rows.Next() {
		userContent := &models.UserContent{}
		err := rows.Scan(
			&userContent.ID,
			&userContent.UserID,
			&userContent.ContentID,
			&userContent.Status,
			&userContent.ScrollPosition,
			&userContent.IsFavorite,
			&userContent.AddedAt,
			&userContent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user-content: %w", err)
		}
		userContents = append(userContents, userContent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user-content rows: %w", err)
	}

	return userContents, nil
}

// ListByUserWithCursor retrieves content for a user with optional filters and keyset pagination.
func (r *userContentRepository) ListByUserWithCursor(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]*models.UserContent, error) {
	query := `
		SELECT id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		FROM user_contents
		WHERE user_id = $1
	`

	args := []interface{}{userID}
	argPos := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	if isFavorite != nil {
		query += fmt.Sprintf(" AND is_favorite = $%d", argPos)
		args = append(args, *isFavorite)
		argPos++
	}

	if cursorTime != nil && cursorID != nil {
		query += fmt.Sprintf(" AND (added_at, id) < ($%d, $%d)", argPos, argPos+1)
		args = append(args, *cursorTime, *cursorID)
		argPos += 2
	}

	query += fmt.Sprintf(" ORDER BY added_at DESC, id DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list user contents with cursor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var userContents []*models.UserContent
	for rows.Next() {
		userContent := &models.UserContent{}
		err := rows.Scan(
			&userContent.ID,
			&userContent.UserID,
			&userContent.ContentID,
			&userContent.Status,
			&userContent.ScrollPosition,
			&userContent.IsFavorite,
			&userContent.AddedAt,
			&userContent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user-content: %w", err)
		}
		userContents = append(userContents, userContent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user-content rows: %w", err)
	}

	return userContents, nil
}

// Update updates an existing user-content record
func (r *userContentRepository) Update(ctx context.Context, userContent *models.UserContent) error {
	query := `
		UPDATE user_contents
		SET
			status = $2,
			scroll_position = $3,
			is_favorite = $4,
			updated_at = $5
		WHERE id = $1
		RETURNING updated_at
	`

	// Update timestamp
	userContent.UpdatedAt = time.Now()

	err := r.db.QueryRowContext(
		ctx, query,
		userContent.ID,
		userContent.Status,
		userContent.ScrollPosition,
		userContent.IsFavorite,
		userContent.UpdatedAt,
	).Scan(&userContent.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("user-content not found for update: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to update user-content: %w", err)
	}

	return nil
}

// UpdateWithTx updates an existing user-content record within a transaction
func (r *userContentRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, userContent *models.UserContent) error {
	query := `
		UPDATE user_contents
		SET
			status = $2,
			scroll_position = $3,
			is_favorite = $4,
			updated_at = $5
		WHERE id = $1
		RETURNING updated_at
	`

	// Update timestamp
	userContent.UpdatedAt = time.Now()

	err := tx.QueryRowContext(
		ctx, query,
		userContent.ID,
		userContent.Status,
		userContent.ScrollPosition,
		userContent.IsFavorite,
		userContent.UpdatedAt,
	).Scan(&userContent.UpdatedAt)

	if err == sql.ErrNoRows {
		return fmt.Errorf("user-content not found for update: %w", err)
	}
	if err != nil {
		return fmt.Errorf("failed to update user-content with transaction: %w", err)
	}

	return nil
}

// UpdateMetadata updates only the metadata fields (status, scroll_position, is_favorite)
func (r *userContentRepository) UpdateMetadata(ctx context.Context, id uuid.UUID, status *string, scrollPosition *int, isFavorite *bool) error {
	// Build dynamic update query based on which fields are provided
	query := "UPDATE user_contents SET updated_at = $1"
	args := []interface{}{time.Now()}
	argPos := 2

	if status != nil {
		query += fmt.Sprintf(", status = $%d", argPos)
		args = append(args, *status)
		argPos++
	}

	if scrollPosition != nil {
		query += fmt.Sprintf(", scroll_position = $%d", argPos)
		args = append(args, *scrollPosition)
		argPos++
	}

	if isFavorite != nil {
		query += fmt.Sprintf(", is_favorite = $%d", argPos)
		args = append(args, *isFavorite)
		argPos++
	}

	query += fmt.Sprintf(" WHERE id = $%d", argPos)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update user-content metadata: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user-content not found for metadata update")
	}

	return nil
}

// Delete deletes a user-content relationship
func (r *userContentRepository) Delete(ctx context.Context, userID, contentID uuid.UUID) error {
	query := `
		DELETE FROM user_contents
		WHERE user_id = $1 AND content_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, userID, contentID)
	if err != nil {
		return fmt.Errorf("failed to delete user-content: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user-content not found for deletion")
	}

	return nil
}

// DeleteWithTx deletes a user-content relationship within a transaction
func (r *userContentRepository) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID, contentID uuid.UUID) error {
	query := `
		DELETE FROM user_contents
		WHERE user_id = $1 AND content_id = $2
	`

	result, err := tx.ExecContext(ctx, query, userID, contentID)
	if err != nil {
		return fmt.Errorf("failed to delete user-content with transaction: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user-content not found for deletion")
	}

	return nil
}

// CountByUser counts the total number of content items for a user
func (r *userContentRepository) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `
		SELECT COUNT(*)
		FROM user_contents
		WHERE user_id = $1
	`

	var count int64
	err := r.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user contents: %w", err)
	}

	return count, nil
}

// Search searches user's content by title and author using full-text search
func (r *userContentRepository) Search(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]*models.UserContent, error) {
	sqlQuery := `
		SELECT uc.id, uc.user_id, uc.content_id, uc.status, uc.scroll_position, uc.is_favorite, uc.added_at, uc.updated_at
		FROM user_contents uc
		JOIN contents c ON uc.content_id = c.id
		WHERE uc.user_id = $1
		AND to_tsvector('english', coalesce(c.title, '') || ' ' || coalesce(c.author, '')) @@ plainto_tsquery('english', $2)
		ORDER BY uc.added_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.QueryContext(ctx, sqlQuery, userID, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search user contents: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var userContents []*models.UserContent
	for rows.Next() {
		userContent := &models.UserContent{}
		err := rows.Scan(
			&userContent.ID,
			&userContent.UserID,
			&userContent.ContentID,
			&userContent.Status,
			&userContent.ScrollPosition,
			&userContent.IsFavorite,
			&userContent.AddedAt,
			&userContent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user-content: %w", err)
		}
		userContents = append(userContents, userContent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user-content rows: %w", err)
	}

	return userContents, nil
}

// SearchWithCursor searches user's content using full-text search with keyset pagination.
func (r *userContentRepository) SearchWithCursor(ctx context.Context, userID uuid.UUID, query string, limit int, cursorTime *time.Time, cursorID *uuid.UUID) ([]*models.UserContent, error) {
	sqlQuery := `
		SELECT uc.id, uc.user_id, uc.content_id, uc.status, uc.scroll_position, uc.is_favorite, uc.added_at, uc.updated_at
		FROM user_contents uc
		JOIN contents c ON uc.content_id = c.id
		WHERE uc.user_id = $1
		AND to_tsvector('english', coalesce(c.title, '') || ' ' || coalesce(c.author, '')) @@ plainto_tsquery('english', $2)
	`

	args := []interface{}{userID, query}
	argPos := 3

	if cursorTime != nil && cursorID != nil {
		sqlQuery += fmt.Sprintf(" AND (uc.added_at, uc.id) < ($%d, $%d)", argPos, argPos+1)
		args = append(args, *cursorTime, *cursorID)
		argPos += 2
	}

	sqlQuery += fmt.Sprintf(" ORDER BY uc.added_at DESC, uc.id DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search user contents with cursor: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var userContents []*models.UserContent
	for rows.Next() {
		userContent := &models.UserContent{}
		err := rows.Scan(
			&userContent.ID,
			&userContent.UserID,
			&userContent.ContentID,
			&userContent.Status,
			&userContent.ScrollPosition,
			&userContent.IsFavorite,
			&userContent.AddedAt,
			&userContent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user-content in search: %w", err)
		}
		userContents = append(userContents, userContent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating user-content search rows: %w", err)
	}

	return userContents, nil
}

// BulkCreate creates multiple user-content relationships in a transaction
func (r *userContentRepository) BulkCreate(ctx context.Context, userContents []*models.UserContent) error {
	if len(userContents) == 0 {
		return nil
	}

	// Start transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Prepare the insert statement
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO user_contents (
			id, user_id, content_id, status, scroll_position, is_favorite, added_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (user_id, content_id) DO NOTHING
		RETURNING id, added_at, updated_at
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	now := time.Now()

	// Insert each user-content relationship
	for _, uc := range userContents {
		// Generate UUID if not provided
		if uc.ID == uuid.Nil {
			uc.ID = uuid.New()
		}

		// Set timestamps
		uc.AddedAt = now
		uc.UpdatedAt = now

		// Set default status if not provided
		if uc.Status == "" {
			uc.Status = models.StatusUnread
		}

		err := stmt.QueryRowContext(
			ctx,
			uc.ID,
			uc.UserID,
			uc.ContentID,
			uc.Status,
			uc.ScrollPosition,
			uc.IsFavorite,
			uc.AddedAt,
			uc.UpdatedAt,
		).Scan(&uc.ID, &uc.AddedAt, &uc.UpdatedAt)

		// ON CONFLICT DO NOTHING returns no rows, which is fine
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("failed to insert user-content in bulk: %w", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit bulk create transaction: %w", err)
	}

	return nil
}
