# Cairn Desktop Web App Requirements

## Overview

This document specifies requirements for a **desktop web application** for Cairn, a read-it-later product. The web app delivers the same core experience as the existing React Native/Expo mobile app (`apps/mobile`) — saving, reading, and discovering long-form content — optimized for desktop browsers and larger viewports.

The guiding principle is **maximum reuse of the mobile app**: its feature set, backend service contracts, data models, service-layer logic, design tokens, and information architecture should be carried over directly. The web app is a new **presentation layer** over the same backend, not a new product.

### Goals

- Provide a full-featured reading experience in the browser on desktop (and degrade gracefully to tablet/mobile widths).
- Reach feature parity with the mobile app's authenticated experience.
- Reuse the mobile app's backend integration contracts and data transforms verbatim wherever possible, so a single backend serves both clients.
- Match the mobile app's visual identity (typography, color, spacing) so the two clients feel like one product.

### Non-Goals (initial release)

- Offline-first / PWA installability
- Push notifications
- A separate web backend or BFF — the web app talks to the **existing** services.
- Reimplementing or changing any backend service. This is a frontend-only effort.
- Device-ID anonymous login (not viable on web).

## Reuse Strategy (Borrow From Mobile)

The mobile app lives at `apps/mobile` and is organized as:
`config/`, `constants/` (theme), `contexts/` (AuthContext), `services/`
(auth, read, explore, storage), `types/`, `screens/`, `components/`, `navigation/`.

What carries over to the web app, and how:

| Mobile layer | Reuse on web | Notes |
|---|---|---|
| `services/auth.ts`, `services/read.ts`, `services/explore.ts` | **Reuse logic ~verbatim** | Pure `fetch` + JSON; only the storage backend and device-ID paths change. |
| `types/` (`article.ts`, `read.ts`, `auth.ts`) | **Reuse as-is** | Plain TypeScript interfaces with no RN dependency. |
| `config/api.ts` | **Reuse, swap storage** | Replace `AsyncStorage` with `localStorage`. |
| `constants/theme.ts` | **Reuse token values** | Same colors/spacing/fonts; map to CSS/web styling. |
| `contexts/AuthContext.tsx` | **Reuse pattern** | Same context shape; web equivalent. |
| Screens & components (`*.tsx` using `react-native`) | **Re-implement** | Rebuilt with web primitives; same layout, copy, and behavior. |
| `navigation/` (React Navigation) | **Replace** | Use a web router; preserve the same screen graph. |

### Shared code recommendation

Extract the framework-agnostic layers — `types/`, the API/service logic, and the theme token values — into **Shared package**: a `apps/shared` workspace containing types, service logic (parameterized over a storage adapter), and theme tokens, imported by both `apps/mobile` and `apps/web`.

## Technology Stack

