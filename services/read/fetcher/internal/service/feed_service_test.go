package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFeedRepository is a mock implementation of repository.FeedRepository
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

func (m *MockFeedRepository) GetByURL(ctx context.Context, url string) (*models.Feed, error) {
	args := m.Called(ctx, url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) Update(ctx context.Context, feed *models.Feed) error {
	args := m.Called(ctx, feed)
	return args.Error(0)
}

func (m *MockFeedRepository) GetFeedsDueForPolling(ctx context.Context, limit int) ([]*models.Feed, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.FeedStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockFeedRepository) ReleaseClaim(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdatePollingInfo(ctx context.Context, id uuid.UUID, lastFetched *time.Time, lastPublished *time.Time, nextPoll time.Time, tier models.PollingTier) error {
	args := m.Called(ctx, id, lastFetched, lastPublished, nextPoll, tier)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdateErrorInfo(ctx context.Context, id uuid.UUID, consecutiveErrorDays int, lastErrorAt *time.Time, lastError *string) error {
	args := m.Called(ctx, id, consecutiveErrorDays, lastErrorAt, lastError)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdateTier(ctx context.Context, id uuid.UUID, tier models.PollingTier) error {
	args := m.Called(ctx, id, tier)
	return args.Error(0)
}

func (m *MockFeedRepository) GetFeedsForTierUpdate(ctx context.Context) ([]*models.Feed, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Feed), args.Error(1)
}

func (m *MockFeedRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// MockFeedSubscriptionRepository is a mock implementation of repository.FeedSubscriptionRepository
type MockFeedSubscriptionRepository struct {
	mock.Mock
}

func (m *MockFeedSubscriptionRepository) Create(ctx context.Context, subscription *models.FeedSubscription) error {
	args := m.Called(ctx, subscription)
	return args.Error(0)
}

func (m *MockFeedSubscriptionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.FeedSubscription, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedSubscription), args.Error(1)
}

func (m *MockFeedSubscriptionRepository) GetByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) (*models.FeedSubscription, error) {
	args := m.Called(ctx, userID, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.FeedSubscription), args.Error(1)
}

func (m *MockFeedSubscriptionRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*models.FeedSubscription, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedSubscription), args.Error(1)
}

func (m *MockFeedSubscriptionRepository) GetByFeedID(ctx context.Context, feedID uuid.UUID) ([]*models.FeedSubscription, error) {
	args := m.Called(ctx, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedSubscription), args.Error(1)
}

func (m *MockFeedSubscriptionRepository) GetSubscriberIDs(ctx context.Context, feedID uuid.UUID) ([]uuid.UUID, error) {
	args := m.Called(ctx, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]uuid.UUID), args.Error(1)
}

func (m *MockFeedSubscriptionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFeedSubscriptionRepository) DeleteByUserAndFeed(ctx context.Context, userID, feedID uuid.UUID) error {
	args := m.Called(ctx, userID, feedID)
	return args.Error(0)
}

func (m *MockFeedSubscriptionRepository) CountByUserID(ctx context.Context, userID uuid.UUID) (int, error) {
	args := m.Called(ctx, userID)
	return args.Int(0), args.Error(1)
}

// Note: TestSubscribe_NewFeed would require mocking the HTTP client
// We'll focus on testable components with existing feed instead

// TestSubscribe_ExistingFeed tests subscribing to an existing feed
func TestSubscribe_ExistingFeed_Success(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedURL := "https://example.com/feed.xml"

	existingFeed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: feedURL,
		Status:  models.FeedStatusActive,
	}

	// Mock subscription count check
	mockSubRepo.On("CountByUserID", ctx, userID).Return(5, nil)

	// Mock feed lookup (feed exists)
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// Mock subscription check (user not yet subscribed)
	mockSubRepo.On("GetByUserAndFeed", ctx, userID, existingFeed.ID).Return(nil, nil)

	// Mock subscription creation
	mockSubRepo.On("Create", ctx, mock.MatchedBy(func(s *models.FeedSubscription) bool {
		return s.UserID == userID && s.FeedID == existingFeed.ID
	})).Return(nil)

	result, err := service.Subscribe(ctx, userID, feedURL)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, existingFeed.ID, result.Feed.ID)
	assert.False(t, result.IsNewFeed)
	assert.Equal(t, userID, result.Subscription.UserID)
	mockFeedRepo.AssertExpectations(t)
	mockSubRepo.AssertExpectations(t)
}

// TestSubscribe_FeedLimitReached tests error when user reaches feed limit
func TestSubscribe_FeedLimitReached(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedURL := "https://example.com/feed.xml"

	// Mock subscription count check (user has max subscriptions)
	mockSubRepo.On("CountByUserID", ctx, userID).Return(MaxFeedsPerUser, nil)

	result, err := service.Subscribe(ctx, userID, feedURL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "maximum feed limit")
	mockSubRepo.AssertExpectations(t)
	// GetByURL should not be called
	mockFeedRepo.AssertNotCalled(t, "GetByURL")
}

