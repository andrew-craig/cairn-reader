# CLAUDE.md - iOS App (native)

Native Swift/SwiftUI iOS client, replacing the React Native app's iOS build (epic_bc48). Android stays on the Expo app in `apps/mobile`, which also keeps shipping iOS until cutover. Same backend, same bundle id (`com.cairnapp.cairnreader`).

## Layout

```
apps/ios/
├── CairnReader.xcodeproj         # Hand-written; folders below are file-system-synchronized groups
│   └── xcshareddata/xcschemes/   # Shared CairnReader scheme (build + CairnReaderTests)
├── CairnReader/                  # App target
│   ├── Info.plist                # Only keys with no build setting (URL scheme, encryption flag, launch screen);
│   │                             #   everything else is INFOPLIST_KEY_* in the project. Excluded from target membership.
│   ├── App/                      # @main app and root view
│   ├── Models/                   # API and domain types
│   ├── Services/                 # Auth, read API, sync
│   ├── Store/                    # On-device store and outbox (GRDB)
│   ├── Features/<Screen>/        # One folder per screen
│   └── Resources/                # Assets.xcassets, CairnIcon.icon, PrivacyInfo.xcprivacy
└── CairnReaderTests/             # Unit tests (Swift Testing), hosted in the app
```

Add a Swift file by creating it in the right folder; the synchronized groups pick it up, so `project.pbxproj` does not change. Folders that do not exist yet are created by the first task that needs them.

## Settings

- iOS 26 minimum, iPhone and iPad; iPhone is portrait only.
- Swift 6 language mode, default actor isolation `MainActor` (app target) and approachable concurrency. Mark service code `nonisolated` where it must run off the main actor.
- `PrivacyInfo.xcprivacy` declares no required-reason APIs. Add the entry (and reason code) in the same change that first uses one, for example `UserDefaults` (`CA92.1`).
- `LaunchBackground` colorset holds a copy of `background` from `design/tokens/colors.json` (light and dark); update it by hand if that changes.

## Build and test

Verify through CI. Cloud agent sessions run on Linux and cannot build or test this app: push and read the **iOS App Checks** workflow (`.github/workflows/ios-checks.yml`, `macos-26`), which builds and tests on an iOS 26 simulator with warnings as errors. On-device checks are the owner's.

On a Mac (Xcode 26):

```bash
cd apps/ios
xcodebuild test -project CairnReader.xcodeproj -scheme CairnReader \
  -destination 'platform=iOS Simulator,name=iPhone 17' CODE_SIGNING_ALLOWED=NO
```
