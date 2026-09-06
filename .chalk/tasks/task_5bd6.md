---
id: task_5bd6
title: Mobile: offline-aware login and loading states
type: task
status: open
priority: 3
labels: [mobile,offline,auth]
blocked_by: []
parent: feature_90a5
remote_task_url: null
created_at: 2026-09-06T07:46:17Z
updated_at: 2026-09-06T21:35:50Z
---
Gap identified during task_c87c review. The OfflineBanner added by task_c87c is rendered as a sibling of the authenticated Stack.Navigator in RootNavigator, so it only appears once the user is logged in. RootNavigator returns early for the two unauthenticated branches (the isLoading spinner and the LoginScreen), and neither shows any connectivity state. This was correctly scoped out of task_c87c rather than folded in, because a bare banner is the wrong answer here: unlike the authenticated screens, which can fall back to locally stored articles, login genuinely cannot proceed offline. The user needs to be told why the attempt will fail, not just that they are offline. Current behaviour: a login attempt while offline surfaces whatever the raw fetch rejection produces. Note that the auth entry points (loginWithDevice, registerWithDevice, loginWithEmail, registerWithEmail at auth.ts:95/123/149/175) are among the raw fetch call sites covered by chore_1089, so land that first and this task consumes the NetworkError it produces rather than string-matching. Scope: decide and implement the offline state for the isLoading and LoginScreen branches. Options worth weighing rather than assuming: extend the banner to those branches, add offline-specific copy plus a disabled or retrying submit button on LoginScreen, or both. Cold start offline while already holding valid tokens must keep working exactly as task_cab7 made it work. Do not regress that. Verify: test asserts LoginScreen shows an offline-specific message rather than a raw network error when a login is attempted offline, and that the already-authenticated cold-start-offline path is unchanged.
