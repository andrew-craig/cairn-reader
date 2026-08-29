package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/email/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockRawEmailRepo struct {
	deleteFunc func(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)
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
func (m *mockRawEmailRepo) ReleaseClaim(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockRawEmailRepo) DeleteProcessed(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, olderThan, batchSize)
	}
	return 0, nil
}

type mockOutboxRepo struct {
	deleteFunc func(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error)
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
func (m *mockOutboxRepo) ReleaseClaim(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockOutboxRepo) DeleteDelivered(ctx context.Context, olderThan time.Duration, batchSize int) (int64, error) {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, olderThan, batchSize)
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
		deleteFunc: func(_ context.Context, olderThan time.Duration, _ int) (int64, error) {
			capturedDuration = olderThan
			return 0, nil
		},
	}

	job, err := NewRawEmailCleanupJob(repo, "0 5 * * *", 7)
	require.NoError(t, err)
	job.run(context.Background())

	assert.Equal(t, 7*24*time.Hour, capturedDuration)
}

// TestRawEmailCleanupJob_Run_BatchesUntilExhausted proves the job keeps
// calling DeleteProcessed with the batch size until a call reports zero
// deletions, instead of assuming one call clears the whole backlog.
func TestRawEmailCleanupJob_Run_BatchesUntilExhausted(t *testing.T) {
	var capturedBatchSizes []int
	calls := 0
	repo := &mockRawEmailRepo{
		deleteFunc: func(_ context.Context, _ time.Duration, batchSize int) (int64, error) {
			capturedBatchSizes = append(capturedBatchSizes, batchSize)
			calls++
			if calls < 3 {
				return int64(batchSize), nil
			}
			return 0, nil
		},
	}

	job, err := NewRawEmailCleanupJob(repo, "0 5 * * *", 7)
	require.NoError(t, err)
	job.run(context.Background())

	assert.Equal(t, 3, calls, "must keep batching until a call returns zero")
	for _, bs := range capturedBatchSizes {
		assert.Equal(t, defaultRawEmailCleanupBatchSize, bs, "every call must be bounded by the batch size")
	}
}

// TestRawEmailCleanupJob_Run_StopsOnContextCancel proves the inter-batch
// delay is context-aware: with a backlog that would otherwise take many
// more passes, cancelling ctx must stop the loop instead of sleeping
// through the shutdown signal.
func TestRawEmailCleanupJob_Run_StopsOnContextCancel(t *testing.T) {
	calls := 0
	repo := &mockRawEmailRepo{
		deleteFunc: func(_ context.Context, _ time.Duration, batchSize int) (int64, error) {
			calls++
			// Always report a full batch, so an unbounded loop would never exit on its own.
			return int64(batchSize), nil
		},
	}

	job, err := NewRawEmailCleanupJob(repo, "0 5 * * *", 7)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		job.run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop after context cancellation")
	}

	assert.Equal(t, 1, calls, "must stop after the in-flight batch instead of continuing to drain")
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
		deleteFunc: func(_ context.Context, olderThan time.Duration, _ int) (int64, error) {
			capturedDuration = olderThan
			return 0, nil
		},
	}

	job, err := NewOutboxCleanupJob(repo, "0 6 * * *", 14)
	require.NoError(t, err)
	job.run(context.Background())

	assert.Equal(t, 14*24*time.Hour, capturedDuration)
}

// TestOutboxCleanupJob_Run_BatchesUntilExhausted proves the job keeps
// calling DeleteDelivered with the batch size until a call reports zero
// deletions, instead of assuming one call clears the whole backlog.
func TestOutboxCleanupJob_Run_BatchesUntilExhausted(t *testing.T) {
	var capturedBatchSizes []int
	calls := 0
	repo := &mockOutboxRepo{
		deleteFunc: func(_ context.Context, _ time.Duration, batchSize int) (int64, error) {
			capturedBatchSizes = append(capturedBatchSizes, batchSize)
			calls++
			if calls < 3 {
				return int64(batchSize), nil
			}
			return 0, nil
		},
	}

	job, err := NewOutboxCleanupJob(repo, "0 6 * * *", 14)
	require.NoError(t, err)
	job.run(context.Background())

	assert.Equal(t, 3, calls, "must keep batching until a call returns zero")
	for _, bs := range capturedBatchSizes {
		assert.Equal(t, defaultOutboxCleanupBatchSize, bs, "every call must be bounded by the batch size")
	}
}

// TestOutboxCleanupJob_Run_StopsOnContextCancel proves the inter-batch
// delay is context-aware: with a backlog that would otherwise take many
// more passes, cancelling ctx must stop the loop instead of sleeping
// through the shutdown signal.
func TestOutboxCleanupJob_Run_StopsOnContextCancel(t *testing.T) {
	calls := 0
	repo := &mockOutboxRepo{
		deleteFunc: func(_ context.Context, _ time.Duration, batchSize int) (int64, error) {
			calls++
			// Always report a full batch, so an unbounded loop would never exit on its own.
			return int64(batchSize), nil
		},
	}

	job, err := NewOutboxCleanupJob(repo, "0 6 * * *", 14)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	go func() {
		job.run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("run did not stop after context cancellation")
	}

	assert.Equal(t, 1, calls, "must stop after the in-flight batch instead of continuing to drain")
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