// TestSubscribe_AlreadySubscribed tests error when user is already subscribed
func TestSubscribe_AlreadySubscribed(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedURL := "https://example.com/feed.xml"

	existingFeed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: feedURL,
	}

	existingSubscription := &models.FeedSubscription{
		ID:     uuid.New(),
		UserID: userID,
		FeedID: existingFeed.ID,
	}

	// Mock subscription count check
	mockSubRepo.On("CountByUserID", ctx, userID).Return(5, nil)

	// Mock feed lookup
	mockFeedRepo.On("GetByURL", ctx, feedURL).Return(existingFeed, nil)

	// Mock subscription check (user already subscribed)
	mockSubRepo.On("GetByUserAndFeed", ctx, userID, existingFeed.ID).Return(existingSubscription, nil)

	result, err := service.Subscribe(ctx, userID, feedURL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "already subscribed")
	mockFeedRepo.AssertExpectations(t)
	mockSubRepo.AssertExpectations(t)
	// Create should not be called
	mockSubRepo.AssertNotCalled(t, "Create")
}

// TestSubscribe_CountError tests error during subscription count check
func TestSubscribe_CountError(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedURL := "https://example.com/feed.xml"

	// Mock subscription count to return error
	mockSubRepo.On("CountByUserID", ctx, userID).Return(0, errors.New("database error"))

	result, err := service.Subscribe(ctx, userID, feedURL)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to count user subscriptions")
	mockSubRepo.AssertExpectations(t)
}

// TestUnsubscribe_Success tests successful unsubscribe
func TestUnsubscribe_Success(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedID := uuid.New()

	existingSubscription := &models.FeedSubscription{
		ID:     uuid.New(),
		UserID: userID,
		FeedID: feedID,
	}

	// Mock subscription check
	mockSubRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(existingSubscription, nil)

	// Mock deletion
	mockSubRepo.On("DeleteByUserAndFeed", ctx, userID, feedID).Return(nil)

	err := service.Unsubscribe(ctx, userID, feedID)

	assert.NoError(t, err)
	mockSubRepo.AssertExpectations(t)
}

// TestUnsubscribe_NotFound tests error when subscription doesn't exist
func TestUnsubscribe_NotFound(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedID := uuid.New()

	// Mock subscription check (not found)
	mockSubRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(nil, nil)

	err := service.Unsubscribe(ctx, userID, feedID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "subscription not found")
	mockSubRepo.AssertExpectations(t)
	// Delete should not be called
	mockSubRepo.AssertNotCalled(t, "DeleteByUserAndFeed")
}

