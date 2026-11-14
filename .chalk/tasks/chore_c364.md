---
id: chore_c364
title: [iOS native 19] Cutover: retire the iOS side of apps/mobile (Expo stays for Android), switch release, update docs
type: chore
status: open
priority: 1
labels: [ios,mobile,docs]
blocked_by: [task_fe45]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
iOS ships only from `apps/ios`. `apps/mobile` remains as the Expo app for Android (decision 6) with its iOS-specific parts removed, so there is one owner per platform and no dead iOS path left in the Expo app (root CLAUDE.md: remove obsolete paths).

## Plan
- [ ] Delete the committed Expo iOS project `apps/mobile/ios` (including its fastlane directory, which task 17 re-created under `apps/ios`).
- [ ] Delete the RN iOS workflows `.github/workflows/ios-testflight-fastlane.yml` and `ios-testflight-local-build.yml` (the `mobile-v*` tag trigger goes with them); the `apps/ios` workflows are the only iOS release path. Update `.github/workflows/README.md`.
- [ ] `apps/mobile/app.json`: remove the `ios` block and the `CairnIcon.icon` asset reference; `package.json`: remove the `ios` script.
- [ ] Remove iOS-only branches from the Expo code so it is plainly Android-only: `Platform.OS === 'ios'` paths and `ActionSheetIOS` in `ArticleContent.tsx`, `getIosIdForVendorAsync` in `services/auth.ts`, and the platform switches in `App.tsx`, `LoginScreen`, `AccountScreen`, `AddArticleScreen`, `HeaderPopover`, `CustomTabBar`, `BottomActionMenu`, `TopBlurGradient`. Update the affected tests.
- [ ] `.github/workflows/mobile-checks.yml`: the bundle job exports `--platform android` instead of `ios`. The other jobs stay.
- [ ] Keep unchanged: the `apps/mobile` npm workspace, `apps/shared` (still consumed by web and the Expo app), and the `mobile` TypeScript platform in `design/tokens/build.js` alongside the new `ios` platform.
- [ ] Docs: root `CLAUDE.md` (overview and architecture now list a native iOS app, an Expo Android app and the web app; add the `apps/ios/CLAUDE.md` link), finish `apps/ios/CLAUDE.md`, rewrite `apps/mobile/CLAUDE.md` and `AGENTS.md` as Android-only, and update `docs/ARCHITECTURE.md`, `docs/TESTING.md`, `docs/TROUBLESHOOTING.md`, `docs/CONFIGURATION.md`, `design/tokens/README.md`.
- [ ] Backlog: go through the open "Mobile:" feature tasks (`feature_2920`, `feature_cf80`, `feature_d632`, `feature_e3b4`, `feature_ec3e`, `feature_753f`, `feature_a85d`, `feature_a849`) and state on each whether it applies to iOS, Android or both. `chore_3ea1` and `chore_88f9` stay valid (the Expo lockfile churn remains).
- [ ] Release the native build to all TestFlight testers with a note that they will need to sign in again.

## Notes
- No `android/` directory or Android release workflow is committed today; adding one is out of scope for this epic.
- After this task every user-facing change has two mobile implementations (Swift for iOS, Expo for Android). Record that in the root `CLAUDE.md` so future tasks say which platforms they cover.

## Verify
- `apps/mobile`: `npm run type-check`, `npm run lint`, `npm test`, knip and the Android bundle export are green.
- `grep -rn "Platform.OS\|ActionSheetIOS\|IosIdForVendor" apps/mobile/src apps/mobile/App.tsx` returns nothing.
- Web checks, selfhost build and `ios-checks` are green.
