---
id: feature_b604
title: [iOS native 15] You tab, Account and About
type: feature
status: open
priority: 2
labels: [ios,ui]
blocked_by: [feature_0b9a,task_ec50]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
The profile hub and its two settings screens as standard grouped lists and forms.

## Source
`apps/mobile/src/screens/{YouScreen,AccountScreen,AboutScreen}.tsx`, `components/common/{QuickAccessButton,Button,IconButton}.tsx`.

## Plan
- [ ] You: `List` (inset grouped) with the account name header and `NavigationLink` rows for Bookmarks, RSS, Newsletters (each with its count as trailing text), Account, About. Counts from `listAllSubscriptions` and `countUserContents(is_favorite: true)` fetched concurrently, each failing independently; refetch on appear only when older than 30 s.
- [ ] Account: `Form`. Device accounts get "Save account" (email + password, `upgradeAccount`); email accounts get "Change password" (current, new, confirm; same validation rules: required fields, 8 characters minimum, match). `SecureField` with `.textContentType(.newPassword)`. Log out as a destructive button with a confirmation dialog.
- [ ] About: `LabeledContent` rows for app version (from the bundle) and server version (`SystemService`), plus whatever links the RN screen shows.
- [ ] No custom button or row components: `Button` with system styles, `LabeledContent`, `NavigationLink`.

## Verify
- View-model tests: partial stats failure still shows the other count; TTL prevents refetch; account validation rules; upgrade and change-password error messages.