- **Framework**: React (matches mobile's React/TSX skill set and component model).
- **Language**: TypeScript, strict mode (matches mobile conventions).
- **Build/dev**: Vite (fast dev server, simple SPA build). Next.js is an alternative only if SSR/SEO of public pages becomes a requirement — not needed for an authenticated reader.
- **Routing**: A standard client-side router (e.g. React Router) replacing React
  Navigation. The screen graph below maps 1:1 to routes.
- **Styling**: Reuse the mobile theme token *values*. implemented in CSS. the requirement is that the tokens (color, spacing, font, radius) match `constants/theme.ts`
- **HTML article rendering**: Render `cleaned_html` directly in the DOM, replacing the mobile `react-native-render-html` / WebView approach. Content **must be sanitized** before injection (see Security).
- **Fonts**: Inter (body/UI) and Crimson Pro (headings), matching mobile.

The app is a **single-page authenticated client** that calls the existing Cairn
backend over REST with JWT bearer auth.

## Architecture

```
Desktop Browser (React SPA)
  → REST + JWT bearer  →  Existing Cairn backend (cairn.seatrain.net by default)
                          ├── User Service   (auth, account)
                          ├── Explore Service (recommendations, votes)
                          └── Read Service    (content, subscriptions, newsletters)
```

- The web app uses the **same base URL and endpoints** as mobile (default
  `https://cairn.seatrain.net`, configurable). All endpoints are unchanged.
- Authentication is JWT bearer (RS256), identical to mobile: access token + refresh
  token, with proactive refresh and 401-retry fallback.
- No new backend, no new database access. The web app is a pure API consumer.

### CORS dependency

Browsers enforce CORS; the mobile app (native `fetch`) does not. The backend services **must** return appropriate `Access-Control-Allow-Origin` (and related) headers for the web app's origin(s), and handle preflight `OPTIONS` for the authenticated `Authorization`-header requests. If CORS is not already configured for a browser origin, that is a backend prerequisite and must be tracked as a dependency (see Dependencies). This is the single most likely integration blocker.

## Authentication Requirements

The web app reuses the mobile `AuthService` logic and the same User Service endpoints, with one deliberate difference: **no device-ID login**.

### Supported flows

1. **Email/password login** — `POST /api/v1/auth/login`.
2. **Email/password registration** — `POST /api/v1/auth/register`.
3. **Token refresh** — `POST /api/v1/auth/refresh`, proactive (5-minute expiry buffer) plus reactive on 401, mirroring `AuthService.ensureValidToken` / `refreshAccessToken`.
4. **Logout** — `POST /api/v1/auth/logout` (revokes refresh token), then clear local tokens.
5. **Change password** — `PUT /api/v1/user/{id}/password`.

### Explicitly excluded

- **Device-ID / anonymous login** (`/auth/login/mobile`, `/auth/register/mobile`). These depend on `expo-application` device identifiers that don't exist in a browser. Web users authenticate with email/password only.
- **Account upgrade** (`POST /api/v1/user/{id}/upgrade`, device→email) is not applicable since web has no anonymous accounts. A web user who started anonymously on mobile can sign in on web once they have upgraded their mobile account to email/password.

### Token storage

- Tokens (`access_token`, `refresh_token`, `expires_at`) and the cached `user`
  object are persisted in the browser. Replace mobile's `AsyncStorage` keys with the
  equivalent `localStorage` keys (`@cairn:access_token`, `@cairn:refresh_token`,
  `@cairn:token_expires_at`, `@cairn:user`).
- **Security note**: `localStorage` is readable by any script on the origin and is
  therefore exposed to XSS. Given the app injects remote article HTML, sanitization
  (see Security) is mandatory. The trade-off vs. httpOnly cookies should be recorded
  as an explicit decision; `localStorage` is the pragmatic reuse path because the
  existing services issue bearer tokens in JSON rather than setting cookies.
- The auth state must be exposed via an `AuthContext` equivalent with the same shape
  as mobile (`user`, `isAuthenticated`, `isLoading`, `login`, `logout`) and the same
  "clear tokens → broadcast logout" listener behavior.

### Unauthenticated state

When not authenticated, the app shows the login/register screen (mirroring
`RootNavigator`, which renders `LoginScreen` when `!isAuthenticated`). All
authenticated routes redirect to login when there is no valid session.

## Information Architecture & Navigation

The mobile app uses a bottom tab bar (`Read`, `Explore`, `You`) plus a stack of
detail/secondary screens. The web app preserves the **same screen graph** but maps
the primary navigation to a persistent **left sidebar** (or top nav bar) with the three primary destinations: **Read**, **Explore**, **You**. This replaces `CustomTabBar`.

The top quick actions bars (add, search) will remain at the top of the list
- There will be a near full width search bar across the width of the read article list
- Non-search actions will be buttons (in the same style) to the right of the search box

The sub-sections of the You page (Account, Feeds, Newsletters), will be shown as sub-items in the left navigation menu. 
- Clicking the You item in the left nav will expand this list
- Clicking on an individual item (e.g. Account) will open a page for that account

### Route map

| Route | Screen (mobile equivalent) | Purpose |
|---|---|---|
| `/login` | `LoginScreen` | Email/password login & registration |
| `/read` | `ReadScreen` | The user's saved reading list (default landing) |
| `/explore` | `ExploreScreen` | Recommended articles to discover |
| `/you` | `YouScreen` | Account hub with counts & links |
| `/read/:id` | `ReadArticleDetailScreen` | Full article reader (saved item) |
| `/explore/:id` | `ExploreArticleDetailScreen` | Full article reader (recommendation) |
| `/you/account` | `AccountScreen` | Profile, change password, log out |
| `/you/feeds` | `FeedsScreen` | RSS/feed subscriptions management |
| `/you/newsletters` | `NewslettersScreen` | Newsletter (email-ingest) address & subs |
| `/you/bookmarks` | `BookmarksScreen` | Saved/bookmarked items |
| `/you/votes` | `VotesScreen` | Articles the user up/down-voted |
| Add-link modal | `AddArticleScreen` / `AddLinkModal` | Add a URL (page or feed) |
| Search modal | `SearchModal` | Search saved content |

Add-link and Search present as **modal/overlay** on web, matching mobile. Implementation of these should be light-weight as they will be improved as a fast follow.

## Functional Requirements

All endpoints below already exist and are consumed by the mobile service layer; the
web app reuses the same request/response handling and `Article` transforms.

### FR-1: Reading List (Read)

- Display the user's saved content via `GET /api/v1/content/user/{userId}` with
  cursor-based pagination (`ReadService.listUserContents`).
- Support **infinite scroll / "load more"** using the returned `cursor` / `has_more`
  (mobile `PAGE_SIZE = 20`).
- Each item renders title, source/author, description/excerpt, lead image, and
  estimated reading time (`word_count / 200`), reusing the `transformToArticle`
  mapping.
- Provide a **refresh** action that reloads the first page.
- Clicking an item opens the article reader (`/read/:id`), passing along the current
  list and index to enable in-reader navigation (mobile passes `{ article, articles,
  currentIndex }`).
- Reflect changes made elsewhere (archive, favorite) on return to the list, matching
  mobile's focus-reload behavior.

### FR-2: Search

- Search saved content via `GET /api/v1/content/user/{userId}/search?q=...`
  (`ReadService.searchUserContents`).
- Present as a modal/overlay; show results in the same list layout; provide a clear
  action to return to the full list.

### FR-3: Add Link

- Reuse the mobile add flow (`AddLinkModal` + `ReadService.detectURL` /
  `discoverFeed` / `addURL`):
  - User enters a URL.
  - Optionally call `POST /api/v1/content/detect` (10s timeout) and
    `POST /api/v1/content/discover-feed` (15s timeout) to classify the URL as a
    **page** vs **feed** and surface discovered feeds.
  - Submit via `POST /api/v1/content/user/{userId}` (`addURL`), which returns either
    a saved page (`type: 'page'`) or a created feed subscription (`type: 'feed'`).
- On success, refresh the reading list.

### FR-4: Article Reader

- Render the article from the `Article` model. Saved articles use
  `content.cleaned_html`; recommendations use the explore `content` field.
- **Render sanitized HTML natively in the DOM** (replacing mobile's RN HTML renderer
  / WebView). See Security for sanitization requirements.
- Reading experience requirements:
  - Comfortable measure (max content width ~65–75ch), Crimson Pro headings, Inter
    body, generous line height — a desktop reading column, centered.
  - Display title, author/site, published date, and reading time.
  - **Reading progress / scroll position**: persist scroll position via
    `PATCH /api/v1/content/user/{userId}/{contentId}` (`scroll_position`) and restore
    it on reopen, mirroring mobile.
  - **Status transitions**: mark `reading` / `completed` / `archived` via the same
    PATCH endpoint (`status`).
  - **Favorite / unfavorite** via the same PATCH (`is_favorite`).
  - **Delete** a saved item via `DELETE /api/v1/content/user/{userId}/{contentId}`.
  - **Open original** in a new tab (the `original_url`).
  - **Previous/next** navigation through the list passed from the Read view.
- For recommendation articles (`/explore/:id`): support **mark as read**
  (`POST /api/v1/explore/article/{id}/read`) and **up/down vote** (see FR-6), plus
  saving the article to the reading list.

### FR-5: Explore (Discovery)

- Fetch recommendations via `GET /api/v1/explore/recommendation?offset=...`
  (`ExploreService.getRecommendations`), paginated by offset.
- Display recommended articles in a card layout suited to desktop (e.g. a responsive
  grid or a centered feed column).
- Best-effort **"shown" reporting** via `POST /api/v1/explore/shown` (batched
  article IDs), matching mobile telemetry behavior.
- Support **up/down vote** directly from the feed and from the reader (FR-6).

### FR-6: Voting

- Up/down vote and remove vote on recommendation articles:
  - `POST /api/v1/explore/article/{id}/vote` with `{ vote_type: 'upvote' | 'downvote' }`.
  - `DELETE /api/v1/explore/article/{id}/vote` to remove.
  - `GET /api/v1/explore/article/{id}/vote` for counts and the user's current vote.
- Optimistic UI for vote toggles is acceptable, matching the responsive feel of
  mobile.

### FR-7: You (Left Nav)

- Clicking You in the Left nav will show the `YouScreen` items as a sub-menu **Account**, **Feeds**, **Newsletters**, **Bookmarks**, **Votes**, and **Log out**.
- Where appropriate, these items will have a count next to them in the sidebar
- Counts are fetched in parallel with **independent failure tolerance**
  (`Promise.allSettled`), so one failing endpoint doesn't hide the others:
  - Vote stats — `ExploreService.getUserVoteStats`.
  - Subscriptions (feeds vs newsletters split by `type === 'email'`) —
    `ReadService.listAllSubscriptions` (`GET /api/v1/content/user/{userId}/subscriptions`).
  - Bookmarks total — `ReadService.listUserContents({ limit: 1 })` total count.
- Refresh counts when re-opening the sub-menu

### FR-8: Feeds (Subscriptions)

- List the user's feed subscriptions via the unified subscriptions endpoint
  (`listAllSubscriptions`), filtering to non-email types.
- **Unsubscribe** from an RSS feed via
  `DELETE /api/v1/content/user/{userId}/subscriptions/rss/{feedId}`
  (existing items in the reading list are preserved).

### FR-9: Newsletters (Email Ingest)

- Show the user's newsletter ingest email address, creating it on demand via
  `POST /api/v1/source/email/user/{userId}/address` (`getOrCreateEmailAddress`).
- Provide **copy to clipboard** for the address (web Clipboard API; mobile uses
  `expo-clipboard`).
- List email-type subscriptions from `listAllSubscriptions`.

### FR-10: Bookmarks

- List bookmarked/saved items (mobile `BookmarksScreen`) using the content list
  endpoints, in the same list layout as Read.

### FR-11: Account

- Show profile info (email or "Anonymous user").
- **Change password** via `PUT /api/v1/user/{userId}/password`
  (`current_password`, `new_password`).
- **Log out**.

## Data Models

Reuse the mobile types unchanged. Core model (`types/article.ts`):

```typescript
interface Article {
  id: string;
  url: string;
  title: string;
  description?: string;
  content?: string;       // cleaned HTML
  imageUrl?: string;
  author?: string;
  publishedDate?: string;
  readingTime?: number;
  tags: string[];
  isRead: boolean;
  isFavorite: boolean;
  addedAt: number;
  readAt?: number;
  notes?: string;
  scrollPosition?: number;
}
```

Read service types (`types/read.ts`) — `UserContentResponse`, `ContentResponse`,
`UnifiedSubscription`, `DetectURLResponse`, `AddURLResponse`, etc. — and auth types
(`types/auth.ts`) are reused verbatim. The `transformToArticle` (read) and
`transformArticle` (explore) mappings carry over directly.

## Design System

Match the mobile visual identity by reusing the token *values* from
`constants/theme.ts`.

### Color (light / dark)

| Token | Light | Dark |
|---|---|---|
| primary | `#0F0C0B` | `#FDFCFC` |
| background | `#FDFCFC` | `#0F0C0B` |
| card / floating | `#FBFAF9` | `#1C1C1E` |
| text | `#0F0C0B` | `#FDFCFC` |
| textSecondary | `#696563` | `#8E8E93` |
| border / line | `#F1EFEE` | `#38383A` |
| hover | `#EDEAE9` | `#2C2C2E` |
| error | `#C63E06` | `#FF453A` |
| success | `#34C759` | `#30D158` |
| warning | `#FF9500` | `#FF9F0A` |

### Typography

- **Body/UI**: Inter (400/500/600/700).
- **Headings**: Crimson Pro (400/500/600/700).
- Font sizes: `xs 12, sm 14, md 16, lg 18, xl 24, xxl 32`. Web may introduce
  additional larger heading sizes for the desktop reading column, but base tokens
  match mobile.

### Spacing & radius

- Spacing scale: `xs 4, sm 8, md 16, lg 24, xl 32, xxl 48`.
- Border radius: `sm 4, md 8, lg 12, xl 16, full 9999`.

### Dark mode

- Follow system preference by default (mobile uses `useColorScheme`). A manual
  light/dark toggle is a desirable addition for desktop but optional for v1.

### Desktop layout adaptations (replacing mobile safe-area model)

The mobile app's entire safe-area strategy (`useSafeAreaInsets`, floating tab bar,
edge-to-edge scrolling) is **mobile-specific and does not carry over**. The web app
instead must:

