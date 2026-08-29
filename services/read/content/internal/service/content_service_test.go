package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/content/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockContentRepository is a mock implementation of repository.ContentRepository
type MockContentRepository struct {
	mock.Mock
}

func (m *MockContentRepository) Create(ctx context.Context, content *models.Content) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockContentRepository) CreateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	args := m.Called(ctx, tx, content)
	return args.Error(0)
}

func (m *MockContentRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Content, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentRepository) GetByContentHashAndFeedID(ctx context.Context, contentHash string, feedID uuid.UUID) (*models.Content, error) {
	args := m.Called(ctx, contentHash, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentRepository) GetByContentHashAndURL(ctx context.Context, contentHash string, originalURL string) (*models.Content, error) {
	args := m.Called(ctx, contentHash, originalURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Content), args.Error(1)
}

func (m *MockContentRepository) GetByContentHashesAndFeedID(ctx context.Context, hashes []string, feedID uuid.UUID) (map[string]*models.Content, error) {
	args := m.Called(ctx, hashes, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.Content), args.Error(1)
}

func (m *MockContentRepository) Update(ctx context.Context, content *models.Content) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockContentRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	args := m.Called(ctx, tx, content)
	return args.Error(0)
}

func (m *MockContentRepository) List(ctx context.Context, limit, offset int) ([]*models.Content, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Content), args.Error(1)
}

func (m *MockContentRepository) BulkCreate(ctx context.Context, contents []*models.Content) error {
	args := m.Called(ctx, contents)
	return args.Error(0)
}

func (m *MockContentRepository) GetByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]*models.Content, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[uuid.UUID]*models.Content), args.Error(1)
}

func (m *MockContentRepository) DeleteOrphaned(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Get(0).(int64), args.Error(1)
}

// TestCreateFromHTML_Success tests successful content creation from HTML
func TestCreateFromHTML_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	url := "https://example.com/article"
	html := "<html><head><title>Test Article</title></head><body><p>Test content</p></body></html>"
	sourceType := "web"

	// Mock repository to return nil for duplicate check (no duplicate)
	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, url).Return(nil, nil)

	// Mock repository to succeed on Create
	mockRepo.On("Create", ctx, mock.MatchedBy(func(c *models.Content) bool {
		return c.OriginalURL == url && c.Title == "Test Article"
	})).Return(nil)

	content, err := service.CreateFromHTML(ctx, url, html, sourceType, nil, nil)

	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.Equal(t, url, content.OriginalURL)
	assert.Equal(t, "Test Article", content.Title)
	assert.Equal(t, sourceType, content.SourceType)
	assert.NotEmpty(t, content.ContentHash)
	mockRepo.AssertExpectations(t)
}

// TestCreateFromHTML_WithFeedID tests content creation with RSS feed source
func TestCreateFromHTML_WithFeedID(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	url := "https://example.com/article"
	html := "<html><head><title>RSS Article</title></head><body><p>RSS content</p></body></html>"
	sourceType := models.SourceTypeRSS
	feedID := uuid.New()
	publishedAt := time.Now()

	// Mock repository to return nil for duplicate check
	mockRepo.On("GetByContentHashAndFeedID", ctx, mock.Anything, feedID).Return(nil, nil)

	// Mock repository to succeed on Create
	mockRepo.On("Create", ctx, mock.MatchedBy(func(c *models.Content) bool {
		return c.SourceFeedID != nil && *c.SourceFeedID == feedID
	})).Return(nil)

	content, err := service.CreateFromHTML(ctx, url, html, sourceType, &feedID, &publishedAt)

	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.NotNil(t, content.SourceFeedID)
	assert.Equal(t, feedID, *content.SourceFeedID)
	assert.NotNil(t, content.PublishedAt)
	mockRepo.AssertExpectations(t)
}

