# CLAUDE.md - Mobile App

This file provides guidance to Claude Code when working with the Cairn mobile application.

## Service Overview

The Cairn mobile app is a React Native application built with Expo (iOS and Android; `npm run web` also works) that provides a read-it-later experience. It talks to the Cairn backend (User and Read services) through a single server origin.

**Key Features:**
- Device ID or email/password authentication (JWT, tokens in AsyncStorage)
- Two lists: **Feed** (online-only skim list) and **Reads** (offline-first read/triage list); the user picks the list per source
- Favorites (Bookmarks), archive, RSS and newsletter subscriptions
- Offline reading of Reads articles (SQLite); edits queue in a mutation outbox and sync on reconnect
- Dark mode (follows system preference)

## Project Structure

```
apps/mobile/
├── App.tsx / index.js        # Entry (index.js imports src/config/init first, then registers App)
├── app.json                  # Expo config (no app.config.*, no EXPO_PUBLIC_* env vars)
├── ios/                      # Committed native project; ios/fastlane lanes: beta (TestFlight), certificates, build_only
├── jest.config.js, jest.setup.ts, __mocks__/   # Test setup
├── knip.json                 # Unused-code check config
└── src/
    ├── components/           # AddLinkModal, ArticleListScreen, SubscriptionListScreen,
    │   │                     #   ErrorBoundary, LogoMark; tests co-located (*.test.tsx)
    │   ├── common/           # ArticleContent, ArticleRow, BottomActionMenu, Button, CustomTabBar, HeaderPopover, HeaderSearch,
    │   │                     #   IconButton, ListChoice (explained Feed/Reads), ListToggle (Feed|Reads), OfflineBanner, ScreenHeader,
    │   │                     #   SyncTriggerEffect, TopBlurGradient, ...
    │   └── icons/            # SVG icons
    ├── config/               # init.ts (wires @cairn/shared at startup), storage.ts (AsyncStorage adapter + DEFAULT_SERVER_URL)
    ├── constants/            # theme.ts (Colors, Spacing, FontSizes, FontFamily, BorderRadius, Layout), globalStyles.ts
    ├── contexts/AuthContext.tsx
    ├── hooks/                # useCursorArticleList, useNetworkStatus, useSyncTrigger
    ├── navigation/           # RootNavigator (stack), TabNavigator (tabs)
    ├── screens/              # Feed, Reads, You (tabs); ReadArticleDetail, AddArticle, Bookmarks, Account, About,
    │                         #   Rss, Newsletters, Login
    ├── services/             # auth, read, system, articleStore, articlePrefetch, articleMutations, outbox,
    │                         #   syncTrigger, feedCache, readsInvalidation, db (SQLite connection + migrations)
    ├── types/                # navigation.ts (client-specific); index.ts re-exports @cairn/shared types
    └── utils/                # helpers, network, retry
```

## Shared code (`@cairn/shared`, `apps/shared`)

Framework-agnostic code shared with the web app (npm workspace, consumed as source): data-model types (`Article`, `ContentList`, `User`, `LoginResponse`, ...), the server-URL/persistence layer (`getServerUrl`, `setServerUrl`, `configureStorage`, `configureDefaultServerUrl`), the base `AuthService` (token refresh state machine, email login/register, logout, `changePassword`, `fetchWithAuth`), `fetchOrNetworkError`/`NetworkError`/`HttpError`, `throttle`, and date helpers (`formatDate`, `formatPublishedDate`, `pluralize` in `utils/date.ts` — used by `ArticleRow` and `ArticleContent`; web uses the same). Check there before adding a util to mobile or web. Run its tests with `npm test` in `apps/shared` (vitest).

## Architecture and Patterns

### Navigation
React Navigation stack + bottom tabs. **Explore was removed; the tabs are Feed, Reads, You.**

```
RootNavigator (Stack)               src/navigation/RootNavigator.tsx
├── MainTabs (Bottom Tabs)          src/navigation/TabNavigator.tsx
│   ├── Feed → FeedScreen
│   ├── Reads → ReadsScreen
│   └── You → YouScreen
├── ArticleDetail → ReadArticleDetailScreen
├── AddArticle (modal), Bookmarks, Account, About, RSS, Newsletters
```

