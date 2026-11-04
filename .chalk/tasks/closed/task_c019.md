---
id: task_c019
title: Warm Leather Binding color palette
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-04T21:39:48Z
updated_at: 2026-11-04T21:40:26Z
---


Apply the "Leather Binding" palette (option 2 from the warm-palette exploration): a subtle tan/russet tint on the
neutral tokens, inspired by aged paper and leather book covers. Accent colors (success, info, error, warning) unchanged.

## Plan
- [x] Update neutral tokens in `design/tokens/colors.json` (light + dark), with hover desaturated ~30% from the proposal
      (light `#EBE2DC` → `#E9E2DE`, dark `#332A25` → `#312B27`)
- [x] Replace slate shadows (`#0F172A`) with warm brown (`#2B150D`)
- [x] Regenerate web/mobile color files (`npm run build -w @cairn/tokens`)
- [x] Update `apps/mobile/app.json` splash/icon background to new `light.background`
- [x] Verify contrast: all text pairs ≥ 4.5:1 (lowest: light textSecondary on hover, 4.6:1)

## Review
Tokens only; no component changes. Token descriptions still cite Figma "red/*" names, which no longer match these values.