// TestCreateFromHTML_DuplicateDetection tests that duplicates are returned instead of created
func TestCreateFromHTML_DuplicateDetection(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	url := "https://example.com/article"
	html := "<html><head><title>Duplicate Article</title></head><body><p>Content</p></body></html>"
	sourceType := models.SourceTypeRSS
	feedID := uuid.New()

	existingContent := &models.Content{
		ID:           uuid.New(),
		ContentHash:  "existing-hash",
		OriginalURL:  url,
		Title:        "Duplicate Article",
		SourceFeedID: &feedID,
	}

	// Mock repository to return existing content
	mockRepo.On("GetByContentHashAndFeedID", ctx, mock.Anything, feedID).Return(existingContent, nil)

	content, err := service.CreateFromHTML(ctx, url, html, sourceType, &feedID, nil)

	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.Equal(t, existingContent.ID, content.ID)
	// Create should not be called since duplicate was found
	mockRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	mockRepo.AssertExpectations(t)
}

// TestCreateFromHTML_DuplicateCheckError tests error handling during duplicate check
func TestCreateFromHTML_DuplicateCheckError(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	url := "https://example.com/article"
	html := "<html><head><title>Article</title></head><body><p>Content</p></body></html>"
	sourceType := models.SourceTypeRSS
	feedID := uuid.New()

	// Mock repository to return error on duplicate check
	mockRepo.On("GetByContentHashAndFeedID", ctx, mock.Anything, feedID).Return(nil, errors.New("database error"))

	content, err := service.CreateFromHTML(ctx, url, html, sourceType, &feedID, nil)

	assert.Error(t, err)
	assert.Nil(t, content)
	assert.Contains(t, err.Error(), "failed to check for duplicate")
	mockRepo.AssertExpectations(t)
}

// TestCreateFromHTML_CreateError tests error handling during content creation
func TestCreateFromHTML_CreateError(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	url := "https://example.com/article"
	html := "<html><head><title>Article</title></head><body><p>Content</p></body></html>"
	sourceType := "web"

	// Mock repository to return nil for duplicate check (no duplicate)
	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, url).Return(nil, nil)

	// Mock repository to return error on Create
	mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("insert failed"))

	content, err := service.CreateFromHTML(ctx, url, html, sourceType, nil, nil)

	assert.Error(t, err)
	assert.Nil(t, content)
	assert.Contains(t, err.Error(), "failed to create content")
	mockRepo.AssertExpectations(t)
}

// TestUpdateContent_Success tests successful content update
func TestUpdateContent_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	contentID := uuid.New()
	url := "https://example.com/updated"
	html := "<html><head><title>Updated Title</title></head><body><p>Updated content</p></body></html>"
	publishedAt := time.Now()

	existingContent := &models.Content{
		ID:          contentID,
		ContentHash: "old-hash",
		Title:       "Old Title",
		OriginalURL: "https://example.com/old",
	}

	// Mock repository to return existing content
	mockRepo.On("GetByID", ctx, contentID).Return(existingContent, nil)

	// Mock repository to succeed on Update
	mockRepo.On("Update", ctx, mock.MatchedBy(func(c *models.Content) bool {
		return c.ID == contentID && c.Title == "Updated Title" && c.OriginalURL == url
	})).Return(nil)

	updatedContent, err := service.UpdateContent(ctx, contentID, url, html, &publishedAt)

	assert.NoError(t, err)
	assert.NotNil(t, updatedContent)
	assert.Equal(t, contentID, updatedContent.ID)
	assert.Equal(t, "Updated Title", updatedContent.Title)
	assert.Equal(t, url, updatedContent.OriginalURL)
	assert.NotEqual(t, "old-hash", updatedContent.ContentHash)
	mockRepo.AssertExpectations(t)
}

// TestUpdateContent_NotFound tests update when content doesn't exist
func TestUpdateContent_NotFound(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	contentID := uuid.New()
	url := "https://example.com/article"
	html := "<html><body>Content</body></html>"

	// Mock repository to return not found error
	mockRepo.On("GetByID", ctx, contentID).Return(nil, sql.ErrNoRows)

	updatedContent, err := service.UpdateContent(ctx, contentID, url, html, nil)

	assert.Error(t, err)
	assert.Nil(t, updatedContent)
	assert.Contains(t, err.Error(), "failed to get existing content")
	mockRepo.AssertExpectations(t)
}

