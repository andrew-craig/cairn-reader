/**
 * Marks the Reads list stale after something outside it adds an item (Save to
 * Reads from the Feed reader). ReadsScreen refetches on focus only once its
 * 30s TTL lapses; without this a just-saved item wouldn't show up at the top
 * of Reads until then.
 */
let invalidatedAt = 0;

export const invalidateReads = (): void => {
  invalidatedAt = Date.now();
};

/** True when Reads was invalidated after `fetchedAt` (null = never fetched). */
export const readsInvalidatedSince = (fetchedAt: number | null): boolean =>
  fetchedAt !== null && invalidatedAt >= fetchedAt;