Route params are typed in `src/types/navigation.ts` (`RootStackParamList`, `MainTabParamList`); update them when adding a screen. `OfflineBanner` is mounted in `RootNavigator`'s loading, logged-out and authenticated states; `SyncTriggerEffect` only in the authenticated tree.

### State management
Hooks plus Context; no global store.
- `AuthContext` (`src/contexts/AuthContext.tsx`): auth state, login/logout. On logout it calls `AuthService.logout()`, then `ArticleStore.clear()` (also empties the outbox) and `FeedCache.clear()`.
- Persistence: `ArticleStore` and `Outbox` (one SQLite DB, `db.ts`), `FeedCache` (AsyncStorage), tokens/user/server URL (AsyncStorage).

### Service layer
Services are static classes or plain objects (no instantiation) in `src/services/`. Backend models are transformed to the shared `Article` in `ReadService.transformToArticle` / `transformDetailToArticle` (`word_count / 200` → `readingTime`, `status === 'completed'` → `isRead`, `image_urls[0]` → `imageUrl`, `list` carried through).

## Screens

- **FeedScreen** — items from sources routed to Feed (`list=feed`). Online-only: never written to `ArticleStore` or prefetch. Keeps the last page in `FeedCache` so a failed refresh shows a stale banner. In the reader, "Save to Reads" (`ReadService.moveToReads`) replaces Archive, and progress tracking (status, completion, scroll) is off, so only Favorite can reach the outbox for a Feed item.
- **ReadsScreen** — `list=reads`, store-first render (`ArticleStore.listRecent`) with a "Showing cached data" banner when the background refetch fails. Every list sync (including pull-to-refresh) calls `ArticleStore.upsertMany` then `SyncTrigger.run()`; `readsInvalidation` marks the list stale after Save to Reads.
- **Search** (Feed, Reads, Bookmarks) — `HeaderSearch` in `ScreenHeader`, mirroring web: the round search button morphs into a pill over the title. `useCursorArticleList` owns the open state and query and runs a debounced live search (`searchUserContents`); results replace the list while a query is held, and clearing it restores the pre-search list and refreshes it quietly.
- **ReadArticleDetailScreen** — reader (`ArticleContent`, `react-native-render-html`). Resolves the article by id from `ArticleStore`, skips the fetch when the stored `contentHash` matches, and shows "Not available offline" when offline with nothing cached.
- **YouScreen** — profile hub (stats, links to Bookmarks, Account, About, RSS, Newsletters). **RssScreen / NewslettersScreen** render `SubscriptionListScreen`, which has a per-source Feed/Reads toggle (`ReadService.setSourceList`; affects new items only). **AddLinkModal** asks Feed or Reads (`ListChoice`, which explains both) when subscribing to a feed. RssScreen's "Add feed" button opens it with `mode="feed"`: a non-feed URL can only go through Find feed, never be saved as an article.
- **LoginScreen** — device login, email login/registration, and a server URL field (`setServerUrl()`, no rebuild). **AccountScreen** — logout, device → email/password upgrade, password change.

## Services (offline model)

