---
id: chore_c364
title: [iOS native 19] Cutover: delete apps/mobile, switch CI and release, update docs
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
One mobile client in the repo. No compatibility layer left behind (root CLAUDE.md: remove obsolete paths).

## Plan
- [ ] Delete `apps/mobile` entirely (decision 6: this removes the Expo Android target too).
- [ ] Root `package.json`: drop the `apps/mobile` workspace; regenerate `package-lock.json`.
- [ ] `design/tokens/build.js`: remove the `mobile` (TypeScript) platform; update `design-tokens.yml` and `design/tokens/README.md`.
- [ ] Delete `.github/workflows/mobile-checks.yml` and the two RN TestFlight workflows; the native workflows become the only ones. Update `.github/workflows/README.md`.
- [ ] `apps/shared` now has a single consumer (web). Leave it in place and file a follow-up chore to decide whether to fold it into `apps/web`; do not do it here.
- [ ] Docs: root `CLAUDE.md` (overview, architecture, service links), finish `apps/ios/CLAUDE.md` to replace `apps/mobile/CLAUDE.md` and `AGENTS.md`, `docs/ARCHITECTURE.md`, `docs/TESTING.md`, `docs/TROUBLESHOOTING.md`, `docs/CONFIGURATION.md`, `apps/web/CLAUDE.md` references to mobile.
- [ ] Backlog: close `chore_3ea1` and `chore_88f9` if the mobile lockfile churn they describe is gone; re-read the open "Mobile:" feature tasks (`feature_2920`, `feature_cf80`, `feature_d632`, `feature_e3b4`, `feature_ec3e`, `feature_753f`, `feature_a85d`, `feature_a849`) and reword them for the native app.
- [ ] Release the native build to all TestFlight testers with a note that they will need to sign in again.

## Verify
- `grep -ri "apps/mobile\|expo\|react native"` over the repo (excluding `.chalk/tasks/closed` and `docs/archive`) returns nothing unexpected.
- Web checks, selfhost build and `ios-checks` are green.
