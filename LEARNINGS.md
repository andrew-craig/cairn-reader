# LEARNINGS

Corrections worth remembering, captured as they happen. Newest first.

## 2026-09-06 — Adding a throwing call to an existing try block silently widens its blast radius (task_cab7, then task_a8a4)

**Correction:** The same defect was caught twice in one feature, by two different
reviewers, in two different files.

First, in PR #381 (task_cab7): `AuthContext.checkAuthStatus`'s new `NetworkError`
branch called `AuthService.getUser()` — which does `JSON.parse` on persisted data
and can throw on corrupt storage — inside the catch handler, unguarded.

Then, in task_a8a4: `AuthContext.logout` added `await ArticleStore.clear()`
between `AuthService.logout()` and `setUser(null)`, inside a try whose catch
rethrows and never reaches `setUser(null)`. A SQLite failure would wipe the tokens
but leave the context reporting the user as authenticated — the app keeps
rendering while every request 401s — and leave the articles on disk, so on a
shared device the next user would see the previous user's articles.

**Why it happens:** a try block reads as "this is the error handling, so my new
call is covered." The opposite is true. Existing catch/finally logic was written
against the set of failures the *original* statements could produce. Dropping a
new call in — especially one that does I/O — hands that handler a failure mode it
was never designed for, and any statement *after* the new call silently becomes
conditional on the new call succeeding.

**How to apply:** when adding a call inside an existing try, ask two questions
before anything else. What can this new call throw? And what statements after it
stop running if it does? If the answer to the second is "something that must
always happen" — clearing auth state, releasing a lock, navigating away — then the
new call must not be able to alter the existing control flow:

```js
await ArticleStore.clear().catch((err) =>
  console.error('Failed to clear local articles on logout:', err),
);
```

The added call still runs and still reports failure, but the pre-existing
behaviour stays byte-for-byte what it was. Prefer this to restructuring the
handler, which quietly changes behaviour the task never asked you to touch.

**Process note:** both instances were caught by review, not by tests — the failure
needs the new call to throw, which no existing test provoked. Both fixes shipped
with a test that forces the rejection. That test is cheap and is the thing that
stops a third occurrence.

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
