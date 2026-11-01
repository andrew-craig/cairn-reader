---
id: task_0bdf
title: Shared design tokens: single color palette for web and mobile
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-01T23:32:41Z
updated_at: 2026-11-01T23:39:41Z
---

## Context

Audit (2026-11-01) found no shared palette: mobile `constants/theme.ts` and web `index.css` held hand-copied
values that had drifted in dark mode (web warm hsl, mobile cool iOS grays), plus hardcoded colors in components.
Mobile will later migrate to native Swift, so the source of truth must be platform-neutral.

Decisions: dark mode uses the warm palette (web's values); border/hover/floating made warm to match.

## Plan

- [x] `design/tokens/colors.json` (W3C DTCG) as the single source, light + dark
- [x] Style Dictionary build (`@cairn/tokens` workspace) → web CSS vars + mobile TS `Colors`
- [x] Web: import generated CSS, drop hand-copied color vars and hardcoded rgba/fallbacks
- [x] Mobile: theme.ts re-exports generated Colors; replace hardcoded colors; drop unused tabIcon* tokens
- [x] CI check that generated files are up to date
- [x] Verify: type-check, lint, tests for web + mobile; knip

## Review

- Source of truth: `design/tokens/colors.json` (DTCG); `@cairn/tokens` workspace runs Style Dictionary 5 with two
  small custom formats (CSS with a `prefers-color-scheme: dark` block; TS `Colors` object). Build fails if light and
  dark define different token names. Generated files committed; `design-tokens.yml` CI fails on drift.
- New tokens replacing literals: `onAccent`, `floating`, `overlay`, `shadow`, `shadowSoft`. Removed unused
  `tabIconDefault`/`tabIconSelected`, the web `#22863a`/`#c0392b` fallbacks, and the icon `#0F0C0B` defaults
  (`color` is now required; every caller already passed one).
- Alignments: dark mode = web's warm palette (mobile changes visibly); dark border/hover/floating warmed
  (#3D3A34/#302D27/card@85%); web FAB hover uses `--color-hover` instead of black/white alpha; modal shadow uses the
  Figma slate base (#0F172A) at 12% instead of black.
- Selfhost Dockerfile now prunes `design/tokens` from the workspace list along with `apps/mobile`.
- Not tokenized: `app.json` splash background (static config, documented as a manual copy of light.background);
  `TopBlurGradient` mask `'black'`/`'transparent'` (alpha mask, not a rendered color).
- Verified: mobile tsc/eslint/jest (318)/knip; web tsc/eslint/vitest (58)/build; web login screenshot in both schemes;
  selfhost lockfile prune simulated with npm 11; web Docker scoped install simulated.
