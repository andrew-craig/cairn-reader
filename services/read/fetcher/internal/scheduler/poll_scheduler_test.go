package scheduler

import (
	"bytes"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/models"
	"github.com/andrew-craig/cairn-reader/services/read/fetcher/internal/repository"
	"github.com/stretchr/testify/assert"
)

// lockedBuffer is a bytes.Buffer safe for concurrent slog writes and test reads.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureSlog redirects the default slog logger to a buffer for the test.
func captureSlog(t *testing.T) *lockedBuffer {
	t.Helper()
	buf := &lockedBuffer{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return buf
}

// stubFeedRepo implements only GetFeedsDueForPolling; any other method panics
// via the nil embedded interface.
type stubFeedRepo struct {
	repository.FeedRepository
	getDue func() ([]*models.Feed, error)
}

func (r *stubFeedRepo) GetFeedsDueForPolling(context.Context, int) ([]*models.Feed, error) {
	return r.getDue()
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timed out waiting for %s", what)
	}
}

func runUntilStopped(s *PollScheduler) {
	s.wg.Add(1)
	go s.run()
}

func TestPollScheduler_IdleTickLogsHeartbeat(t *testing.T) {
	buf := captureSlog(t)
	ticks := make(chan struct{}, 16)
	s := NewPollScheduler(&PollSchedulerConfig{BatchSize: 1, PollInterval: time.Millisecond},
		&stubFeedRepo{getDue: func() ([]*models.Feed, error) {
			ticks <- struct{}{}
			return nil, nil
		}}, nil)

	runUntilStopped(s)
	waitFor(t, ticks, "first poll")
	waitFor(t, ticks, "second poll")
	s.Stop()

	assert.Contains(t, buf.String(), `msg="Poll scheduler tick" due=0 polled=0`)
}

func TestPollScheduler_RecoversFromPanicAndContinues(t *testing.T) {
	buf := captureSlog(t)
	calls := 0
	second := make(chan struct{})
	s := NewPollScheduler(&PollSchedulerConfig{BatchSize: 1, PollInterval: time.Millisecond},
		&stubFeedRepo{getDue: func() ([]*models.Feed, error) {
			calls++
			if calls == 1 {
				panic("boom in scheduler")
			}
			if calls == 2 {
				close(second)
			}
			return nil, nil
		}}, nil)

	runUntilStopped(s)
	waitFor(t, second, "poll after panic")
	s.Stop()

	out := buf.String()
	assert.Contains(t, out, "level=ERROR")
	assert.Contains(t, out, "Panic in poll scheduler tick")
	assert.Contains(t, out, "boom in scheduler")
	assert.Contains(t, out, "stack=")
}
