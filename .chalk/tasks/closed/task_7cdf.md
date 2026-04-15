---
id: task_7cdf
title: Archived article still shows on Read page after archiving
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: https://github.com/cairn-app/cairn-reader/issues/150
created_at: 2026-04-10T07:57:55Z
updated_at: 2026-04-15T10:16:59Z
---

## Reproduction

1. Open the Read tab.
2. Tap an article to open it in the detail screen.
3. Tap the archive action and confirm.
4. Navigate back to the Read tab.

**Expected:** The archived article is no longer in the Read list.
**Actual:** The archived article is still visible in the Read list until the app is reloaded or the list is pull-to-refreshed.

## Root cause

- `apps/mobile/src/screens/ReadScreen.tsx` loads articles via `useEffect(() => { loadReadArticles(); }, [])`. That effect only fires once, on the initial mount of the tab. The screen never re-fetches when it regains focus after the detail screen is dismissed.
- `apps/mobile/src/screens/ReadArticleDetailScreen.tsx` (`handleArchive`) already calls `ReadService.deleteUserContent(article.id)` which successfully removes the article on the backend, then calls `navigation.goBack()`. But because `ReadScreen` keeps its stale `articles` state, the user sees the archived article still in the list.
- `apps/mobile/src/screens/BookmarksScreen.tsx:97-103` already demonstrates the correct pattern — it uses `useFocusEffect` so the list reloads whenever the screen regains focus. `VotesScreen.tsx` and `FeedsScreen.tsx` follow the same pattern.

## Plan

- [x] Convert `ReadScreen.tsx` to use `useFocusEffect` (from `@react-navigation/native`) to reload the list on focus, mirroring `BookmarksScreen`.
- [x] Wrap `loadReadArticles` in `useCallback` so it can be used safely as a `useFocusEffect` dependency.
- [x] Guard the focus refresh behind `!searchQuery` so an active search is not clobbered when returning to the tab, matching `BookmarksScreen`.
- [x] Remove the now-redundant `useEffect(() => { loadReadArticles(); }, [])` (focus effect fires on mount too).
- [x] Verify via `npm run type-check`, `npm run lint`, and `npm test` in `apps/mobile`.

Scope is intentionally limited. The semantics of "archive = backend DELETE" are unchanged — that matches the developer's current comment ("archiving = deleting") and the absence of any archive-viewing screen. A separate change would be needed to move to a true `status=archived` model with its own archive screen.

## Review

**Files changed:**
- `apps/mobile/src/screens/ReadScreen.tsx` — swapped mount-only `useEffect` for `useFocusEffect`; wrapped `loadReadArticles` in `useCallback`.

**Verification:**
- `npm run type-check` → passes (no errors).
- `npm run lint` → 0 errors; total warnings decreased from 17 → 16. The two new `react-hooks/exhaustive-deps` warnings on lines 125/132 (`handleRefresh`, `handleLoadMore` missing `loadReadArticles`) exactly mirror the long-standing warnings on `BookmarksScreen.tsx:102/112`, so the code matches the established convention on the sibling screen.
- `npm test` → 51/51 passing (`storage.test.ts`, `read.test.ts`, `helpers.test.ts`).

**Behavioural diff vs. `main`:**
- Before: after archiving an article in `ReadArticleDetailScreen`, the backend `DELETE /api/v1/content/user/{id}/{contentId}` succeeds, but `ReadScreen`'s local `articles` state is still the initial load. On `navigation.goBack()`, the archived article is still rendered until the user pulls to refresh or restarts the app.
- After: `useFocusEffect` fires whenever `ReadScreen` regains focus (including after the detail screen is dismissed), calling `loadReadArticles(true)` which refetches the list from the backend. The archived article is gone immediately.

**Not reproducible in an emulator from this sandbox** — no simulator available. The fix is a targeted mirror of the `BookmarksScreen` / `VotesScreen` / `FeedsScreen` pattern that already works correctly for the same class of refresh-on-return bugs.

