// Package worker provides background workers for the email ingest service.
package worker

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/client"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
)

// OutboxWorkerConfig holds configuration for the OutboxWorker.
type OutboxWorkerConfig struct {
	BatchSize    int
	PollInterval time.Duration
	MaxRetries   int
}

// OutboxWorker delivers processed content to the Content Service.
type OutboxWorker struct {
	outboxRepo    repository.OutboxRepository
	contentClient *client.ContentServiceClient

	batchSize    int
	pollInterval time.Duration
}

// NewOutboxWorker creates a new OutboxWorker.
func NewOutboxWorker(
	outboxRepo repository.OutboxRepository,
	contentClient *client.ContentServiceClient,
	cfg OutboxWorkerConfig,
) *OutboxWorker {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 10
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 10 * time.Second
	}
	return &OutboxWorker{
		outboxRepo:    outboxRepo,
		contentClient: contentClient,
		batchSize:     cfg.BatchSize,
		pollInterval:  cfg.PollInterval,
	}
}

// Start begins delivering outbox entries. It blocks until ctx is cancelled.
func (w *OutboxWorker) Start(ctx context.Context) {
	slog.Info("outbox worker starting")

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("outbox worker stopped")
			return
		case <-ticker.C:
			w.deliverBatch(ctx)
		}
	}
}

func (w *OutboxWorker) deliverBatch(ctx context.Context) {
	entries, err := w.outboxRepo.GetPendingEntries(ctx, w.batchSize)
	if err != nil {
		slog.Error("failed to fetch pending outbox entries", slog.Any("error", err))
		return
	}
	if len(entries) == 0 {
		return
	}

	slog.Info("delivering outbox batch", slog.Int("count", len(entries)))

	for i, entry := range entries {
		if ctx.Err() != nil {
			// entries[i:] were already claimed into 'sending' by
			// GetPendingEntries but never delivered -- release them so the
			// next instance picks them up immediately instead of waiting out
			// the claim lease.
			for _, e := range entries[i:] {
				if err := w.outboxRepo.ReleaseClaim(context.Background(), e.ID); err != nil {
					slog.Error("failed to release outbox claim on shutdown",
						slog.String("outbox_id", e.ID.String()),
						slog.Any("error", err),
					)
				}
			}
			return
		}
		w.deliverEntry(ctx, entry)
	}
}

func (w *OutboxWorker) deliverEntry(ctx context.Context, entry *models.ContentOutbox) {
	item, err := outboxToContentItem(entry)
	if err != nil {
		slog.Error("failed to build content item",
			slog.String("outbox_id", entry.ID.String()),
			slog.Any("error", err),
		)
		w.recordFailure(ctx, entry, err)
		return
	}

	contentServiceID, err := w.contentClient.DeliverContent(ctx, []client.EmailContentItem{item})
	if err != nil {
		slog.Error("failed to deliver content",
			slog.String("outbox_id", entry.ID.String()),
			slog.Any("error", err),
		)
		w.recordFailure(ctx, entry, err)
		return
	}

	now := time.Now()
	if updateErr := w.outboxRepo.UpdateDeliveryStatus(ctx, entry.ID, models.DeliveryStatusDelivered, &contentServiceID, &now); updateErr != nil {
		slog.Error("failed to mark outbox entry delivered",
			slog.String("outbox_id", entry.ID.String()),
			slog.Any("error", updateErr),
		)
	}

	slog.Info("outbox entry delivered",
		slog.String("outbox_id", entry.ID.String()),
		slog.String("content_service_id", contentServiceID.String()),
	)
}

func (w *OutboxWorker) recordFailure(ctx context.Context, entry *models.ContentOutbox, err error) {
	newRetryCount := entry.RetryCount + 1
	nextRetryAt := time.Now().Add(outboxBackoff(newRetryCount))
	if updateErr := w.outboxRepo.UpdateRetryInfo(ctx, entry.ID, newRetryCount, nextRetryAt, err.Error()); updateErr != nil {
		slog.Error("failed to update outbox retry info",
			slog.String("outbox_id", entry.ID.String()),
			slog.Any("error", updateErr),
		)
	}
}

// outboxBackoff returns exponential backoff duration for the given retry count.
// Produces: 1m, 2m, 4m, 8m, 16m, 32m, ...
func outboxBackoff(retryCount int) time.Duration {
	exp := math.Pow(2, float64(retryCount-1))
	return time.Duration(float64(time.Minute) * exp)
}

// outboxToContentItem converts a ContentOutbox entry to a client.EmailContentItem.
func outboxToContentItem(entry *models.ContentOutbox) (client.EmailContentItem, error) {
	p := entry.ContentPayload

	url, _ := p["url"].(string)
	if url == "" {
		return client.EmailContentItem{}, fmt.Errorf("missing url in content payload for outbox entry %s", entry.ID)
	}

	html, _ := p["html"].(string)
	title, _ := p["title"].(string)
	author, _ := p["author"].(string)
	sourceType, _ := p["source_type"].(string)

	return client.EmailContentItem{
		UserID:     entry.UserID,
		URL:        url,
		Type:       "email",
		HTML:       html,
		Title:      title,
		Author:     author,
		SourceType: sourceType,
	}, nil
}
