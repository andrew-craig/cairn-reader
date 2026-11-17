---
id: feature_4e37
title: [iOS native 12] Reader: web-view article, progress tracking, toolbar actions
type: feature
status: in_progress
priority: 1
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-17T11:52:48Z
---
## Goal
The reading screen. The highest-risk UI task: scroll restore and completion tracking must behave as they do today.

## Source
`apps/mobile/src/screens/ReadArticleDetailScreen.tsx`, `components/common/{ArticleContent,BottomActionMenu}.tsx`; tests `ReadArticleDetailScreen.test.tsx`, `ReadArticleDetailScreen.mutations.test.tsx`, `ArticleContent.test.tsx`.

## Native mapping
| RN | Native |
|----|--------|
| `react-native-render-html` + `tagsStyles` | SwiftUI `WebView` / `WebPage` loading a bundled HTML template with `cleaned_html` injected (decision 2) |
| `BottomActionMenu` floating pill | `.toolbar` items in `.bottomBar` |
| `ActionSheetIOS` on link tap | `.confirmationDialog` |
| vertical card transition, custom back | standard push, system back button and swipe |
| `Alert.alert` | `.alert` |
| route params `article`, `articles`, `currentIndex`, list callbacks | route carries the `Article`; the presenting list model answers `article(after:)` |

## Plan
- [x] `ReaderModel` (`@Observable`): starts from the route's `Article`. When the store has a row for the id (every Reads item) it follows `ArticleStore.observeArticle(id)`, so favourite, read state and a re-synced value after a rejection are never held twice; otherwise (Feed, and Bookmarks or search results outside the store) it keeps the route's value and applies its own edits to it.
- [x] Body: skip the fetch when the stored `contentHash` matches, never fetch offline, retry when connectivity returns, save the fetched body to the store. States: loading, content, "Not available offline" (`ContentUnavailableView`).
- [x] `reader.html` + `reader.css` in the bundle: header (title, author, domain, published date, reading time), body, and the styles from `tagsStyles` (paragraph spacing, links, headings, blockquote, `pre`/`code`, images max-width 100%, lists). Colours as CSS variables written from the token values with `prefers-color-scheme`; body in Atkinson Hyperlegible Next and content headings in Inter bold (as in `tagsStyles`) via `@font-face` from the bundle; base size follows the user's Dynamic Type setting (passed into the template as a CSS variable from `UIFontMetrics`).
- [x] Security: no JavaScript from article content (`cleaned_html` is server-sanitised; still disable content JavaScript and load with a `nil`/about:blank base URL), and the web view never navigates away: every link activation is intercepted.
- [x] Link tap: haptic, then a dialog titled with the link's domain: Save to Reads (`addURL`), Open in Browser (`openURL`), Copy Link (`UIPasteboard`).
- [x] Progress (Reads items only; off for Feed items): restore the saved fraction once content height makes the target reachable, do not persist the pre-restore offset, save throttled to 1 s while scrolling and once on leaving, mark completed at 95% (`(offset + viewport) / contentHeight`), mark reading on open if unread. Read scroll geometry with the web view's scroll APIs rather than injected script.
- [x] Toolbar, all through `ArticleMutations` (decision 9), none of which wait for or handle the backend's answer:
  - favourite toggle;
  - Archive for Reads items or Save to Reads for Feed items, then pop. The originating list updates itself from the store (decision 8), so the reader calls nothing on it;
  - Next article when the presenting model has one: ask `article(after:)` before archiving, then replace the current reader;
  - `ShareLink` for the article URL.
- [ ] "Report an issue with the article" entry (toolbar menu item) presenting the sheet from the report task; ship as a no-op hook if that task has not landed.