- **ArticleStore** (`articleStore.ts`) — SQLite (`expo-sqlite`) store of Reads articles: metadata, user state, and an opportunistically cached body diffed by `content_hash`. A read-through cache for initial render, not a query engine: `useCursorArticleList` still paginates against the network. Upsert-only sync; rows are removed by archive (`remove`) or `clear()`. Schema changes go through the `PRAGMA user_version` ladder in `db.ts`.
- **ArticlePrefetchService** — downloads `cleaned_html` for unread/reading Reads articles lacking a body: 3 workers, newest first, capped at the 100 most recent, evicts bodies outside the cap. Skips when offline, aborts the batch on the first `NetworkError`, single-flight. Only invoked via `SyncTrigger`.
- **Outbox** (`outbox.ts`) — SQLite queue of PATCH payloads keyed on `(article_id, field)` (`field`: `status` | `is_favorite` | `scroll_position`; an archive is a `status: 'archived'` row), so repeats coalesce. `drain()` replays in `created_at` order and stops at the first row that cannot succeed: 2xx removes the row; a definitive 4xx other than 401 drops it; `NetworkError`/401/5xx/unrecognized errors keep it and bump `attempts` (no backoff or drop-after-N).
- **ArticleMutations** (`articleMutations.ts`) — what reading screens call for `markCompleted`, `markReading`, `saveScrollPosition`, `setFavorite`, `archive`: writes the store first, then the backend, and enqueues in `Outbox` on a retryable error (`isRetryable` in `outbox.ts`: `NetworkError`, `HttpError` 401/5xx). A definitive 4xx rethrows. Add-URL stays online-only.
- **SyncTrigger** (`syncTrigger.ts`) — `Outbox.drain()` then `ArticlePrefetchService.run()`, each isolated from the other's failure, single-flight. Fired by `useSyncTrigger` on offline→online and background→foreground *transitions*, and by `ReadsScreen` after every list sync.
- **Connectivity** — `useNetworkStatus()` (hook) and `isOffline()` (`utils/network.ts`, non-React code) wrap `expo-network`; unknown state counts as online.
- **Errors** — `NetworkError` (unreachable, timeout, unparseable body) vs `HttpError` (definitive non-2xx, has `status`), from `@cairn/shared`; offline logic keys off this distinction.

### AuthService (`src/services/auth.ts`)
Subclasses the shared `AuthService` and adds only mobile-specific parts: `getDeviceId()` (`expo-application`: `getIosIdForVendorAsync()` on iOS, `getAndroidId()` on Android), `loginWithDevice()`, `registerWithDevice()`, `upgradeAccount(email, password)` and `fetchWithAuthAndRetry()` (also retries 5xx/network errors). Everything else (`initialize`, `loginWithEmail`, `logout`, `getAccessToken`, `refreshAccessToken`, `fetchWithAuth`, `changePassword`, ...) lives in `apps/shared/src/services/auth.ts`. Session state is module-level there.

AsyncStorage keys: `@cairn:access_token`, `@cairn:refresh_token`, `@cairn:token_expires_at`, `@cairn:user`, `@cairn:server_url`, `@cairn:feed_cache`.

### ReadService (`src/services/read.ts`)
All calls use `AuthService.fetchWithAuth` against `${getServerUrl()}`. Methods and endpoints (all exist in the Go routers):

| Method | Endpoint |
|--------|----------|
| `listUserContents` | `GET /api/v1/content/user/{user_id}` (cursor pagination, `list` filter) |
| `addContentToUser`, `addURL` | `POST .../user/{user_id}` |
| `countUserContents` | `GET .../user/{user_id}/count` |
| `searchUserContents` | `GET .../user/{user_id}/search` |
| `getContentById` | `GET .../user/{user_id}/{content_id}` |
| `updateUserContent`, `moveToReads` | `PATCH .../user/{user_id}/{content_id}` (`moveToReads` sends `{ list: 'reads' }`) |
| `detectURL`, `discoverFeed` | `POST /api/v1/content/detect`, `/api/v1/content/discover-feed` |
| `listAllSubscriptions` | `GET .../user/{user_id}/subscriptions` |
| `unsubscribeFromRSSFeed` | `DELETE .../user/{user_id}/subscriptions/rss/{feed_id}` |
| `setSourceList(type, key, list)` | `PUT .../user/{user_id}/subscriptions/{type}/{key}/list` |
| `getOrCreateEmailAddress` | `POST /api/v1/source/email/user/{user_id}/address` |

Auth/user endpoints used by the shared and mobile `AuthService`: `/api/v1/auth/{login,register,login/mobile,register/mobile,refresh,logout}`, `POST /api/v1/user/{id}/upgrade`, `PUT /api/v1/user/{id}/password`. `SystemService.getServerVersion()` reads `version` from `GET /health/live`. See the OpenAPI specs in the root CLAUDE.md for request/response shapes.

## Configuration

