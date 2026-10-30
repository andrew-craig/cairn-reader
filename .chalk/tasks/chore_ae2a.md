---
id: chore_ae2a
title: Declare iOS export compliance in app config so TestFlight builds skip the encryption questions
type: chore
status: open
priority: 2
labels: [mobile,ios]
blocked_by: []
parent: null
remote_task_url: null
created_at: 2026-10-30T10:19:00Z
updated_at: 2026-10-30T10:19:00Z
---
Each TestFlight upload shows 'Missing Compliance' and requires answering the encryption questions in App Store Connect. Set ios.config.usesNonExemptEncryption=false in apps/mobile/app.json so expo prebuild writes ITSAppUsesNonExemptEncryption=false into Info.plist (CI regenerates ios/ with prebuild --clean, so the setting must live in app.json). Cairn only uses HTTPS via Apple's system networking, which is exempt.

## Plan
- [x] Add `ios.config.usesNonExemptEncryption: false` to `apps/mobile/app.json`
- [x] Verify `npx expo config --type introspect` emits `ITSAppUsesNonExemptEncryption = false` in the iOS Info.plist
- [ ] Confirm the next TestFlight build arrives without "Missing Compliance"

## Review
`expo config --type introspect` (Expo SDK 54) shows `ios.infoPlist.ITSAppUsesNonExemptEncryption: false`. CI's `expo prebuild --clean` therefore writes the key on every build, covering both the Fastlane and local-build workflows. Remaining check needs a real upload.
