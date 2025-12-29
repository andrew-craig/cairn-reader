package fetcher

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-read/fetcher/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockFeedRepository is a mock implementation of FeedRepository
type MockFeedRepository struct {
	mock.Mock
}

func (m *MockFeedRepository) Create(ctx context.Context, feed *models.Feed) error {
	args := m.Called(ctx, feed)
	return args.Error(0)
}

func (m *MockFeedRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Feed, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) GetByURL(ctx context.Context, feedURL string) (*models.Feed, error) {
	args := m.Called(ctx, feedURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) List(ctx context.Context, limit, offset int) ([]*models.Feed, error) {
	args := m.Called(ctx, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) Update(ctx context.Context, feed *models.Feed) error {
	args := m.Called(ctx, feed)
	return args.Error(0)
}

func (m *MockFeedRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFeedRepository) GetFeedsDueForPolling(ctx context.Context, limit int) ([]*models.Feed, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) UpdatePollingInfo(ctx context.Context, feedID uuid.UUID, lastFetchedAt, lastPublishedAt *time.Time, nextPollAt time.Time, tier models.PollingTier) error {
	args := m.Called(ctx, feedID, lastFetchedAt, lastPublishedAt, nextPollAt, tier)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdateErrorInfo(ctx context.Context, feedID uuid.UUID, consecutiveErrorDays int, lastErrorAt *time.Time, lastErrorMessage *string) error {
	args := m.Called(ctx, feedID, consecutiveErrorDays, lastErrorAt, lastErrorMessage)
	return args.Error(0)
}

func (m *MockFeedRepository) DisableFeed(ctx context.Context, feedID uuid.UUID) error {
	args := m.Called(ctx, feedID)
	return args.Error(0)
}

func (m *MockFeedRepository) EnableFeed(ctx context.Context, feedID uuid.UUID) error {
	args := m.Called(ctx, feedID)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdateMetadata(ctx context.Context, feedID uuid.UUID, title, description, siteURL *string) error {
	args := m.Called(ctx, feedID, title, description, siteURL)
	return args.Error(0)
}

func (m *MockFeedRepository) GetFeedsForTierUpdate(ctx context.Context) ([]*models.Feed, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.FeedStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

// MockFeedItemRepository is a mock implementation of FeedItemRepository
type MockFeedItemRepository struct {
	mock.Mock
}

func (m *MockFeedItemRepository) Create(ctx context.Context, item *models.FeedItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockFeedItemRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FeedItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedItem), args.Error(1)
}

func (m *MockFeedItemRepository) GetByFeedAndGUID(ctx context.Context, feedID uuid.UUID, guid string) (*models.FeedItem, error) {
	args := m.Called(ctx, feedID, guid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedItem), args.Error(1)
}

func (m *MockFeedItemRepository) Update(ctx context.Context, item *models.FeedItem) error {
	args := m.Called(ctx, item)
	return args.Error(0)
}

func (m *MockFeedItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFeedItemRepository) GetPendingItems(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
}


func (m *MockFeedItemRepository) DeleteOldCompletedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockFeedItemRepository) DeleteOldFailedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Int(0), args.Error(1)
}

func (m *MockFeedItemRepository) GetItemsForUpdateCheck(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
}

func (m *MockFeedItemRepository) GetByFeedID(ctx context.Context, feedID uuid.UUID, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, feedID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
}

func (m *MockFeedItemRepository) GetMetrics(ctx context.Context) (*repository.FeedItemMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.FeedItemMetrics), args.Error(1)
}

func (m *MockFeedItemRepository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, contentHash *string, contentServiceID *uuid.UUID, processedAt *time.Time) error {
	args := m.Called(ctx, id, status, contentHash, contentServiceID, processedAt)
	return args.Error(0)
}

func (m *MockFeedItemRepository) UpdateContentUpdateInfo(ctx context.Context, id uuid.UUID, httpLastModified, httpETag *string, lastCheckedAt *time.Time) error {
	args := m.Called(ctx, id, httpLastModified, httpETag, lastCheckedAt)
	return args.Error(0)
}

func (m *MockFeedItemRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID, lastError string) error {
	args := m.Called(ctx, id, lastError)
	return args.Error(0)
}

// Test helper to create a valid RSS feed response
func createTestRSSFeed(title, description string, itemCount int) string {
	items := ""
	for i := 0; i < itemCount; i++ {
		items += fmt.Sprintf(`
    <item>
      <title>Article %d</title>
      <link>https://example.com/article%d</link>
      <description>Article %d description</description>
      <guid>https://example.com/article%d</guid>
      <pubDate>Mon, %02d Jan 2024 12:00:00 GMT</pubDate>
    </item>`, i+1, i+1, i+1, i+1, i+1)
	}

	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>%s</title>
    <description>%s</description>
    <link>https://example.com</link>%s
  </channel>
</rss>`, title, description, items)
}

// TestDefaultFeedFetcherConfig tests the default configuration
func TestDefaultFeedFetcherConfig(t *testing.T) {
	config := DefaultFeedFetcherConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 10, config.MaxRedirects)
	assert.Equal(t, "Cairn-RSS-Fetcher/1.0", config.UserAgent)
	assert.True(t, config.VerifySSL)
}

// TestNewFeedFetcher_WithConfig tests creating a fetcher with custom config
func TestNewFeedFetcher_WithConfig(t *testing.T) {
	config := &FeedFetcherConfig{
		Timeout:      15 * time.Second,
		MaxRedirects: 5,
		UserAgent:    "CustomAgent/1.0",
		VerifySSL:    false,
	}

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(config, mockFeedRepo, mockFeedItemRepo)

	assert.NotNil(t, fetcher)
	assert.Equal(t, config, fetcher.config)
	assert.NotNil(t, fetcher.httpClient)
	assert.NotNil(t, fetcher.parser)
}

// TestNewFeedFetcher_WithNilConfig tests creating a fetcher with nil config uses defaults
func TestNewFeedFetcher_WithNilConfig(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)

	assert.NotNil(t, fetcher)
	assert.NotNil(t, fetcher.config)
	assert.Equal(t, 30*time.Second, fetcher.config.Timeout)
}

// TestProcessFeed_Success tests successful feed processing
func TestProcessFeed_Success(t *testing.T) {
	// Create test server
	rssFeed := createTestRSSFeed("Test Feed", "Test Description", 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	// Create mocks
	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	// Create fetcher
	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	// Override http client to use test server client
	fetcher.httpClient = server.Client()

	// Create test feed
	feedID := uuid.New()
	title := "Old Title"
	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     server.URL,
		Title:       &title,
		PollingTier: models.PollingTierActive,
	}

	// Setup expectations - both items are new
	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "https://example.com/article1").
		Return(nil, nil)
	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "https://example.com/article2").
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.MatchedBy(func(item *models.FeedItem) bool {
		return item.FeedID == feedID && item.ProcessingStatus == models.ProcessingStatusPending
	})).Return(nil).Times(2)

	// Expect polling info update after success
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	// Execute
	err := fetcher.ProcessFeed(context.Background(), feed)

	// Assert
	require.NoError(t, err)
	mockFeedItemRepo.AssertExpectations(t)
	mockFeedRepo.AssertExpectations(t)
}

// TestProcessFeed_UpdatesMetadata tests that feed metadata is updated
func TestProcessFeed_UpdatesMetadata(t *testing.T) {
	rssFeed := createTestRSSFeed("New Title", "New Description", 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	fetcher.httpClient = server.Client()

	feedID := uuid.New()
	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     server.URL,
		Title:       nil, // No title initially
		Description: nil, // No description initially
		PollingTier: models.PollingTierActive,
	}

	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, mock.Anything).
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.NoError(t, err)
	assert.NotNil(t, feed.Title)
	assert.Equal(t, "New Title", *feed.Title)
	assert.NotNil(t, feed.Description)
	assert.Equal(t, "New Description", *feed.Description)
}

// TestProcessFeed_SkipsDuplicateItems tests that duplicate items are skipped
func TestProcessFeed_SkipsDuplicateItems(t *testing.T) {
	rssFeed := createTestRSSFeed("Test Feed", "Test Description", 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	fetcher.httpClient = server.Client()

	feedID := uuid.New()
	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     server.URL,
		PollingTier: models.PollingTierActive,
	}

	// First item exists, second is new
	existingItem := &models.FeedItem{ID: uuid.New()}
	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "https://example.com/article1").
		Return(existingItem, nil)
	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "https://example.com/article2").
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.Anything).Return(nil).Once()
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.NoError(t, err)
	// Only one Create call should be made (for the new item)
	mockFeedItemRepo.AssertNumberOfCalls(t, "Create", 1)
}

// TestProcessFeed_FetchError tests error handling during feed fetch
func TestProcessFeed_FetchError(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	fetcher.httpClient = server.Client()

	feedID := uuid.New()
	feed := &models.Feed{
		ID:                    feedID,
		FeedURL:               server.URL,
		PollingTier:           models.PollingTierActive,
		ConsecutiveErrorDays:  0,
	}

	// Expect error info update
	mockFeedRepo.On("UpdateErrorInfo", mock.Anything, feedID, 1, mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*string")).
		Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.Error(t, err)
	mockFeedRepo.AssertExpectations(t)
}

// TestProcessFeed_IncrementConsecutiveErrors tests consecutive error tracking
func TestProcessFeed_IncrementConsecutiveErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	fetcher.httpClient = server.Client()

	feedID := uuid.New()
	feed := &models.Feed{
		ID:                    feedID,
		FeedURL:               server.URL,
		PollingTier:           models.PollingTierActive,
		ConsecutiveErrorDays:  3, // Already had errors
	}

	// Expect consecutive errors to increment to 4
	mockFeedRepo.On("UpdateErrorInfo", mock.Anything, feedID, 4, mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*string")).
		Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.Error(t, err)
	mockFeedRepo.AssertExpectations(t)
}

// TestProcessFeed_HTTPTimeout tests timeout handling
func TestProcessFeed_HTTPTimeout(t *testing.T) {
	// Create server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &FeedFetcherConfig{
		Timeout:      100 * time.Millisecond, // Very short timeout
		MaxRedirects: 10,
		UserAgent:    "Test/1.0",
		VerifySSL:    true,
	}

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(config, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	feed := &models.Feed{
		ID:                    feedID,
		FeedURL:               server.URL,
		PollingTier:           models.PollingTierActive,
		ConsecutiveErrorDays:  0,
	}

	mockFeedRepo.On("UpdateErrorInfo", mock.Anything, feedID, 1, mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*string")).
		Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	mockFeedRepo.AssertExpectations(t)
}

// TestProcessFeed_MaxRedirects tests redirect limit enforcement
func TestProcessFeed_MaxRedirects(t *testing.T) {
	redirectCount := 0
	maxRedirects := 3

	// Create server that redirects infinitely
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectCount++
		http.Redirect(w, r, "/redirect", http.StatusMovedPermanently)
	}))
	defer server.Close()

	config := &FeedFetcherConfig{
		Timeout:      5 * time.Second,
		MaxRedirects: maxRedirects,
		UserAgent:    "Test/1.0",
		VerifySSL:    true,
	}

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(config, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	feed := &models.Feed{
		ID:                    feedID,
		FeedURL:               server.URL,
		PollingTier:           models.PollingTierActive,
		ConsecutiveErrorDays:  0,
	}

	mockFeedRepo.On("UpdateErrorInfo", mock.Anything, feedID, 1, mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*string")).
		Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "stopped after")
	mockFeedRepo.AssertExpectations(t)
}

// TestProcessFeed_FollowsValidRedirects tests that valid redirects are followed
func TestProcessFeed_FollowsValidRedirects(t *testing.T) {
	rssFeed := createTestRSSFeed("Test Feed", "Test Description", 1)
	redirectCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/redirect") {
			redirectCount++
			if redirectCount < 3 {
				http.Redirect(w, r, fmt.Sprintf("/redirect%d", redirectCount+1), http.StatusFound)
				return
			}
			// After 3 redirects, serve the actual feed
			http.Redirect(w, r, "/feed", http.StatusFound)
			return
		}
		// Serve the feed
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     server.URL + "/redirect1",
		PollingTier: models.PollingTierActive,
	}

	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, mock.Anything).
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, redirectCount, 3)
}

// TestProcessFeed_SSLVerification tests SSL certificate verification
func TestProcessFeed_SSLVerification(t *testing.T) {
	t.Skip("Skipping SSL verification test - requires self-signed certificate setup")
	// This test would require setting up a test HTTPS server with a self-signed cert
	// For now, we verify the TLS config is properly set in the NewFeedFetcher test
}

// TestProcessFeed_SSLConfig tests that SSL verification config is applied
func TestProcessFeed_SSLConfig(t *testing.T) {
	// Test with SSL verification enabled
	configWithSSL := &FeedFetcherConfig{
		Timeout:      30 * time.Second,
		MaxRedirects: 10,
		UserAgent:    "Test/1.0",
		VerifySSL:    true,
	}

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcherWithSSL := NewFeedFetcher(configWithSSL, mockFeedRepo, mockFeedItemRepo)
	transport := fetcherWithSSL.httpClient.Transport.(*http.Transport)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)

	// Test with SSL verification disabled
	configWithoutSSL := &FeedFetcherConfig{
		Timeout:      30 * time.Second,
		MaxRedirects: 10,
		UserAgent:    "Test/1.0",
		VerifySSL:    false,
	}

	fetcherWithoutSSL := NewFeedFetcher(configWithoutSSL, mockFeedRepo, mockFeedItemRepo)
	transport = fetcherWithoutSSL.httpClient.Transport.(*http.Transport)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

// TestProcessFeed_TLSConfig tests that TLS configuration is properly set
func TestProcessFeed_TLSConfig(t *testing.T) {
	config := &FeedFetcherConfig{
		Timeout:      30 * time.Second,
		MaxRedirects: 10,
		UserAgent:    "Test/1.0",
		VerifySSL:    true,
	}

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(config, mockFeedRepo, mockFeedItemRepo)

	// Verify that TLS config is set
	transport, ok := fetcher.httpClient.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)

	assert.IsType(t, &tls.Config{}, transport.TLSClientConfig)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
}

// TestProcessFeed_ItemCreationError tests handling of item creation errors
func TestProcessFeed_ItemCreationError(t *testing.T) {
	rssFeed := createTestRSSFeed("Test Feed", "Test Description", 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	fetcher.httpClient = server.Client()

	feedID := uuid.New()
	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     server.URL,
		PollingTier: models.PollingTierActive,
	}

	// First item fails to create, second succeeds
	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "https://example.com/article1").
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.MatchedBy(func(item *models.FeedItem) bool {
		return strings.Contains(*item.ItemGUID, "article1")
	})).Return(errors.New("database error")).Once()

	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "https://example.com/article2").
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.MatchedBy(func(item *models.FeedItem) bool {
		return strings.Contains(*item.ItemGUID, "article2")
	})).Return(nil).Once()

	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	// Should succeed overall even if one item fails
	require.NoError(t, err)
	mockFeedItemRepo.AssertExpectations(t)
}

// TestProcessFeed_EmptyFeed tests handling of empty feeds
func TestProcessFeed_EmptyFeed(t *testing.T) {
	rssFeed := createTestRSSFeed("Empty Feed", "No items", 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(rssFeed))
	}))
	defer server.Close()

	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)
	fetcher.httpClient = server.Client()

	feedID := uuid.New()
	feed := &models.Feed{
		ID:          feedID,
		FeedURL:     server.URL,
		PollingTier: models.PollingTierActive,
	}

	// Expect no item creation
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierActive).
		Return(nil)

	err := fetcher.ProcessFeed(context.Background(), feed)

	require.NoError(t, err)
	mockFeedItemRepo.AssertNotCalled(t, "Create")
}

// TestHandleFetchError tests error handling logic
func TestHandleFetchError(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	feed := &models.Feed{
		ID:                    feedID,
		FeedURL:               "https://example.com/feed",
		PollingTier:           models.PollingTierModerate,
		ConsecutiveErrorDays:  2,
	}

	testError := errors.New("network timeout")

	mockFeedRepo.On("UpdateErrorInfo", mock.Anything, feedID, 3, mock.AnythingOfType("*time.Time"), mock.AnythingOfType("*string")).
		Return(nil)
	mockFeedRepo.On("UpdatePollingInfo", mock.Anything, feedID, mock.Anything, mock.Anything, mock.Anything, models.PollingTierModerate).
		Return(nil)

	err := fetcher.handleFetchError(context.Background(), feed, testError)

	require.Error(t, err)
	assert.Equal(t, testError, err)
	mockFeedRepo.AssertExpectations(t)
}

// TestProcessFeedItem_NewItem tests creating a new feed item
func TestProcessFeedItem_NewItem(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	item := &ParsedItem{
		Title:       "Test Article",
		Author:      "John Doe",
		PublishedAt: nil,
		Description: "Test description",
		URL:         "https://example.com/article",
		GUID:        "test-guid",
	}

	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "test-guid").
		Return(nil, nil)
	mockFeedItemRepo.On("Create", mock.Anything, mock.MatchedBy(func(feedItem *models.FeedItem) bool {
		return feedItem.FeedID == feedID &&
			*feedItem.ItemGUID == "test-guid" &&
			feedItem.ProcessingStatus == models.ProcessingStatusPending &&
			*feedItem.Title == "Test Article" &&
			*feedItem.Author == "John Doe"
	})).Return(nil)

	isNew, err := fetcher.processFeedItem(context.Background(), feedID, item)

	require.NoError(t, err)
	assert.True(t, isNew)
	mockFeedItemRepo.AssertExpectations(t)
}

// TestProcessFeedItem_ExistingItem tests skipping existing items
func TestProcessFeedItem_ExistingItem(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	item := &ParsedItem{
		GUID: "existing-guid",
		URL:  "https://example.com/article",
	}

	existingItem := &models.FeedItem{
		ID:     uuid.New(),
		FeedID: feedID,
	}

	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "existing-guid").
		Return(existingItem, nil)

	isNew, err := fetcher.processFeedItem(context.Background(), feedID, item)

	require.NoError(t, err)
	assert.False(t, isNew)
	mockFeedItemRepo.AssertNotCalled(t, "Create")
}

// TestProcessFeedItem_DatabaseError tests handling of database errors
func TestProcessFeedItem_DatabaseError(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockFeedItemRepo := new(MockFeedItemRepository)

	fetcher := NewFeedFetcher(nil, mockFeedRepo, mockFeedItemRepo)

	feedID := uuid.New()
	item := &ParsedItem{
		GUID: "test-guid",
		URL:  "https://example.com/article",
	}

	mockFeedItemRepo.On("GetByFeedAndGUID", mock.Anything, feedID, "test-guid").
		Return(nil, errors.New("database connection error"))

	isNew, err := fetcher.processFeedItem(context.Background(), feedID, item)

	require.Error(t, err)
	assert.False(t, isNew)
	assert.Contains(t, err.Error(), "failed to check for existing item")
}