- Use a responsive layout with a persistent sidebar/top nav on desktop widths.
- Constrain reading content to a centered, readable column.
- Use a multi-column or grid layout for list/discovery views where horizontal space
  allows, collapsing to a single column at narrow widths.
- Provide standard desktop affordances: hover states (the theme already defines a
  `hover` color), keyboard focus rings, and scrollbars.

## Non-Functional Requirements

### Responsive design

- **Primary target**: desktop browsers (≥1024px), the focus of this effort.
- Must remain usable down to tablet (~768px)
- At mobile widths, mirror the mobile app more closely. Note this will require a `You` page or modal that is only shown at mobile widths
- Latest evergreen browsers (Chrome, Firefox, Safari, Edge).

### Performance

- Initial authenticated view (Read list first page) should render quickly; lists use
  pagination/virtualization as needed for long lists.
- Article HTML rendering should not block interaction; large articles render
  progressively where feasible.

### Accessibility

- Keyboard navigable throughout (nav, lists, reader controls, modals).
- Semantic HTML for article content and controls; visible focus states; sufficient
  contrast (the token palette is high-contrast by design).
- Reader supports browser zoom and respects user font-size preferences.

### Security

- **HTML sanitization is mandatory.** Article `cleaned_html` and any
  description/excerpt HTML originate from remote sources and are injected into the
  DOM; they must be sanitized (e.g. DOMPurify) before rendering to prevent XSS.
  External links open with `rel="noopener noreferrer"`; consider opening in a new
  tab.
