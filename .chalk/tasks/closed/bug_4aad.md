---
id: bug_4aad
title: Reader: link tapped while the previous link dialog is dismissing opens no dialog
type: bug
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-18T06:56:55Z
updated_at: 2026-11-18T07:22:39Z
---

Found in the on-device check of feature_4e37 (task 12). Tap a link, dismiss its dialog, then quickly tap another link: no dialog opens. The owner feels the tap haptic, and a tap a second later works again.
## Diagnosis
The haptic fires on `link` changing to a non-nil URL, so the second tap reaches `onLink` and sets `link`. The tap lands while the first dialog's popover is still animating away. There are two possible mechanisms, and nothing on Linux tells them apart:
- SwiftUI writes `false` to the `isPresented` binding when the dismissal finishes. The setter clears whatever `link` holds by then, so it wipes the new URL.
- Or the binding is already false. Setting `link` asks SwiftUI to present while UIKit is still dismissing the old popover, and UIKit drops the request ("Attempt to present … while a presentation is in progress").
Either way, the cause is a single SwiftUI presentation slot shared by consecutive taps, which can't wait for the previous dismissal to finish.
## Plan
- [ ] `ArticleWebView` presents the link dialog itself: a `UIAlertController` (`.actionSheet`, popover from the web view at the touch-down point), one per tap. Its actions come from the reader (`onLink(url) -> LinkDialog?`, nil for the title link, which opens directly). If the presenter is mid-transition, present in the transition coordinator's completion. Each alert has its own actions, so an old dismissal can't clear a new link.
- [ ] Medium impact haptic in the coordinator (replaces the `sensoryFeedback` on `link`).
- [ ] Remove `link`, `linkPoint`, `linkAnchor` and the `.confirmationDialog` from `ReaderScreen`.
- [ ] Update `apps/ios/CLAUDE.md` (Reader bullet).
- [ ] CI: iOS App Checks green. Owner on device: rapid link taps each open a dialog; the dialog still points at the tapped link; Save to Reads / Open in Browser / Copy Link still work.
## Resolution
Won't fix: a SwiftUI limitation. A binding-driven `.confirmationDialog` has one presentation slot. SwiftUI gives no signal when a dismissal finishes and can't queue a presentation behind one, so a tap during the roughly 0.3 s closing animation is lost. The impact is cosmetic: the haptic fires, no dialog appears, and the next tap works. The owner accepted it rather than replacing the dialog with a UIKit `UIAlertController` presented from the web view. That would be an extra UIKit special case, depend on finding the presenting view controller, and not be unit-testable. The plan above remains the fix if this becomes a problem in real use.
