---
id: task_aae1
title: Migrate Fastlane build process to iOS 26 SDK
type: task
status: closed
priority: 2
labels: []
blocked_by: []
parent: null
created_at: 2026-04-03T23:50:57Z
updated_at: 2026-04-03T23:52:45Z
---

## Context

Apple requires all new apps and updates to be built with Xcode 26 / iOS 26 SDK by **April 28, 2026**. Our current CI uses `macos-latest` and does not pin Xcode version. This task migrates the Fastlane-based TestFlight build pipeline to iOS 26 SDK.

## Research Findings

- **Xcode 26** ships with iOS 26 SDK and Swift 6.2. Requires macOS 15.6+.
- **GitHub Actions**: `macos-26` runner is GA (Feb 2026), runs ARM64 natively.
- **CocoaPods**: Xcode 26 uses project object version 70 which older `xcodeproj` gem doesn't recognize. Need updated gem or workaround.
- **Swift Explicit Modules**: Xcode 26 enables `SWIFT_ENABLE_EXPLICIT_MODULES` by default, can break CocoaPods. Workaround: disable in post_install.
- **Fastlane**: Known silent upload failures with Xcode 26 `altool` changes. Need latest fastlane + `--verbose` flag.
- **Expo SDK 54 + RN 0.81.5**: Already compatible with iOS 26 SDK.
- **Liquid Glass**: Auto-applies to native UIKit components. Custom `CustomTabBar` may need attention (separate task).
- **Deployment target**: Can remain at 15.1 — SDK requirement is compile-time only.

## Plan

- [x] 1. Update GitHub Actions workflows
  - Changed `runs-on: macos-latest` → `runs-on: macos-26` (both workflows)
  - Added Xcode 26 selection step (`sudo xcode-select -s /Applications/Xcode_26.app`)
  - Updated Ruby version from 3.2 → 3.3
- [x] 2. Update Gemfile to pin compatible fastlane/cocoapods versions
  - Pinned `fastlane` to `~> 2.227` (Xcode 26 fixes)
  - Pinned `cocoapods` to `~> 1.17` (object version 70 support)
  - Added `xcodeproj` gem `~> 1.28` (object version 70 support)
- [x] 3. Update Podfile for Xcode 26 compatibility
  - Added `SWIFT_ENABLE_EXPLICIT_MODULES = NO` in post_install to prevent build failures
- [x] 4. Update Fastfile for Xcode 26 compatibility
  - Added `verbose: true` to `upload_to_testflight` to catch silent upload failures
- [x] 5. Update ExportOptions.plist (removed deprecated bitcode fields)
- [x] 6. Commit and push changes