// TestUpdateContent_UpdateError tests error during update operation
func TestUpdateContent_UpdateError(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	contentID := uuid.New()
	url := "https://example.com/article"
	html := "<html><head><title>Title</title></head><body>Content</body></html>"

	existingContent := &models.Content{
		ID:    contentID,
		Title: "Old Title",
	}

	// Mock repository to return existing content
	mockRepo.On("GetByID", ctx, contentID).Return(existingContent, nil)

	// Mock repository to fail on Update
	mockRepo.On("Update", ctx, mock.Anything).Return(errors.New("update failed"))

	updatedContent, err := service.UpdateContent(ctx, contentID, url, html, nil)

	assert.Error(t, err)
	assert.Nil(t, updatedContent)
	assert.Contains(t, err.Error(), "failed to update content")
	mockRepo.AssertExpectations(t)
}

// TestGetByID_Success tests successful content retrieval
func TestGetByID_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	contentID := uuid.New()

	expectedContent := &models.Content{
		ID:    contentID,
		Title: "Test Article",
	}

	mockRepo.On("GetByID", ctx, contentID).Return(expectedContent, nil)

	content, err := service.GetByID(ctx, contentID)

	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.Equal(t, contentID, content.ID)
	mockRepo.AssertExpectations(t)
}

// TestCheckDuplicate tests duplicate checking functionality
func TestCheckDuplicate(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	contentHash := "test-hash"
	feedID := uuid.New()

	expectedContent := &models.Content{
		ID:          uuid.New(),
		ContentHash: contentHash,
	}

	mockRepo.On("GetByContentHashAndFeedID", ctx, contentHash, feedID).Return(expectedContent, nil)

	content, err := service.CheckDuplicate(ctx, contentHash, feedID)

	assert.NoError(t, err)
	assert.NotNil(t, content)
	assert.Equal(t, contentHash, content.ContentHash)
	mockRepo.AssertExpectations(t)
}

// TestListContents tests content listing with pagination
func TestListContents(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	limit := 20
	offset := 0

	expectedContents := []*models.Content{
		{ID: uuid.New(), Title: "Article 1"},
		{ID: uuid.New(), Title: "Article 2"},
	}

	mockRepo.On("List", ctx, limit, offset).Return(expectedContents, nil)

	contents, err := service.ListContents(ctx, limit, offset)

	assert.NoError(t, err)
	assert.Len(t, contents, 2)
	mockRepo.AssertExpectations(t)
}

// TestBulkCreateFromHTML_Success tests successful bulk content creation
func TestBulkCreateFromHTML_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	items := []BulkContentItem{
		{
			URL:        "https://example.com/article1",
			HTML:       "<html><head><title>Article 1</title></head><body>Content 1</body></html>",
			SourceType: "web",
		},
		{
			URL:        "https://example.com/article2",
			HTML:       "<html><head><title>Article 2</title></head><body>Content 2</body></html>",
			SourceType: "web",
		},
	}

	// Mock repository to return nil for duplicate check (no duplicate)
	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, mock.Anything).Return(nil, nil)

	// Mock BulkCreate to succeed
	mockRepo.On("BulkCreate", ctx, mock.MatchedBy(func(contents []*models.Content) bool {
		return len(contents) == 2 && contents[0].Title == "Article 1" && contents[1].Title == "Article 2"
	})).Return(nil)

	contents, errs, err := service.BulkCreateFromHTML(ctx, items)

	assert.NoError(t, err)
	assert.Len(t, contents, 2)
	assert.Empty(t, errs)
	assert.Equal(t, "Article 1", contents[0].Title)
	assert.Equal(t, "Article 2", contents[1].Title)
	mockRepo.AssertExpectations(t)
}

