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

// mockAddressService is a test mock for AddressService.
type mockAddressService struct {
	resolveRecipientFunc func(ctx context.Context, recipient string) (uuid.UUID, error)
}

func (m *mockAddressService) GetOrCreate(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
	return nil, nil
}

func (m *mockAddressService) GetByUserID(_ context.Context, _ uuid.UUID) (*models.EmailAddress, error) {
	return nil, nil
}

func (m *mockAddressService) ResolveRecipient(ctx context.Context, recipient string) (uuid.UUID, error) {
	if m.resolveRecipientFunc != nil {
		return m.resolveRecipientFunc(ctx, recipient)
	}
	return uuid.Nil, ErrRecipientNotFound
}

// mockRawEmailRepository is a test mock for repository.RawEmailRepository.
type mockRawEmailRepository struct {
	createFunc func(ctx context.Context, email *models.RawEmail) error
}

func (m *mockRawEmailRepository) Create(ctx context.Context, email *models.RawEmail) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, email)
	}
	return nil
}

func (m *mockRawEmailRepository) GetByID(_ context.Context, _ uuid.UUID) (*models.RawEmail, error) {
	return nil, nil
}

func (m *mockRawEmailRepository) GetPendingEmails(_ context.Context, _ int) ([]*models.RawEmail, error) {
	return nil, nil
}

func (m *mockRawEmailRepository) UpdateStatus(_ context.Context, _ uuid.UUID, _ models.ProcessingStatus, _ *time.Time) error {
	return nil
}

func (m *mockRawEmailRepository) UpdateError(_ context.Context, _ uuid.UUID, _ int, _ string) error {
	return nil
}

func (m *mockRawEmailRepository) DeleteProcessed(_ context.Context, _ time.Duration, _ int) (int64, error) {
	return 0, nil
}

func TestEmailService_IngestEmail_Success(t *testing.T) {
	userID := uuid.New()
	recipient := "k7m2x9pq@read.cairnapp.com"
	senderEmail := "newsletter@example.com"
	senderName := "Example Newsletter"
	subject := "Weekly Update"
	htmlBody := "<p>Hello world</p>"
	textBody := "Hello world"
	receivedAt := time.Now().UTC()

	addressSvc := &mockAddressService{
		resolveRecipientFunc: func(_ context.Context, r string) (uuid.UUID, error) {
			assert.Equal(t, recipient, r)
			return userID, nil
		},
	}

	var storedEmail *models.RawEmail
	rawEmailRepo := &mockRawEmailRepository{
		createFunc: func(_ context.Context, email *models.RawEmail) error {
			storedEmail = email
			return nil
		},
	}

	svc := NewEmailService(addressSvc, rawEmailRepo)
	accepted, err := svc.IngestEmail(context.Background(), IngestEmailInput{
		Recipient:   recipient,
		SenderEmail: senderEmail,
		SenderName:  senderName,
		Subject:     subject,
		HTMLBody:    htmlBody,
		TextBody:    textBody,
		ReceivedAt:  receivedAt,
	})

	require.NoError(t, err)
	assert.True(t, accepted)
	require.NotNil(t, storedEmail)

	assert.Equal(t, userID, storedEmail.UserID)
	assert.Equal(t, recipient, storedEmail.Recipient)
	assert.Equal(t, senderEmail, storedEmail.SenderEmail)
	assert.Equal(t, &senderName, storedEmail.SenderName)
	assert.Equal(t, &subject, storedEmail.Subject)
	assert.Equal(t, &htmlBody, storedEmail.HTMLBody)
	assert.Equal(t, &textBody, storedEmail.TextBody)
	assert.Equal(t, receivedAt, storedEmail.ReceivedAt)
	assert.Equal(t, models.ProcessingStatusPending, storedEmail.ProcessingStatus)
}

