package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/models"
	"github.com/andrew-craig/cairn/services/read/content/internal/processor"
	"github.com/andrew-craig/cairn/services/read/content/internal/repository"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ContentService defines the interface for content business logic
type ContentService interface {
	// CreateFromURL fetches and processes content from a URL, then stores it
	CreateFromURL(ctx context.Context, url string, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error)

	// CreateFromHTML processes raw HTML and stores it
	CreateFromHTML(ctx context.Context, url string, html string, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error)

	// GetByID retrieves content by ID
	GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error)

	// UpdateContent updates existing content (preserves user-content relationships)
	UpdateContent(ctx context.Context, id uuid.UUID, url string, html string, publishedAt *time.Time) (*models.Content, error)

	// CheckDuplicate checks if content already exists by content hash and feed ID
	CheckDuplicate(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error)

	// ListContents retrieves contents with pagination
	ListContents(ctx context.Context, limit, offset int) ([]*models.Content, error)

	// BulkCreateFromHTML processes multiple raw HTML contents and stores them
	BulkCreateFromHTML(ctx context.Context, items []BulkContentItem) ([]*models.Content, []BulkCreateError, error)

	// CheckDuplicates checks if multiple contents already exist by content hash and feed ID
	CheckDuplicates(ctx context.Context, items []DuplicateCheckItem) (map[string]*models.Content, error)
}

// BulkContentItem represents a single content item in a bulk creation request
type BulkContentItem struct {
	URL          string
	HTML         string
	SourceType   string
	SourceFeedID *uuid.UUID
	PublishedAt  *time.Time
}

// BulkCreateError represents an error for a specific bulk item
type BulkCreateError struct {
	Index   int
	URL     string
	Error   error
	Message string
}

// DuplicateCheckItem represents an item to check for duplicates
type DuplicateCheckItem struct {
	ContentHash  string
	SourceFeedID uuid.UUID
}

// contentService implements ContentService
type contentService struct {
	contentRepo repository.ContentRepository
	processor   *processor.ContentProcessor
	db          *sql.DB
}

// NewContentService creates a new ContentService
func NewContentService(contentRepo repository.ContentRepository, db *sql.DB) ContentService {
	return &contentService{
		contentRepo: contentRepo,
		processor:   processor.NewContentProcessor(),
		db:          db,
	}
}

// CreateFromURL fetches and processes content from a URL, then stores it
func (s *contentService) CreateFromURL(ctx context.Context, url string, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error) {
	// Process the URL
	processed, err := s.processor.ProcessURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to process URL: %w", err)
	}

	// Check for duplicates if this is from an RSS feed
	if sourceType == models.SourceTypeRSS && sourceFeedID != nil {
		existing, err := s.contentRepo.GetByContentHashAndFeedID(ctx, processed.ContentHash, *sourceFeedID)
		if err != nil {
			return nil, fmt.Errorf("failed to check for duplicate: %w", err)
		}
		if existing != nil {
			// Content already exists, return it
			return existing, nil
		}
	}

	// Create the content model
	content := &models.Content{
		ContentHash:  processed.ContentHash,
		CleanedHTML:  processed.CleanedHTML,
		OriginalURL:  url,
		CanonicalURL: &processed.CanonicalURL,
		Title:        processed.Title,
		SourceType:   sourceType,
		SourceFeedID: sourceFeedID,
		PublishedAt:  publishedAt,
	}

	// Set optional fields if they're not empty
	if processed.Author != "" {
		content.Author = &processed.Author
	}
	if processed.Description != "" {
		content.Description = &processed.Description
	}
	if processed.ImageURL != "" {
		content.ImageURLs = pq.StringArray{processed.ImageURL}
	}

	// Store in database
	err = s.contentRepo.Create(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create content: %w", err)
	}

	return content, nil
}

// CreateFromHTML processes raw HTML and stores it
func (s *contentService) CreateFromHTML(ctx context.Context, url string, html string, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error) {
	// Process the HTML
	processed, err := s.processor.ProcessHTML(url, html)
	if err != nil {
		return nil, fmt.Errorf("failed to process HTML: %w", err)
	}

	// Check for duplicates if this is from an RSS feed
	if sourceType == models.SourceTypeRSS && sourceFeedID != nil {
		existing, err := s.contentRepo.GetByContentHashAndFeedID(ctx, processed.ContentHash, *sourceFeedID)
		if err != nil {
			return nil, fmt.Errorf("failed to check for duplicate: %w", err)
		}
		if existing != nil {
			// Content already exists, return it
			return existing, nil
		}
	}

	// Create the content model
	content := &models.Content{
		ContentHash:  processed.ContentHash,
		CleanedHTML:  processed.CleanedHTML,
		OriginalURL:  url,
		CanonicalURL: &processed.CanonicalURL,
		Title:        processed.Title,
		SourceType:   sourceType,
		SourceFeedID: sourceFeedID,
		PublishedAt:  publishedAt,
	}

	// Set optional fields if they're not empty
	if processed.Author != "" {
		content.Author = &processed.Author
	}
	if processed.Description != "" {
		content.Description = &processed.Description
	}
	if processed.ImageURL != "" {
		content.ImageURLs = pq.StringArray{processed.ImageURL}
	}

	// Store in database
	err = s.contentRepo.Create(ctx, content)
	if err != nil {
		return nil, fmt.Errorf("failed to create content: %w", err)
	}

	return content, nil
}

