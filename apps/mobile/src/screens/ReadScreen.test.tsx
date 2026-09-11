import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { ReadScreen } from './ReadScreen';
import { ReadService } from '../services/read';
import { ArticleStore } from '../services/articleStore';
import { SyncTrigger } from '../services/syncTrigger';
import { Article } from '../types';

// task_a8a4: ReadScreen must render articles already in the local SQLite
// store immediately on focus, before the network page for the same list has
// resolved — the store is the read-through cache for the initial render.
// task_c55c: prefetch is triggered from this screen's sync callback only —
// after upsertMany, per decision 4 — and the screen itself knows nothing
// else about it.
// task_ebf1 (item G): pull-to-refresh routes through SyncTrigger.run() (outbox
// drain, then prefetch) instead of calling ArticlePrefetchService directly,
// so the outbox gets a drain on this path too.

jest.mock('../services/read', () => ({
  ReadService: {
    listUserContents: jest.fn(),
  },
}));

jest.mock('../services/articleStore', () => ({
  ArticleStore: {
    listRecent: jest.fn(),
    upsertMany: jest.fn(),
    remove: jest.fn(),
  },
}));

jest.mock('../services/syncTrigger', () => ({
  SyncTrigger: {
    run: jest.fn(),
  },
}));

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn() }),
  // Minimal stand-in: run the focus callback on mount / when it changes.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;
const mockedSyncTrigger = SyncTrigger as jest.Mocked<typeof SyncTrigger>;

const article = (id: string): Article => ({
  id,
  url: `https://example.com/${id}`,
  title: `Stored Article ${id}`,
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: Date.now(),
});

describe('ReadScreen offline-first render', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedArticleStore.upsertMany.mockResolvedValue(undefined);
    mockedArticleStore.remove.mockResolvedValue(undefined);
    mockedSyncTrigger.run.mockResolvedValue(undefined);
  });

  it('renders stored articles before the network call resolves', async () => {
    mockedArticleStore.listRecent.mockResolvedValue([article('stored-1')]);
    // Never resolves during this test.
    mockedReadService.listUserContents.mockReturnValue(new Promise(() => {}));

    render(<ReadScreen />);

    expect(await screen.findByText('Stored Article stored-1')).toBeTruthy();
    expect(mockedReadService.listUserContents).toHaveBeenCalled();
  });

  it('triggers SyncTrigger.run() (outbox drain, then prefetch) after a successful sync upserts the new page', async () => {
    mockedArticleStore.listRecent.mockResolvedValue([]);
    mockedReadService.listUserContents.mockResolvedValue({
      contents: [],
      total_count: 0,
      limit: 20,
      cursor: '',
      has_more: false,
    });

    render(<ReadScreen />);
    await screen.findByText('No saved articles yet');

    // SyncTrigger.run() must run only after upsertMany's write has resolved,
    // not alongside it — prefetch (one of its consumers) depends on the
    // post-sync store state.
    await waitFor(() => expect(mockedSyncTrigger.run).toHaveBeenCalled());
    const upsertOrder = mockedArticleStore.upsertMany.mock.invocationCallOrder[0];
    const syncTriggerOrder = mockedSyncTrigger.run.mock.invocationCallOrder[0];
    expect(syncTriggerOrder).toBeGreaterThan(upsertOrder);
  });
});
