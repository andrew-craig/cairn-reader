package dto

import (
	"errors"
	"time"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/google/uuid"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
)

// maxFeedbackCommentLength is the longest comment (in characters) a report may carry.
const maxFeedbackCommentLength = 2000

// ExtractionFeedbackRequest represents the request body for reporting a bad extraction
type ExtractionFeedbackRequest struct {
	Categories []string `json:"categories"`
	Comment    *string  `json:"comment,omitempty"`
}

// Validate validates the ExtractionFeedbackRequest
func (f ExtractionFeedbackRequest) Validate() error {
	allowed := make([]interface{}, len(models.FeedbackCategories))
	for i, c := range models.FeedbackCategories {
		allowed[i] = c
	}

	return validation.ValidateStruct(&f,
		validation.Field(&f.Categories,
			validation.Required.Error("At least one category is required"),
			validation.Each(validation.Required.Error("Unknown category"), validation.In(allowed...).Error("Unknown category")),
			validation.By(uniqueCategories),
		),
		validation.Field(&f.Comment,
			validation.When(f.Comment != nil,
				validation.RuneLength(0, maxFeedbackCommentLength).Error("Comment must be at most 2000 characters"),
			),
		),
	)
}

func uniqueCategories(value interface{}) error {
	categories, _ := value.([]string)
	seen := make(map[string]struct{}, len(categories))
	for _, c := range categories {
		if _, dup := seen[c]; dup {
			return errors.New("categories must be unique")
		}
		seen[c] = struct{}{}
	}
	return nil
}

// ExtractionFeedbackResponse is a user's own report. It never carries the evidence snapshot.
type ExtractionFeedbackResponse struct {
	ContentID  uuid.UUID `json:"content_id"`
	Categories []string  `json:"categories"`
	Comment    *string   `json:"comment"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// NewExtractionFeedbackResponse converts a stored report to its API response
func NewExtractionFeedbackResponse(f *models.ContentFeedback) *ExtractionFeedbackResponse {
	return &ExtractionFeedbackResponse{
		ContentID:  f.ContentID,
		Categories: f.Categories,
		Comment:    f.Comment,
		Status:     f.Status,
		CreatedAt:  f.CreatedAt,
		UpdatedAt:  f.UpdatedAt,
	}
}
