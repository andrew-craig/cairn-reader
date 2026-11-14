---
id: feature_4e37
title: [iOS native 12] Reader: web-view article, progress tracking, toolbar actions
type: feature
status: open
priority: 1
labels: [ios,ui]
blocked_by: [feature_33bc]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
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

## Plan
- [ ] `ReaderModel` (`@Observable`): resolve the article by id from `ArticleStore`, merge with the list metadata, skip the fetch when the stored `contentHash` matches, never fetch offline, retry when connectivity returns, save the fetched body to the store. States: loading, content, "Not available offline" (`ContentUnavailableView`).
- [ ] `reader.html` + `reader.css` in the bundle: header (title, author, domain, published date, reading time), body, and the styles from `tagsStyles` (paragraph spacing, links, headings, blockquote, `pre`/`code`, images max-width 100%, lists). Colours as CSS variables written from the token values with `prefers-color-scheme`; body in Atkinson Hyperlegible Next and content headings in Inter bold (as in `tagsStyles`) via `@font-face` from the bundle; base size follows the user's Dynamic Type setting (passed into the template as a CSS variable from `UIFontMetrics`).
- [ ] Security: no JavaScript from article content (`cleaned_html` is server-sanitised; still disable content JavaScript and load with a `nil`/about:blank base URL), and the web view never navigates away: every link activation is intercepted.
- [ ] Link tap: haptic, then a dialog titled with the link's domain: Save to Reads (`addContentToUser`), Open in Browser (`openURL`), Copy Link (`UIPasteboard`).
- [ ] Progress (Reads items only; off for Feed items): restore the saved fraction once content height makes the target reachable, do not persist the pre-restore offset, save throttled to 1 s while scrolling and once on leaving, mark completed at 95% (`(offset + viewport) / contentHeight`), mark reading on open if unread. Read scroll geometry with the web view's scroll APIs rather than injected script.
- [ ] Toolbar: favourite toggle (optimistic, rolled back on a definitive failure), Archive for Reads items or Save to Reads for Feed items (pop, remove from the originating list, alert on definitive failure), Next article when the list has one (replaces the current reader), `ShareLink` for the article URL.
- [ ] "Report an issue with the article" entry (toolbar menu item) presenting the sheet from the report task; ship as a no-op hook if that task has not landed.

## Verify
- `ReaderModel` tests ported from the two detail-screen test files: hash-match skips fetch, offline with and without a stored body, reconnect retry, favourite rollback, archive and save-to-reads flows, progress not tracked for Feed items, completion fires once.
- A pure-function test for the restore-reachability rule (`isScrollTargetReachable`).
- Owner on device: long article restores position after relaunch; images, tables and code blocks render in light and dark; largest Dynamic Type size is readable.
