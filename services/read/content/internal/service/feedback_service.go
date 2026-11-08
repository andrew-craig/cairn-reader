package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/content/internal/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ErrUserContentNotFound is returned when the user does not have the content.
var ErrUserContentNotFound = errors.New("user content not found")

// FeedbackService records users' extraction reports together with the
// evidence needed to reproduce them.
type FeedbackService struct {
	userContentRepo repository.UserContentRepository
	contentRepo     repository.ContentRepository
	feedbackRepo    repository.ContentFeedbackRepository
}

// NewFeedbackService creates a new FeedbackService
func NewFeedbackService(
	userContentRepo repository.UserContentRepository,
	contentRepo repository.ContentRepository,
	feedbackRepo repository.ContentFeedbackRepository,
) *FeedbackService {
	return &FeedbackService{userContentRepo: userContentRepo, contentRepo: contentRepo, feedbackRepo: feedbackRepo}
}

// Submit saves the user's report for a content they have, snapshotting the
// extraction output and, while it is still retained, the raw input.
func (s *FeedbackService) Submit(ctx context.Context, userID, contentID uuid.UUID, categories []string, comment *string) (*models.ContentFeedback, error) {
	userContent, err := s.userContentRepo.GetByUserAndContent(ctx, userID, contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user content: %w", err)
	}
	if userContent == nil {
		return nil, ErrUserContentNotFound
	}

	content, err := s.contentRepo.GetByID(ctx, contentID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch content: %w", err)
	}
	source, err := s.contentRepo.GetSource(ctx, contentID)
	if err != nil {
		return nil, err
	}

	snapshot := &models.FeedbackSnapshot{
		OriginalURL: content.OriginalURL,
		Title:       content.Title,
		Author:      content.Author,
		Description: content.Description,
		CleanedHTML: content.CleanedHTML,
	}
	if len(content.ImageURLs) > 0 {
		snapshot.ImageURL = &content.ImageURLs[0]
	}
	if source != nil {
		snapshot.RawHTML = &source.RawHTML
		snapshot.ReadabilityFailed = &source.ReadabilityFailed
	}

	return s.feedbackRepo.Upsert(ctx, &models.ContentFeedback{
		UserID:     userID,
		ContentID:  contentID,
		Categories: pq.StringArray(categories),
		Comment:    comment,
		Snapshot:   snapshot,
	})
}
