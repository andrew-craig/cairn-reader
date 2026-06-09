---
id: task_07ff
title: Web: Design system — CSS tokens, Inter + Crimson Pro fonts, dark/light mode
type: task
status: open
priority: 2
labels: []
blocked_by: [task_f734]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:32:47Z
updated_at: 2026-06-08T11:32:47Z
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
