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

// defaultRawEmailCleanupBatchSize bounds each DELETE transaction so cleanup
// never holds a long lock on the raw_emails table.
const defaultRawEmailCleanupBatchSize = 1000

// RawEmailCleanupJob periodically cleans up processed raw emails.
type RawEmailCleanupJob struct {
	rawEmailRepo  repository.RawEmailRepository
	retentionDays int
	hour          int
	minute        int
	batchSize     int
}

// NewRawEmailCleanupJob creates a new RawEmailCleanupJob.
// cronExpr should be in "M H * * *" format (e.g. "0 5 * * *" for 5 AM).
func NewRawEmailCleanupJob(rawEmailRepo repository.RawEmailRepository, cronExpr string, retentionDays int) (*RawEmailCleanupJob, error) {
	hour, minute, err := parseDailyCron(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression %q: %w", cronExpr, err)
	}
	if retentionDays <= 0 {
		retentionDays = 7
	}
	return &RawEmailCleanupJob{
		rawEmailRepo:  rawEmailRepo,
		retentionDays: retentionDays,
		hour:          hour,
		minute:        minute,
		batchSize:     defaultRawEmailCleanupBatchSize,
	}, nil
}

// Start runs the cleanup job on its daily schedule until ctx is cancelled.
func (j *RawEmailCleanupJob) Start(ctx context.Context) {
	slog.Info("raw email cleanup job scheduled",
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
			slog.Info("raw email cleanup job stopped")
			return
		case <-timer.C:
			j.run(ctx)
		}
	}
}

func (j *RawEmailCleanupJob) run(ctx context.Context) {
	retention := time.Duration(j.retentionDays) * 24 * time.Hour

	totalDeleted := int64(0)
	for {
		count, err := j.rawEmailRepo.DeleteProcessed(ctx, retention, j.batchSize)
		if err != nil {
			slog.Error("raw email cleanup failed", slog.Any("error", err), slog.Int64("total_deleted_so_far", totalDeleted))
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

	slog.Info("raw email cleanup completed", slog.Int64("deleted", totalDeleted))
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
