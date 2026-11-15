---
id: bug_f546
title: [iOS native] Reader: scroll position not restored, and a partly read article shows as read
type: bug
status: open
priority: 1
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T22:06:46Z
updated_at: 2026-11-15T22:06:46Z
---
## Feedback (owner, TestFlight testing of task 12)
Scroll position persistence doesn't seem to work. Opening an article and scrolling part way leaves the list row styled as fully read.
## What "fully read" means in code
`ArticleRow`: regular weight + `textSecondary` only when `isRead` (`status == .completed`). A partly read article is bold + `textSecondary`. So the reader most likely queued `markCompleted` (`ReaderModel.scrolled`, `completedThreshold` 0.95). Once an article is completed, the restore is pointless, so both symptoms may share one cause. Confirm with the owner that the row was in Reads: Feed items don't track progress (`tracksProgress`).
## Leading hypothesis (not reproduced; needs a device or simulator)
`ArticleWebView` reports scroll readings before the HTML has laid out. At that point `scrollView.contentSize.height` is about the frame height, so `(offset + viewport) / contentHeight` is about 1. For an article with no saved position (`restoreFraction == 0`), the restore counts as done at once, so `isRestoring` is false and the first `scrollViewDidScroll` (inset adjustment at load) marks the article completed and saves fraction 0. Check also:
- `adjustedContentInset` under the iOS 26 bars versus the offset/viewport sums (`ArticleWebView.swift` ~L125).
- whether `leave()`/throttled saves actually reach the store (`savedFraction` skip; `isFinished`).
- `isScrollTargetReachable` against late image loads for the restore half.
## Plan
- [ ] Reproduce in the simulator with logging of every `Scroll` reading, `restoreFraction`, and completion/save calls.
- [ ] Only accept readings (for completion and saving) once the page has finished loading (`didFinish`) and the user has scrolled (drag began). Base the fix on the root cause you find.
- [ ] Unit tests in `ReaderModelTests`: a pre-layout reading with contentHeight ≈ viewport doesn't complete or save; a part scroll saves its fraction, which reopening restores.
## Verify
- Owner on device: scroll a long Reads article part way, leave, and the row is bold muted (started). Reopen, and after a relaunch, it comes back to the same place. Scroll to the end and it's marked read.
