# Cairn Web App — Agent Notes

## Test account

A reusable account for verifying authenticated flows against the default backend:

- **Email**: `cairn.web.test@seatrain.net`
- **Password**: `CairnWebTest!2026`

Seeded data: 2 RSS feed subscriptions (Hacker News, The Verge) and 1 saved page.

## Backend URL in development

The app defaults to the origin it is served from. Under `npm run dev` that origin
has no API, so set `VITE_API_URL` (copy `.env.example` to `.env`) to point at a
backend, e.g. `VITE_API_URL=http://localhost:8099`.

## Routes

Tabs are Feed | Reads | You (`src/App.tsx`). `/feed` and `/reads` render the same `ArticleList` with `list="feed"` or
`list="reads"`; items open at `/feed/:id` and `/reads/:id`. Under `/you`: `rss`, `newsletters`, `bookmarks`, `account`
and `about`. The RSS and Newsletters pages choose Feed or Reads per source (`SourceListToggle`); a change applies to
new items only. Directly saved links always go to Reads.

## Commands

Run from the repo root with `-w @cairn/web` (npm workspace), or from `apps/web`: `npm run dev`, `build`
(`tsc --noEmit && vite build`), `type-check`, `test` (`vitest run`), `lint`, `preview`.

## Layout and shared code

- `src/App.tsx` (router, `RequireAuth`), `src/main.tsx` (wires storage and default server URL into `@cairn/shared`)
- `src/routes/` pages, `src/components/`, `src/contexts/AuthContext.tsx`, `src/utils/` (`sanitize.ts` uses DOMPurify)
- `src/services/read.ts` (`ReadService`, mirrors `apps/mobile/src/services/read.ts`), `storage.ts` (localStorage
  adapter), `system.ts` (server version from `/health/live`)
- Auth, server-URL config, API types and HTTP helpers come from `apps/shared` (`@cairn/shared`, consumed from source
  via the alias in `vite.config.ts`; no build step). Authenticated calls go through `AuthService.fetchWithAuth`.

## Testing

Vitest with jsdom and Testing Library (`vitest.config.ts`, `vitest.setup.ts`); tests are colocated as `*.test.ts(x)`.
To drive the app in a browser against a remote backend, use the `webapp-staging-test` skill
(`.agents/skills/webapp-staging-test/SKILL.md`).

## Build and serve

`Dockerfile` builds the SPA (workspace `web` + `shared` only) and serves `dist` with Caddy on port 8080 using
`Caddyfile` (SPA fallback to `index.html`, `/healthz`). `/api/*` routing and TLS are done by the front proxy.

## Design decisions

See [docs/decisions/web-app-adr.md](/docs/decisions/web-app-adr.md) and
[docs/detailed_requirements/web_app_requirements.md](/docs/detailed_requirements/web_app_requirements.md).
