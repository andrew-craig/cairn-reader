package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/cairn-app/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockRawEmailRepo struct {
	deleteFunc func(ctx context.Context, olderThan time.Duration) (int64, error)
}

func (m *mockRawEmailRepo) Create(_ context.Context, _ *models.RawEmail) error { return nil }
func (m *mockRawEmailRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.RawEmail, error) {
	return nil, nil
}
func (m *mockRawEmailRepo) GetPendingEmails(_ context.Context, _ int) ([]*models.RawEmail, error) {
	return nil, nil
}
func (m *mockRawEmailRepo) UpdateStatus(_ context.Context, _ uuid.UUID, _ models.ProcessingStatus, _ *time.Time) error {
	return nil
}
func (m *mockRawEmailRepo) UpdateError(_ context.Context, _ uuid.UUID, _ int, _ string) error {
	return nil
}
func (m *mockRawEmailRepo) DeleteProcessed(ctx context.Context, olderThan time.Duration) (int64, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, olderThan)
	}
	return 0, nil
}

type mockOutboxRepo struct {
	deleteFunc func(ctx context.Context, olderThan time.Duration) (int64, error)
}

func (m *mockOutboxRepo) Create(_ context.Context, _ *models.ContentOutbox) error { return nil }
func (m *mockOutboxRepo) GetByID(_ context.Context, _ uuid.UUID) (*models.ContentOutbox, error) {
	return nil, nil
}
func (m *mockOutboxRepo) GetPendingEntries(_ context.Context, _ int) ([]*models.ContentOutbox, error) {
	return nil, nil
}
func (m *mockOutboxRepo) UpdateDeliveryStatus(_ context.Context, _ uuid.UUID, _ models.DeliveryStatus, _ *uuid.UUID, _ *time.Time) error {
	return nil
}
func (m *mockOutboxRepo) UpdateRetryInfo(_ context.Context, _ uuid.UUID, _ int, _ time.Time, _ string) error {
	return nil
}
func (m *mockOutboxRepo) DeleteDelivered(ctx context.Context, olderThan time.Duration) (int64, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, olderThan)
	}
	return 0, nil
}

// --- parseDailyCron ---

func TestParseDailyCron_Valid(t *testing.T) {
	cases := []struct {
		expr           string
		expectedHour   int
		expectedMinute int
	}{
		{"0 5 * * *", 5, 0},
		{"0 6 * * *", 6, 0},
		{"30 3 * * *", 3, 30},
		{"0 0 * * *", 0, 0},
	}

	for _, tc := range cases {
		hour, minute, err := parseDailyCron(tc.expr)
		require.NoError(t, err, "expr: %s", tc.expr)
		assert.Equal(t, tc.expectedHour, hour, "hour for %s", tc.expr)
		assert.Equal(t, tc.expectedMinute, minute, "minute for %s", tc.expr)
	}
}

func TestParseDailyCron_Invalid(t *testing.T) {
	cases := []string{
		"",
		"5 * * *",
		"not a cron",
		"abc 5 * * *",
		"0 abc * * *",
	}
	for _, expr := range cases {
		_, _, err := parseDailyCron(expr)
		assert.Error(t, err, "expected error for expr: %q", expr)
	}
}

// --- nextDailyRun ---

func TestNextDailyRun_Future(t *testing.T) {
	// Pick a time far in the future (23:59) — next run should be tomorrow
	next := nextDailyRun(23, 59)
	assert.True(t, next.After(time.Now()), "next run should be in the future")
}

func TestNextDailyRun_PastTime_ReturnsNextDay(t *testing.T) {
	// 00:01 — if current time is after midnight+1m, should be tomorrow
	next := nextDailyRun(0, 1)
	assert.True(t, next.After(time.Now()))
	// Should be within the next 25 hours
	assert.True(t, next.Before(time.Now().Add(25*time.Hour)))
}

// --- RawEmailCleanupJob ---

func TestNewRawEmailCleanupJob_InvalidCron(t *testing.T) {
	repo := &mockRawEmailRepo{}
	_, err := NewRawEmailCleanupJob(repo, "bad cron", 7)
	assert.Error(t, err)
}

func TestRawEmailCleanupJob_Run_UsesRetentionDuration(t *testing.T) {
	var capturedDuration time.Duration
	repo := &mockRawEmailRepo{
		deleteFunc: func(_ context.Context, olderThan time.Duration) (int64, error) {
			capturedDuration = olderThan
			return 5, nil
		},
	}

	job, err := NewRawEmailCleanupJob(repo, "0 5 * * *", 7)
	require.NoError(t, err)
	job.run(context.Background())

	assert.Equal(t, 7*24*time.Hour, capturedDuration)
}

// --- OutboxCleanupJob ---

func TestNewOutboxCleanupJob_InvalidCron(t *testing.T) {
	repo := &mockOutboxRepo{}
	_, err := NewOutboxCleanupJob(repo, "bad cron", 7)
	assert.Error(t, err)
}

func TestOutboxCleanupJob_Run_UsesRetentionDuration(t *testing.T) {
	var capturedDuration time.Duration
	repo := &mockOutboxRepo{
		deleteFunc: func(_ context.Context, olderThan time.Duration) (int64, error) {
			capturedDuration = olderThan
			return 3, nil
		},
	}

	job, err := NewOutboxCleanupJob(repo, "0 6 * * *", 14)
	require.NoError(t, err)
	job.run(context.Background())

	assert.Equal(t, 14*24*time.Hour, capturedDuration)
}

func TestRawEmailCleanupJob_Start_StopsOnContextCancel(t *testing.T) {
	repo := &mockRawEmailRepo{}
	job, err := NewRawEmailCleanupJob(repo, "0 5 * * *", 7)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		job.Start(ctx)
		close(done)
	}()

	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("cleanup job did not stop after context cancellation")
	}
}