// TestUnsubscribe_DeleteError tests error during deletion
func TestUnsubscribe_DeleteError(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()
	feedID := uuid.New()

	existingSubscription := &models.FeedSubscription{
		ID:     uuid.New(),
		UserID: userID,
		FeedID: feedID,
	}

	// Mock subscription check
	mockSubRepo.On("GetByUserAndFeed", ctx, userID, feedID).Return(existingSubscription, nil)

	// Mock deletion error
	mockSubRepo.On("DeleteByUserAndFeed", ctx, userID, feedID).Return(errors.New("delete failed"))

	err := service.Unsubscribe(ctx, userID, feedID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to delete subscription")
	mockSubRepo.AssertExpectations(t)
}

// TestListUserSubscriptions_Success tests listing user subscriptions
func TestListUserSubscriptions_Success(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()

	feed1 := &models.Feed{ID: uuid.New(), FeedURL: "https://example.com/feed1.xml"}
	feed2 := &models.Feed{ID: uuid.New(), FeedURL: "https://example.com/feed2.xml"}

	subscriptions := []*models.FeedSubscription{
		{ID: uuid.New(), UserID: userID, FeedID: feed1.ID},
		{ID: uuid.New(), UserID: userID, FeedID: feed2.ID},
	}

	// Mock subscription list
	mockSubRepo.On("GetByUserID", ctx, userID).Return(subscriptions, nil)

	// Mock feed lookups
	mockFeedRepo.On("GetByID", ctx, feed1.ID).Return(feed1, nil)
	mockFeedRepo.On("GetByID", ctx, feed2.ID).Return(feed2, nil)

	result, err := service.ListUserSubscriptions(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, feed1.ID, result[0].Feed.ID)
	assert.Equal(t, feed2.ID, result[1].Feed.ID)
	mockFeedRepo.AssertExpectations(t)
	mockSubRepo.AssertExpectations(t)
}

// TestListUserSubscriptions_Empty tests listing with no subscriptions
func TestListUserSubscriptions_Empty(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()

	// Mock empty subscription list
	mockSubRepo.On("GetByUserID", ctx, userID).Return([]*models.FeedSubscription{}, nil)

	result, err := service.ListUserSubscriptions(ctx, userID)

	assert.NoError(t, err)
	assert.Empty(t, result)
	mockSubRepo.AssertExpectations(t)
}

// TestListUserSubscriptions_FeedDeleted tests handling of deleted feed (defensive)
func TestListUserSubscriptions_FeedDeleted(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	userID := uuid.New()

	feed1 := &models.Feed{ID: uuid.New(), FeedURL: "https://example.com/feed1.xml"}
	deletedFeedID := uuid.New()

	subscriptions := []*models.FeedSubscription{
		{ID: uuid.New(), UserID: userID, FeedID: feed1.ID},
		{ID: uuid.New(), UserID: userID, FeedID: deletedFeedID}, // Feed was deleted
	}

	// Mock subscription list
	mockSubRepo.On("GetByUserID", ctx, userID).Return(subscriptions, nil)

	// Mock feed lookups
	mockFeedRepo.On("GetByID", ctx, feed1.ID).Return(feed1, nil)
	mockFeedRepo.On("GetByID", ctx, deletedFeedID).Return(nil, nil) // Feed deleted

	result, err := service.ListUserSubscriptions(ctx, userID)

	assert.NoError(t, err)
	assert.Len(t, result, 1) // Only one feed should be in result
	assert.Equal(t, feed1.ID, result[0].Feed.ID)
	mockFeedRepo.AssertExpectations(t)
	mockSubRepo.AssertExpectations(t)
}

// TestEnableFeed_Success tests successful feed re-enable
func TestEnableFeed_Success(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	feedID := uuid.New()

	disabledFeed := &models.Feed{
		ID:     feedID,
		Status: models.FeedStatusDisabled,
	}

	// Mock feed lookup
	mockFeedRepo.On("GetByID", ctx, feedID).Return(disabledFeed, nil)

	// Mock status update
	mockFeedRepo.On("UpdateStatus", ctx, feedID, models.FeedStatusActive).Return(nil)

	// Mock error info reset
	mockFeedRepo.On("UpdateErrorInfo", ctx, feedID, 0, (*time.Time)(nil), (*string)(nil)).Return(nil)

	// Mock polling info update
	mockFeedRepo.On("UpdatePollingInfo", ctx, feedID, (*time.Time)(nil), (*time.Time)(nil), mock.AnythingOfType("time.Time"), models.PollingTierActive).Return(nil)

	err := service.EnableFeed(ctx, feedID)

	assert.NoError(t, err)
	mockFeedRepo.AssertExpectations(t)
}

// TestEnableFeed_NotFound tests error when feed doesn't exist
func TestEnableFeed_NotFound(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	feedID := uuid.New()

	// Mock feed lookup (not found)
	mockFeedRepo.On("GetByID", ctx, feedID).Return(nil, nil)

	err := service.EnableFeed(ctx, feedID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "feed not found")
	mockFeedRepo.AssertExpectations(t)
	// No updates should be called
	mockFeedRepo.AssertNotCalled(t, "UpdateStatus")
}

// TestEnableFeed_AlreadyActive tests error when feed is already active
func TestEnableFeed_AlreadyActive(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	feedID := uuid.New()

	activeFeed := &models.Feed{
		ID:     feedID,
		Status: models.FeedStatusActive,
	}

	// Mock feed lookup
	mockFeedRepo.On("GetByID", ctx, feedID).Return(activeFeed, nil)

	err := service.EnableFeed(ctx, feedID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already active")
	mockFeedRepo.AssertExpectations(t)
	// No updates should be called
	mockFeedRepo.AssertNotCalled(t, "UpdateStatus")
}

// TestEnableFeed_UpdateStatusError tests error during status update
func TestEnableFeed_UpdateStatusError(t *testing.T) {
	mockFeedRepo := new(MockFeedRepository)
	mockSubRepo := new(MockFeedSubscriptionRepository)
	service := &feedService{
		feedRepo:         mockFeedRepo,
		subscriptionRepo: mockSubRepo,
	}

	ctx := context.Background()
	feedID := uuid.New()

	disabledFeed := &models.Feed{
		ID:     feedID,
		Status: models.FeedStatusDisabled,
	}

	// Mock feed lookup
	mockFeedRepo.On("GetByID", ctx, feedID).Return(disabledFeed, nil)

	// Mock status update error
	mockFeedRepo.On("UpdateStatus", ctx, feedID, models.FeedStatusActive).Return(errors.New("update failed"))

	err := service.EnableFeed(ctx, feedID)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update feed status")
	mockFeedRepo.AssertExpectations(t)
}