// TestBulkCreateFromHTML_WithDuplicates tests bulk creation with some duplicates
func TestBulkCreateFromHTML_WithDuplicates(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	feedID := uuid.New()
	items := []BulkContentItem{
		{
			URL:          "https://example.com/article1",
			HTML:         "<html><head><title>Article 1</title></head><body>Content 1</body></html>",
			SourceType:   models.SourceTypeRSS,
			SourceFeedID: &feedID,
		},
		{
			URL:          "https://example.com/article2",
			HTML:         "<html><head><title>Article 2</title></head><body>Content 2</body></html>",
			SourceType:   models.SourceTypeRSS,
			SourceFeedID: &feedID,
		},
	}

	existingContent := &models.Content{
		ID:    uuid.New(),
		Title: "Article 1",
	}

	// First item is duplicate, second is new
	mockRepo.On("GetByContentHashAndFeedID", ctx, mock.Anything, feedID).Return(existingContent, nil).Once()
	mockRepo.On("GetByContentHashAndFeedID", ctx, mock.Anything, feedID).Return(nil, nil).Once()

	// BulkCreate must only receive the genuinely new item — the existing one
	// already carries a real DB id and re-inserting it would PK-collide.
	mockRepo.On("BulkCreate", ctx, mock.MatchedBy(func(contents []*models.Content) bool {
		if len(contents) != 1 {
			return false
		}
		return contents[0].ID != existingContent.ID
	})).Return(nil)

	contents, errs, err := service.BulkCreateFromHTML(ctx, items)

	assert.NoError(t, err)
	assert.Len(t, contents, 2) // Both items returned (1 existing, 1 new)
	assert.Empty(t, errs)
	mockRepo.AssertExpectations(t)
}

// TestBulkCreateFromHTML_CallerSuppliedTitleWins verifies that a caller-supplied
// title overrides whatever readability extracts from the HTML.
func TestBulkCreateFromHTML_CallerSuppliedTitleWins(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	suppliedTitle := "Caller-Supplied Title"
	suppliedAuthor := "Caller-Supplied Author"
	items := []BulkContentItem{
		{
			URL:        "https://example.com/article",
			HTML:       "<html><head><title>Readability Title</title></head><body><p>Content</p></body></html>",
			SourceType: "web",
			Title:      &suppliedTitle,
			Author:     &suppliedAuthor,
		},
	}

	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, mock.Anything).Return(nil, nil)

	mockRepo.On("BulkCreate", ctx, mock.MatchedBy(func(contents []*models.Content) bool {
		if len(contents) != 1 {
			return false
		}
		if contents[0].Title != suppliedTitle {
			return false
		}
		if contents[0].Author == nil || *contents[0].Author != suppliedAuthor {
			return false
		}
		return true
	})).Return(nil)

	contents, errs, err := service.BulkCreateFromHTML(ctx, items)

	assert.NoError(t, err)
	assert.Empty(t, errs)
	assert.Len(t, contents, 1)
	assert.Equal(t, suppliedTitle, contents[0].Title)
	if assert.NotNil(t, contents[0].Author) {
		assert.Equal(t, suppliedAuthor, *contents[0].Author)
	}
	mockRepo.AssertExpectations(t)
}

// TestBulkCreateFromHTML_FallsBackToReadabilityTitle verifies that when no
// caller-supplied title is provided, readability output is used.
func TestBulkCreateFromHTML_FallsBackToReadabilityTitle(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	items := []BulkContentItem{
		{
			URL:        "https://example.com/article",
			HTML:       "<html><head><title>Readability Title</title></head><body><p>Content</p></body></html>",
			SourceType: "web",
		},
	}

	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, mock.Anything).Return(nil, nil)

	mockRepo.On("BulkCreate", ctx, mock.MatchedBy(func(contents []*models.Content) bool {
		return len(contents) == 1 && contents[0].Title == "Readability Title"
	})).Return(nil)

	contents, errs, err := service.BulkCreateFromHTML(ctx, items)

	assert.NoError(t, err)
	assert.Empty(t, errs)
	assert.Len(t, contents, 1)
	assert.Equal(t, "Readability Title", contents[0].Title)
	mockRepo.AssertExpectations(t)
}

