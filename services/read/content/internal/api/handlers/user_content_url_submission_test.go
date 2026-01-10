package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cairn-app/cairn-reader/services/read/content/internal/api/dto"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/repository"
	"github.com/cairn-app/cairn-reader/services/read/content/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Mock implementations for testing
type mockContentService struct{}

func (m *mockContentService) CreateFromURL(ctx interface{}, url string, sourceType string, sourceFeedID *uuid.UUID, publishedAt interface{}) (*models.Content, error) {
	// Return a mock content
	return &models.Content{
		ID:          uuid.New(),
		Title:       "Test Article",
		OriginalURL: url,
		CleanedHTML: "<p>Test content</p>",
	}, nil
}

func (m *mockContentService) CreateFromHTML(ctx interface{}, url string, html string, sourceType string, sourceFeedID *uuid.UUID, publishedAt interface{}) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) GetByID(ctx interface{}, id uuid.UUID) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) UpdateContent(ctx interface{}, id uuid.UUID, url string, html string, publishedAt interface{}) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) CheckDuplicate(ctx interface{}, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) ListContents(ctx interface{}, limit, offset int) ([]*models.Content, error) {
	return nil, nil
}

func (m *mockContentService) BulkCreateFromHTML(ctx interface{}, items []service.BulkContentItem) ([]*models.Content, []service.BulkCreateError, error) {
	return nil, nil, nil
}

func (m *mockContentService) CheckDuplicates(ctx interface{}, items []service.DuplicateCheckItem) (map[string]*models.Content, error) {
	return nil, nil
}

type mockURLDetector struct {
	detectionType service.URLType
}

func (m *mockURLDetector) DetectURL(ctx interface{}, url string) (*service.URLDetectionResult, error) {
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

func (m *mockIngestRSSClient) SubscribeUserToFeed(ctx interface{}, userID, feedURL string) (*service.FeedSubscriptionResponse, error) {
	if m.shouldFail {
		return nil, &service.IngestRSSError{
			Error:   "already_subscribed",
			Message: "Already subscribed to this feed",
		}
	}
	return &service.FeedSubscriptionResponse{
		Subscription: struct {
			ID           string      `json:"id"`
			UserID       string      `json:"user_id"`
			FeedID       string      `json:"feed_id"`
			SubscribedAt interface{} `json:"subscribed_at"`
		}{
			ID:     uuid.NewString(),
			UserID: userID,
			FeedID: uuid.NewString(),
		},
		Feed: struct {
			ID          string      `json:"id"`
			FeedURL     string      `json:"feed_url"`
			Title       string      `json:"title"`
			Description string      `json:"description"`
			SiteURL     string      `json:"site_url"`
			PollingTier string      `json:"polling_tier"`
			Status      string      `json:"status"`
			CreatedAt   interface{} `json:"created_at"`
			UpdatedAt   interface{} `json:"updated_at"`
		}{
			ID:      uuid.NewString(),
			FeedURL: feedURL,
			Title:   "Test Feed",
		},
		IsNewFeed: true,
	}, nil
}

type mockUserContentRepo struct{}

func (m *mockUserContentRepo) Create(ctx interface{}, uc *models.UserContent) error {
	uc.ID = uuid.New()
	return nil
}

func (m *mockUserContentRepo) GetByID(ctx interface{}, id uuid.UUID) (*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) GetByUserAndContent(ctx interface{}, userID, contentID uuid.UUID) (*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) ListByUserWithFilter(ctx interface{}, userID uuid.UUID, status *string, isFavorite *bool, limit, offset int) ([]*models.UserContent, error) {
	return nil, nil
}

func (m *mockUserContentRepo) CountByUser(ctx interface{}, userID uuid.UUID) (int64, error) {
	return 0, nil
}

func (m *mockUserContentRepo) UpdateMetadata(ctx interface{}, id uuid.UUID, status *string, scrollPosition *int, isFavorite *bool) error {
	return nil
}

func (m *mockUserContentRepo) Delete(ctx interface{}, userID, contentID uuid.UUID) error {
	return nil
}

func (m *mockUserContentRepo) Search(ctx interface{}, userID uuid.UUID, query string, limit, offset int) ([]*models.UserContent, error) {
	return nil, nil
}

type mockContentRepo struct{}

func (m *mockContentRepo) Create(ctx interface{}, content *models.Content) error {
	return nil
}

func (m *mockContentRepo) GetByID(ctx interface{}, id uuid.UUID) (*models.Content, error) {
	return &models.Content{
		ID:          id,
		Title:       "Test Article",
		OriginalURL: "https://example.com",
		CleanedHTML: "<p>Test content</p>",
	}, nil
}

func (m *mockContentRepo) GetByContentHashAndFeedID(ctx interface{}, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	return nil, nil
}

func (m *mockContentRepo) Update(ctx interface{}, content *models.Content) error {
	return nil
}

func (m *mockContentRepo) List(ctx interface{}, limit, offset int) ([]*models.Content, error) {
	return nil, nil
}

func (m *mockContentRepo) BulkCreate(ctx interface{}, contents []*models.Content) error {
	return nil
}

func (m *mockContentRepo) CheckDuplicatesByHashAndFeed(ctx interface{}, items []repository.DuplicateCheckItem) (map[string]*models.Content, error) {
	return nil, nil
}

// TestURLBasedSubmission_Page tests URL submission for a regular web page
func TestURLBasedSubmission_Page(t *testing.T) {
	// Setup
	userID := uuid.New()
	url := "https://example.com/article"

	handler := NewUserContentHandler(
		&mockUserContentRepo{},
		&mockContentRepo{},
		&mockContentService{},
		&mockURLDetector{detectionType: service.URLTypePage},
		&mockIngestRSSClient{},
	)

	// Create request
	reqBody := dto.AddContentToUserRequest{
		URL: &url,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/user/"+userID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	// Add user_id to request context
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(chi.NewRouter().Context())

	w := httptest.NewRecorder()

	// Execute
	handler.AddContentToUser(w, req)

	// Verify
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", w.Code)
	}

	var response dto.AddPageResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Type != "page" {
		t.Errorf("Expected type 'page', got '%s'", response.Type)
	}

	if response.Content == nil {
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
		&mockIngestRSSClient{},
	)

	// Create request with neither URL nor ContentID
	reqBody := dto.AddContentToUserRequest{}
	bodyBytes, _ := json.Marshal(reqBody)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/content/user/"+userID.String(), bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", userID.String())
	req = req.WithContext(chi.NewRouter().Context())

	w := httptest.NewRecorder()

	// Execute
	handler.AddContentToUser(w, req)

	// Verify - should return 400 Bad Request
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
