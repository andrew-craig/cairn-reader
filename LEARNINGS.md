# LEARNINGS

Corrections worth remembering, captured as they happen. Newest first.

## 2026-09-06 — "Keep the tokens" was not the same as "keep the user logged in" (task_cab7)

**Correction:** The task, and then my own scope clarification on it, both stopped
short of the change that actually fixed the user-visible bug.

task_cab7 described the offline-logout bug as living in one function,
`doRefreshAccessToken`, which called `clearTokens()` on any error. Reviewing it, I
found the failure actually propagated through three functions and re-scoped the
task to cover `ensureValidToken` (which collapsed "server rejected" and "server
unreachable" into a single `false`) and `fetchWithAuth` (which converted both into
the load-bearing string `'Session expired. Please log in again.'`).

That was still wrong. There was a fourth layer:
`AuthContext.checkAuthStatus` catches every error from `ensureValidToken` and calls
`setUser(null)` — "On error, clear auth state to force re-login." Making
`ensureValidToken` throw a `NetworkError` simply routed the offline case into that
generic catch. The tokens survived on disk and the user was still shown the login
screen. Three of the four layers fixed, and the symptom was completely unchanged.

**Why it happened:** I traced the chain outward from the reported function until the
tokens were safe, and stopped there. But the bug was defined in terms of the tokens
("clears tokens on any error"), not in terms of what the user sees. Preserved state
that nothing reads is not a fix.

**How to apply:** When re-scoping a bug, trace to the *user-visible symptom*, not to
the state the ticket happens to name. Then check what consumes the new behaviour —
here, making a function throw where it previously never threw pushed the failure
into a pre-existing catch-all one level up. Widening an error's blast radius is a
change to every handler above it; enumerate the callers before assuming the fix
terminates.

**Process note that worked:** requiring the implementer to prove each new test fails
against the old code, and re-running that proof myself, is what would have exposed
this had I missed it a second time — the AuthContext test fails on the old code for
a different reason than the service tests do.