// TestBulkCreateFromHTML_EmptyCallerTitleFallsBack verifies that an empty
// caller-supplied title is treated as "not supplied" and we fall back to
// readability output.
func TestBulkCreateFromHTML_EmptyCallerTitleFallsBack(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	emptyTitle := ""
	items := []BulkContentItem{
		{
			URL:        "https://example.com/article",
			HTML:       "<html><head><title>Readability Title</title></head><body><p>Content</p></body></html>",
			SourceType: "web",
			Title:      &emptyTitle,
		},
	}

	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, mock.Anything).Return(nil, nil)

	mockRepo.On("BulkCreate", ctx, mock.MatchedBy(func(contents []*models.Content) bool {
		return len(contents) == 1 && contents[0].Title == "Readability Title"
	})).Return(nil)

	contents, _, err := service.BulkCreateFromHTML(ctx, items)
	assert.NoError(t, err)
	assert.Len(t, contents, 1)
	assert.Equal(t, "Readability Title", contents[0].Title)
	mockRepo.AssertExpectations(t)
}

// TestBulkCreateFromHTML_BulkCreateError tests error during bulk creation
func TestBulkCreateFromHTML_BulkCreateError(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	items := []BulkContentItem{
		{
			URL:        "https://example.com/article",
			HTML:       "<html><head><title>Article</title></head><body>Content</body></html>",
			SourceType: "web",
		},
	}

	mockRepo.On("GetByContentHashAndURL", ctx, mock.Anything, mock.Anything).Return(nil, nil)

	// Mock BulkCreate to fail
	mockRepo.On("BulkCreate", ctx, mock.Anything).Return(errors.New("bulk insert failed"))

	contents, errs, err := service.BulkCreateFromHTML(ctx, items)

	assert.Error(t, err)
	assert.Nil(t, contents)
	assert.Nil(t, errs)
	assert.Contains(t, err.Error(), "failed to bulk create contents")
	mockRepo.AssertExpectations(t)
}

// TestCheckDuplicates_Success tests checking multiple items for duplicates
func TestCheckDuplicates_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	feedID := uuid.New()
	items := []DuplicateCheckItem{
		{ContentHash: "hash1", SourceFeedID: feedID},
		{ContentHash: "hash2", SourceFeedID: feedID},
	}

	expectedMap := map[string]*models.Content{
		"hash1": {ID: uuid.New(), ContentHash: "hash1"},
	}

	mockRepo.On("GetByContentHashesAndFeedID", ctx, []string{"hash1", "hash2"}, feedID).Return(expectedMap, nil)

	result, err := service.CheckDuplicates(ctx, items)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Contains(t, result, "hash1")
	mockRepo.AssertExpectations(t)
}

// TestCheckDuplicates_Empty tests checking with empty items list
func TestCheckDuplicates_Empty(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	items := []DuplicateCheckItem{}

	result, err := service.CheckDuplicates(ctx, items)

	assert.NoError(t, err)
	assert.Empty(t, result)
	mockRepo.AssertNotCalled(t, "GetByContentHashesAndFeedID")
}

// TestCheckDuplicates_MultipleFeedGroups tests checking duplicates across multiple feeds
func TestCheckDuplicates_MultipleFeedGroups(t *testing.T) {
	mockRepo := new(MockContentRepository)
	service := NewContentService(mockRepo, &sql.DB{})

	ctx := context.Background()
	feedID1 := uuid.New()
	feedID2 := uuid.New()
	items := []DuplicateCheckItem{
		{ContentHash: "hash1", SourceFeedID: feedID1},
		{ContentHash: "hash2", SourceFeedID: feedID1},
		{ContentHash: "hash3", SourceFeedID: feedID2},
	}

	map1 := map[string]*models.Content{
		"hash1": {ID: uuid.New(), ContentHash: "hash1"},
	}
	map2 := map[string]*models.Content{
		"hash3": {ID: uuid.New(), ContentHash: "hash3"},
	}

	mockRepo.On("GetByContentHashesAndFeedID", ctx, mock.MatchedBy(func(hashes []string) bool {
		return len(hashes) == 2
	}), feedID1).Return(map1, nil)

	mockRepo.On("GetByContentHashesAndFeedID", ctx, []string{"hash3"}, feedID2).Return(map2, nil)

	result, err := service.CheckDuplicates(ctx, items)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Contains(t, result, "hash1")
	assert.Contains(t, result, "hash3")
	mockRepo.AssertExpectations(t)
}
