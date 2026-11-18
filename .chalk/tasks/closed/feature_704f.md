---
id: feature_704f
title: [iOS native 14] Add link / subscribe sheet
type: feature
status: closed
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-18T11:58:36Z
---
## Goal
Save a URL to Reads or subscribe to a feed, from the Reads toolbar and from the RSS screen.

## Source
`apps/mobile/src/components/AddLinkModal.tsx` (+ test), `screens/AddArticleScreen.tsx`, `components/common/{ListChoice,HeaderPopover}.tsx`.

## Plan
- [x] One `.sheet` with a `Form`: URL field (`.keyboardType(.URL)`, no autocapitalisation, paste button), debounced `detectURL` with cancellation of superseded runs, result summary (article vs feed), "Find feed" via `discoverFeed` for non-feed pages.
- [x] When the target is a feed: a Feed/Reads `Picker` (inline style) whose section footer carries the explanation text from `ListChoice`. Default Reads.
- [x] `mode: .link | .feed`. In `.feed` mode a non-feed URL can only go through Find feed and is never saved as an article.
- [x] Submit through `addURL` / `addContentToUser`; on success dismiss and refresh the presenting list. Online-only: disabled with a note when offline.
- [x] `AddArticleScreen` is registered in the RN navigator but check whether anything still navigates to it; if not, do not port it and list it as dead code in the PR.

## Verify
- View-model tests ported from `AddLinkModal.test.tsx`: detection debounce and cancellation, feed mode restriction, list choice sent on subscribe, error states.

## Review
Implemented on `worktree-agent-a5895369805b30739` (from d5e0774). Built and tested locally on the iOS 26 simulator (Xcode 26.6): `xcodebuild test`, 411 test cases passed, 0 failed (25 of them new in `AddLinkModelTests`), no warnings.

