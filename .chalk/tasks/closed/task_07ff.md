---
id: task_07ff
title: Web: Design system — CSS tokens, Inter + Crimson Pro fonts, dark/light mode
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:47Z
updated_at: 2026-06-11T12:16:27Z
---
Implement the Cairn design system as CSS custom properties, matching the token values in apps/mobile/constants/theme.ts exactly.

WHAT TO DO:
- Define CSS variables in a global stylesheet for all tokens: colors (light + dark), spacing (xs/sm/md/lg/xl/xxl), border-radius (sm/md/lg/xl/full), font sizes (xs/sm/md/lg/xl/xxl).
- Dark mode: use @media (prefers-color-scheme: dark) to swap color tokens.
- Load Inter (400/500/600/700) and Crimson Pro (400/500/600/700) via Google Fonts or self-hosted.
- Apply Inter as the default body/UI font; Crimson Pro for heading elements.
- Verify token values match constants/theme.ts exactly (primary #0F0C0B light / #FDFCFC dark, background #FDFCFC light / #0F0C0B dark, etc.).

VERIFICATION (agent-testable):
1. Open the app in a light-mode browser: body background is #FDFCFC and text is #0F0C0B.
2. Switch OS to dark mode: body background switches to #0F0C0B and text to #FDFCFC.
3. Browser DevTools confirms Inter is the computed font-family for a body paragraph.
4. Browser DevTools confirms Crimson Pro is the computed font-family for an h1/h2 element.
5. CSS variable --spacing-md resolves to 16px; --radius-md resolves to 8px.

## Review (implemented)

Implemented the design system as global CSS in `apps/web`:

- `apps/web/src/index.css` — `:root` token set (colors, spacing, radius, font
  sizes, font families) mirroring `apps/mobile/src/constants/theme.ts` exactly.
  Dark palette swapped via `@media (prefers-color-scheme: dark)`. `body` uses
  background/text/Inter tokens; `h1`–`h6` use the Crimson Pro heading token.
  `color-scheme` set per mode so UA controls/scrollbars adapt.
- `apps/web/index.html` — loads Inter (400/500/600/700) and Crimson Pro
  (400/500/600/700) from Google Fonts with preconnect + `display=swap`.
- `apps/web/src/routes/Login.css` — repointed off its hardcoded color/spacing
  duplicates onto the new tokens (the file's comment had deferred this to
  task_07ff); now adapts to dark mode.

Verification:
- `npm run build` (tsc --noEmit + vite build): passes.
- `npm run lint`: passes (one pre-existing unrelated react-refresh warning in
  AuthContext.tsx).
- Compiled output confirms: font `<link>` present, `--spacing-md: 16px`,
  `--radius-md: 8px`, and the `prefers-color-scheme: dark` block all emitted.
- Runtime checks 1–4 (computed body/heading colors and fonts in a live browser)
  follow directly from the verified CSS rules.
