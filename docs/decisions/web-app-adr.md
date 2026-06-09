# ADR: Cairn Desktop Web App — Foundational Decisions

- **Status**: Accepted
- **Date**: 2026-06-09
- **Context**: Epic "Desktop web app" (`epic_f54e`). See
  [`docs/detailed_requirements/web_app_requirements.md`](../detailed_requirements/web_app_requirements.md).

Two decisions gate the web app scaffold and the tasks that depend on it
(`task_f734` project scaffold, `task_f61e` authentication). This ADR records both
choices and their rationale so subsequent tasks have a single source of truth.

---

## Decision 1 — Shared code strategy

**Decision: Extract an `apps/shared` workspace package.**

The framework-agnostic layers — `types/` (`article.ts`, `read.ts`, `auth.ts`), the
API/service logic (`auth`, `read`, `explore`, parameterized over a storage adapter),
and the theme token *values* — are extracted into an `apps/shared` package consumed by
both `apps/mobile` and `apps/web`.

### Options considered

- **A) Extract `apps/shared`** (chosen): a workspace package of types, service logic,
  and theme tokens imported by both clients.
- **B) Copy-and-adapt**: duplicate the shared files into `apps/web` and converge later.

### Rationale

- **Maximum reuse is the guiding principle** of the epic: the web app is a new
  presentation layer over the *same* backend contracts, not a new product. A single
  shared package makes the backend contracts, `Article` transforms, and design tokens
  literally one source of truth, so the two clients cannot drift on API shapes or
  token values.
- **Maintenance overhead** is lowest long-term: a change to a backend contract, a
  service transform, or a design token is made once and both clients pick it up,
  instead of being applied twice and policed for parity.
- **The shared surface is genuinely framework-agnostic**: the reuse table in the
  requirements doc marks `types/`, the service logic, and theme token values as
  "reuse as-is / reuse logic ~verbatim" — they carry no React Native dependency, so
  they belong in a neutral package rather than inside either client.

### Trade-offs accepted

- **Build tooling complexity**: this introduces a root workspace and converts the
  currently-standalone Expo app (`apps/mobile`, which today has its own
  `package-lock.json` and no root workspace) into a workspace member. Metro must be
  configured to resolve the `apps/shared` package (Expo supports monorepos
  officially; this is a one-time setup cost).
- **Mobile disruption risk**: touching the shipping mobile build to wire in the
  shared package carries some risk. This is mitigated by keeping the extraction
  mechanical (move framework-agnostic files unchanged, re-point imports) and by
  verifying the mobile build/tests after the move. The alternative (copy-and-adapt)
  was rejected because it trades this one-time, contained risk for ongoing drift and
  a "converge later" effort that tends not to happen.

### Consequences for downstream tasks

- The scaffold task (`task_f734`) consumes types and service logic from `apps/shared`
  rather than copying them into `apps/web`.
- The shared service/config layer is parameterized over a **storage adapter** so each
  client supplies its own backend: `AsyncStorage` on mobile, `localStorage` on web
  (see Decision 2).
- Theme token *values* live in `apps/shared`; each client maps them to its own
  styling system (RN `StyleSheet` on mobile, CSS custom properties on web).

---

## Decision 2 — Token storage

**Decision: Persist auth tokens in `localStorage`, with mandatory HTML sanitization.**

The web client stores `access_token`, `refresh_token`, `expires_at`, and the cached
`user` object in `localStorage` under the keys `@cairn:access_token`,
`@cairn:refresh_token`, `@cairn:token_expires_at`, and `@cairn:user` — the
browser-equivalent of mobile's `AsyncStorage` keys.

### Options considered

- **A) `localStorage`** (chosen): reuse the existing bearer-token flow; XSS-exposed,
  mitigated by mandatory sanitization of injected article HTML.
- **B) httpOnly cookie auth model**: tokens in httpOnly cookies; requires backend
  changes to set and clear cookies on the auth endpoints.

### Rationale

- **The existing services issue bearer tokens in JSON, not cookies.** The auth
  endpoints (`/api/v1/auth/login`, `/refresh`, `/logout`) return tokens in the
  response body and expect an `Authorization: Bearer <token>` header. `localStorage`
  reuses this contract verbatim; the mobile `AuthService` logic (proactive 5-minute
  buffer refresh, 401-retry) ports across with only the storage backend swapped.
- **httpOnly cookies would require backend changes**, which is out of scope: a stated
  non-goal of this epic is "no new backend… frontend-only effort; reimplementing or
  changing any backend service" is excluded.
- **CORS is already configured for bearer-token auth, not cookies.** The completed
  CORS work (`task_1318`) deliberately uses wildcard origin `*` with
  `AllowCredentials: false` — correct for `Authorization`-header requests. A cookie
  model needs `AllowCredentials: true` with an explicit (non-wildcard) origin, i.e.
  the very backend change ruled out above.
- The requirements doc names `localStorage` the **pragmatic reuse path** and the
  default for exactly these reasons.

### Trade-offs accepted

- **`localStorage` is readable by any script on the origin and is therefore exposed
  to XSS.** Because the app injects remote article HTML into the DOM (replacing
  mobile's RN HTML renderer / WebView), this risk is real and must be contained.
- **Mitigation is mandatory, not optional**: all remote HTML (`cleaned_html` for
  saved articles, the explore `content` field for recommendations) **must** be
  sanitized with DOMPurify before injection. This is a hard requirement on the
  article reader task (`task_e0f7`), not a nicety. Without sanitization this decision
  is not safe.

### Revisit criteria

If cookie-based auth is ever introduced backend-side, revisit this decision: switch
the web client to httpOnly cookies and switch CORS to an explicit origin with
`AllowCredentials: true` (wildcard `*` + credentials is rejected by browsers).
