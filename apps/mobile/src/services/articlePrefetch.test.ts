import { ArticlePrefetchService } from './articlePrefetch';
import { ArticleStore } from './articleStore';
import { ReadService } from './read';
import { isOffline } from '../utils/network';
import { NetworkError } from '@cairn/shared';
import { Article } from '../types';
import type { UserContentDetailResponse } from '@cairn/shared';

// task_c55c: after each successful list sync, download cleaned_html for
// unread/reading articles that lack a body — bounded concurrency, newest
// first, capped at the 100 most recent Read-list articles, run from a
// service rather than a screen (decision 4), never while offline (decision
// 7), aborting the batch on the first NetworkError (decision 7), and
// single-flight (decision 8).

jest.mock('./articleStore', () => ({
  ArticleStore: {
    listPrefetchCandidates: jest.fn(),
    saveBody: jest.fn(),
    evictBodiesOutsideCap: jest.fn(),
  },
}));

jest.mock('./read', () => ({
  ReadService: {
    getContentById: jest.fn(),
    transformDetailToArticle: jest.fn(),
  },
}));

jest.mock('../utils/network', () => ({
  isOffline: jest.fn(),
}));

const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;
const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedIsOffline = isOffline as jest.Mock;

const makeCandidate = (id: string): Article => ({
  id,
  url: `https://example.com/${id}`,
  title: id,
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: 1000,
});

const detailFor = (id: string): UserContentDetailResponse =>
  ({ content_id: id }) as unknown as UserContentDetailResponse;

describe('ArticlePrefetchService.run', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedIsOffline.mockResolvedValue(false);
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([]);
    mockedArticleStore.saveBody.mockResolvedValue(undefined);
    mockedArticleStore.evictBodiesOutsideCap.mockResolvedValue(undefined);
    mockedReadService.transformDetailToArticle.mockImplementation(
      (detail) => ({ ...makeCandidate(detail.content_id), content: `body-${detail.content_id}` }),
    );
  });

  it('never prefetches while offline', async () => {
    mockedIsOffline.mockResolvedValue(true);
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([makeCandidate('a1')]);

    await ArticlePrefetchService.run();

    expect(mockedArticleStore.listPrefetchCandidates).not.toHaveBeenCalled();
    expect(mockedReadService.getContentById).not.toHaveBeenCalled();
    expect(mockedArticleStore.evictBodiesOutsideCap).not.toHaveBeenCalled();
  });

  it('downloads and stores a body for every candidate', async () => {
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([
      makeCandidate('a1'),
      makeCandidate('a2'),
    ]);
    mockedReadService.getContentById.mockImplementation((id) =>
      Promise.resolve(detailFor(id)),
    );

    await ArticlePrefetchService.run();

    expect(mockedArticleStore.saveBody).toHaveBeenCalledWith('a1', 'body-a1');
    expect(mockedArticleStore.saveBody).toHaveBeenCalledWith('a2', 'body-a2');
  });

  it('evicts bodies outside the cap after the batch', async () => {
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([makeCandidate('a1')]);
    mockedReadService.getContentById.mockResolvedValue(detailFor('a1'));

    await ArticlePrefetchService.run();

    expect(mockedArticleStore.evictBodiesOutsideCap).toHaveBeenCalledWith(100);
  });

  it('skips a candidate that fails with a non-network error and continues the batch', async () => {
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([
      makeCandidate('bad'),
      makeCandidate('good'),
    ]);
    mockedReadService.getContentById.mockImplementation((id) =>
      id === 'bad'
        ? Promise.reject(new Error('server said no'))
        : Promise.resolve(detailFor(id)),
    );
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});

    await ArticlePrefetchService.run();

    expect(mockedArticleStore.saveBody).toHaveBeenCalledWith('good', 'body-good');
    expect(mockedArticleStore.saveBody).not.toHaveBeenCalledWith('bad', expect.anything());
    expect(consoleErrorSpy).toHaveBeenCalled();
    consoleErrorSpy.mockRestore();
  });

  it('aborts the remaining batch on the first NetworkError', async () => {
    // A single worker (CONCURRENCY is fixed at 3, but one candidate means
    // one worker) makes the abort point deterministic: candidate 1 fails
    // with NetworkError, candidates 2 and 3 must never be attempted.
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([makeCandidate('only')]);
    mockedReadService.getContentById.mockRejectedValue(new NetworkError());

    await ArticlePrefetchService.run();

    expect(mockedArticleStore.saveBody).not.toHaveBeenCalled();
    // Eviction is bookkeeping independent of the network pass — still runs.
    expect(mockedArticleStore.evictBodiesOutsideCap).toHaveBeenCalledWith(100);
  });

  it('never starts a candidate beyond the initial wave once aborted', async () => {
    // 5 candidates against a fixed 3-worker pool: workers 0-2 have already
    // grabbed a1-a3 synchronously before any of their fetches can settle —
    // that first wave is unavoidable and acceptable (decision 7's "abort the
    // *remaining* work"). What must never happen is a worker looping back
    // for a4 or a5 after the abort. a1 rejects with NetworkError; a2/a3
    // resolve normally, so this also proves an abort stops workers that
    // weren't the one that failed, not just the failing one.
    const attempted: string[] = [];
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([
      makeCandidate('a1'),
      makeCandidate('a2'),
      makeCandidate('a3'),
      makeCandidate('a4'),
      makeCandidate('a5'),
    ]);
    mockedReadService.getContentById.mockImplementation(async (id) => {
      attempted.push(id);
      if (id === 'a1') throw new NetworkError();
      return detailFor(id);
    });

    await ArticlePrefetchService.run();

    expect(attempted).toContain('a1');
    expect(attempted).not.toContain('a4');
    expect(attempted).not.toContain('a5');
    expect(mockedArticleStore.saveBody).not.toHaveBeenCalledWith('a1', expect.anything());
  });

  it('is single-flight: a run already in progress is not restarted by a second call', async () => {
    let resolveCandidates: (articles: Article[]) => void;
    mockedArticleStore.listPrefetchCandidates.mockReturnValue(
      new Promise((resolve) => {
        resolveCandidates = resolve;
      }),
    );

    const first = ArticlePrefetchService.run();
    const second = ArticlePrefetchService.run();

    // The second call must return without waiting on the first — a no-op,
    // not a queued run.
    await second;
    expect(mockedArticleStore.listPrefetchCandidates).toHaveBeenCalledTimes(1);

    resolveCandidates!([]);
    await first;
    expect(mockedArticleStore.listPrefetchCandidates).toHaveBeenCalledTimes(1);
  });

  it('allows a new run once the previous one has finished', async () => {
    mockedArticleStore.listPrefetchCandidates.mockResolvedValue([]);

    await ArticlePrefetchService.run();
    await ArticlePrefetchService.run();

    expect(mockedArticleStore.listPrefetchCandidates).toHaveBeenCalledTimes(2);
  });
});
