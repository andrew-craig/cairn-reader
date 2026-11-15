---
id: decision_1686
title: [iOS native 00] Confirm migration decisions (min iOS, reader, persistence, cutover)
type: decision
status: closed
priority: 1
labels: [ios,mobile]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:52Z
updated_at: 2026-11-14T06:33:17Z
---
Owner sign-off on the choices the rest of epic_bc48 is built on. Each has a recommended default; the tasks are written against the defaults. Change one here and the affected task descriptions need a matching edit before work starts.

| # | Decision | Recommended | Alternative | Affects |
|---|----------|-------------|-------------|---------|
| 1 | Minimum iOS | **DECIDED (owner, 2026-11-14): iOS 26** (RN app is 15.1). Gives the SwiftUI `WebView`/`WebPage`, the `Tab` API and the system Liquid Glass bars with no wrappers. | iOS 18: reader needs a `UIViewRepresentable` around `WKWebView` (one more custom component). | scaffold, reader |
| 2 | Reader rendering | **DECIDED (owner, 2026-11-14): web view** showing `cleaned_html` inside a bundled HTML/CSS template. Tables, code, images and lists render correctly for free. | Native text from parsed HTML: a large custom renderer, against the "minimal custom components" goal. | reader |
| 3 | On-device store | **DECIDED (owner, 2026-11-14): GRDB** (SQLite). The store and outbox are SQL today (`UPSERT` with outbox guards, `json_extract`); they port statement for statement and so do their tests. Only third-party dependency in the app. | SwiftData: zero dependencies, but the guarded upsert has to be re-expressed in fetch-then-write code. | store, outbox |
| 4 | Existing installs (iOS) | **DECIDED (owner, 2026-11-14): no data migration** (root CLAUDE.md: no backward compatibility). Same bundle id, so the update installs in place: the AsyncStorage session is not read, the user signs in again. Device-ID accounts come back on their own because `identifierForVendor` survives an in-place update and is what the RN app sends as `expo_device_id`. The offline cache re-downloads. Edits still queued in the RN outbox at update time are lost; the old AsyncStorage and SQLite files stay on disk unused. | Read the old tokens / drain the old outbox once on first launch (throwaway compatibility code). | auth, cutover |
| 5 | Typography | **DECIDED (owner, 2026-11-14): keep the brand fonts.** Inter for UI text, Crimson Pro for the wordmark/headings where the RN app uses it (Login), Atkinson Hyperlegible Next for article body. Bundled and applied through one typography file, scaling with Dynamic Type. | ~~System fonts~~ (rejected) | tokens, every screen |
| 6 | RN app and Android | **DECIDED (owner, 2026-11-14): `apps/mobile` is kept; Expo continues to be used for Android.** New iOS app in `apps/ios`. At cutover only the iOS side of `apps/mobile` is retired (committed `ios/` project, RN TestFlight workflows, iOS-only code branches). `apps/shared` keeps two consumers. | ~~Delete `apps/mobile`~~ (rejected) | cutover, release pipeline |
| 7 | Visual fidelity | **DECIDED (owner, 2026-11-14): platform look wins.** The floating pill tab bar, the search button that morphs into a pill, the floating action pill and the top blur gradient are replaced by the system tab bar, `.searchable`, toolbars and the navigation bar. The colour palette is kept. | Rebuild those as custom SwiftUI views. | every screen |

## Done when
- [x] Rows 5 and 6 decided by the owner on 2026-11-14 (see table).
- [x] Rows 1, 2, 3, 4 and 7 confirmed by the owner on 2026-11-14 as recommended, and recorded here and in the epic.

## Review (2026-11-14)
All seven rows are decided; no task description needed rewriting because the tasks were written against the confirmed defaults. Three conditions came out of the review and were added to the affected tasks:
- Row 4, task_5092: the user service matches `expo_device_id` as an exact string, so the Swift app sends `identifierForVendor.uuidString` unmodified. A differently-cased value would create a new empty account instead of restoring the existing one.
- Row 4, task_c194: the RN `cairnreader.db` stays on disk, so the GRDB database uses a different file (already in that task's plan).
- Rows 5 and 7, task_fe45: check on device that the brand fonts apply to the system bars and the search field at the largest Dynamic Type size.
Accepted costs: devices that cannot run iOS 26 (older than iPhone 11) stay on the last RN build; email users sign in once; edits still queued offline at the update are lost; iOS chrome diverges from the Android app.
