---
id: task_0f03
title: Web: Account screen (FR-11) — profile, change password, logout
type: task
status: open
priority: 2
labels: []
blocked_by: [task_58b2]
parent: epic_f54e
remote_task_url: null
created_at: 2026-06-08T11:34:09Z
updated_at: 2026-06-08T11:34:09Z
---
Implement /you/account: the user's profile screen.

WHAT TO DO:
- Display the logged-in user's email address (or 'Anonymous user' if no email).
- Change password form: current_password, new_password inputs → PUT /api/v1/user/{userId}/password.
- Show success/error feedback for password change.
- Log out button → POST /api/v1/auth/logout (revokes refresh token), clear localStorage tokens, redirect to /login.

VERIFICATION (agent-testable):
1. /you/account shows the logged-in user's email address.
2. Submitting the change-password form with the correct current password and a new password returns a success message.
3. Submitting with the wrong current password returns a user-visible error (not just a console error).
4. After a successful password change, logging out and logging back in with the new password succeeds.
5. Clicking Log out on the account screen clears tokens and redirects to /login.
