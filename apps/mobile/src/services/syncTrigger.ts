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

export const SyncTrigger = {
  async run(): Promise<void> {
    if (inFlight) return;
    inFlight = true;
    try {
      for (const consumer of consumers) {
        await consumer();
      }
    } finally {
      inFlight = false;
    }
  },
};