- Tokens in `localStorage` are XSS-exposed — sanitization above is the primary
  mitigation. Record the `localStorage`-vs-cookie decision explicitly.
- All API calls require `Authorization: Bearer <token>`; never log full tokens (the
  mobile service logs only short previews — match that).
- Backend **CORS** must allow the web origin (see Architecture → CORS dependency).

### Error handling

- Match mobile patterns: try/catch around async calls, user-friendly error messages
  (no stack traces), graceful network-error handling.
- On refresh failure / expired session, clear tokens and route to login (mirroring
  `ensureValidToken` → logout broadcast).

### Configuration

- Server base URL configurable, defaulting to `https://cairn.seatrain.net`
  (mobile `DEFAULT_SERVER_URL`), with the same normalization rules (`config/api.ts`).
  Expose this in settings for self-hosting parity, persisted in `localStorage`.

## Testing Requirements

- **Unit tests** for reused service logic and transforms (mobile already tests
  `read`, `storage`, `helpers`; web should retain/port these).
- **Component tests** for the reader, list, add-link, and search flows.
- Type checking (`tsc --noEmit`) and linting must pass, matching mobile's
  `type-check` / `lint` gates.
- Manual verification of auth, save, read (with scroll persistence), archive,
  favorite, delete, vote, subscribe/unsubscribe, and newsletter-address flows
  against a running backend.

