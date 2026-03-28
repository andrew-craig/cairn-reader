package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/processor"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/service"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockSenderService struct {
	upsertFunc func(ctx context.Context, userID uuid.UUID, senderEmail, senderName string, receivedAt time.Time) (*models.EmailSender, error)
}

func (m *mockSenderService) UpsertOnReceipt(ctx context.Context, userID uuid.UUID, senderEmail, senderName string, receivedAt time.Time) (*models.EmailSender, error) {
	if m.upsertFunc != nil {
		return m.upsertFunc(ctx, userID, senderEmail, senderName, receivedAt)
	}
	name := senderName
	return &models.EmailSender{
		ID:          uuid.New(),
		UserID:      userID,
		SenderEmail: senderEmail,
		SenderName:  &name,
	}, nil
}

func (m *mockSenderService) ListByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*models.EmailSender, error) {
	return nil, nil
}

var _ service.SenderService = (*mockSenderService)(nil)

type mockRawEmailRepo struct {
	getPendingFunc  func(ctx context.Context, limit int) ([]*models.RawEmail, error)
	updateStatusFunc func(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, processedAt *time.Time) error
	updateErrorFunc func(ctx context.Context, id uuid.UUID, retryCount int, errorMsg string) error
}

func (m *mockRawEmailRepo) Create(ctx context.Context, email *models.RawEmail) error { return nil }
func (m *mockRawEmailRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.RawEmail, error) {
	return nil, nil
}
func (m *mockRawEmailRepo) GetPendingEmails(ctx context.Context, limit int) ([]*models.RawEmail, error) {
	if m.getPendingFunc != nil {
		return m.getPendingFunc(ctx, limit)
	}
	return nil, nil
}
func (m *mockRawEmailRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status models.ProcessingStatus, processedAt *time.Time) error {
	if m.updateStatusFunc != nil {
		return m.updateStatusFunc(ctx, id, status, processedAt)
	}
	return nil
}
func (m *mockRawEmailRepo) UpdateError(ctx context.Context, id uuid.UUID, retryCount int, errorMsg string) error {
	if m.updateErrorFunc != nil {
		return m.updateErrorFunc(ctx, id, retryCount, errorMsg)
	}
	return nil
}
func (m *mockRawEmailRepo) DeleteProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}

type mockOutboxRepo struct {
	createFunc func(ctx context.Context, outbox *models.ContentOutbox) error
}

func (m *mockOutboxRepo) Create(ctx context.Context, outbox *models.ContentOutbox) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, outbox)
	}
	return nil
}
func (m *mockOutboxRepo) GetByID(ctx context.Context, id uuid.UUID) (*models.ContentOutbox, error) {
	return nil, nil
}
func (m *mockOutboxRepo) GetPendingEntries(ctx context.Context, limit int) ([]*models.ContentOutbox, error) {
	return nil, nil
}
func (m *mockOutboxRepo) UpdateDeliveryStatus(ctx context.Context, id uuid.UUID, status models.DeliveryStatus, contentServiceID *uuid.UUID, deliveredAt *time.Time) error {
	return nil
}
func (m *mockOutboxRepo) UpdateRetryInfo(ctx context.Context, id uuid.UUID, retryCount int, nextRetryAt time.Time, lastError string) error {
	return nil
}
func (m *mockOutboxRepo) DeleteDelivered(ctx context.Context, olderThan time.Duration) (int64, error) {
	return 0, nil
}

// --- helpers ---

func makeWorker(rawRepo *mockRawEmailRepo, outRepo *mockOutboxRepo, sender *mockSenderService) *EmailProcessorWorker {
	return NewEmailProcessorWorker(
		sender,
		processor.NewEmailCleaner(),
		processor.NewContentExtractor(),
		rawRepo,
		outRepo,
		EmailProcessorConfig{BatchSize: 5, WorkerCount: 1, PollInterval: time.Second},
	)
}

func makeEmail() *models.RawEmail {
	subject := "Test Newsletter"
	html := "<html><body><p>Hello world</p></body></html>"
	name := "Sender Name"
	return &models.RawEmail{
		ID:               uuid.New(),
		UserID:           uuid.New(),
		SenderEmail:      "sender@example.com",
		SenderName:       &name,
		Subject:          &subject,
		HTMLBody:         &html,
		ReceivedAt:       time.Now(),
		ProcessingStatus: models.ProcessingStatusPending,
		RetryCount:       0,
	}
}

// --- tests ---

