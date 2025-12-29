package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-read/services/rss-fetcher-service/internal/models"
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

func (m *MockFeedRepository) UpdatePollingInfo(ctx context.Context, id uuid.UUID, lastFetchedAt, lastPublishedAt *time.Time, nextPollAt time.Time, tier models.PollingTier) error {
	args := m.Called(ctx, id, lastFetchedAt, lastPublishedAt, nextPollAt, tier)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdateErrorInfo(ctx context.Context, id uuid.UUID, consecutiveErrorDays int, lastErrorAt *time.Time, lastErrorMessage *string) error {
	args := m.Called(ctx, id, consecutiveErrorDays, lastErrorAt, lastErrorMessage)
	return args.Error(0)
}

func (m *MockFeedRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.FeedStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockFeedRepository) GetFeedsDueForPolling(ctx context.Context, limit int) ([]*models.Feed, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Feed), args.Error(1)
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

// MockFeedProcessor is a mock implementation of FeedProcessor
type MockFeedProcessor struct {
	mock.Mock
}

func (m *MockFeedProcessor) ProcessFeed(ctx context.Context, feed *models.Feed) error {
	args := m.Called(ctx, feed)
	return args.Error(0)
}

func TestFeedWorker_Start_Stop(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := DefaultFeedWorkerConfig()

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	// Start worker
	worker.Start()

	// Let it run briefly
	time.Sleep(50 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Worker should stop gracefully
	assert.NotNil(t, worker)
}

func TestFeedWorker_Submit_Success(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := DefaultFeedWorkerConfig()

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	feed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/feed.xml",
		Status:  models.FeedStatusActive,
	}

	// Mock processor to succeed
	mockProcessor.On("ProcessFeed", mock.Anything, feed).
		Return(nil).Once()

	// Start worker
	worker.Start()

	// Submit feed
	worker.Submit(feed)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Verify processor was called
	mockProcessor.AssertExpectations(t)
}

func TestFeedWorker_Submit_ProcessorError(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := DefaultFeedWorkerConfig()

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	feed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/feed.xml",
		Status:  models.FeedStatusActive,
	}

	// Mock processor to return error
	mockProcessor.On("ProcessFeed", mock.Anything, feed).
		Return(errors.New("failed to fetch feed")).Once()

	// Start worker
	worker.Start()

	// Submit feed
	worker.Submit(feed)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Verify processor was called despite error
	mockProcessor.AssertExpectations(t)
}

func TestFeedWorker_Submit_NilProcessor(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	config := DefaultFeedWorkerConfig()

	// Create worker with nil processor
	worker := NewFeedWorker(config, mockRepo, nil)

	feed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/feed.xml",
		Status:  models.FeedStatusActive,
	}

	// Start worker
	worker.Start()

	// Submit feed - should not panic
	assert.NotPanics(t, func() {
		worker.Submit(feed)
	})

	// Wait briefly
	time.Sleep(50 * time.Millisecond)

	// Stop worker
	worker.Stop()
}

func TestFeedWorker_Submit_QueueFull(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := &FeedWorkerConfig{
		WorkerCount: 1,
		QueueSize:   2,
	}

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	// Mock processor to block (simulate slow processing)
	mockProcessor.On("ProcessFeed", mock.Anything, mock.Anything).
		Return(nil).
		Run(func(args mock.Arguments) {
			time.Sleep(500 * time.Millisecond)
		}).Maybe()

	// Start worker
	worker.Start()

	// Fill queue to capacity
	for i := 0; i < config.QueueSize; i++ {
		feed := &models.Feed{
			ID:      uuid.New(),
			FeedURL: "https://example.com/feed.xml",
		}
		worker.Submit(feed)
	}

	// Try to submit one more - should not block or panic
	extraFeed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/extra.xml",
	}

	assert.NotPanics(t, func() {
		worker.Submit(extraFeed)
	})

	// Stop worker
	worker.Stop()
}

func TestFeedWorker_Submit_AfterStop(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := DefaultFeedWorkerConfig()

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	feed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/feed.xml",
	}

	// Start and immediately stop worker
	worker.Start()
	worker.Stop()

	// Try to submit after stop - should not panic
	assert.NotPanics(t, func() {
		worker.Submit(feed)
	})
}

