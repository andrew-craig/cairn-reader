---
id: task_a13d
title: [iOS native 02] Colours and typography: design-token colorsets and bundled brand fonts
type: task
status: closed
priority: 2
labels: [ios,design]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T08:51:16Z
---
## Goal
The Swift app reads the same palette as web, switching light/dark natively. No colour literal in Swift code.

## Source
`design/tokens/build.js`, `design/tokens/README.md` (already anticipates a Swift platform), `apps/mobile/src/constants/theme.ts`.

## Plan
- [x] Add an `ios` platform to `build.js` that writes one `.colorset` per token (light + dark appearance) into `apps/ios/CairnReader/Resources/Colors.xcassets`. Xcode's generated asset symbols then give `Color(.textSecondary)` with no hand-written accessor file.
- [x] Set `AccentColor` from the `primary` token so system controls tint correctly.
- [x] Extend `.github/workflows/design-tokens.yml` (paths and the `git diff --exit-code` list) to cover the generated colorsets.
- [x] Update `design/tokens/README.md` table.
- [x] Typography (decision 5: keep the brand fonts). Bundle the same faces the RN app loads (`FontFamily` in `theme.ts`): Inter 400/500/600/700, Crimson Pro 400/500/600/700, Atkinson Hyperlegible Next regular/italic/bold, with their OFL licence files, under `Resources/Fonts`, registered through `UIAppFonts`.
- [x] One `Typography.swift`: a `Font` per role (body, medium, semibold, bold, heading, and so on) built with `Font.custom(_:size:relativeTo:)` so every role scales with Dynamic Type. Sizes start from `FontSizes` in `theme.ts`.
- [x] Apply Inter once at the root with `.font(...)` so `Text`, `Label`, `Button`, `List` rows and `Form` controls inherit it without per-view modifiers; set the navigation bar title and tab bar item fonts once through `UINavigationBar` / `UITabBarItem` appearance in the app initialiser. No per-screen font wrappers and no custom bar components.
- [x] Do not port `Spacing`, `BorderRadius`, `Layout` or `globalStyles.ts`: system list, form and bar metrics replace them.

## Verify
- `npm run build -w @cairn/tokens` is idempotent and the design-tokens workflow passes.
- A preview/test resolves every token name in both appearances.
- A test asserts every bundled PostScript font name used in `Typography.swift` loads (`UIFont(name:size:)` is non-nil), so a missing or misnamed font file fails CI instead of silently falling back to the system font.

## Review
- `build.js` gains an `ios` platform: a Style Dictionary action that recreates `apps/ios/CairnReader/Resources/Colors.xcassets` with one colorset per token (light, plus a dark luminosity appearance; alpha from `#RRGGBBAA`). Recreated on every build, so removed tokens disappear. Running the build twice leaves the tree unchanged, and the web/mobile outputs are unchanged.
- `primary` clashes with SwiftUI's `Color.primary` in the generated `Color.<name>` extensions (a build error), so `ASSETCATALOG_COMPILER_GENERATE_SWIFT_ASSET_SYMBOL_EXTENSIONS = NO`. The `ColorResource` symbols remain: `Color(.textSecondary)`. Token names stay the same as on web.
- Accent: `ASSETCATALOG_COMPILER_GLOBAL_ACCENT_COLOR_NAME = primary` (points at the generated colorset; no duplicate `AccentColor`).
- LaunchBackground: the hand copy and `Assets.xcassets` (it held nothing else) are gone; `UILaunchScreen.UIColorName` is `background`, from the generator. The README "Not generated" entry and the apps/ios/CLAUDE.md note are removed.
- design-tokens.yml: paths cover the colorsets; the check uses `git status --porcelain` so added or removed colorsets fail too, not just modified ones; also runs on PRs into `feature/ios-native-migration`, like ios-checks.
- Fonts: the exact static TTFs the RN app loads, from the `@expo-google-fonts/*` packages (Inter 0.4.2, Crimson Pro 0.4.2, Atkinson Hyperlegible Next 0.4.1), with each package's OFL `LICENSE_FONT` saved as `<Family>-OFL.txt` (bundled alongside). Registered in Info.plist `UIAppFonts` (no build setting).
- `App/Typography.swift`: `BrandFont` (PostScript names, `CaseIterable`) and `Typography` roles (largeTitle/title/heading in Crimson Pro; body/bodyMedium/bodySemiBold/bodyBold/subheadline/caption in Inter; content/contentItalic/contentBold in Atkinson), all `Font.custom(_:size:relativeTo:)`. `Typography.body` is applied at the root; `applyBarAppearance()` (called from `CairnReaderApp.init`) sets the nav bar title/large title and tab bar item fonts, scaled through `UIFontMetrics`.
- Tests (`DesignTokenTests`): every token in `colors.json` (read from the repo via `#filePath`, so there is no hand-kept list) resolves with its exact light and dark hex; every `BrandFont` loads with `UIFont(name:size:)`. Changing a token without regenerating makes the token test fail (checked, then reverted).
- Verified locally (Xcode 26.6, iPhone 17 Pro iOS 26.2, CI flags): `** TEST SUCCEEDED **`, 3 tests. The built bundle has the 11 TTFs, the OFL files, `NSAccentColorName = primary` and `UILaunchScreen` → `background`; the app launches. The placeholder `ContentUnavailableView` keeps its own system title font; real screens inherit Inter.
- Not yet verified: the GitHub runs of design-tokens and ios-checks (they run when a PR into the feature branch is opened).
