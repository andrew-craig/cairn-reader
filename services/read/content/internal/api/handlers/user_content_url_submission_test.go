package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/pkg/auth"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Mock implementations for testing
type mockContentService struct{}

func (m *mockContentService) CreateFromURL(ctx context.Context, url string, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error) {
	return &models.Content{
		ID:          uuid.New(),
		Title:       "Test Article",
		OriginalURL: url,
		CleanedHTML: "<p>Test content</p>",
	}, nil
}

func (m *mockContentService) CreateFromHTML(ctx context.Context, url string, html string, sourceType string, sourceFeedID *uuid.UUID, publishedAt *time.Time) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) UpdateContent(ctx context.Context, id uuid.UUID, url string, html string, publishedAt *time.Time) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) CheckDuplicate(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) ListContents(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) BulkCreateFromHTML(ctx context.Context, items []service.BulkContentItem) ([]*models.Content, []service.BulkCreateError, error) {
	return nil, nil, nil
}

func (m *mockContentService) CheckDuplicates(ctx context.Context, items []service.DuplicateCheckItem) (map[string]*models.Content, error) {
	return nil, nil
}

type mockURLDetector struct {
	detectionType service.URLType
}

func (m *mockURLDetector) DetectURL(ctx context.Context, url string) (*service.URLDetectionResult, error) {
	title := "Test Title"
	return &service.URLDetectionResult{
		URL:   url,
		Type:  m.detectionType,
		Title: &title,
	}, nil
}

type mockIngestRSSClient struct {
	shouldFail bool
}

func (m *mockIngestRSSClient) SubscribeUserToFeed(ctx context.Context, userID, feedURL string) (*service.FeedSubscriptionResponse, error) {
	if m.shouldFail {
		return nil, fmt.Errorf("already subscribed to this feed")
	}
	return &service.FeedSubscriptionResponse{
		Subscription: struct {
			ID           string    `json:"id"`
			UserID       string    `json:"user_id"`
			FeedID       string    `json:"feed_id"`
			SubscribedAt time.Time `json:"subscribed_at"`
		}{
			ID:           uuid.NewString(),
			UserID:       userID,
			FeedID:       uuid.NewString(),
			SubscribedAt: time.Now(),
		},
		Feed: struct {
			ID          string    `json:"id"`
			FeedURL     string    `json:"feed_url"`
			Title       string    `json:"title"`
			Description string    `json:"description"`
			SiteURL     string    `json:"site_url"`
			PollingTier string    `json:"polling_tier"`
			Status      string    `json:"status"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
		}{
			ID:        uuid.NewString(),
			FeedURL:   feedURL,
			Title:     "Test Feed",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		IsNewFeed: true,
	}, nil
}

type mockUserContentRepo struct{}

func (m *mockUserContentRepo) Create(ctx context.Context, uc *models.UserContent) error {
	uc.ID = uuid.New()
	return nil
}

func (m *mockUserContentRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, uc *models.UserContent) error {
	return nil
}

func (m *mockUserContentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) GetByUserAndContent(ctx context.Context, userID, contentID uuid.UUID) (*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) ListByUserWithFilter(ctx context.Context, userID uuid.UUID, status *string, isFavorite *bool, limit, offset int) ([]*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) CountByUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockUserContentRepo) Update(ctx context.Context, uc *models.UserContent) error {
	return nil
}

func (m *mockUserContentRepo) UpdateWithTx(ctx context.Context, tx *sql.Tx, uc *models.UserContent) error {
	return nil
}

func (m *mockUserContentRepo) UpdateMetadata(ctx context.Context, id uuid.UUID, status *string, scrollPosition *int, isFavorite *bool) error {
	return nil
}

func (m *mockUserContentRepo) Delete(ctx context.Context, userID, contentID uuid.UUID) error {
	return nil
}

func (m *mockUserContentRepo) DeleteWithTx(ctx context.Context, tx *sql.Tx, userID, contentID uuid.UUID) error {
	return nil
}

func (m *mockUserContentRepo) Search(ctx context.Context, userID uuid.UUID, query string, limit, offset int) ([]*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) BulkCreate(ctx context.Context, userContents []*models.UserContent) error {
	return nil
}

type mockContentRepo struct{}

func (m *mockContentRepo) Create(ctx context.Context, content *models.Content) error {
	return nil
}

func (m *mockContentRepo) CreateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	return nil
}

func (m *mockContentRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	return &models.Content{
		ID:          id,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		CleanedHTML: "<p>Test content</p>",
	}, nil
}

func (m *mockContentRepo) GetByContentHashAndFeedID(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentRepo) Update(ctx context.Context, content *models.Content) error {
	return nil
}

func (m *mockContentRepo) UpdateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	return nil
}

func (m *mockContentRepo) DeleteOrphaned(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockContentRepo) List(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	return nil, nil
}

func (m *mockContentRepo) GetByContentHashesAndFeedID(ctx context.Context, contentHashes []string, feedID uuid.UUID) (map[string]*models.Content, error) {
	return nil, nil
}

func (m *mockContentRepo) BulkCreate(ctx context.Context, contents []*models.Content) error {
	return nil
}

// TestURLBasedSubmission_Page tests URL submission for a regular web page
func TestURLBasedSubmission_Page(t *testing.T) {
	// Setup
	userID := uuid.New()
	url := "https://example.com/article"

	// The handler expects *service.IngestRSSClient (concrete type), so we pass nil
	// since this test exercises the page path, not the feed path.
	handler := NewUserContentHandler(
		&mockUserContentRepo{},
		&mockContentRepo{},
		&mockContentService{},
		&mockURLDetector{detectionType: service.URLTypePage},
		nil, // ingestRSSClient not needed for page submissions
	)

	// Create request
	reqBody := dto.AddContentToUserRequest{
		URL: &url,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/user/"+userID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add user_id to chi route context and authenticated user ID to auth context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.UserIDContextKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	handler.AddContentToUser(w, req)

	// Verify
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Response is wrapped by api.WriteSuccess: {"data": {...}, "meta": {...}}
	var wrapped struct {
		Data dto.AddPageResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&wrapped); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if wrapped.Data.Type != "page" {
		t.Errorf("Expected type 'page', got '%s'", wrapped.Data.Type)
	}

	if wrapped.Data.Content == nil {
		t.Error("Expected content in response, got nil")
	}
}

// TestURLBasedSubmission_RequiresURLOrContentID tests validation
func TestURLBasedSubmission_RequiresURLOrContentID(t *testing.T) {
	// Setup
	userID := uuid.New()

	handler := NewUserContentHandler(
		&mockUserContentRepo{},
		&mockContentRepo{},
		&mockContentService{},
		&mockURLDetector{detectionType: service.URLTypePage},
		nil,
	)

	// Create request with neither URL nor ContentID
	reqBody := dto.AddContentToUserRequest{}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/user/"+userID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, auth.UserIDContextKey, userID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	// Execute
	handler.AddContentToUser(w, req)

	// Verify - should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}
