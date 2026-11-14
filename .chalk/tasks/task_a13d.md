---
id: task_a13d
title: [iOS native 02] Colours and typography: asset-catalog output from design tokens
type: task
status: open
priority: 2
labels: [ios,design]
blocked_by: [task_8792]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
The Swift app reads the same palette as web, switching light/dark natively. No colour literal in Swift code.

## Source
`design/tokens/build.js`, `design/tokens/README.md` (already anticipates a Swift platform), `apps/mobile/src/constants/theme.ts`.

## Plan
- [ ] Add an `ios` platform to `build.js` that writes one `.colorset` per token (light + dark appearance) into `apps/ios/CairnReader/Resources/Colors.xcassets`. Xcode's generated asset symbols then give `Color(.textSecondary)` with no hand-written accessor file.
- [ ] Set `AccentColor` from the `primary` token so system controls tint correctly.
- [ ] Extend `.github/workflows/design-tokens.yml` (paths and the `git diff --exit-code` list) to cover the generated colorsets.
- [ ] Update `design/tokens/README.md` table.
- [ ] Typography (decision 5): no font constants. UI uses system text styles (`.headline`, `.subheadline`, ...). Bundle Atkinson Hyperlegible Next (regular, italic, bold; OFL licence file alongside) under `Resources/Fonts` for the reader template only.
- [ ] Do not port `Spacing`, `BorderRadius`, `Layout` or `globalStyles.ts`: system list, form and bar metrics replace them.

## Verify
- `npm run build -w @cairn/tokens` is idempotent and the design-tokens workflow passes.
- A preview/test resolves every token name in both appearances.
