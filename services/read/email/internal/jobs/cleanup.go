// Package jobs provides scheduled background jobs for the email ingest service.
package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/repository"
)

// defaultCleanupBatchSize bounds each DELETE transaction so a cleanup pass
// never holds a long lock on its target table.
const defaultCleanupBatchSize = 1000

// defaultRetentionDays is used when a caller passes a non-positive retention.
const defaultRetentionDays = 7

// deleteBatchFunc deletes up to batchSize rows older than olderThan and returns
// how many it deleted. Both repository.OutboxRepository.DeleteDelivered and
// repository.RawEmailRepository.DeleteProcessed satisfy it.
type deleteBatchFunc func(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)

// CleanupJob is the shared shape of the email service's daily retention jobs:
// on a daily schedule it calls deleteBatch repeatedly — each call bounded by
// batchSize — until a pass deletes nothing, pacing between passes and honouring
// context cancellation. The two concrete jobs (outbox delivery entries, raw
// emails) differ only in their name and which delete function they call.
type CleanupJob struct {
	name          string // used in log lines, e.g. "outbox cleanup"
	deleteBatch   deleteBatchFunc
	retentionDays int
	hour          int
	minute        int
	batchSize     int
}

// newCleanupJob builds a CleanupJob. cronExpr must be "M H * * *" (e.g.
// "0 6 * * *" for 6 AM); only the minute and hour fields are honoured.
func newCleanupJob(name string, deleteBatch deleteBatchFunc, cronExpr string, retentionDays int) (*CleanupJob, error) {
	hour, minute, err := parseDailyCron(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}
	if retentionDays <= 0 {
		retentionDays = defaultRetentionDays
	}
	return &CleanupJob{
		name:          name,
		deleteBatch:   deleteBatch,
		retentionDays: retentionDays,
		hour:          hour,
		minute:        minute,
		batchSize:     defaultCleanupBatchSize,
	}, nil
}

// Start runs the cleanup job on its daily schedule until ctx is cancelled.
func (j *CleanupJob) Start(ctx context.Context) {
	slog.Info(j.name+" job scheduled",
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
			slog.Info(j.name + " job stopped")
			return
		case <-timer.C:
			j.run(ctx)
		}
	}
}

func (j *CleanupJob) run(ctx context.Context) {
	retention := time.Duration(j.retentionDays) * 24 * time.Hour

	totalDeleted := int64(0)
	for {
		count, err := j.deleteBatch(ctx, retention, j.batchSize)
		if err != nil {
			slog.Error(j.name+" failed", slog.Any("error", err), slog.Int64("total_deleted_so_far", totalDeleted))
			return
		}

		totalDeleted += count
		if count == 0 {
			break
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(100 * time.Millisecond):
		}
	}

	slog.Info(j.name+" completed", slog.Int64("deleted", totalDeleted))
}

// NewOutboxCleanupJob returns a daily job that deletes delivered content_outbox
// entries older than retentionDays.
func NewOutboxCleanupJob(outboxRepo repository.OutboxRepository, cronExpr string, retentionDays int) (*CleanupJob, error) {
	return newCleanupJob("outbox cleanup", outboxRepo.DeleteDelivered, cronExpr, retentionDays)
}

// NewRawEmailCleanupJob returns a daily job that deletes processed raw_emails
// older than retentionDays.
func NewRawEmailCleanupJob(rawEmailRepo repository.RawEmailRepository, cronExpr string, retentionDays int) (*CleanupJob, error) {
	return newCleanupJob("raw email cleanup", rawEmailRepo.DeleteProcessed, cronExpr, retentionDays)
}

// parseDailyCron extracts hour and minute from a "M H * * *" cron expression.
func parseDailyCron(expr string) (hour, minute int, err error) {
	parts := strings.Fields(expr)
	if len(parts) != 5 {
		return 0, 0, fmt.Errorf("expected 5 fields, got %d", len(parts))
	}
	minute, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid minute %q", parts[0])
	}
	hour, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid hour %q", parts[1])
	}
	return hour, minute, nil
}

// nextDailyRun returns the next time the job should run at the given hour/minute (local time).
func nextDailyRun(hour, minute int) time.Time {
	now := time.Now()
	next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !next.After(now) {
		next = next.Add(24 * time.Hour)
	}
	return next
}
