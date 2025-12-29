package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-read/fetcher/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOutboxRepository is a mock implementation of repository.OutboxRepository
type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) Create(ctx context.Context, outbox *models.ContentOutbox) error {
	args := m.Called(ctx, outbox)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ContentOutbox), args.Error(1)
}

func (m *MockOutboxRepository) GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ContentOutbox), args.Error(1)
}

func (m *MockOutboxRepository) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time, lastError *string) error {
	args := m.Called(ctx, id, status, contentServiceID, deliveredAt, lastError)
	return args.Error(0)
}

func (m *MockOutboxRepository) IncrementRetryCount(ctx context.Context, id uuid.UUID, nextRetryAt time.Time, lastError string) error {
	args := m.Called(ctx, id, nextRetryAt, lastError)
	return args.Error(0)
}

func (m *MockOutboxRepository) DeleteOldDeliveredEntries(ctx context.Context, olderThan time.Time, batchSize int) (int, error) {
	args := m.Called(ctx, olderThan, batchSize)
	return args.Get(0).(int), args.Error(1)
}

func (m *MockOutboxRepository) GetFailedEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	args := m.Called(ctx, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ContentOutbox), args.Error(1)
}

func (m *MockOutboxRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockOutboxRepository) GetMetrics(ctx context.Context) (*repository.OutboxMetrics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.OutboxMetrics), args.Error(1)
}

func TestOutboxCleanupJob_Run_Success(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := DefaultOutboxCleanupJobConfig()
	job := NewOutboxCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   5,
		SendingCount:   2,
		DeliveredCount: 100,
		FailedCount:    3,
		TotalCount:     110,
	}, nil).Once()

	// Mock deletion of old delivered entries
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(50, nil).Once()
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock failed entries retrieval
	mockRepo.On("GetFailedEntries", mock.Anything, config.FailedEntryLogLimit).
		Return([]*models.ContentOutbox{}, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   5,
		SendingCount:   2,
		DeliveredCount: 50,
		FailedCount:    3,
		TotalCount:     60,
	}, nil).Once()

	// Run the job
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestOutboxCleanupJob_Run_WithFailedEntries(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := DefaultOutboxCleanupJobConfig()
	job := NewOutboxCleanupJob(config, mockRepo)

	feedItemID := uuid.New()
	errorMsg := "Content Service API error"
	failedEntry := &models.ContentOutbox{
		ID:             uuid.New(),
		FeedItemID:     feedItemID,
		DeliveryStatus: models.DeliveryStatusFailed,
		RetryCount:     6,
		UserIDs:        []uuid.UUID{uuid.New(), uuid.New()},
		LastError:      &errorMsg,
		CreatedAt:      time.Now().Add(-24 * time.Hour),
	}

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 0,
		FailedCount:    1,
		TotalCount:     1,
	}, nil).Once()

	// Mock deletion of old delivered entries
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock failed entries retrieval
	mockRepo.On("GetFailedEntries", mock.Anything, config.FailedEntryLogLimit).
		Return([]*models.ContentOutbox{failedEntry}, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 0,
		FailedCount:    1,
		TotalCount:     1,
	}, nil).Once()

	// Run the job
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestOutboxCleanupJob_Run_DeleteError(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := DefaultOutboxCleanupJobConfig()
	job := NewOutboxCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 100,
		FailedCount:    0,
		TotalCount:     100,
	}, nil).Once()

	// Mock deletion error
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, errors.New("database connection error")).Once()

	// Mock failed entries retrieval
	mockRepo.On("GetFailedEntries", mock.Anything, config.FailedEntryLogLimit).
		Return([]*models.ContentOutbox{}, nil).Once()

	// Mock metrics after cleanup (should still be called)
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 100,
		FailedCount:    0,
		TotalCount:     100,
	}, nil).Once()

	// Run the job - should handle error gracefully
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestOutboxCleanupJob_Run_GetFailedEntriesError(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := DefaultOutboxCleanupJobConfig()
	job := NewOutboxCleanupJob(config, mockRepo)

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 0,
		FailedCount:    5,
		TotalCount:     5,
	}, nil).Once()

	// Mock deletion of old delivered entries
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock failed entries retrieval error
	mockRepo.On("GetFailedEntries", mock.Anything, config.FailedEntryLogLimit).
		Return(nil, errors.New("database error")).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 0,
		FailedCount:    5,
		TotalCount:     5,
	}, nil).Once()

	// Run the job - should handle error gracefully
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestOutboxCleanupJob_GetMetrics_Success(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := DefaultOutboxCleanupJobConfig()
	job := NewOutboxCleanupJob(config, mockRepo)

	expectedMetrics := &repository.OutboxMetrics{
		PendingCount:   10,
		SendingCount:   5,
		DeliveredCount: 200,
		FailedCount:    2,
		TotalCount:     217,
	}

	mockRepo.On("GetMetrics", mock.Anything).Return(expectedMetrics, nil).Once()

	metrics, err := job.GetMetrics()

	assert.NoError(t, err)
	assert.Equal(t, expectedMetrics, metrics)
	mockRepo.AssertExpectations(t)
}

