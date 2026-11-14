---
id: decision_1686
title: [iOS native 00] Confirm migration decisions (min iOS, reader, persistence, cutover)
type: decision
status: open
priority: 1
labels: [ios,mobile]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:52Z
updated_at: 2026-11-14T02:35:52Z
---
Owner sign-off on the choices the rest of epic_bc48 is built on. Each has a recommended default; the tasks are written against the defaults. Change one here and the affected task descriptions need a matching edit before work starts.

| # | Decision | Recommended | Alternative | Affects |
|---|----------|-------------|-------------|---------|
| 1 | Minimum iOS | **iOS 26** (RN app is 15.1). Gives the SwiftUI `WebView`/`WebPage`, the `Tab` API and the system Liquid Glass bars with no wrappers. | iOS 18: reader needs a `UIViewRepresentable` around `WKWebView` (one more custom component). | scaffold, reader |
| 2 | Reader rendering | **Web view** showing `cleaned_html` inside a bundled HTML/CSS template. Tables, code, images and lists render correctly for free. | Native text from parsed HTML: a large custom renderer, against the "minimal custom components" goal. | reader |
| 3 | On-device store | **GRDB** (SQLite). The store and outbox are SQL today (`UPSERT` with outbox guards, `json_extract`); they port statement for statement and so do their tests. Only third-party dependency in the app. | SwiftData: zero dependencies, but the guarded upsert has to be re-expressed in fetch-then-write code. | store, outbox |
| 4 | Existing installs | **No data migration** (root CLAUDE.md: no backward compatibility). Same bundle id, so the update installs in place: the AsyncStorage session is not read, the user signs in again. Device-ID accounts come back on their own because `identifierForVendor` survives an in-place update and is what the RN app sends as `expo_device_id`. The offline cache re-downloads. Edits still queued in the RN outbox at update time are lost; the old AsyncStorage and SQLite files stay on disk unused. | Read the old tokens / drain the old outbox once on first launch (throwaway compatibility code). | auth, cutover |
| 5 | Typography | **System font with Dynamic Type for all UI**; bundle only Atkinson Hyperlegible Next for the article body in the reader. Inter and Crimson Pro are dropped. | Keep Inter/Crimson Pro across the UI via custom font modifiers. | tokens, every screen |
| 6 | RN app and Android | New app in `apps/ios`. `apps/mobile` stays shippable until cutover, then is **deleted**, which also removes the Expo Android target (no `android/` is committed; Android is out of scope). | Keep `apps/mobile` for Android: two mobile clients to maintain. | cutover |
| 7 | Visual fidelity | **Platform look wins.** The floating pill tab bar, the search button that morphs into a pill, the floating action pill and the top blur gradient are replaced by the system tab bar, `.searchable`, toolbars and the navigation bar. The colour palette is kept. | Rebuild those as custom SwiftUI views. | every screen |

## Done when
- [ ] Each row is confirmed or changed by the owner, and the outcome is recorded in this task and in the epic.
