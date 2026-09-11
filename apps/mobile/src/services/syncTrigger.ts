import { ArticlePrefetchService } from './articlePrefetch';

// Consumers run in this fixed order every time the trigger fires. The order
// is this module's contract, not the caller's — see useSyncTrigger, which
// only decides *when* to fire, never what runs or in what order.
// task_ebf1 adds the outbox drain here, as the first entry, ahead of
// prefetch — no restructuring needed, just insert it above.
const consumers: (() => Promise<void>)[] = [
  () => ArticlePrefetchService.run(),
];

// A trigger firing mid-run (e.g. the app resumes onto a connection that was
// already restored) must not stack a second pass through the consumers.
// This guards the composite sequence only — it does not duplicate
// ArticlePrefetchService's own inFlight guard, which still protects that
// consumer against being entered concurrently from any other caller.
let inFlight = false;

// Runs each consumer in order, isolating one from the next: a rejection
// (e.g. the outbox drain task_ebf1 adds ahead of prefetch) must not stop the
// rest of the fixed order from running, and must not escape as an unhandled
// rejection into useSyncTrigger's `void SyncTrigger.run()`. Exported
// separately from SyncTrigger.run() only so this isolation behavior can be
// exercised directly in tests against more than the one real consumer that
// exists today — callers still only ever see SyncTrigger.run().
export async function runConsumersIsolated(fns: (() => Promise<void>)[]): Promise<void> {
  for (const fn of fns) {
    try {
      await fn();
    } catch (error) {
      console.error('SyncTrigger consumer failed:', error);
    }
  }
}

export const SyncTrigger = {
  async run(): Promise<void> {
    if (inFlight) return;
    inFlight = true;
    try {
      await runConsumersIsolated(consumers);
    } finally {
      inFlight = false;
    }
  },
};
