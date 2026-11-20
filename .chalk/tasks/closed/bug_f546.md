---
id: bug_f546
title: [iOS native] Reader: scroll position not restored, and a partly read article shows as read
type: bug
status: closed
priority: 1
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-15T22:06:46Z
updated_at: 2026-11-16T07:20:45Z
---
## Feedback (owner, TestFlight testing of task 12)
Scroll position persistence doesn't seem to work. Opening an article and scrolling part way leaves the list row styled as fully read.
## What "fully read" means in code
`ArticleRow`: regular weight + `textSecondary` only when `isRead` (`status == .completed`). A partly read article is bold + `textSecondary`. So the reader most likely queued `markCompleted` (`ReaderModel.scrolled`, `completedThreshold` 0.95). Once an article is completed, the restore is pointless, so both symptoms may share one cause. The owner confirmed (2026-11-15) the article was in Reads, so progress tracking was on.
## Leading hypothesis (not reproduced; needs a device or simulator)
`ArticleWebView` reports scroll readings before the HTML has laid out. At that point `scrollView.contentSize.height` is about the frame height, so `(offset + viewport) / contentHeight` is about 1. For an article with no saved position (`restoreFraction == 0`), the restore counts as done at once, so `isRestoring` is false and the first `scrollViewDidScroll` (inset adjustment at load) marks the article completed and saves fraction 0. Check also:
- `adjustedContentInset` under the iOS 26 bars versus the offset/viewport sums (`ArticleWebView.swift` ~L125).
- whether `leave()`/throttled saves actually reach the store (`savedFraction` skip; `isFinished`).
- `isScrollTargetReachable` against late image loads for the restore half.
## Plan
- [x] Reproduce in the simulator with logging of every `Scroll` reading, `restoreFraction`, and completion/save calls. *Not possible here (Linux agent, no Swift toolchain or simulator); the cause was traced from the code instead, see Review.*
- [x] Only accept readings (for completion and saving) once the page has finished loading (`didFinish`) and the user has scrolled (drag began). Base the fix on the root cause you find.
- [x] Unit tests in `ReaderModelTests`: a pre-layout reading with contentHeight ≈ viewport doesn't complete or save; a part scroll saves its fraction, which reopening restores.
## Verify
- Owner on device: scroll a long Reads article part way, leave, and the row is bold muted (started). Reopen, and after a relaunch, it comes back to the same place. Scroll to the end and it's marked read.
## Review
**Root cause (from the code; not reproduced on a simulator).** The `hasSettled` gate in `ArticleWebView` treated `didFinish` as "the layout is final". `WKWebView` passes the web process's layout to `scrollView.contentSize` asynchronously, so at `didFinish` the content height can still be about the viewport's. `didFinish` then did two things wrong:
- It ended the restore (`hasRestored = true`) against that pre-layout size, where the clamped target is offset 0. When the real layout arrived, nothing moved to the saved position, so it was not restored.
- It set `hasSettled` and reported that pre-layout reading as confirmed: `(0 + viewport) / contentHeight ≈ 1 ≥ 0.95`, so `ReaderModel.scrolled` queued `markCompleted` (the row shows as read) and saved fraction 0 over the saved position. Later inset changes (`scrollViewDidScroll` with no drag) were confirmed too.

`adjustedContentInset` handling is consistent: offset + viewport equals the content height at the bottom, and the restore subtracts `insets.top`. `leave()` and the throttled saves do reach `ArticleMutations.saveScrollPosition`. They were skipped only when the fraction equalled `savedFraction`, which hid nothing once the bad 0 had been saved.

**Change.**
- `ReaderModel.Scroll` drops `isRestoring` for `hasLoaded` (`didFinish`) and `hasDragged`. `scrolled` counts a reading only when both are true, so completion, saves and `leave()` see only positions the user scrolled to. "Do not persist the pre-restore offset" still holds: before a drag, nothing is confirmed.
- `ArticleWebView`: `didFinish` only sets `hasLoaded` and reports. The restore moves to `fraction × contentHeight` on every content growth until the user drags, not until `isScrollTargetReachable` or `didFinish`. The fraction was measured on the finished layout, which can land after `didFinish` or after a first text-only layout that could already reach the target (late images). `isScrollTargetReachable` and its tests are removed because nothing uses them now.
- Tests: `aPreLayoutReadingNeitherCompletesNorSaves` and `aPartScrollIsSavedAndReopeningRestoresIt`. `thePreRestoreOffsetIsNeverSaved` now uses `hasDragged: false`. `apps/ios/CLAUDE.md` Reader bullet updated.

**Trade-offs, to check on device.** A reading counts only after a touch drag, so:
- A short article that fits on one screen completes only after a drag. The RN app completes it unconditionally; on iOS the page rubber-bands, so any small drag will do.
- Scrolling with no drag (a hardware keyboard on iPad, VoiceOver, a tap on the status bar) is not saved or completed.