// GetByID retrieves content by ID
func (s *contentService) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	return s.contentRepo.GetByID(ctx, id)
}

// UpdateContent updates existing content (preserves user-content relationships)
func (s *contentService) UpdateContent(ctx context.Context, id uuid.UUID, url string, html string, publishedAt *time.Time) (*models.Content, error) {
	// Get existing content
	existing, err := s.contentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing content: %w", err)
	}

	// Process the new HTML
	processed, err := s.processor.ProcessHTML(url, html)
	if err != nil {
		return nil, fmt.Errorf("failed to process HTML: %w", err)
	}

	// Update the content fields
	existing.ContentHash = processed.ContentHash
	existing.CleanedHTML = processed.CleanedHTML
	existing.OriginalURL = url
	existing.CanonicalURL = &processed.CanonicalURL
	existing.Title = processed.Title
	existing.PublishedAt = publishedAt

	// Update optional fields
	if processed.Author != "" {
		existing.Author = &processed.Author
	} else {
		existing.Author = nil
	}
	if processed.Description != "" {
		existing.Description = &processed.Description
	} else {
		existing.Description = nil
	}
	if processed.ImageURL != "" {
		existing.ImageURLs = pq.StringArray{processed.ImageURL}
	} else {
		existing.ImageURLs = nil
	}

	// Update in database
	err = s.contentRepo.Update(ctx, existing)
	if err != nil {
		return nil, fmt.Errorf("failed to update content: %w", err)
	}

	return existing, nil
}

// CheckDuplicate checks if content already exists by content hash and feed ID
func (s *contentService) CheckDuplicate(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	return s.contentRepo.GetByContentHashAndFeedID(ctx, contentHash, feedID)
}

// ListContents retrieves contents with pagination
func (s *contentService) ListContents(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	return s.contentRepo.List(ctx, limit, offset)
}

// BulkCreateFromHTML processes multiple raw HTML contents and stores them
func (s *contentService) BulkCreateFromHTML(ctx context.Context, items []BulkContentItem) ([]*models.Content, []BulkCreateError, error) {
	var contents []*models.Content
	var errors []BulkCreateError

	// Process each item
	for i, item := range items {
		// Process the HTML
		processed, err := s.processor.ProcessHTML(item.URL, item.HTML)
		if err != nil {
			errors = append(errors, BulkCreateError{
				Index:   i,
				URL:     item.URL,
				Error:   err,
				Message: fmt.Sprintf("Failed to process HTML: %v", err),
			})
			continue
		}

		// Check for duplicates if this is from an RSS feed
		if item.SourceType == models.SourceTypeRSS && item.SourceFeedID != nil {
			existing, err := s.contentRepo.GetByContentHashAndFeedID(ctx, processed.ContentHash, *item.SourceFeedID)
			if err != nil {
				errors = append(errors, BulkCreateError{
					Index:   i,
					URL:     item.URL,
					Error:   err,
					Message: fmt.Sprintf("Failed to check for duplicate: %v", err),
				})
				continue
			}
			if existing != nil {
				// Content already exists, add to results
				contents = append(contents, existing)
				continue
			}
		}

		// Create the content model
		content := &models.Content{
			ContentHash:  processed.ContentHash,
			CleanedHTML:  processed.CleanedHTML,
			OriginalURL:  item.URL,
			CanonicalURL: &processed.CanonicalURL,
			Title:        processed.Title,
			SourceType:   item.SourceType,
			SourceFeedID: item.SourceFeedID,
			PublishedAt:  item.PublishedAt,
		}

		// Set optional fields if they're not empty
		if processed.Author != "" {
			content.Author = &processed.Author
		}
		if processed.Description != "" {
			content.Description = &processed.Description
		}
		if processed.ImageURL != "" {
			content.ImageURLs = pq.StringArray{processed.ImageURL}
		}

		contents = append(contents, content)
	}

	// Bulk create all new contents
	if len(contents) > 0 {
		err := s.contentRepo.BulkCreate(ctx, contents)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to bulk create contents: %w", err)
		}
	}

	return contents, errors, nil
}

// CheckDuplicates checks if multiple contents already exist by content hash and feed ID
func (s *contentService) CheckDuplicates(ctx context.Context, items []DuplicateCheckItem) (map[string]*models.Content, error) {
	if len(items) == 0 {
		return make(map[string]*models.Content), nil
	}

	// Group items by feed ID
	feedGroups := make(map[uuid.UUID][]string)
	for _, item := range items {
		feedGroups[item.SourceFeedID] = append(feedGroups[item.SourceFeedID], item.ContentHash)
	}

	// Check each feed group
	result := make(map[string]*models.Content)
	for feedID, hashes := range feedGroups {
		contents, err := s.contentRepo.GetByContentHashesAndFeedID(ctx, hashes, feedID)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicates for feed %s: %w", feedID, err)
		}

		// Merge results
		for hash, content := range contents {
			result[hash] = content
		}
	}

	return result, nil
}
