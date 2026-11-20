---
id: feature_b604
title: [iOS native 15] You tab, Account and About
type: feature
status: closed
priority: 2
labels: [ios,ui]
blocked_by: []
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-20T10:44:11Z
---
## Goal
The profile hub and its two settings screens as standard grouped lists and forms.

## Source
`apps/mobile/src/screens/{YouScreen,AccountScreen,AboutScreen}.tsx`, `components/common/{QuickAccessButton,Button,IconButton}.tsx`.

## Plan
- [x] You: `List` (inset grouped) with the account name header and `NavigationLink` rows for Bookmarks, RSS, Newsletters (each with its count as trailing text), Account, About. Counts from `listAllSubscriptions` and `countUserContents(is_favorite: true)` fetched concurrently, each failing independently; refetch on appear only when older than 30 s.
- [x] Account: `Form`. Device accounts get "Save account" (email + password, `upgradeAccount`); email accounts get "Change password" (current, new, confirm; same validation rules: required fields, 8 characters minimum, match). `SecureField` with `.textContentType(.newPassword)`. Log out as a destructive button with a confirmation dialog.
- [x] About: `LabeledContent` rows for app version (from the bundle) and server version (`SystemService`), plus whatever links the RN screen shows.
- [x] No custom button or row components: `Button` with system styles, `LabeledContent`, `NavigationLink`. Text picks up Inter from the root font (decision 5).

### Refinements (implementation)
- [x] `Route.swift`: `RouteDestination` replaces `RoutePlaceholder` as the one route-to-view mapping for all three stacks (Account and About real; article, Bookmarks, RSS, Newsletters stay `ContentUnavailableView` until tasks 11, 12, 16). `YouPlaceholder` and its temporary Sign out go.
- [x] `SessionModel`: `user` becomes stored and is refreshed by `login()` even when the id is unchanged, so an upgraded device account shows its email (RN calls `login()` after `upgradeAccount`); `auth` is readable for `AccountModel`.
- [x] `CairnReaderApp`: one `HTTPClient` shared by `AuthService` and a `SystemService` put in the environment for About (no service built in a view).
- [x] `YouModel` (counts stay at their last value on failure; first error shown, subscriptions before bookmarks, with the RN strings; the 30 s clock only advances on a fully successful fetch, as in RN). `AccountModel` (RN validation and alert strings; the server's message on failure).
- [x] Tests: `YouModelTests`, `AccountModelTests`, About server-version text, `SessionModelTests` case for the refreshed user.

## Verify
- View-model tests: partial stats failure still shows the other count; TTL prevents refetch; account validation rules; upgrade and change-password error messages.

## Review
- You (`Features/You/YouView.swift`, `YouModel.swift`): inset grouped `List`, account name (email or "Anonymous user") as the section header, `NavigationLink` rows for Bookmarks, RSS and Newsletters with the count as `LabeledContent` trailing text, then Account and About. Counts fetched concurrently with `async let`, each failing independently (RN strings, subscriptions error first); counts keep their last value on failure; refetch on appear only when the last fully successful fetch is over 30 s old (RN only advances the clock on full success). `.onAppear` + `Task` rather than `.task`, so leaving the tab mid-fetch does not cancel it into an error.
- Account (`Features/Account/AccountView.swift`, `AccountModel.swift`): `Form` with the account label, then "Save account" (email, password, `upgradeAccount`) for device accounts or "Change password" (current, new, confirm) for email accounts, RN validation and Error/Success alert wording, the server's message on failure. Log out is a destructive button behind a `confirmationDialog`. After an upgrade `SessionModel.login()` now refreshes `SessionModel.user` (made a stored property) without replacing the session, so the form switches to Change password and You shows the email.
- About (`Features/About/AboutView.swift`): `LabeledContent` for app version (`CFBundleShortVersionString`) and server version (Loading… / version / Unknown / Unavailable, as RN). RN shows no links. `SystemService` is built in `CairnReaderApp` on the shared `HTTPClient` and passed through `\.systemService`. No required-reason API, so `PrivacyInfo.xcprivacy` is unchanged.
- `RouteDestination` (`Features/Main/Route.swift`) replaces `RoutePlaceholder` as the single route-to-screen mapping for all stacks; `YouPlaceholder` and its temporary Sign out are gone.
- Tests: `YouModelTests` (all counts, each partial failure, both failing, TTL, partial failure not starting the clock), `AccountModelTests` (save and change-password validation tables, success, server message, offline message), `AboutViewTests`, and a `SessionModelTests` case for the upgraded user keeping its session.
- Known, not changed (task 04 territory, same as RN): a wrong current password gets a 401 from `PUT /user/{id}/password`, which `AuthService.authorizedRequest` treats as an expired token (refresh, retry, then sign out on the second 401).

- CI: iOS App Checks green on PR #467 (85c43fb). Close after merge and on-device check.
- Local run (2026-11-20, Xcode 27.2, iPhone 17 simulator on iOS 27.2, branch at 2f8ed73): `YouModelTests`, `AccountModelTests` and `AboutViewTests` all pass (17 tests in 3 suites). Closed at the owner's request.
