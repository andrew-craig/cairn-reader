---
id: task_0c2b
title: Mobile: use Icon Composer app icon in iOS builds
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-07T06:16:50Z
updated_at: 2026-11-07T07:18:25Z
---


## Plan
- [x] Add the Icon Composer bundle at `apps/mobile/assets/CairnIcon.icon`
- [x] Set `ios.icon` in `app.json` so `expo prebuild --clean` (TestFlight workflows) picks it up
- [x] Apply only the icon entries to the tracked `apps/mobile/ios` project
- [x] Confirm a built app contains the icon (Assets.car)
- [x] Replace the old logo: favicons, Logo.svg, LogoMark, icon/adaptive/splash rasters, tracked iOS images

## Review
- No workflow change needed: both TestFlight workflows regenerate `ios/` from `app.json`; the last Fastlane run used Xcode 26.6.
- Clean prebuild output sets `ASSETCATALOG_COMPILER_APPICON_NAME = CairnIcon` and passes the bundle to actool, which reports `CFBundleIconName = CairnIcon`.
- Verified after installing the iOS 26.5 platform: a Release simulator build from a clean prebuild succeeded (signing off). Assets.car holds CairnIcon with light, dark and tinted appearances, and the app has CairnIcon PNG fallbacks.
- icon.png, adaptive-icon.png and splash.png were rendered from cairn_icon.svg at 1024px; icon and splash use the #FDFAF8 background.
- LogoMark uses the `brand` token and omits the SVG's drop shadow.
- Seen on an iOS 26.5 iPhone 17 Pro simulator: light-mode home screen icon, and the login logo in light and dark.
- The welcome screen wordmark also uses `brand`, and the logo is shifted down 16pt towards it.
- Not verified: a signed TestFlight build, the dark and tinted home screen icons, or the 16pt logo shift on a device (the last build predates it).
