package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
)

// ContentFeedbackRepository stores users' extraction reports. Reads never
// return the evidence snapshot.
type ContentFeedbackRepository interface {
	// Upsert creates the user's report for a content, or replaces the existing
	// one: categories, comment and snapshot are refreshed and the status goes
	// back to open. Returns the saved report without its snapshot.
	Upsert(ctx context.Context, feedback *models.ContentFeedback) (*models.ContentFeedback, error)

	// GetByUserAndContent returns the user's report for a content, or nil when
	// there is none.
	GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.ContentFeedback, error)
}

type contentFeedbackRepository struct {
	db *sql.DB
}

// NewContentFeedbackRepository creates a new ContentFeedbackRepository
func NewContentFeedbackRepository(db *sql.DB) ContentFeedbackRepository {
	return &contentFeedbackRepository{db: db}
}

const feedbackReportColumns = `id, user_id, content_id, categories, comment, status, created_at, updated_at`

func scanFeedbackReport(row *sql.Row) (*models.ContentFeedback, error) {
	f := &models.ContentFeedback{}
	err := row.Scan(&f.ID, &f.UserID, &f.ContentID, &f.Categories, &f.Comment, &f.Status, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (r *contentFeedbackRepository) Upsert(ctx context.Context, feedback *models.ContentFeedback) (*models.ContentFeedback, error) {
	s := feedback.Snapshot
	if s == nil {
		return nil, errors.New("feedback snapshot is required")
	}

	saved, err := scanFeedbackReport(r.db.QueryRowContext(ctx, `
		INSERT INTO content_feedback (
			user_id, content_id, categories, comment,
			original_url, title, author, description, image_url,
			cleaned_html, raw_html, readability_failed
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (user_id, content_id) DO UPDATE SET
			categories = EXCLUDED.categories,
			comment = EXCLUDED.comment,
			original_url = EXCLUDED.original_url,
			title = EXCLUDED.title,
			author = EXCLUDED.author,
			description = EXCLUDED.description,
			image_url = EXCLUDED.image_url,
			cleaned_html = EXCLUDED.cleaned_html,
			raw_html = EXCLUDED.raw_html,
			readability_failed = EXCLUDED.readability_failed,
			status = 'open',
			updated_at = NOW()
		RETURNING `+feedbackReportColumns,
		feedback.UserID, feedback.ContentID, feedback.Categories, feedback.Comment,
		s.OriginalURL, s.Title, s.Author, s.Description, s.ImageURL,
		s.CleanedHTML, s.RawHTML, s.ReadabilityFailed,
	))
	if err != nil {
		return nil, fmt.Errorf("failed to upsert content feedback: %w", err)
	}
	return saved, nil
}

func (r *contentFeedbackRepository) GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.ContentFeedback, error) {
	f, err := scanFeedbackReport(r.db.QueryRowContext(ctx, `
		SELECT `+feedbackReportColumns+`
		FROM content_feedback
		WHERE user_id = $1 AND content_id = $2
	`, userID, contentID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get content feedback: %w", err)
	}
	return f, nil
}
