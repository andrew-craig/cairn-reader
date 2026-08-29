package jobs

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
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

func (m *MockContentRepository) Update(ctx context.Context, content *models.Content) error {
	args := m.Called(ctx, content)
	return args.Error(0)
}

func (m *MockContentRepository) UpdateWithTx(ctx context.Context, tx *sql.Tx, content *models.Content) error {
	args := m.Called(ctx, tx, content)
	return args.Error(0)
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

func TestCleanupJob_Run_Success(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	job := NewCleanupJob(mockRepo, logger, 1000)

	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(5), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(0), nil).Once()

	job.Run()

	mockRepo.AssertExpectations(t)
}

func TestCleanupJob_Run_NoContentToDelete(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	job := NewCleanupJob(mockRepo, logger, 1000)

	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(0), nil).Once()

	job.Run()

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNumberOfCalls(t, "DeleteOrphaned", 1)
}

func TestCleanupJob_Run_Error(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	job := NewCleanupJob(mockRepo, logger, 1000)

	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(0), errors.New("database connection error")).Once()

	// Run the job - should log error but not panic
	job.Run()

	mockRepo.AssertExpectations(t)
}

// TestCleanupJob_Run_BatchesUntilExhausted proves the finding: the job must
// keep calling DeleteOrphaned with the configured batch size until a call
// reports zero deletions, rather than assuming one call clears everything.
func TestCleanupJob_Run_BatchesUntilExhausted(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	job := NewCleanupJob(mockRepo, logger, 1000)

	// First batch: delete 1000 rows (a full batch, so there may be more)
	// Second batch: delete 500 rows (a full batch, so there may be more)
	// Third batch: delete 0 rows (done)
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(1000), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(500), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(0), nil).Once()

	job.Run()

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNumberOfCalls(t, "DeleteOrphaned", 3)
}

func TestCleanupJob_Run_ErrorInMiddleOfBatching(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	job := NewCleanupJob(mockRepo, logger, 1000)

	// First batch succeeds, second batch fails
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(500), nil).Once()
	mockRepo.On("DeleteOrphaned", mock.Anything, 90*24*time.Hour, 1000).
		Return(int64(0), errors.New("database lock timeout")).Once()

	job.Run()

	mockRepo.AssertExpectations(t)
	mockRepo.AssertNumberOfCalls(t, "DeleteOrphaned", 2)
}

func TestCleanupJob_NewCleanupJob(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	job := NewCleanupJob(mockRepo, logger, 500)

	assert.NotNil(t, job)
	assert.Equal(t, mockRepo, job.contentRepo)
	assert.Equal(t, logger, job.logger)
	assert.Equal(t, 500, job.batchSize)
}

func TestCleanupJob_NewCleanupJob_DefaultsBatchSize(t *testing.T) {
	mockRepo := new(MockContentRepository)
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	job := NewCleanupJob(mockRepo, logger, 0)

	assert.Equal(t, defaultCleanupBatchSize, job.batchSize)
}
