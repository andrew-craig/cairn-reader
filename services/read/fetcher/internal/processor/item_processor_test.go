package processor

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Mock implementations are scoped to this test file to avoid leaking test-only
// types into the production package. They cover only the surface that
// processItem touches.

type mockFeedItemRepo struct{ mock.Mock }

func (m *mockFeedItemRepo) Create(ctx context.Context, item *models.FeedItem) error {
	return m.Called(ctx, item).Error(0)
}
func (m *mockFeedItemRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.FeedItem, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedItem), args.Error(1)
}
func (m *mockFeedItemRepo) GetByFeedAndGUID(ctx context.Context, feedID uuid.UUID, guid string) (*models.FeedItem, error) {
	args := m.Called(ctx, feedID, guid)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedItem), args.Error(1)
}
func (m *mockFeedItemRepo) Update(ctx context.Context, item *models.FeedItem) error {
	return m.Called(ctx, item).Error(0)
}
func (m *mockFeedItemRepo) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, contentHash *string, contentServiceID *uuid.UUID, processedAt *time.Time) error {
	return m.Called(ctx, id, status, contentHash, contentServiceID, processedAt).Error(0)
}
func (m *mockFeedItemRepo) UpdateContentUpdateInfo(ctx context.Context, id uuid.UUID, httpLastModified, httpETag *string, lastCheckedAt *time.Time) error {
	return m.Called(ctx, id, httpLastModified, httpETag, lastCheckedAt).Error(0)
}
func (m *mockFeedItemRepo) IncrementRetryCount(ctx context.Context, id uuid.UUID, lastError string) error {
	return m.Called(ctx, id, lastError).Error(0)
}
func (m *mockFeedItemRepo) GetPendingItems(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
}
func (m *mockFeedItemRepo) GetItemsForUpdateCheck(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
}
func (m *mockFeedItemRepo) GetByFeedID(ctx context.Context, feedID uuid.UUID, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, feedID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
}
func (m *mockFeedItemRepo) DeleteOldCompletedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Int(0), args.Error(1)
}
func (m *mockFeedItemRepo) DeleteOldFailedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Int(0), args.Error(1)
}
func (m *mockFeedItemRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockFeedItemRepo) GetMetrics(ctx context.Context) (*repository.FeedItemMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.FeedItemMetrics), args.Error(1)
}

type mockSubscriptionRepo struct{ mock.Mock }

func (m *mockSubscriptionRepo) Create(ctx context.Context, s *models.FeedSubscription) error {
	return m.Called(ctx, s).Error(0)
}
func (m *mockSubscriptionRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.FeedSubscription, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedSubscription), args.Error(1)
}
func (m *mockSubscriptionRepo) GetByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) (*models.FeedSubscription, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedSubscription), args.Error(1)
}
func (m *mockSubscriptionRepo) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.FeedSubscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedSubscription), args.Error(1)
}
func (m *mockSubscriptionRepo) GetByFeedID(ctx context.Context, feedID uuid.UUID) ([]*models.FeedSubscription, error) {
	args := m.Called(ctx, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedSubscription), args.Error(1)
}
func (m *mockSubscriptionRepo) GetSubscriberIDs(ctx context.Context, feedID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}
func (m *mockSubscriptionRepo) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}
func (m *mockSubscriptionRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockSubscriptionRepo) DeleteByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) error {
	return m.Called(ctx, userID, feedID).Error(0)
}

type mockOutboxRepo struct{ mock.Mock }