func TestFeedWorker_QueueSize(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := DefaultFeedWorkerConfig()

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	// Initially queue should be empty
	assert.Equal(t, 0, worker.QueueSize())

	// Add some feeds to the queue directly
	feed1 := &models.Feed{ID: uuid.New(), FeedURL: "https://example.com/1"}
	feed2 := &models.Feed{ID: uuid.New(), FeedURL: "https://example.com/2"}

	worker.feedQueue <- feed1
	worker.feedQueue <- feed2

	// Queue size should be 2
	assert.Equal(t, 2, worker.QueueSize())

	// Drain queue
	<-worker.feedQueue
	<-worker.feedQueue

	// Queue should be empty again
	assert.Equal(t, 0, worker.QueueSize())
}

func TestFeedWorker_NewFeedWorker_WithNilConfig(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)

	worker := NewFeedWorker(nil, mockRepo, mockProcessor)

	assert.NotNil(t, worker)
	assert.NotNil(t, worker.config)
	assert.Equal(t, 5, worker.config.WorkerCount)
	assert.Equal(t, 100, worker.config.QueueSize)
}

func TestFeedWorker_NewFeedWorker_WithCustomConfig(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	customConfig := &FeedWorkerConfig{
		WorkerCount: 10,
		QueueSize:   200,
	}

	worker := NewFeedWorker(customConfig, mockRepo, mockProcessor)

	assert.NotNil(t, worker)
	assert.Equal(t, customConfig, worker.config)
	assert.Equal(t, 10, worker.config.WorkerCount)
	assert.Equal(t, 200, worker.config.QueueSize)
}

func TestDefaultFeedWorkerConfig(t *testing.T) {
	config := DefaultFeedWorkerConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 5, config.WorkerCount)
	assert.Equal(t, 100, config.QueueSize)
}

func TestFeedWorker_MultipleWorkers(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	mockProcessor := new(MockFeedProcessor)
	config := &FeedWorkerConfig{
		WorkerCount: 3,
		QueueSize:   10,
	}

	worker := NewFeedWorker(config, mockRepo, mockProcessor)

	// Create multiple feeds
	feeds := []*models.Feed{
		{ID: uuid.New(), FeedURL: "https://example.com/1"},
		{ID: uuid.New(), FeedURL: "https://example.com/2"},
		{ID: uuid.New(), FeedURL: "https://example.com/3"},
		{ID: uuid.New(), FeedURL: "https://example.com/4"},
		{ID: uuid.New(), FeedURL: "https://example.com/5"},
	}

	// Mock processor to succeed for all feeds
	mockProcessor.On("ProcessFeed", mock.Anything, mock.Anything).
		Return(nil).Times(len(feeds))

	// Start worker
	worker.Start()

	// Submit all feeds
	for _, feed := range feeds {
		worker.Submit(feed)
	}

	// Wait for all to process
	time.Sleep(200 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Verify all feeds were processed
	mockProcessor.AssertExpectations(t)
}

func TestNoOpFeedProcessor_ProcessFeed(t *testing.T) {
	processor := &NoOpFeedProcessor{}

	feed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/feed.xml",
	}

	// Should not return error
	err := processor.ProcessFeed(context.Background(), feed)
	assert.NoError(t, err)
}

func TestFeedWorker_ProcessFeed_Integration(t *testing.T) {
	mockRepo := new(MockFeedRepository)
	config := DefaultFeedWorkerConfig()

	// Use the NoOpFeedProcessor for integration testing
	processor := &NoOpFeedProcessor{}
	worker := NewFeedWorker(config, mockRepo, processor)

	feed := &models.Feed{
		ID:      uuid.New(),
		FeedURL: "https://example.com/feed.xml",
		Status:  models.FeedStatusActive,
	}

	// Start worker
	worker.Start()

	// Submit feed
	worker.Submit(feed)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	// Stop worker
	worker.Stop()

	// Should complete without errors
	assert.NotNil(t, worker)
}
