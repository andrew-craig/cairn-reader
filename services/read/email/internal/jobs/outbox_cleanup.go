// Package jobs provides scheduled background jobs for the email ingest service.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/repository"
)

// OutboxCleanupJob periodically cleans up delivered outbox entries.
type OutboxCleanupJob struct {
	outboxRepo    repository.OutboxRepository
	retentionDays int
	hour          int
	minute        int
}

// NewOutboxCleanupJob creates a new OutboxCleanupJob.
// cronExpr should be in "M H * * *" format (e.g. "0 6 * * *" for 6 AM).
func NewOutboxCleanupJob(outboxRepo repository.OutboxRepository, cronExpr string, retentionDays int) (*OutboxCleanupJob, error) {
	hour, minute, err := parseDailyCron(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}
	if retentionDays <= 0 {
		retentionDays = 7
	}
	return &OutboxCleanupJob{
		outboxRepo:    outboxRepo,
		retentionDays: retentionDays,
		hour:          hour,
		minute:        minute,
	}, nil
}

// Start runs the cleanup job on its daily schedule until ctx is cancelled.
func (j *OutboxCleanupJob) Start(ctx context.Context) {
	slog.Info("outbox cleanup job scheduled",
		slog.Int("hour", j.hour),
		slog.Int("minute", j.minute),
		slog.Int("retention_days", j.retentionDays),
	)

	for {
		next := nextDailyRun(j.hour, j.minute)
		timer := time.NewTimer(time.Until(next))

		select {
		case <-ctx.Done():
			timer.Stop()
			slog.Info("outbox cleanup job stopped")
			return
		case <-timer.C:
			j.run(ctx)
		}
	}
}

func (j *OutboxCleanupJob) run(ctx context.Context) {
	retention := time.Duration(j.retentionDays) * 24 * time.Hour
	count, err := j.outboxRepo.DeleteDelivered(ctx, retention)
	if err != nil {
		slog.Error("outbox cleanup failed", slog.Any("error", err))
		return
	}
	slog.Info("outbox cleanup completed", slog.Int64("deleted", count))
}
