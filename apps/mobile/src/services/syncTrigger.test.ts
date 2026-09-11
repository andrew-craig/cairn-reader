import { SyncTrigger, runConsumersIsolated } from './syncTrigger';
import { ArticlePrefetchService } from './articlePrefetch';
import { Outbox } from './outbox';

// task_06e5: SyncTrigger.run() is the single, fixed-order entry point the
// app-foreground/reconnect trigger calls into. task_ebf1 adds the outbox
// drain as a consumer, ahead of ArticlePrefetchService.

jest.mock('./articlePrefetch', () => ({
  ArticlePrefetchService: {
    run: jest.fn(),
  },
}));

jest.mock('./outbox', () => ({
  Outbox: {
    drain: jest.fn(),
  },
}));

const mockedArticlePrefetchService = ArticlePrefetchService as jest.Mocked<typeof ArticlePrefetchService>;
const mockedOutbox = Outbox as jest.Mocked<typeof Outbox>;

describe('SyncTrigger.run', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedArticlePrefetchService.run.mockResolvedValue(undefined);
    mockedOutbox.drain.mockResolvedValue(undefined);
  });

  it('runs ArticlePrefetchService.run() as a consumer', async () => {
    await SyncTrigger.run();
    expect(mockedArticlePrefetchService.run).toHaveBeenCalledTimes(1);
  });

  it('drains the outbox before running prefetch', async () => {
    await SyncTrigger.run();

    expect(mockedOutbox.drain).toHaveBeenCalledTimes(1);
    const drainOrder = mockedOutbox.drain.mock.invocationCallOrder[0];
    const prefetchOrder = mockedArticlePrefetchService.run.mock.invocationCallOrder[0];
    expect(drainOrder).toBeLessThan(prefetchOrder);
  });

  it('does not stack concurrent runs', async () => {
    let resolveRun: () => void;
    mockedArticlePrefetchService.run.mockReturnValue(
      new Promise((resolve) => {
        resolveRun = resolve;
      }),
    );

    const first = SyncTrigger.run();
    const second = SyncTrigger.run();

    // The second call must return without waiting on the first — a no-op,
    // not a queued run.
    await second;
    expect(mockedArticlePrefetchService.run).toHaveBeenCalledTimes(1);

    resolveRun!();
    await first;
    expect(mockedArticlePrefetchService.run).toHaveBeenCalledTimes(1);
  });

  it('allows a new run once the previous one has finished', async () => {
    await SyncTrigger.run();
    await SyncTrigger.run();

    expect(mockedArticlePrefetchService.run).toHaveBeenCalledTimes(2);
  });

  it('does not reject when its consumer rejects', async () => {
    mockedArticlePrefetchService.run.mockRejectedValue(new Error('boom'));
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    await expect(SyncTrigger.run()).resolves.toBeUndefined();

    expect(consoleErrorSpy).toHaveBeenCalled();
    consoleErrorSpy.mockRestore();
  });
});

describe('runConsumersIsolated', () => {
  it('runs every consumer even when an earlier one rejects, and does not reject itself', async () => {
    const first = jest.fn().mockRejectedValue(new Error('boom'));
    const second = jest.fn().mockResolvedValue(undefined);
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    await expect(runConsumersIsolated([first, second])).resolves.toBeUndefined();

    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
    expect(consoleErrorSpy).toHaveBeenCalled();
    consoleErrorSpy.mockRestore();
  });
});
