# Design tokens

`colors.json` is the single source of the Cairn color palette for every app. It uses the
[W3C Design Tokens](https://www.designtokens.org/tr/drafts/format/) format and defines a `light` and a `dark`
mode with the same token names (the build fails if they differ).

`build.js` ([Style Dictionary](https://styledictionary.com)) generates each app's color file:

| App | Generated file | Used as |
|-----|----------------|---------|
| Web | `apps/web/src/theme/colors.css` | `var(--color-text-secondary)`; dark values under `prefers-color-scheme: dark` |
| Mobile (React Native) | `apps/mobile/src/constants/colors.ts` | `Colors[useColorScheme() ?? 'light'].textSecondary` |
| iOS (native) | `apps/ios/CairnReader/Resources/Colors.xcassets` (one colorset per token) | `Color(.textSecondary)`; light/dark switch natively. The app's accent colour is `primary` and its launch screen background is `background` |

## Changing a color

1. Edit `colors.json` (both modes).
2. Run `npm run build -w @cairn/tokens` from the repo root.
3. Commit `colors.json` together with the regenerated files. CI (`.github/workflows/design-tokens.yml`) fails if
   the generated files do not match the source.

Never hardcode a color in an app. If no token fits, add one here.

## Not generated

- `apps/mobile/app.json` (`backgroundColor` for the splash and Android icon) is static Expo config and cannot import
  tokens. It holds a copy of `light.background`; update it by hand if that changes.
