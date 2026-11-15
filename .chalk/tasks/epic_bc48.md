---
id: epic_bc48
title: Native iOS app: migrate the mobile client from React Native to Swift/SwiftUI
type: epic
status: open
priority: 1
labels: [ios,mobile]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-11-14T02:33:16Z
updated_at: 2026-11-14T02:33:16Z
---
## Goal
Move iOS from the React Native/Expo app (`apps/mobile`) to a native Swift app (`apps/ios`) that uses system components for navigation and chrome, keeps custom views to a minimum, and behaves the same against the unchanged backend. Android is out of scope: `apps/mobile` is kept and Expo continues to be used for Android.

## Principles
- **System containers only for navigation:** `TabView`, `NavigationStack`, `.sheet`, `.toolbar`, `.searchable`, `.swipeActions`, `.refreshable`, `.confirmationDialog`, `Form`, `List`, `ContentUnavailableView`.
- **Custom view budget:** `ArticleRow` (list row content), the offline/status banner, and the reader's HTML template. Anything else custom needs a reason in its PR.
- **Port behaviour, not code structure.** The service layer (auth refresh, store, outbox, sync) carries the app's hard-won rules; its jest suites are the specification and are ported case for case. The UI layer is rewritten idiomatically.
- **No backend change, no compatibility layer.** Same REST API, same bundle id (`com.cairnapp.cairnreader`), no migration of on-device RN data.
- **Brand fonts are kept** (Inter, Crimson Pro, Atkinson Hyperlegible Next), applied from one typography file at the root and through bar appearances, so system components are still used unmodified.
- **`apps/mobile` keeps shipping iOS until cutover.** Cutover then removes only its iOS side; it stays as the Expo Android app.

## What the review found
- ~19k lines in `apps/mobile` (about half tests). 10 screens, a service layer of ~1.7k lines, and 15 common components.
- Most common components exist only because RN lacks the system control and disappear in the port:

| RN component | Native replacement |
|---|---|
| `CustomTabBar`, `TabNavigator` | `TabView` |
| `RootNavigator`, `ScreenHeader`, `TopBlurGradient` | `NavigationStack`, navigation bar |
| `HeaderSearch` | `.searchable` |
| `SwipeAction` | `.swipeActions` |
| `BottomActionMenu` | bottom `.toolbar` |
| `HeaderPopover`, `AddLinkModal`, `ReportIssueModal` | `.sheet` + `Form` |
| `ListToggle`, `ListChoice` | `Picker` |
| `Button`, `IconButton`, `QuickAccessButton`, `icons/*` | `Button`, `NavigationLink`, SF Symbols |
| `FontFamily` constants on every `Text` | root `.font` + bar appearances from one `Typography.swift` |
| `ErrorBoundary`, safe-area constants, `globalStyles` | nothing |

- Native API replacements: AsyncStorage tokens to Keychain; `expo-sqlite` to GRDB; `expo-network` to `NWPathMonitor`; `react-native-render-html` to a web view; `expo-application` device id to `identifierForVendor`; `expo-haptics`/`expo-clipboard` to UIKit feedback generators / `UIPasteboard`; AppState to `scenePhase`.
- `design/tokens` already anticipates a Swift output; `apps/shared` (TypeScript) cannot be shared with Swift, so its auth state machine, error types and transforms are ported; it stays shared by web and the Expo Android app.
- The committed `apps/mobile/ios` project is Expo-generated (CocoaPods, deployment target 15.1) and is not a starting point; only the icon, privacy manifest and fastlane configuration carry over.

## Constraints and risks
- **Verification needs macOS.** Cloud agent sessions run on Linux and cannot build or test the app. Task 01 sets up a macOS CI job that every later task verifies through; on-device checks are the owner's.
- **Reader scroll restore and completion tracking** (task 12) is the riskiest port: web-view content height settles late (images), and the rule "do not persist the pre-restore offset" must hold.
- **Existing installs sign in again** and lose unsynced queued edits at the update (decision 4).
- **Minimum iOS rises from 15.1 to 26** (decision 1). Devices that cannot run iOS 26 stay on the last RN build.
- **Two mobile codebases after cutover** (Swift for iOS, Expo for Android): every later mobile feature is implemented twice, and service-layer rules (outbox, sync) can drift between them.

## Decisions
All seven decisions in decision_1686 were made by the owner on 2026-11-14:
1. Minimum iOS 26.
2. The reader is a web view showing `cleaned_html` in a bundled HTML/CSS template.
3. GRDB (SQLite) for the on-device store and outbox.
4. No migration of on-device RN data: existing installs sign in again, device-ID accounts return through the unchanged `identifierForVendor`, unsynced queued edits are lost.
5. Brand fonts are kept.
6. `apps/mobile` is kept as the Expo Android app.
7. Platform look wins over the RN app's custom chrome; the colour palette is kept.

## Tasks (in dependency order)
Foundation
- decision_1686: 00 Confirm migration decisions
- task_8792: 01 Scaffold `apps/ios` Xcode project and macOS CI
- task_a13d: 02 Colours and typography from design tokens
- task_ce49: 03 Models, errors, server URL config, HTTP client

Services (ported with their tests)
- task_5092: 04 AuthService (Keychain, refresh, device and email login)
- task_ec50: 05 ReadService and SystemService
- task_c194: 06 Local database and ArticleStore
- task_387d: 07 Outbox and ArticleMutations
- task_4e17: 08 Connectivity, SyncTrigger, prefetch, FeedCache

UI
- feature_0b9a: 09 App shell: auth gate, tabs, navigation stacks, Login
- feature_33bc: 10 Article list model, ArticleRow, Reads tab
- feature_75bc: 11 Feed tab and Bookmarks
- feature_4e37: 12 Reader
- feature_65dc: 13 Report extraction issue sheet
- feature_704f: 14 Add link / subscribe sheet
- feature_b604: 15 You, Account, About
- feature_983c: 16 Subscriptions (RSS, Newsletters)

Ship
- task_4713: 17 fastlane and TestFlight workflows (parallel, needs only 01)
- task_fe45: 18 Parity and platform QA pass
- chore_c364: 19 Cutover: retire the iOS side of `apps/mobile`, switch release, update docs

Parallel tracks after 03: {04, 05} and {06} can proceed side by side and meet at 07. After 10: 11, 12 and 14 are independent. 15 needs only 09 and 05.

## Done when
- iOS is released only from `apps/ios` through TestFlight with every item in task 18's checklist passing, and `apps/mobile` is an Android-only Expo app that still passes its checks.
