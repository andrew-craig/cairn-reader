---
id: feature_0b9a
title: [iOS native 09] App shell: auth gate, TabView, per-tab NavigationStack, Login
type: feature
status: open
priority: 1
labels: [ios,ui]
blocked_by: [task_a13d,task_5092]
parent: epic_bc48
remote_task_url: null
created_at: 2026-11-14T02:35:53Z
updated_at: 2026-11-14T02:35:53Z
---
## Goal
The navigation skeleton, built only from system containers. First task that produces something the owner can run and sign in to.

## Source
`apps/mobile/App.tsx`, `src/navigation/{RootNavigator,TabNavigator}.tsx`, `src/screens/LoginScreen.tsx`, `src/components/common/{CustomTabBar,OfflineBanner}.tsx`; tests `RootNavigator.test.tsx`, `TabNavigator.test.tsx`, `LoginScreen.test.tsx`.

## Native mapping
| RN | Native |
|----|--------|
| `RootNavigator` loading / logged-out / authenticated switch | root view switching on `SessionModel` |
| `CustomTabBar` (floating pill, custom SVG icons) | `TabView` with `Tab` values, SF Symbols, system tab bar |
| stack navigator with `headerShown: false` + `ScreenHeader` | one `NavigationStack` per tab with typed `navigationDestination(for:)`, system navigation bar and back gesture |
| `ErrorBoundary` | none (no equivalent needed) |
| `GestureHandlerRootView`, safe-area maths, `StatusBar` setup | none |

## Plan
- [ ] Root view: progress view while `SessionModel.isLoading`, `LoginView` when signed out, `MainTabs` when signed in.
- [ ] `MainTabs`: Feed, Reads, You; opens on Reads; selection held in `@State` (or `@SceneStorage`) of an enum.
- [ ] A `Route` enum per stack (article, bookmarks, account, about, rss, newsletters) so destinations are typed in one place.
- [ ] `LoginView` as a `Form`: continue with this device (login, falling back to register as today), email sign in / create account with `.textContentType` set for password autofill, and a server URL field under an "Advanced" disclosure. Errors shown inline in the form footer.
- [ ] Offline indicator: one small banner view attached with `.safeAreaInset(edge: .top)` on the root, shown in all three root states. This is one of the few custom views.
- [ ] Placeholder views for the three tabs.

## Verify
- Tests for `SessionModel`-driven root switching and the login view model (device login falls back to register on the "not found" response; server URL change clears the session).
- Owner: sign in with device and with email against the self-host stack on `http://localhost:8099`, switch tabs, sign out via a temporary button.
