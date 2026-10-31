---
id: chore_ae2a
title: Declare iOS export compliance in app config so TestFlight builds skip the encryption questions
type: chore
status: in_progress
priority: 2
labels: [mobile,ios]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-30T10:19:00Z
updated_at: 2026-10-31T00:49:07Z
---
Each TestFlight upload shows 'Missing Compliance' and requires answering the encryption questions in App Store Connect. Set ios.config.usesNonExemptEncryption=false in apps/mobile/app.json so expo prebuild writes ITSAppUsesNonExemptEncryption=false into Info.plist (CI regenerates ios/ with prebuild --clean, so the setting must live in app.json). Cairn only uses HTTPS via Apple's system networking, which is exempt.

## Plan
- [x] Add `ios.config.usesNonExemptEncryption: false` to `apps/mobile/app.json`
- [x] Verify `npx expo config --type introspect` emits `ITSAppUsesNonExemptEncryption = false` in the iOS Info.plist
- [x] Add the same key to the committed `ios/CairnReader/Info.plist`, so builds from the committed project (Xcode archive, `expo run:ios`, a local `build_only` run without prebuild) carry it too
- [x] Confirm the exemption claim: no crypto in `apps/mobile/src` or `apps/shared/src`, `expo-sqlite` without SQLCipher, all traffic over HTTPS (ATS on, `NSAllowsArbitraryLoads=false`)
- [ ] Confirm the next TestFlight build arrives without "Missing Compliance"

## Review
`expo config --type introspect` (Expo SDK 54) shows `ios.infoPlist.ITSAppUsesNonExemptEncryption: false`. CI's `expo prebuild --clean` therefore writes the key on every build, covering both the Fastlane and local-build workflows. Remaining check needs a real upload.

Validation (second pass): the app.json change was correct, but `apps/mobile/ios/` is committed and its Info.plist did not have the key, so only the two CI workflows (both run `expo prebuild --clean`) were covered. Added the key there too; `plistlib` parses it. Note: `expo config --type introspect` runs config mods and rewrites `project.pbxproj` and adds a `cairnreader.entitlements` in the committed ios/ as a side effect, so revert those after running it. The committed Info.plist is already out of date in other ways (e.g. `CFBundleDisplayName` is `cairnreader`, not `Cairn Reader`). That is left out of scope here.

Review follow-up (PR #442): the reviewer said `ios.config.usesNonExemptEncryption` is a legacy ExpoKit field that prebuild ignores. That is wrong for SDK 54: `@expo/config-plugins` has `IOSConfig.UsesNonExemptEncryption.withUsesNonExemptEncryption`, which reads `ios.config.usesNonExemptEncryption` and is in `@expo/prebuild-config`'s default plugin chain. Running `npx expo prebuild --platform ios --clean --no-install` on a `git archive` copy writes `ITSAppUsesNonExemptEncryption = false` to the regenerated `ios/CairnReader/Info.plist`. No change needed.
