---
id: task_a13d
title: [iOS native 02] Colours and typography: design-token colorsets and bundled brand fonts
type: task
status: open
priority: 2
labels: [ios,design]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T07:15:57Z
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
- [ ] Typography (decision 5: keep the brand fonts). Bundle the same faces the RN app loads (`FontFamily` in `theme.ts`): Inter 400/500/600/700, Crimson Pro 400/500/600/700, Atkinson Hyperlegible Next regular/italic/bold, with their OFL licence files, under `Resources/Fonts`, registered through `UIAppFonts`.
- [ ] One `Typography.swift`: a `Font` per role (body, medium, semibold, bold, heading, and so on) built with `Font.custom(_:size:relativeTo:)` so every role scales with Dynamic Type. Sizes start from `FontSizes` in `theme.ts`.
- [ ] Apply Inter once at the root with `.font(...)` so `Text`, `Label`, `Button`, `List` rows and `Form` controls inherit it without per-view modifiers; set the navigation bar title and tab bar item fonts once through `UINavigationBar` / `UITabBarItem` appearance in the app initialiser. No per-screen font wrappers and no custom bar components.
- [ ] Do not port `Spacing`, `BorderRadius`, `Layout` or `globalStyles.ts`: system list, form and bar metrics replace them.

## Verify
- `npm run build -w @cairn/tokens` is idempotent and the design-tokens workflow passes.
- A preview/test resolves every token name in both appearances.
- A test asserts every bundled PostScript font name used in `Typography.swift` loads (`UIFont(name:size:)` is non-nil), so a missing or misnamed font file fails CI instead of silently falling back to the system font.
