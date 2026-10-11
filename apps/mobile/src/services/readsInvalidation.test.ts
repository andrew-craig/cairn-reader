describe('readsInvalidation', () => {
  beforeEach(() => {
    jest.resetModules();
    jest.useFakeTimers();
  });
  afterEach(() => jest.useRealTimers());

  const load = () =>
    jest.requireActual<typeof import('./readsInvalidation')>('./readsInvalidation');

  it('is not stale before anything invalidates Reads', () => {
    const { readsInvalidatedSince } = load();
    jest.setSystemTime(10_000);
    expect(readsInvalidatedSince(5_000)).toBe(false);
  });

  it('is never stale for a screen that has not fetched yet', () => {
    const { invalidateReads, readsInvalidatedSince } = load();
    invalidateReads();
    expect(readsInvalidatedSince(null)).toBe(false);
  });

  it('is stale for a fetch that predates the invalidation, not for one after it', () => {
    const { invalidateReads, readsInvalidatedSince } = load();
    jest.setSystemTime(10_000);
    invalidateReads();
    expect(readsInvalidatedSince(9_000)).toBe(true);
    jest.setSystemTime(11_000);
    expect(readsInvalidatedSince(11_000)).toBe(false);
  });
});