func TestEmailProcessorWorker_ProcessEmail_Success(t *testing.T) {
	var statusUpdates []models.ProcessingStatus
	var outboxCreated *models.ContentOutbox

	email := makeEmail()

	rawRepo := &mockRawEmailRepo{
		updateStatusFunc: func(_ context.Context, _ uuid.UUID, status models.ProcessingStatus, _ *time.Time) error {
			statusUpdates = append(statusUpdates, status)
			return nil
		},
	}
	outRepo := &mockOutboxRepo{
		createFunc: func(_ context.Context, outbox *models.ContentOutbox) error {
			outboxCreated = outbox
			return nil
		},
	}
	sender := &mockSenderService{}

	w := makeWorker(rawRepo, outRepo, sender)
	err := w.processEmail(context.Background(), email)
	require.NoError(t, err)

	// Should mark processing then completed
	require.Len(t, statusUpdates, 2)
	assert.Equal(t, models.ProcessingStatusProcessing, statusUpdates[0])
	assert.Equal(t, models.ProcessingStatusCompleted, statusUpdates[1])

	// Outbox entry should be created with expected fields
	require.NotNil(t, outboxCreated)
	assert.Equal(t, email.ID, outboxCreated.RawEmailID)
	assert.Equal(t, email.UserID, outboxCreated.UserID)
	assert.Equal(t, models.DeliveryStatusPending, outboxCreated.DeliveryStatus)

	url, _ := outboxCreated.ContentPayload["url"].(string)
	assert.Equal(t, "email://"+email.ID.String(), url)

	title, _ := outboxCreated.ContentPayload["title"].(string)
	assert.Equal(t, "Test Newsletter", title)

	sourceType, _ := outboxCreated.ContentPayload["source_type"].(string)
	assert.Equal(t, "email", sourceType)
}

func TestEmailProcessorWorker_ProcessEmail_SenderError(t *testing.T) {
	var errorRecorded bool
	email := makeEmail()

	rawRepo := &mockRawEmailRepo{
		updateErrorFunc: func(_ context.Context, _ uuid.UUID, _ int, _ string) error {
			errorRecorded = true
			return nil
		},
	}
	outRepo := &mockOutboxRepo{}
	sender := &mockSenderService{
		upsertFunc: func(_ context.Context, _ uuid.UUID, _, _ string, _ time.Time) (*models.EmailSender, error) {
			return nil, errors.New("sender upsert failed")
		},
	}

	w := makeWorker(rawRepo, outRepo, sender)
	err := w.processEmail(context.Background(), email)
	assert.Error(t, err)
	assert.True(t, errorRecorded)
}

func TestEmailProcessorWorker_ProcessEmail_OutboxCreateError(t *testing.T) {
	var errorRecorded bool
	email := makeEmail()

	rawRepo := &mockRawEmailRepo{
		updateErrorFunc: func(_ context.Context, _ uuid.UUID, _ int, _ string) error {
			errorRecorded = true
			return nil
		},
	}
	outRepo := &mockOutboxRepo{
		createFunc: func(_ context.Context, _ *models.ContentOutbox) error {
			return errors.New("db error")
		},
	}
	sender := &mockSenderService{}

	w := makeWorker(rawRepo, outRepo, sender)
	err := w.processEmail(context.Background(), email)
	assert.Error(t, err)
	assert.True(t, errorRecorded)
}

func TestEmailProcessorWorker_ProcessEmail_RetryCountIncremented(t *testing.T) {
	var recordedRetryCount int
	email := makeEmail()
	email.RetryCount = 2

	rawRepo := &mockRawEmailRepo{
		updateErrorFunc: func(_ context.Context, _ uuid.UUID, retryCount int, _ string) error {
			recordedRetryCount = retryCount
			return nil
		},
	}
	outRepo := &mockOutboxRepo{
		createFunc: func(_ context.Context, _ *models.ContentOutbox) error {
			return errors.New("fail")
		},
	}
	sender := &mockSenderService{}

	w := makeWorker(rawRepo, outRepo, sender)
	w.processEmail(context.Background(), email)
	assert.Equal(t, 3, recordedRetryCount)
}

func TestEmailProcessorWorker_ProcessEmail_NilHTMLBody(t *testing.T) {
	email := makeEmail()
	email.HTMLBody = nil

	rawRepo := &mockRawEmailRepo{}
	outRepo := &mockOutboxRepo{}
	sender := &mockSenderService{}

	w := makeWorker(rawRepo, outRepo, sender)
	err := w.processEmail(context.Background(), email)
	// Should succeed even with no HTML body
	assert.NoError(t, err)
}

func TestEmailProcessorWorker_Start_StopsOnContextCancel(t *testing.T) {
	rawRepo := &mockRawEmailRepo{}
	outRepo := &mockOutboxRepo{}
	sender := &mockSenderService{}

	w := NewEmailProcessorWorker(
		sender,
		processor.NewEmailCleaner(),
		processor.NewContentExtractor(),
		rawRepo,
		outRepo,
		EmailProcessorConfig{PollInterval: 100 * time.Millisecond},
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}