func (m *mockOutboxRepo) Create(ctx context.Context, outbox *models.ContentOutbox) error {
	return m.Called(ctx, outbox).Error(0)
}
func (m *mockOutboxRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentOutbox), args.Error(1)
}
func (m *mockOutboxRepo) GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ContentOutbox), args.Error(1)
}
func (m *mockOutboxRepo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time, lastError *string) error {
	return m.Called(ctx, id, status, contentServiceID, deliveredAt, lastError).Error(0)
}
func (m *mockOutboxRepo) IncrementRetryCount(ctx context.Context, id uuid.UUID, nextRetryAt time.Time, lastError string) error {
	return m.Called(ctx, id, nextRetryAt, lastError).Error(0)
}
func (m *mockOutboxRepo) DeleteOldDeliveredEntries(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Int(0), args.Error(1)
}
func (m *mockOutboxRepo) GetFailedEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ContentOutbox), args.Error(1)
}
func (m *mockOutboxRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockOutboxRepo) GetMetrics(ctx context.Context) (*repository.OutboxMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.OutboxMetrics), args.Error(1)
}

// sentinelRawHTML deliberately includes <meta> and class attributes — both
// stripped by bluemonday's UGCPolicy. If they survive into the outbox payload,
// the fetcher is no longer sanitizing.
const sentinelRawHTML = `<html>
  <head>
    <title>Sentinel Title</title>
    <meta name="author" content="Sentinel Author">
  </head>
  <body>
    <article class="post">
      <p>Body paragraph.</p>
    </article>
  </body>
</html>`

func newProcessorWithStubServer(t *testing.T) (*ItemProcessor, *mockFeedItemRepo, *mockSubscriptionRepo, *mockOutboxRepo, *httptest.Server) {
	t.Helper()

	feedItemRepo := new(mockFeedItemRepo)
	subRepo := new(mockSubscriptionRepo)
	outboxRepo := new(mockOutboxRepo)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.NotEmpty(t, r.Header.Get("User-Agent"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(sentinelRawHTML))
	}))
	t.Cleanup(srv.Close)

	p := NewItemProcessor(DefaultItemProcessorConfig(), feedItemRepo, subRepo, outboxRepo)
	return p, feedItemRepo, subRepo, outboxRepo, srv
}

func TestItemProcessor_OutboxPayloadCarriesRawHTML(t *testing.T) {
	p, feedItemRepo, subRepo, outboxRepo, srv := newProcessorWithStubServer(t)

	feedID := uuid.New()
	itemID := uuid.New()
	userID := uuid.New()

	title := "RSS title"
	author := "RSS author"
	publishedAt := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)

	item := &models.FeedItem{
		ID:               itemID,
		FeedID:           feedID,
		ItemURL:          srv.URL + "/article",
		Title:            &title,
		Author:           &author,
		PublishedAt:      &publishedAt,
		ProcessingStatus: models.ProcessingStatusPending,
	}

	subRepo.On("GetByFeedID", mock.Anything, feedID).Return([]*models.FeedSubscription{
		{ID: uuid.New(), UserID: userID, FeedID: feedID},
	}, nil).Once()

	var capturedOutbox *models.ContentOutbox
	outboxRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.ContentOutbox")).
		Run(func(args mock.Arguments) {
			capturedOutbox = args.Get(1).(*models.ContentOutbox)
		}).Return(nil).Once()

	feedItemRepo.On("UpdateProcessingStatus", mock.Anything, itemID, models.ProcessingStatusCompleted,
		(*string)(nil), (*uuid.UUID)(nil), mock.AnythingOfType("*time.Time")).Return(nil).Once()

	require.NoError(t, p.processItem(context.Background(), item))
	require.NotNil(t, capturedOutbox)

	// Round-trip through JSON to mirror the outbox repository's JSONB
	// serialization — that's the wire format the worker actually consumes.
	payloadJSON, err := json.Marshal(capturedOutbox.ContentPayload)
	require.NoError(t, err)

	var payload map[string]interface{}
	require.NoError(t, json.Unmarshal(payloadJSON, &payload))

	rawHTML, ok := payload[models.PayloadKeyRawHTML].(string)
	require.True(t, ok)
	assert.Equal(t, sentinelRawHTML, rawHTML)
	assert.Contains(t, rawHTML, `<meta name="author" content="Sentinel Author">`)
	assert.Contains(t, rawHTML, `<article class="post">`)

	assert.Equal(t, title, payload[models.PayloadKeyTitle])
	assert.Equal(t, author, payload[models.PayloadKeyAuthor])
	assert.Equal(t, srv.URL+"/article", payload[models.PayloadKeySourceURL])
	assert.Equal(t, feedID.String(), payload[models.PayloadKeySourceFeedID])

	_, hasCleanedHTML := payload["cleaned_html"]
	assert.False(t, hasCleanedHTML)
	_, hasHash := payload["content_hash"]
	assert.False(t, hasHash)

	feedItemRepo.AssertExpectations(t)
	subRepo.AssertExpectations(t)
	outboxRepo.AssertExpectations(t)
}

