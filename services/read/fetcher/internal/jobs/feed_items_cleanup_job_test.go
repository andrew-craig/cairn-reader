package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrew-craig/cairn/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn/services/read/fetcher/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockFeedItemRepository is a mock implementation of repository.FeedItemRepository
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

func (m *MockFeedItemRepository) UpdateProcessingStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, contentHash *string, contentServiceID *uuid.UUID, processedAt *time.Time) error {
	args := m.Called(ctx, id, status, contentHash, contentServiceID, processedAt)
	return args.Error(0)
}

func (m *MockFeedItemRepository) UpdateContentUpdateInfo(ctx context.Context, id uuid.UUID, httpLastModified, httpETag *string, lastCheckedAt *time.Time) error {
	args := m.Called(ctx, id, httpLastModified, httpETag, lastCheckedAt)
	return args.Error(0)
}

func (m *MockFeedItemRepository) GetPendingItems(ctx context.Context, limit int) ([]*models.FeedItem, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.FeedItem), args.Error(1)
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

func (m *MockFeedItemRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID, lastError string) error {
	args := m.Called(ctx, id, lastError)
	return args.Error(0)
}

func (m *MockFeedItemRepository) DeleteOldCompletedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockFeedItemRepository) DeleteOldFailedItems(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockFeedItemRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockFeedItemRepository) GetMetrics(ctx context.Context) (*repository.FeedItemMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.FeedItemMetrics), args.Error(1)
}

func TestFeedItemsCleanupJob_Run_Success(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := DefaultFeedItemsCleanupJobConfig()
	job := NewFeedItemsCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    10,
		ProcessingCount: 5,
		CompletedCount:  200,
		FailedCount:     15,
		TotalCount:      230,
	}, nil).Once()

	// Mock deletion of old completed items (7 days)
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(150, nil).Once()
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock deletion of old failed items (30 days)
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(10, nil).Once()
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    10,
		ProcessingCount: 5,
		CompletedCount:  50,
		FailedCount:     5,
		TotalCount:      70,
	}, nil).Once()

	// Run the job
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestFeedItemsCleanupJob_Run_NoItemsToDelete(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := DefaultFeedItemsCleanupJobConfig()
	job := NewFeedItemsCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    5,
		ProcessingCount: 2,
		CompletedCount:  10,
		FailedCount:     1,
		TotalCount:      18,
	}, nil).Once()

	// Mock deletion returning 0 items deleted
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    5,
		ProcessingCount: 2,
		CompletedCount:  10,
		FailedCount:     1,
		TotalCount:      18,
	}, nil).Once()

	// Run the job
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestFeedItemsCleanupJob_Run_CompletedDeleteError(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := DefaultFeedItemsCleanupJobConfig()
	job := NewFeedItemsCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    0,
		ProcessingCount: 0,
		CompletedCount:  100,
		FailedCount:     50,
		TotalCount:      150,
	}, nil).Once()

	// Mock deletion error for completed items
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, errors.New("database lock timeout")).Once()

	// Mock deletion of old failed items (should still run even if completed deletion failed)
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(25, nil).Once()
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    0,
		ProcessingCount: 0,
		CompletedCount:  100,
		FailedCount:     25,
		TotalCount:      125,
	}, nil).Once()

	// Run the job - should handle error gracefully
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestFeedItemsCleanupJob_Run_FailedDeleteError(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := DefaultFeedItemsCleanupJobConfig()
	job := NewFeedItemsCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    0,
		ProcessingCount: 0,
		CompletedCount:  100,
		FailedCount:     50,
		TotalCount:      150,
	}, nil).Once()

	// Mock deletion of completed items succeeds
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(75, nil).Once()
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock deletion error for failed items
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, errors.New("database connection error")).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    0,
		ProcessingCount: 0,
		CompletedCount:  25,
		FailedCount:     50,
		TotalCount:      75,
	}, nil).Once()

	// Run the job - should handle error gracefully
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestFeedItemsCleanupJob_GetMetrics_Success(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := DefaultFeedItemsCleanupJobConfig()
	job := NewFeedItemsCleanupJob(config, mockRepo)

	expectedMetrics := &repository.FeedItemMetrics{
		PendingCount:    20,
		ProcessingCount: 10,
		CompletedCount:  500,
		FailedCount:     5,
		TotalCount:      535,
	}

	mockRepo.On("GetMetrics", mock.Anything).Return(expectedMetrics, nil).Once()

	metrics, err := job.GetMetrics()

	assert.NoError(t, err)
	assert.Equal(t, expectedMetrics, metrics)
	mockRepo.AssertExpectations(t)
}

func TestFeedItemsCleanupJob_RunWithCustomRetention(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := DefaultFeedItemsCleanupJobConfig()
	job := NewFeedItemsCleanupJob(config, mockRepo)

	customCompletedDays := 14
	customFailedDays := 60

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    0,
		ProcessingCount: 0,
		CompletedCount:  200,
		FailedCount:     50,
		TotalCount:      250,
	}, nil).Once()

	// Mock deletion with custom retention periods
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.MatchedBy(func(t time.Time) bool {
		expectedTime := time.Now().Add(-14 * 24 * time.Hour)
		diff := expectedTime.Sub(t)
		return diff > -1*time.Minute && diff < 1*time.Minute
	}), config.BatchSize).Return(100, nil).Once()
	mockRepo.On("DeleteOldCompletedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.MatchedBy(func(t time.Time) bool {
		expectedTime := time.Now().Add(-60 * 24 * time.Hour)
		diff := expectedTime.Sub(t)
		return diff > -1*time.Minute && diff < 1*time.Minute
	}), config.BatchSize).Return(25, nil).Once()
	mockRepo.On("DeleteOldFailedItems", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.FeedItemMetrics{
		PendingCount:    0,
		ProcessingCount: 0,
		CompletedCount:  100,
		FailedCount:     25,
		TotalCount:      125,
	}, nil).Once()

	// Run the job with custom retention
	job.RunWithCustomRetention(customCompletedDays, customFailedDays)

	// Verify expectations
	mockRepo.AssertExpectations(t)
	// Verify config was restored
	assert.Equal(t, 7, job.config.CompletedRetentionDays)
	assert.Equal(t, 30, job.config.FailedRetentionDays)
}

func TestFeedItemsCleanupJob_NewFeedItemsCleanupJob_WithNilConfig(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)

	job := NewFeedItemsCleanupJob(nil, mockRepo)

	assert.NotNil(t, job)
	assert.NotNil(t, job.config)
	assert.Equal(t, 7, job.config.CompletedRetentionDays)
	assert.Equal(t, 30, job.config.FailedRetentionDays)
	assert.Equal(t, 1000, job.config.BatchSize)
}

func TestFeedItemsCleanupJob_String(t *testing.T) {
	mockRepo := new(MockFeedItemRepository)
	config := &FeedItemsCleanupJobConfig{
		CompletedRetentionDays: 7,
		FailedRetentionDays:    30,
		BatchSize:              1000,
	}
	job := NewFeedItemsCleanupJob(config, mockRepo)

	expected := "FeedItemsCleanupJob(completed_retention=7d, failed_retention=30d, batch=1000)"
	assert.Equal(t, expected, job.String())
}

func TestDefaultFeedItemsCleanupJobConfig(t *testing.T) {
	config := DefaultFeedItemsCleanupJobConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 7, config.CompletedRetentionDays)
	assert.Equal(t, 30, config.FailedRetentionDays)
	assert.Equal(t, 1000, config.BatchSize)
}
