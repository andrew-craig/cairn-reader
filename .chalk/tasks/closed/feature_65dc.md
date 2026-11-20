---
id: feature_65dc
title: [iOS native 13] Report extraction issue sheet
type: feature
status: closed
priority: 3
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-18T11:16:03Z
---
## Goal
Let the reader report a badly extracted article.

## Source
`apps/mobile/src/components/ReportIssueModal.tsx` and test; `components/common/HeaderPopover.tsx`.

## Plan
- [x] `.sheet` containing a `NavigationStack` + `Form`: a multi-select list of the ten reasons (same labels and category values), a comment `TextEditor` limited to 2000 characters with a counter, Cancel and Send in the toolbar.
- [x] Prefill from `getExtractionFeedback` each time it opens; submit with `submitExtractionFeedback`; inline error on failure; Send disabled with nothing selected and while sending.

## Verify
- View-model tests ported from `ReportIssueModal.test.tsx`: prefill, validation, submit payload, error path.

## Review
Implemented on `worktree-agent-a6e2d7124268fb15b` (from d5e0774). Built and tested locally (Xcode 26.6, iPhone 17 Pro simulator, iOS 26): `xcodebuild test` succeeded with warnings as errors, 394 test cases passed, none failed (whole suite).

What changed:
- `Features/ReportIssue/ReportIssueModel.swift` (new): `@Observable`, one per presentation of the sheet. `reasons` holds the ten categories in server order with the RN labels. `selected` (a set), `comment` (cut to `commentLimit` = 2000 in a `didSet`), `isLoading`, `isSending`, `error`, `canSend` (a reason selected, not loading, not sending). `load()` prefills from `getExtractionFeedback`; a failed or missing report leaves the form empty (RN logged and carried on). `toggle(_:)` clears the error, as RN did. `send()` puts the categories in list order (not tap order) and the trimmed comment (omitted when blank) through `submitExtractionFeedback`; it answers true when accepted, else keeps the form and sets `error` to the error's description. A second send while one is in flight does nothing.
- `Features/ReportIssue/ReportIssueView.swift` (new): `.sheet` content, `NavigationStack` + `Form`. The reasons are a section of rows that toggle a checkmark (`.isSelected` trait), disabled while loading or sending. The details are a vertical `TextField("Details (optional)")` with a `count/2000` counter in the section footer. The inline error is in `error`. Cancel (`cancellationAction`) is disabled while sending. Send (`confirmationAction`) is disabled until `canSend`, and a `ProgressView` replaces it while loading or sending. Interactive dismiss is off while sending. `.task { await model.load() }`.
- `Features/Reader/ReaderView.swift`: the bottom toolbar gains a More `Menu` (`ellipsis`) with "Report an issue with the article" (`ant` SF Symbol: the app has no bug icon), which presents the sheet. A new view per presentation means a new model, so each opening prefills afresh and closing resets it. On success the sheet calls `onSent` and dismisses. Its `onDismiss` then shows the RN "Report sent" / "Thanks — your report was sent." through the reader's existing alert, so the alert does not race the sheet's dismissal.
- `apps/ios/CLAUDE.md`: one sentence in the Reader bullet.

Tests: `ReportIssueModelTests` (send disabled until a reason is selected and again after deselecting; prefill from the existing report, again for a second presentation after the first was edited; a failed load leaves the form empty; categories sent in list order with the trimmed comment; a blank comment is not sent; send disabled while sending, with a second send ignored (held PUT); details limited to 2000 characters; inline error on failure keeps the selection and re-enables Send, and a reason change clears it).

Dropped or re-expressed jest cases:
- "submits categories and comment, confirms and closes": the payload part is kept. The alert and close are the view's response to `send()` answering true, so they are not unit-tested.
- "limits the details to 2000 characters" asserted the `TextInput`'s `maxLength`. Here the model cuts the comment, and the test sets 2001 characters.

Deliberate differences from RN and from the plan:
- An inline sheet with a navigation bar instead of `HeaderPopover` (a blurred popover under the header). Cancel/Send are in the toolbar, not a button row. The button is "Send" (the plan's word) rather than RN's "Submit".
- The entry point is the toolbar's More menu instead of a footer link under the article (the web view's page has no native footer, and task 12 left it for a toolbar menu). It shows in every reader state, including loading and offline, where a send fails with the inline error.
- A vertical `TextField` instead of the plan's `TextEditor`: it has a system placeholder, so there is no hand-placed overlay.
- As in RN, the reasons are disabled while the earlier report loads, and the comment field only while sending (`editable={!sending}`).

Risks:
- On device (owner): the sheet's keyboard handling with the multi-line field, and the alert appearing after the sheet's dismissal.
- The menu holds a single item. If later tasks add reader actions, they belong there.

On-device follow-up (owner): the "Report sent" alert needed dismissing for no reason. It's replaced by a toast: once the sheet has gone, a glass capsule saying "Report sent" sits just above the bottom toolbar for 2 s, then fades. It ignores touches and is announced to VoiceOver. It's local to `ReaderScreen` (`showsReportToast`, cleared by a `.task(id:)` sleep), because nothing else needs a toast yet. Needs a recheck on device.
On-device check (owner): passed, including the "Report sent" toast.