func TestItemProcessor_NoSubscribers_SkipsOutbox(t *testing.T) {
	p, feedItemRepo, subRepo, outboxRepo, srv := newProcessorWithStubServer(t)

	feedID := uuid.New()
	itemID := uuid.New()
	item := &models.FeedItem{
		ID:               itemID,
		FeedID:           feedID,
		ItemURL:          srv.URL + "/article",
		ProcessingStatus: models.ProcessingStatusPending,
	}

	subRepo.On("GetByFeedID", mock.Anything, feedID).Return([]*models.FeedSubscription{}, nil).Once()
	feedItemRepo.On("UpdateProcessingStatus", mock.Anything, itemID, models.ProcessingStatusCompleted,
		(*string)(nil), (*uuid.UUID)(nil), mock.AnythingOfType("*time.Time")).Return(nil).Once()

	require.NoError(t, p.processItem(context.Background(), item))

	feedItemRepo.AssertExpectations(t)
	subRepo.AssertExpectations(t)
	outboxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestItemProcessor_FetchFailureFallsBackToDescription(t *testing.T) {
	feedItemRepo := new(mockFeedItemRepo)
	subRepo := new(mockSubscriptionRepo)
	outboxRepo := new(mockOutboxRepo)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	p := NewItemProcessor(DefaultItemProcessorConfig(), feedItemRepo, subRepo, outboxRepo)

	desc := "<p>fallback description from RSS</p>"
	feedID := uuid.New()
	itemID := uuid.New()
	item := &models.FeedItem{
		ID:               itemID,
		FeedID:           feedID,
		ItemURL:          srv.URL + "/article",
		Description:      &desc,
		ProcessingStatus: models.ProcessingStatusPending,
		DiscoveredAt:     time.Now(),
	}

	subRepo.On("GetByFeedID", mock.Anything, feedID).Return([]*models.FeedSubscription{
		{ID: uuid.New(), UserID: uuid.New(), FeedID: feedID},
	}, nil).Once()

	var capturedOutbox *models.ContentOutbox
	outboxRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.ContentOutbox")).
		Run(func(args mock.Arguments) { capturedOutbox = args.Get(1).(*models.ContentOutbox) }).
		Return(nil).Once()
	feedItemRepo.On("UpdateProcessingStatus", mock.Anything, itemID, models.ProcessingStatusCompleted,
		(*string)(nil), (*uuid.UUID)(nil), mock.AnythingOfType("*time.Time")).Return(nil).Once()

	require.NoError(t, p.processItem(context.Background(), item))
	require.NotNil(t, capturedOutbox)
	assert.Equal(t, desc, capturedOutbox.ContentPayload[models.PayloadKeyRawHTML])
}

// A user who subscribes after an item was published must not receive that
// item — otherwise subscribing to a feed backfills the entire RSS history
// into the user's read list.
func TestItemProcessor_FiltersSubscribersBySubscriptionTime(t *testing.T) {
	p, feedItemRepo, subRepo, outboxRepo, srv := newProcessorWithStubServer(t)

	feedID := uuid.New()
	itemID := uuid.New()
	earlyUserID := uuid.New() // subscribed before the article was published
	lateUserID := uuid.New()  // subscribed after the article was published

	title := "Old article"
	publishedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	item := &models.FeedItem{
		ID:               itemID,
		FeedID:           feedID,
		ItemURL:          srv.URL + "/old-article",
		Title:            &title,
		PublishedAt:      &publishedAt,
		ProcessingStatus: models.ProcessingStatusPending,
		DiscoveredAt:     time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	subRepo.On("GetByFeedID", mock.Anything, feedID).Return([]*models.FeedSubscription{
		{ID: uuid.New(), UserID: earlyUserID, FeedID: feedID,
			SubscribedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
		{ID: uuid.New(), UserID: lateUserID, FeedID: feedID,
			SubscribedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}, nil).Once()

	var capturedOutbox *models.ContentOutbox
	outboxRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.ContentOutbox")).
		Run(func(args mock.Arguments) { capturedOutbox = args.Get(1).(*models.ContentOutbox) }).
		Return(nil).Once()
	feedItemRepo.On("UpdateProcessingStatus", mock.Anything, itemID, models.ProcessingStatusCompleted,
		(*string)(nil), (*uuid.UUID)(nil), mock.AnythingOfType("*time.Time")).Return(nil).Once()

	require.NoError(t, p.processItem(context.Background(), item))
	require.NotNil(t, capturedOutbox)
	assert.Equal(t, []uuid.UUID{earlyUserID}, capturedOutbox.UserIDs)
}

// When every subscriber joined after the item was published, the item still
// gets marked completed but no outbox entry is written — there's no one to
// deliver it to.
func TestItemProcessor_AllSubscribersAfterItem_SkipsOutbox(t *testing.T) {
	p, feedItemRepo, subRepo, outboxRepo, srv := newProcessorWithStubServer(t)

	feedID := uuid.New()
	itemID := uuid.New()
	publishedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	title := "Historical article"

	item := &models.FeedItem{
		ID:               itemID,
		FeedID:           feedID,
		ItemURL:          srv.URL + "/historical",
		Title:            &title,
		PublishedAt:      &publishedAt,
		ProcessingStatus: models.ProcessingStatusPending,
		DiscoveredAt:     time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	subRepo.On("GetByFeedID", mock.Anything, feedID).Return([]*models.FeedSubscription{
		{ID: uuid.New(), UserID: uuid.New(), FeedID: feedID,
			SubscribedAt: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)},
	}, nil).Once()
	feedItemRepo.On("UpdateProcessingStatus", mock.Anything, itemID, models.ProcessingStatusCompleted,
		(*string)(nil), (*uuid.UUID)(nil), mock.AnythingOfType("*time.Time")).Return(nil).Once()

	require.NoError(t, p.processItem(context.Background(), item))

	outboxRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

// Oversized responses must error rather than silently truncate. io.LimitReader
// returns N bytes with nil error on EOF, so a "truncate then ship" approach
// would send partial HTML to the Content Service and break page structure.
func TestItemProcessor_FetchRejectsOversizedBody(t *testing.T) {
	feedItemRepo := new(mockFeedItemRepo)
	subRepo := new(mockSubscriptionRepo)
	outboxRepo := new(mockOutboxRepo)

	body := strings.Repeat("a", 6*1024*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	cfg := DefaultItemProcessorConfig()
	p := NewItemProcessor(cfg, feedItemRepo, subRepo, outboxRepo)

	_, err := p.fetchArticleContent(context.Background(), srv.URL+"/big")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds maximum")
}

// A response exactly at the cap is accepted in full.
func TestItemProcessor_FetchAcceptsBodyAtExactCap(t *testing.T) {
	feedItemRepo := new(mockFeedItemRepo)
	subRepo := new(mockSubscriptionRepo)
	outboxRepo := new(mockOutboxRepo)

	cfg := &ItemProcessorConfig{
		ContentFetchTimeout: 30 * time.Second,
		MaxContentSize:      1024,
		MaxRetries:          3,
	}
	body := strings.Repeat("a", int(cfg.MaxContentSize))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	p := NewItemProcessor(cfg, feedItemRepo, subRepo, outboxRepo)
	got, err := p.fetchArticleContent(context.Background(), srv.URL+"/exact")
	require.NoError(t, err)
	assert.Equal(t, int64(len(got)), cfg.MaxContentSize)
}