## Project Structure (proposed)

Add the web app alongside the mobile app:

```
apps/
  mobile/   # existing React Native/Expo app
  web/      # new React (Vite) desktop web app
    src/
      config/        # api base URL (localStorage-backed)
      constants/     # theme tokens (shared with mobile or mirrored)
      contexts/      # AuthContext (web)
      services/      # auth, read, explore (reused logic, web storage adapter)
      types/         # reused from mobile / shared package
      routes/        # route-level views (Read, Explore, You, Reader, ...)
      components/     # web UI components (ArticleCard, ArticleRow, Modal, ...)
```

If the shared-package approach is chosen, `types/`, `services/` logic, and theme
tokens move to `packages/shared` and both apps import them.

## Dependencies & Prerequisites

1. **Backend CORS** configured for the web origin(s) (preflight + `Authorization`).
   Hard blocker if absent.
2. Confirmation that existing auth/content/explore endpoints are reachable from a
   browser origin with bearer tokens (no mobile-only assumptions).
3. Decision on shared-code strategy (shared package vs. copy-and-adapt).
4. Fonts (Inter, Crimson Pro) available for web (Google Fonts or self-hosted).

## Out of Scope / Future Work

- PWA / offline reading and installability.
- Push/web notifications.
- SSR/SEO for public or shared article pages.
- Browser extension "save to Cairn" (could later reuse the add-URL flow).
- Highlights/annotations (the model has a `notes` field but no current UI).
- Manual dark-mode toggle (optional for v1).

## Open Questions

1. **Shared code**: extract a `packages/shared` now, or copy-and-adapt into
   `apps/web` and converge later?
2. **Token storage**: accept `localStorage` (reuse path, XSS-exposed) or invest in an
   httpOnly-cookie auth model (requires backend changes)?
3. **CORS**: is the backend already configured for any browser origin, or is this net
   new work to schedule first?
4. **Default landing route**: `/read` (parity with mobile's primary list) — confirm.
5. **Reader navigation**: replicate mobile's pass-the-list prev/next, or fetch
   neighbors on demand from the reader route?
