# Design tokens

`colors.json` is the single source of the Cairn color palette for every app. It uses the
[W3C Design Tokens](https://www.designtokens.org/tr/drafts/format/) format and defines a `light` and a `dark`
mode with the same token names (the build fails if they differ).

`build.js` ([Style Dictionary](https://styledictionary.com)) generates each app's color file:

| App | Generated file | Used as |
|-----|----------------|---------|
| Web | `apps/web/src/theme/colors.css` | `var(--color-text-secondary)`; dark values under `prefers-color-scheme: dark` |
| Mobile (React Native) | `apps/mobile/src/constants/colors.ts` | `Colors[useColorScheme() ?? 'light'].textSecondary` |

## Changing a color

1. Edit `colors.json` (both modes).
2. Run `npm run build -w @cairn/tokens` from the repo root.
3. Commit `colors.json` together with the regenerated files. CI (`.github/workflows/design-tokens.yml`) fails if
   the generated files do not match the source.

Never hardcode a color in an app. If no token fits, add one here.

## Not generated

- `apps/mobile/app.json` (`backgroundColor` for the splash and Android icon) is static Expo config and cannot import
  tokens. It holds a copy of `light.background`; update it by hand if that changes.
- `apps/ios/CairnReader/Resources/Assets.xcassets/LaunchBackground.colorset` (iOS launch screen) holds a copy of
  `light.background` and `dark.background`; update it by hand if those change.

## Native Swift

A future Swift app adds a platform to `build.js` (for example an asset-catalog colorset per token, which
switches light/dark natively) without changing `colors.json`.
