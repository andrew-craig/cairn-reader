package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSenderRepository is a test mock for repository.SenderRepository.
type mockSenderRepository struct {
	createFunc              func(ctx context.Context, sender *models.EmailSender) error
	getByIDFunc             func(ctx context.Context, id uuid.UUID) (*models.EmailSender, error)
	getByUserAndEmailFunc   func(ctx context.Context, userID uuid.UUID, senderEmail string) (*models.EmailSender, error)
	upsertFunc              func(ctx context.Context, sender *models.EmailSender) error
	listByUserFunc          func(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error)
	incrementEmailCountFunc func(ctx context.Context, id uuid.UUID, receivedAt time.Time) error
}

func (m *mockSenderRepository) Create(ctx context.Context, sender *models.EmailSender) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, sender)
	}
	return nil
}

func (m *mockSenderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.EmailSender, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockSenderRepository) GetByUserAndEmail(ctx context.Context, userID uuid.UUID, senderEmail string) (*models.EmailSender, error) {
	if m.getByUserAndEmailFunc != nil {
		return m.getByUserAndEmailFunc(ctx, userID, senderEmail)
	}
	return nil, nil
}

func (m *mockSenderRepository) Upsert(ctx context.Context, sender *models.EmailSender) error {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, sender)
	}
	return nil
}

func (m *mockSenderRepository) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
	if m.listByUserFunc != nil {
		return m.listByUserFunc(ctx, userID, limit, offset)
	}
	return nil, nil
}

func (m *mockSenderRepository) IncrementEmailCount(ctx context.Context, id uuid.UUID, receivedAt time.Time) error {
	if m.incrementEmailCountFunc != nil {
		return m.incrementEmailCountFunc(ctx, id, receivedAt)
	}
	return nil
}

func TestSenderService_UpsertOnReceipt_Success(t *testing.T) {
	userID := uuid.New()
	senderEmail := "newsletter@example.com"
	senderName := "Example Newsletter"
	receivedAt := time.Now()

	repo := &mockSenderRepository{
		upsertFunc: func(_ context.Context, sender *models.EmailSender) error {
			assert.Equal(t, userID, sender.UserID)
			assert.Equal(t, senderEmail, sender.SenderEmail)
			assert.Equal(t, &senderName, sender.SenderName)
			assert.Equal(t, 1, sender.EmailCount)
			assert.Equal(t, &receivedAt, sender.LastReceivedAt)
			return nil
		},
	}

	svc := NewSenderService(repo)
	result, err := svc.UpsertOnReceipt(context.Background(), userID, senderEmail, senderName, receivedAt)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, userID, result.UserID)
	assert.Equal(t, senderEmail, result.SenderEmail)
	assert.Equal(t, &senderName, result.SenderName)
}

func TestSenderService_UpsertOnReceipt_EmptyName(t *testing.T) {
	userID := uuid.New()
	senderEmail := "no-name@example.com"
	receivedAt := time.Now()

	repo := &mockSenderRepository{
		upsertFunc: func(_ context.Context, sender *models.EmailSender) error {
			assert.Nil(t, sender.SenderName, "empty sender name should be stored as nil")
			return nil
		},
	}

	svc := NewSenderService(repo)
	result, err := svc.UpsertOnReceipt(context.Background(), userID, senderEmail, "", receivedAt)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Nil(t, result.SenderName)
}

func TestSenderService_UpsertOnReceipt_RepoError(t *testing.T) {
	repo := &mockSenderRepository{
		upsertFunc: func(_ context.Context, _ *models.EmailSender) error {
			return fmt.Errorf("connection refused")
		},
	}

	svc := NewSenderService(repo)
	result, err := svc.UpsertOnReceipt(context.Background(), uuid.New(), "test@example.com", "Test", time.Now())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to upsert sender")
	assert.Contains(t, err.Error(), "connection refused")
}

func TestSenderService_ListByUser_Success(t *testing.T) {
	userID := uuid.New()
	now := time.Now()
	name := "Sender One"
	expected := []*models.EmailSender{
		{
			ID:             uuid.New(),
			UserID:         userID,
			SenderEmail:    "one@example.com",
			SenderName:     &name,
			EmailCount:     5,
			LastReceivedAt: &now,
		},
		{
			ID:          uuid.New(),
			UserID:      userID,
			SenderEmail: "two@example.com",
			EmailCount:  2,
		},
	}

	repo := &mockSenderRepository{
		listByUserFunc: func(_ context.Context, uid uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
			assert.Equal(t, userID, uid)
			assert.Equal(t, 20, limit)
			assert.Equal(t, 0, offset)
			return expected, nil
		},
	}

	svc := NewSenderService(repo)
	result, err := svc.ListByUser(context.Background(), userID, 20, 0)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	assert.Len(t, result, 2)
}

func TestSenderService_ListByUser_Empty(t *testing.T) {
	repo := &mockSenderRepository{
		listByUserFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.EmailSender, error) {
			return nil, nil
		},
	}

	svc := NewSenderService(repo)
	result, err := svc.ListByUser(context.Background(), uuid.New(), 20, 0)

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestSenderService_ListByUser_RepoError(t *testing.T) {
	repo := &mockSenderRepository{
		listByUserFunc: func(_ context.Context, _ uuid.UUID, _, _ int) ([]*models.EmailSender, error) {
			return nil, fmt.Errorf("db timeout")
		},
	}

	svc := NewSenderService(repo)
	result, err := svc.ListByUser(context.Background(), uuid.New(), 20, 0)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to list senders")
	assert.Contains(t, err.Error(), "db timeout")
}