func TestEmailService_IngestEmail_UnknownRecipient(t *testing.T) {
	addressSvc := &mockAddressService{
		resolveRecipientFunc: func(_ context.Context, _ string) (uuid.UUID, error) {
			return uuid.Nil, ErrRecipientNotFound
		},
	}

	rawEmailRepo := &mockRawEmailRepository{
		createFunc: func(_ context.Context, _ *models.RawEmail) error {
			t.Fatal("Create should not be called for unknown recipient")
			return nil
		},
	}

	svc := NewEmailService(addressSvc, rawEmailRepo)
	accepted, err := svc.IngestEmail(context.Background(), IngestEmailInput{
		Recipient:   "unknown@read.cairnapp.com",
		SenderEmail: "sender@example.com",
		ReceivedAt:  time.Now(),
	})

	require.NoError(t, err)
	assert.False(t, accepted)
}

func TestEmailService_IngestEmail_ResolveError(t *testing.T) {
	addressSvc := &mockAddressService{
		resolveRecipientFunc: func(_ context.Context, _ string) (uuid.UUID, error) {
			return uuid.Nil, fmt.Errorf("db connection refused")
		},
	}

	rawEmailRepo := &mockRawEmailRepository{}

	svc := NewEmailService(addressSvc, rawEmailRepo)
	accepted, err := svc.IngestEmail(context.Background(), IngestEmailInput{
		Recipient:   "test@read.cairnapp.com",
		SenderEmail: "sender@example.com",
		ReceivedAt:  time.Now(),
	})

	assert.Error(t, err)
	assert.False(t, accepted)
	assert.Contains(t, err.Error(), "failed to resolve recipient")
	assert.Contains(t, err.Error(), "db connection refused")
}

func TestEmailService_IngestEmail_CreateError(t *testing.T) {
	userID := uuid.New()

	addressSvc := &mockAddressService{
		resolveRecipientFunc: func(_ context.Context, _ string) (uuid.UUID, error) {
			return userID, nil
		},
	}

	rawEmailRepo := &mockRawEmailRepository{
		createFunc: func(_ context.Context, _ *models.RawEmail) error {
			return fmt.Errorf("disk full")
		},
	}

	svc := NewEmailService(addressSvc, rawEmailRepo)
	accepted, err := svc.IngestEmail(context.Background(), IngestEmailInput{
		Recipient:   "k7m2x9pq@read.cairnapp.com",
		SenderEmail: "sender@example.com",
		ReceivedAt:  time.Now(),
	})

	assert.Error(t, err)
	assert.False(t, accepted)
	assert.Contains(t, err.Error(), "failed to store raw email")
	assert.Contains(t, err.Error(), "disk full")
}

func TestEmailService_IngestEmail_OptionalFieldsOmitted(t *testing.T) {
	userID := uuid.New()

	addressSvc := &mockAddressService{
		resolveRecipientFunc: func(_ context.Context, _ string) (uuid.UUID, error) {
			return userID, nil
		},
	}

	var storedEmail *models.RawEmail
	rawEmailRepo := &mockRawEmailRepository{
		createFunc: func(_ context.Context, email *models.RawEmail) error {
			storedEmail = email
			return nil
		},
	}

	svc := NewEmailService(addressSvc, rawEmailRepo)
	accepted, err := svc.IngestEmail(context.Background(), IngestEmailInput{
		Recipient:   "k7m2x9pq@read.cairnapp.com",
		SenderEmail: "sender@example.com",
		ReceivedAt:  time.Now(),
	})

	require.NoError(t, err)
	assert.True(t, accepted)
	require.NotNil(t, storedEmail)

	assert.Nil(t, storedEmail.SenderName, "empty sender name should be nil")
	assert.Nil(t, storedEmail.Subject, "empty subject should be nil")
	assert.Nil(t, storedEmail.HTMLBody, "empty HTML body should be nil")
	assert.Nil(t, storedEmail.TextBody, "empty text body should be nil")
}
