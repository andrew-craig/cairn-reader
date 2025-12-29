package jobs

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/andrew-craig/cairn/services/read/content/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
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

func (m *MockContentRepository) GetByContentHashesAndFeedID(ctx context.Context, contentHashes []string, feedID uuid.UUID) (map[string]*models.Content, error) {
	args := m.Called(ctx, contentHashes, feedID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]*models.Content), args.Error(1)
}

func (m *MockContentRepository) BulkCreate(ctx context.Context, contents []*models.Content) error {
	args := m.Called(ctx, contents)
	return args.Error(0)
}

func (m *MockContentRepository) DeleteOrphaned(ctx context.Context, olderThan time.Duration) (int64, error) {
	args := m.Called(ctx, olderThan)
	return args.Get(0).(int64), args.Error(1)
}

func TestCleanupJob_Run_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()
	job := NewCleanupJob(mockRepo, logger)

	// Expect deletion of content older than 90 days
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(5), nil)

	// Run the job
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestCleanupJob_Run_NoContentToDelete(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()
	job := NewCleanupJob(mockRepo, logger)

	// Expect deletion but return 0 rows deleted
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(0), nil)

	// Run the job
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestCleanupJob_Run_Error(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()
	job := NewCleanupJob(mockRepo, logger)

	// Mock repository to return an error
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(0), errors.New("database connection error"))

	// Run the job - should log error but not panic
	job.Run()

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestCleanupJob_RunWithBatching_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()
	job := NewCleanupJob(mockRepo, logger)

	// Mock multiple batch deletions
	// First batch: delete 1000 rows
	// Second batch: delete 500 rows
	// Third batch: delete 0 rows (done)
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(1000), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(500), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(0), nil).Once()

	// Run the job with batching
	job.RunWithBatching(1000)

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestCleanupJob_RunWithBatching_ErrorInMiddle(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()
	job := NewCleanupJob(mockRepo, logger)

	// First batch succeeds, second batch fails
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(500), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(0), errors.New("database lock timeout")).Once()

	// Run the job with batching
	job.RunWithBatching(1000)

	// Verify expectations
	mockRepo.AssertExpectations(t)
}

func TestCleanupJob_RunWithBatching_NoContentToDelete(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()
	job := NewCleanupJob(mockRepo, logger)

	// First call returns 0 rows deleted
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour).
		Return(int64(0), nil).Once()

	// Run the job with batching
	job.RunWithBatching(1000)

	// Verify expectations - should only be called once
	mockRepo.AssertExpectations(t)
	mockRepo.AssertNumberOfCalls(t, "DeleteOrphaned", 1)
}

func TestCleanupJob_NewCleanupJob(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger, _ := zap.NewDevelopment()

	job := NewCleanupJob(mockRepo, logger)

	assert.NotNil(t, job)
	assert.Equal(t, mockRepo, job.contentRepo)
	assert.Equal(t, logger, job.logger)
}