- **Server URL**: a single backend origin. Default `DEFAULT_SERVER_URL` in `src/config/storage.ts` (`https://cairn.seatrain.net`); users override it at runtime from the Login screen (persisted under `@cairn:server_url`; switching server wipes the session keys). `src/config/init.ts` must be imported first in `index.js` so the shared layer is configured before use.
- **Local backend**: use the [self-hosted stack](/infrastructure/docker/selfhost/README.md), which serves all services on `http://localhost:8099`. iOS simulator: `http://localhost:8099`; Android emulator: `http://10.0.2.2:8099` (or the machine's LAN IP).
- **Expo**: `app.json` (bundle IDs `com.cairnapp.cairnreader`, plugins `expo-web-browser`, `expo-font`, `expo-sqlite`, translucent Android status bar). There are no `EXPO_PUBLIC_*` variables.

## Development Workflow

This is an npm workspace (`apps/web`, `apps/shared`, `apps/mobile`); run `npm install` from the repo root. In `apps/mobile`:

```bash
npm start            # expo start (press i / a / w for simulator / emulator / web)
npm run ios          # expo run:ios
npm run android      # expo run:android
npm run type-check   # tsc --noEmit
npm run lint         # eslint .
npm test             # jest (jest-expo preset, tests co-located as *.test.ts[x])
npm run test:ci      # jest --ci --coverage
```

Testing uses `@testing-library/react-native`; native modules are mocked in `jest.setup.ts` and `__mocks__/`. For general testing philosophy see [docs/TESTING.md](/docs/TESTING.md) and [Engineering Principles](/docs/ENGINEERING_PRINCIPLES.md).

## Conventions

- TypeScript strict mode; type props and function signatures; avoid `any`.
- Functional components with hooks; styles via `StyleSheet.create()` at the bottom of the file using theme constants (`Colors`, `Spacing`, `FontSizes`, `FontFamily`, `BorderRadius`, `Layout` from `src/constants/theme.ts`; get colors with `Colors[useColorScheme() ?? 'light']`). Test dark mode.
- Services: static classes/objects; log with `console.error`, show user-friendly messages, handle loading and error states, and throw `NetworkError`/`HttpError` (not bare `Error`) so offline logic can classify failures.
- Adding a screen: create it in `src/screens/`, export from `src/screens/index.ts`, register it in `RootNavigator.tsx` (or `TabNavigator.tsx`) and add its params to `src/types/navigation.ts`.

### Safe area (edge-to-edge)
Content scrolls behind the status bar and the floating tab bar/action menu. **Do not wrap screens in `SafeAreaView`**; use plain `View`s with `useSafeAreaInsets()`:
- Tab screens: top `insets.top + Spacing.md`; bottom `Layout.tabBarHeight + insets.bottom + Spacing.md`.
- Detail screens: bottom `Layout.bottomActionMenuHeight + insets.bottom + Spacing.md`.
- Modals: `paddingTop: insets.top`, `paddingBottom: insets.bottom`.
- Never hardcode bottom padding (e.g. `100`); scrollable content must always include it or it hides behind floating UI.

`App.tsx` configures `StatusBar` as translucent on Android; `app.json` also sets `androidStatusBar.translucent`.

## Troubleshooting

- Metro/bundle errors ("No bundle URL", "Unable to resolve module"): `npx expo start -c`; reinstall from the repo root if modules are missing.
- Auth errors: check the server URL (Login screen field or `DEFAULT_SERVER_URL`), that the backend is up, and clear `@cairn:*` AsyncStorage keys.
- Navigation errors: check `src/types/navigation.ts` and that the screen is registered in a navigator.
- General: [docs/TROUBLESHOOTING.md](/docs/TROUBLESHOOTING.md).

## Related Documentation

- **Root CLAUDE.md**: `/CLAUDE.md` - Project-wide guidance
- **Architecture**: `/docs/ARCHITECTURE.md`
- **Engineering Principles**: `/docs/ENGINEERING_PRINCIPLES.md` - Coding standards and architecture
- **Web App**: `/apps/web/CLAUDE.md` - Sibling client sharing `@cairn/shared`
- **User Service**: `/services/users/CLAUDE.md` - Authentication backend
- **Read Service**: `/services/read/CLAUDE.md` - Content storage backend
- **Expo Documentation**: https://docs.expo.dev/