func TestOutboxCleanupJob_RunWithCustomRetention(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := DefaultOutboxCleanupJobConfig()
	job := NewOutboxCleanupJob(config, mockRepo)

	customRetention := 14 // 14 days

	// Mock metrics before cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 100,
		FailedCount:    0,
		TotalCount:     100,
	}, nil).Once()

	// Mock deletion - should use custom retention period
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.MatchedBy(func(t time.Time) bool {
		// Verify that the time is approximately 14 days ago
		expectedTime := time.Now().Add(-14 * 24 * time.Hour)
		diff := expectedTime.Sub(t)
		return diff > -1*time.Minute && diff < 1*time.Minute
	}), config.BatchSize).Return(75, nil).Once()
	mockRepo.On("DeleteOldDeliveredEntries", mock.Anything, mock.AnythingOfType("time.Time"), config.BatchSize).
		Return(0, nil).Once()

	// Mock failed entries retrieval
	mockRepo.On("GetFailedEntries", mock.Anything, config.FailedEntryLogLimit).
		Return([]*models.ContentOutbox{}, nil).Once()

	// Mock metrics after cleanup
	mockRepo.On("GetMetrics", mock.Anything).Return(&repository.OutboxMetrics{
		PendingCount:   0,
		SendingCount:   0,
		DeliveredCount: 25,
		FailedCount:    0,
		TotalCount:     25,
	}, nil).Once()

	// Run the job with custom retention
	job.RunWithCustomRetention(customRetention)

	// Verify expectations
	mockRepo.AssertExpectations(t)
	// Verify config was restored
	assert.Equal(t, 7, job.config.RetentionDays)
}

func TestOutboxCleanupJob_NewOutboxCleanupJob_WithNilConfig(t *testing.T) {
	mockRepo := new(MockOutboxRepository)

	job := NewOutboxCleanupJob(nil, mockRepo)

	assert.NotNil(t, job)
	assert.NotNil(t, job.config)
	assert.Equal(t, 7, job.config.RetentionDays)
	assert.Equal(t, 1000, job.config.BatchSize)
	assert.Equal(t, 100, job.config.FailedEntryLogLimit)
}

func TestOutboxCleanupJob_String(t *testing.T) {
	mockRepo := new(MockOutboxRepository)
	config := &OutboxCleanupJobConfig{
		RetentionDays:       7,
		BatchSize:           1000,
		FailedEntryLogLimit: 100,
	}
	job := NewOutboxCleanupJob(config, mockRepo)

	expected := "OutboxCleanupJob(retention=7d, batch=1000)"
	assert.Equal(t, expected, job.String())
}

func TestDefaultOutboxCleanupJobConfig(t *testing.T) {
	config := DefaultOutboxCleanupJobConfig()

	assert.NotNil(t, config)
	assert.Equal(t, 7, config.RetentionDays)
	assert.Equal(t, 1000, config.BatchSize)
	assert.Equal(t, 100, config.FailedEntryLogLimit)
}