## Verify
- `ReaderModel` tests re-expressed from the two detail-screen test files: hash-match skips fetch, offline with and without a stored body, reconnect retry, a stored article follows the store (a favourite changed elsewhere shows; a rejected favourite shows the server's value after the re-sync), an unstored article keeps local state, archive and save-to-reads call the mutation and pop, progress not tracked for Feed items, completion fires once. The optimistic-rollback cases do not port (decision_adb1).
- A pure-function test for the restore-reachability rule (`isScrollTargetReachable`).
- Owner on device: long article restores position after relaunch; images, tables and code blocks render in light and dark; largest Dynamic Type size is readable.

## Review
Implemented on branch `task12` (from 7071697); not built locally (Linux, no Swift toolchain). To be verified by the iOS App Checks workflow.

What changed:
- `Features/Reader/ReaderModel.swift` (new): `@Observable`. `article` is the stored row (`observeArticle`, followed by `follow()` in `.task`), or else the route's value with the reader's own edits; a body fetched here wins over both. `loadBody(isOnline:)` runs in `.task(id: connectivity.isOnline)`, so it retries on reconnect. It does nothing when the route already carried a body (only store rows do, and the upsert's hash rule keeps their body current). Otherwise a stored body with the route's hash ends it. It never fetches offline; it fetches the detail, shows it and saves it with `ArticleStore.saveBody`. A load generation stops an earlier, cancelled load from clearing a newer load's spinner. `display` is `.loading`, `.article` (after a failed online fetch the template shows the description, as RN did) or `.unavailableOffline`. Edits: `toggleFavorite`; `archive` (sets `isFinished` first, so the screen pops before the write); `saveToReads` (pops after the store write; a database failure keeps the reader open with the RN alert); `saveLink` (`addURL`, with RN's "Saved"/"Error" alerts). Progress (`tracksProgress` = route `list == .reads`): `markReadingIfNeeded` runs once and never after completion, because `.task` reruns on reappear. `scrolled(Scroll)` ignores readings marked `isRestoring`, marks completion once at `(offset + viewport) / contentHeight >= 0.95`, and saves through a leading + trailing 1 s throttle (the `@cairn/shared` throttle). `leave()`, from `onDisappear`, cancels the trailing save and saves the last confirmed fraction if one was ever confirmed.
- `Features/Reader/ArticleWebView.swift` (new): `UIViewRepresentable` over `WKWebView`. Content JavaScript and link previews are off. The base URL is `cairn-reader://reader/`, served by `ReaderResources` (a `WKURLSchemeHandler` that answers only `.css`/`.ttf` files from the bundle). Navigation policy: a link to anything outside the page's own scheme goes to `onLink` and is cancelled. Only main-frame loads of the page's scheme (the page and its `#anchors`) are allowed; nothing else loads. Restore: each load reads the fraction to restore from the model. On every `contentSize` change (KVO) before the restore is done, the view scrolls to `min(fraction * contentHeight, maxOffset)`. The restore counts as done once `isScrollTargetReachable` holds, when `didFinish` fires (the content will not grow further, so an unreachable position, e.g. an article shorter than the screen, ends there), or when `scrollViewWillBeginDragging` fires. Readings also count as restoring until `didFinish` or a drag, so a fresh article whose text loads before its images is not completed on open (review fix). Until then every reading (`scrollViewDidScroll`, content-size changes) is reported with `isRestoring`. Geometry comes from the scroll view, with offset and viewport measured inside `adjustedContentInset` (the page runs under the bars with `.ignoresSafeArea()`). If the content process is killed, the page reloads and restores again.
- `Features/Reader/ReaderTemplate.swift` (new) and `Resources/Reader/reader.html`, `reader.css` (new): the header has the title (linked to the article when it is http(s)) and a byline "author or domain | published date | N min read" ("Email" for an email without an author). The body is `cleaned_html` raw, else the escaped description. The CSS comes from `tagsStyles` (paragraph spacing, links in `primary`, Inter bold headings, blockquote, `pre`/`code` on `card`, lists), plus `max-width: 100%` images and scrollable tables and `pre`. Colours are CSS variables written at runtime from the colour assets (`UIColor(resource:)` resolved light and dark, dark under `prefers-color-scheme`), so there is no colour literal outside `design/tokens`. The base size `--base-size` is `UIFontMetrics(.body).scaledValue(for: 16)` at the environment's Dynamic Type size; when it changes the page reloads and restores the reading position. Escaping also covers `{`, so a title cannot smuggle in a later placeholder, and the body goes in last.
- `Features/Reader/ReaderView.swift` (new): `ReaderView` holds the current article and shows `ReaderScreen` keyed by its id. Next article therefore replaces the article with a new model, and the old screen's `onDisappear` saves its progress. `ReaderScreen` shows a loading `ProgressView`, `ContentUnavailableView("Not available offline")`, or the web view. Its bottom `.toolbar` has: Next article when the presenter provides one (disabled when there is no next row); Favorite / Remove from Favorites; Archive or Save to Reads; and `ShareLink` for an http(s) URL. The tab bar is hidden. A link tap gives a medium impact (`sensoryFeedback`), then a `.confirmationDialog` titled with the link's domain (Save to Reads, Open in Browser, Copy Link); a saved link gives a success haptic. There is an `.alert` for the model's alert, and the screen pops on `isFinished`. A tap on the header title opens the article in the browser directly.
- `Features/Main/Route.swift`: `.article` shows `ReaderView` (session from `\.userSession`). `RouteDestination` gains `init(_:nextArticle:)`. `init(_:)` is unchanged, so `destination: RouteDestination.init` still works elsewhere.
- `Features/Reads/ReadsView.swift`: `.navigationDestination(for: Route.self) { RouteDestination($0, nextArticle: model.article(after:)) }`.

Next-article wiring: SwiftUI uses only the `navigationDestination(for: Route.self)` declared closest to the stack's root (a later one for the same type is ignored with a runtime warning), and builds the destination from that declaration. So the presenter's `article(after:)` goes into the destination closure of the tab's root screen, not through the environment of the row that pushes. RN passed `articles`/`currentIndex` only from Reads and Feed (not Bookmarks), so only tab roots need it. Feed (task 11) changes its line to `.navigationDestination(for: Route.self) { RouteDestination($0, nextArticle: model.article(after:)) }` with its `ArticleListModel`, whose `article(after:)` already applies `exclude`. Bookmarks keeps the plain declaration: it is pushed inside You's stack, which uses You's destination, and it has no Next, as in RN. The reader asks the presenter for the next article live in its body (`nextArticle(article.id)`), so the button follows the list and reads the next row while the current row is still listed.

Tests: `ReaderModelTests` (stored body shown while the fetch is held, hash match skips the fetch, another hash shows the stale body then the fetched one and saves it, offline with and without a stored body, reconnect shows the spinner then content, a favourite changed elsewhere shows, a rejected favourite shows the server's value after the re-sync, an unstored article keeps its edit and queues it, archive and save-to-reads queue and pop, Feed items track nothing, mark reading once and never after completion, a read item is not marked reading, completion at 95% fires once, the pre-restore offset is never saved, throttled saves with a trailing save, leaving saves the last position). `ScrollTargetReachabilityTests` (the five ArticleContent.test.tsx cases). `ReaderTemplateTests` (byline uses the published date and omits a missing one, email byline, escaping and body placement, description fallback, stylesheet and fonts in the bundle).

Dropped or re-expressed jest cases:
- mutations test "rolls back both the UI and the store on a definitive failure", "does not touch the store when the mutation succeeds": there is no rollback (decision_adb1). Re-expressed as "a rejected favourite shows the server's value after the re-sync".
- "navigates back immediately without waiting on the backend archive", "surfaces a definitive failure via Alert after navigation", "does not alert when the backend archive succeeds": the archive is a store write, and `isFinished` is set before it. A refusal is the root's alert (`MainTabs`, `Outbox.rejections`), which is already covered.
- Feed "saves the item, marks Reads stale, then leaves the reader": there is no `invalidateReads` or `onArchived` (decision 8: lists follow the store). Kept as "save to Reads queues the move and pops". "stays in the reader and alerts when the move is rejected": the backend's answer is not awaited, and only a database failure keeps the reader open (not unit-tested; it needs a failing database). "ignores a second tap while the move is in flight": covered by the `isFinished` guard, not tested separately.
- "offers Save to Reads instead of Archive" / "keeps Archive for Reads items": view layout (`article.list` picks the button), not unit-tested.
- ArticleContent byline "1 week ago": asserted against `relativeDayLabel` rather than the English string.
- Detail-screen "renders a stored body without waiting for getContentById to resolve": kept, using a held detail response.

Deliberate differences from RN and from the plan:
- `WKWebView` in a `UIViewRepresentable` instead of SwiftUI's `WebView`/`WebPage`: the restore needs three things the scroll view gives directly: content-size growth, the start of a drag, and a synchronous offset write. The fonts and stylesheet need a base URL that is neither nil (relative `@font-face` URLs would not resolve) nor a file or remote URL. Hence the custom scheme instead of the plan's nil/about:blank base.
- Completion is checked only for confirmed readings, not while restoring. In RN, a restore towards an unreachable target scrolls to the bottom of the short, still-loading content. That reads as 100% and could mark an article completed on open.
- A save is skipped when the fraction equals the last one saved (at first, the article's own). Opening an article therefore no longer queues a `scroll_position` row with the same value.
- No progress flush when leaving through Archive or Save to Reads (RN's unmount flush sent a scroll PATCH for the archived item).
- The byline's author/domain is not a link (RN linked it to the site's origin); the title opens the article, as in RN.

Not done:
- "Report an issue with the article" entry (plan box left open): task 13 (feature_65dc) is blocked by this task, and a menu item that opens nothing would be dead UI (task 10 made the same call for add-link). Task 13 adds a toolbar `Menu` item in `ReaderScreen` and its `.sheet`.

Risks:
- Compile risks: the async `WKNavigationDelegate.webView(_:decidePolicyFor:)` and the scheme handler's conformance under Swift 6 default MainActor isolation; the KVO closure (`MainActor.assumeIsolated`); `UIContentSizeCategory(dynamicTypeSize)`; `UIColor(resource: .primary)`; the `@MainActor` closure type for `nextArticle` (Next replaces the article through a binding, not a closure).
- Restore on device (owner): check that `adjustedContentInset` matches what WebKit uses under the iOS 26 bars, and that late image loads trigger the content-size KVO.
- Timing-based tests: the held-detail tests poll a `Mutex` in the stub handler (which now runs off the shared URL-loading thread); the throttle test sends its readings in one turn with a 5 s interval; several cases sleep 50 ms to show that nothing was written.
- CI: iOS App Checks green on PR #468 (3974b0e). Close after merge and on-device check.