What changed:
- `Features/AddLink/AddLinkModel.swift` (new): `@Observable`, built from closures (`Detect`, `Discover`, `Add`), with `init(read:mode:)` wiring them to `ReadService.detectURL`/`discoverFeed`/`addURL` (all three already existed, so `ReadService` is unchanged). `mode` is `.link` or `.feed`. Each change of `url` clears the error and the previous detection, cancels the detection task, and starts a new one after a 300 ms debounce. `isDetecting` is true from the edit until that detection lands, so Add is ignored in the meantime. An invalid URL stops detection at once. A failed detection is `.unknown`, and Add still works. `normalized` trims the URL and adds `https://` (`URL(string:encodingInvalidCharacters: false)` with a host, matching RN's `new URL`). `addURL()` sends `list` only for a detected feed; the success message and error wording are RN's. `findFeed()` gives: no feeds, "No RSS feed found for this site"; one feed, it replaces the URL, which is then re-detected as a feed; several, `feedChoices`. `close()` (on the sheet's disappear) cancels detection, and a discovery that lands afterwards is dropped. `canDismiss` is false only while an add is in flight. `submit()` is the return key: Add where it is offered, else Find feed.
- `Features/AddLink/AddLinkSheet.swift` (new): `NavigationStack` + `Form`. The URL section is a `TextField` (`.keyboardType(.URL)`, `.textContentType(.URL)`, no autocapitalisation or autocorrection, return submits) plus a system `PasteButton`. Its footer shows the offline note, else the error (in `error`), else the detection summary ("RSS feed: <title>", "Article: <title>"). For a detected feed, an inline Feed/Reads `Picker` whose header and footer carry `ListChoice`'s heading and explanation text. A Find feed button (with a progress view while discovering) for anything not detected as a feed. Toolbar: Cancel (`role: .cancel`) and, where offered, Add / Add Feed / Detecting… / Adding…. A `.confirmationDialog` for several feeds ("Multiple feeds found", "Select a feed to use:"). A "Success" alert whose OK dismisses the sheet. `onAdded` runs as soon as the add succeeds, before the alert. `.interactiveDismissDisabled` and a disabled Cancel while adding. Offline, the form is disabled and the footer says why.
- `Features/Reads/ReadsView.swift`: trailing toolbar "Add link" (`plus` SF Symbol: the app has no add icon; RN used Ionicons `add-outline`) opens the sheet in `.link` mode; `onAdded` runs `model.refresh()` (the first-page `Sync.syncReads(after: nil)`, or a rerun of the search while a query is held, as RN's `handleRefresh`).

Presenting API (for task 16, the RSS screen):
```swift
.sheet(isPresented: $isAddingFeed) {
    AddLinkSheet(read: session.read, mode: .feed) { Task { await model.reload() } }
}
```
The sheet reads `Connectivity` from the environment (inherited by sheets).

Tests: `AddLinkModelTests`, ported from AddLinkModal.test.tsx case for case:
- Dismissal while a detection is in flight: allowed, and a late detection is not applied.
- Dismissal blocked while an add is in flight.
- A discovery that lands after close offers no choices.
- A feed defaults to Reads.
- Picking Feed sends `list: feed`, with the success wording.
- A stale feed detection is dropped, and Add is ignored until the new URL is detected (then it sends no list).
- No stuck Detecting when the URL turns invalid.
- A page offers no choice and sends no list.
- In feed mode: the choice shows for a feed, a page only goes through Find feed (also on return), and return runs Find feed while detection is in flight.

Added beyond the jest suite: the debounce (three edits, one detection of the last URL); a superseded detection that lands late is not applied; `https://` is added; a failed detection is unknown and still adds; one found feed replaces the URL and detects as a feed; several are offered; none is an error that editing clears; the RN error wording (parameterised over server messages); the network message; an invalid URL is not sent to discovery.

Dropped or re-expressed jest cases:
- "explains both destinations when a feed is detected": the text is the sheet's section header/footer (view layout, not unit-tested). The model case checks that the choice is offered and defaults to Reads.
- The Modal `requestClose` cases become `canDismiss` plus `close()`. The sheet maps them to `.interactiveDismissDisabled`, the disabled Cancel and `.onDisappear`.
- The Alert assertions become `successMessage` and `feedChoices`.

Deliberate differences from RN:
- Detection is debounced 300 ms (RN detected on every keystroke and cancelled superseded runs). Superseded runs are still cancelled.
- `ReadService.discoverFeed` answers no feeds on failure, so a failed discovery shows "No RSS feed found for this site" rather than the error message.
- The success alert is shown in the sheet and its OK dismisses it (RN closed the popover first and alerted over the screen). The list refreshes at once either way.
- The error mapping is kept as RN had it. Against the current backend, only "Failed to add content to user: …" → "Failed to add article" changes anything: the server already says "Already subscribed to this feed" and "Failed to subscribe to feed".
- Offline: the form and Add are disabled, with "You're offline. Adding a link needs a connection." in the footer (RN had no offline handling here).

Dead code: `AddArticleScreen` is not ported. It is reachable only from the `headerRight` add button on the `MainTabs` stack screen in `RootNavigator.tsx`, and that screen has `headerShown: false`, so the button never renders. Nothing else navigates to `AddArticle`. The screen, its `RootStackParamList` entry and the dead `headerRight` can be removed from `apps/mobile`.

Risks:
- `PasteButton`'s closure is formed in a MainActor view; if the system calls it off the main thread, Swift 6's isolation check would trap. It is not exercised by tests, so check it on the device.
- On device (owner): the `PasteButton` and the inline picker inside the Form; the sheet's keyboard and return-key behaviour; the success alert over the sheet.
- Timing-based tests use a 20 ms debounce, `eventually` polling and two 50 ms sleeps, to show that a late response is not applied.
On-device check (owner): passed. Covered paste, article and feed detection, the Feed/Reads choice, Find feed (one, several and none), the URL keyboard and return key, dismissal while detecting and adding, the duplicate-feed error, offline, and layout in light and dark.
