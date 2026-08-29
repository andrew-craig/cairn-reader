// Package worker provides background workers for the email ingest service.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/processor"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/repository"
	"github.com/andrew-craig/cairn-reader/services/read/email/internal/service"
	"github.com/google/uuid"
)

// EmailProcessorConfig holds configuration for the EmailProcessorWorker.
type EmailProcessorConfig struct {
	BatchSize    int
	WorkerCount  int
	PollInterval time.Duration
}

// EmailProcessorWorker processes raw emails from the database.
type EmailProcessorWorker struct {
	senderService    service.SenderService
	emailCleaner     *processor.EmailCleaner
	contentExtractor *processor.ContentExtractor
	rawEmailRepo     repository.RawEmailRepository
	outboxRepo       repository.OutboxRepository

	batchSize    int
	workerCount  int
	pollInterval time.Duration
}

// NewEmailProcessorWorker creates a new EmailProcessorWorker.
func NewEmailProcessorWorker(
	senderService service.SenderService,
	emailCleaner *processor.EmailCleaner,
	contentExtractor *processor.ContentExtractor,
	rawEmailRepo repository.RawEmailRepository,
	outboxRepo repository.OutboxRepository,
	cfg EmailProcessorConfig,
) *EmailProcessorWorker {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 20
	}
	if cfg.WorkerCount == 0 {
		cfg.WorkerCount = 3
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 5 * time.Second
	}
	return &EmailProcessorWorker{
		senderService:    senderService,
		emailCleaner:     emailCleaner,
		contentExtractor: contentExtractor,
		rawEmailRepo:     rawEmailRepo,
		outboxRepo:       outboxRepo,
		batchSize:        cfg.BatchSize,
		workerCount:      cfg.WorkerCount,
		pollInterval:     cfg.PollInterval,
	}
}

// Start begins processing raw emails. It blocks until ctx is cancelled.
func (w *EmailProcessorWorker) Start(ctx context.Context) {
	slog.Info("email processor worker starting", slog.Int("workers", w.workerCount))

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("email processor worker stopped")
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *EmailProcessorWorker) processBatch(ctx context.Context) {
	emails, err := w.rawEmailRepo.GetPendingEmails(ctx, w.batchSize)
	if err != nil {
		slog.Error("failed to fetch pending emails", slog.Any("error", err))
		return
	}
	if len(emails) == 0 {
		return
	}

	slog.Info("processing email batch", slog.Int("count", len(emails)))

	sem := make(chan struct{}, w.workerCount)
	var wg sync.WaitGroup

	for i, email := range emails {
		if ctx.Err() != nil {
			// emails[i:] were already claimed into 'processing' by
			// GetPendingEmails but never dispatched -- release them so the
			// next instance picks them up immediately instead of waiting out
			// the claim lease.
			for _, e := range emails[i:] {
				if err := w.rawEmailRepo.ReleaseClaim(context.Background(), e.ID); err != nil {
					slog.Error("failed to release email claim on shutdown",
						slog.String("email_id", e.ID.String()),
						slog.Any("error", err),
					)
				}
			}
			break
		}
		sem <- struct{}{}
		wg.Add(1)
		go func(e *models.RawEmail) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := w.processEmail(ctx, e); err != nil {
				slog.Error("failed to process email",
					slog.String("email_id", e.ID.String()),
					slog.Any("error", err),
				)
			}
		}(email)
	}

	wg.Wait()
}

func (w *EmailProcessorWorker) processEmail(ctx context.Context, email *models.RawEmail) error {
	// email arrives already claimed into 'processing' by GetPendingEmails'
	// atomic claim, so no separate status transition is needed here.

	senderName := ""
	if email.SenderName != nil {
		senderName = *email.SenderName
	}
	sender, err := w.senderService.UpsertOnReceipt(ctx, email.UserID, email.SenderEmail, senderName, email.ReceivedAt)
	if err != nil {
		return w.handleError(ctx, email, fmt.Errorf("upsert sender: %w", err))
	}

	htmlBody := ""
	if email.HTMLBody != nil {
		htmlBody = *email.HTMLBody
	}
	cleanedHTML, err := w.emailCleaner.Clean(htmlBody)
	if err != nil {
		return w.handleError(ctx, email, fmt.Errorf("clean email: %w", err))
	}

	content, err := w.contentExtractor.Extract(cleanedHTML)
	if err != nil {
		return w.handleError(ctx, email, fmt.Errorf("extract content: %w", err))
	}

	subject := ""
	if email.Subject != nil {
		subject = *email.Subject
	}
	author := sender.SenderEmail
	if sender.SenderName != nil {
		author = *sender.SenderName
	}

	payload := map[string]interface{}{
		"url":          fmt.Sprintf("email://%s", email.ID.String()),
		"html":         content.SanitizedHTML,
		"title":        subject,
		"author":       author,
		"source_type":  "email",
		"published_at": email.ReceivedAt,
	}

	outbox := &models.ContentOutbox{
		ID:             uuid.New(),
		RawEmailID:     email.ID,
		ContentPayload: payload,
		UserID:         email.UserID,
		DeliveryStatus: models.DeliveryStatusPending,
		RetryCount:     0,
		MaxRetries:     6,
		NextRetryAt:    time.Now(),
	}
	if err := w.outboxRepo.Create(ctx, outbox); err != nil {
		return w.handleError(ctx, email, fmt.Errorf("create outbox entry: %w", err))
	}

	now := time.Now()
	if err := w.rawEmailRepo.UpdateStatus(ctx, email.ID, models.ProcessingStatusCompleted, &now); err != nil {
		slog.Warn("failed to mark email completed",
			slog.String("email_id", email.ID.String()),
			slog.Any("error", err),
		)
	}

	slog.Info("email processed",
		slog.String("email_id", email.ID.String()),
		slog.String("outbox_id", outbox.ID.String()),
	)
	return nil
}

func (w *EmailProcessorWorker) handleError(ctx context.Context, email *models.RawEmail, err error) error {
	newRetryCount := email.RetryCount + 1
	if updateErr := w.rawEmailRepo.UpdateError(ctx, email.ID, newRetryCount, err.Error()); updateErr != nil {
		slog.Error("failed to record email error",
			slog.String("email_id", email.ID.String()),
			slog.Any("error", updateErr),
		)
	}
	return err
}
