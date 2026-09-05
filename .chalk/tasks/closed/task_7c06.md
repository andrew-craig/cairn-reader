---
id: task_7c06
title: [Audit/Tier 4] AddLinkModal cannot be dismissed while a debounced URL detection is in flight
type: task
status: closed
priority: 2
labels: [quality,audit,web,ux]
blocked_by: []
parent: epic_fefa
remote_task_url: null
created_at: 2026-08-17T12:54:16Z
updated_at: 2026-09-05T09:20:13Z
---
**Source:** Cairn Simplification Audit — https://claude.ai/code/artifact/286883fb-3f93-49c4-942f-4880251a409f · located and verified against HEAD `a6c56a1` (the report names these two items by description; file:line below is my own derivation, so re-verify before relying on it).
Read docs/QUALITY_REMEDIATION_STRATEGY.md §0 (rules of engagement) and §2.6 (definition of done) before starting. One finding, one branch, one PR.

**Audit tier:** 4 (frontend) | **Verified.**

## Problem
`apps/web/src/components/AddLinkModal.tsx:198-205`:
```ts
const handleClose = () => {
    if (!loading && !detecting && !discovering) {
        ...
        onClose();
    }
};
```
Gating dismissal on `detecting` traps the user, because **detection is not user-initiated** — it fires from a 400ms debounce on every URL change (`setTimeout(runDetect, 400)`, :112, effect :90-113). While it is in flight **all three dismissal routes are dead**:
- **Escape** — :209 (`if (e.key === 'Escape') handleClose()`)
- **the Cancel button** — :282, and separately `disabled={isBusy}` at :283
- **the backdrop click** — :291

So typing a URL and immediately changing your mind leaves the modal unclosable until the network settles. If the detect request is slow or hangs, the user is stuck behind a request whose result they no longer want.

`loading` (a real submit in flight) is a defensible block. `detecting` and `discovering` are advisory background work and should not be.

## The cancellation machinery already exists
The detection effect already tracks a `cancelled` flag and guards every `setState` behind it (:103-107) — so an in-flight detection is **already safe to abandon**. `handleClose` simply declines to.

## What to do
1. Drop `detecting` and `discovering` from the `handleClose` guard; keep `loading`.
2. Ensure closing cancels the pending debounce timer and marks the in-flight detection cancelled, so no late response calls `setState` on an unmounted modal.
3. Keep `isBusy` for the **submit** affordances (:263, :273) — the fix is about dismissal, not about letting the user submit mid-detection.
4. Check whether `apps/mobile/src/components/AddLinkModal.tsx` has the same guard shape and fix it consistently if so.

## Done when
- Escape, the Cancel button and the backdrop all dismiss the modal during detection; no late detection response warns about setting state after unmount.

## Design note from the audit — do not unify the state fields
The audit explicitly considered and **declined** folding these flags into one union:
> Kept two fields rather than one union in the modal fix: folding an orthogonal background lifecycle into the union would let a late response clobber user-initiated state.

Keep `loading` and `detecting` as separate fields. The whole point is that they have different lifecycles and different authority over the modal.
