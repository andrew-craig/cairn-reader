package worker

import (
	"log/slog"
	"runtime/debug"
)

// runTick runs one poll iteration of a worker loop. A panic inside tick is
// recovered and logged so the loop survives to the next tick; otherwise a
// heartbeat is logged (including idle ticks) so a silent worker is detectable.
// tick returns the number of entries it picked up.
func runTick(name string, tick func() int) {
	defer recoverAndLog(name)
	entries := tick()
	slog.Info("worker heartbeat", slog.String("worker", name), slog.Int("entries", entries))
}

// recoverAndLog must be deferred directly (recover only works there). attrs
// are extra log fields, e.g. the ID of the entry being processed.
func recoverAndLog(name string, attrs ...any) {
	if r := recover(); r != nil {
		slog.Error("worker tick panicked", append([]any{
			slog.String("worker", name),
			slog.Any("panic", r),
			slog.String("stack", string(debug.Stack())),
		}, attrs...)...)
	}
}
