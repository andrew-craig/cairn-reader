import React from 'react';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react-native';
import { FeedScreen } from './FeedScreen';
import { ReadService } from '../services/read';
import { FeedCache } from '../services/feedCache';
import { ArticleStore } from '../services/articleStore';
import { Article } from '../types';

// Feed is online-only: items are fetched with list=feed, the last page is kept
// in FeedCache for a stale render, and nothing touches the offline Reads
// store (ArticleStore), its prefetch or the outbox.

jest.mock('../services/read', () => ({
  ReadService: {
    listUserContents: jest.fn(),
    transformToArticle: jest.requireActual('../services/read').ReadService.transformToArticle,
  },
}));

jest.mock('../services/feedCache', () => ({
  FeedCache: { save: jest.fn(), load: jest.fn(), clear: jest.fn() },
}));

jest.mock('../services/articleStore', () => ({
  ArticleStore: { listRecent: jest.fn(), upsertMany: jest.fn(), remove: jest.fn() },
}));

const mockNavigate = jest.fn();
jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: mockNavigate, goBack: jest.fn() }),
  // Minimal stand-in: run the focus callback on mount / when it changes.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedFeedCache = FeedCache as jest.Mocked<typeof FeedCache>;
const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;

const userContent = (id: string) => ({
  id: `uc-${id}`,
  user_id: 'u',
  content_id: id,
  status: 'unread' as const,
  list: 'feed' as const,
  scroll_position: 0,
  is_favorite: false,
  added_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-01T00:00:00Z',
  content: {
    id,
    content_hash: id,
    original_url: `https://example.com/${id}`,
    title: `Feed Item ${id}`,
    source_type: 'rss',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
  },
});

const page = (ids: string[]) => ({
  contents: ids.map(userContent),
  total_count: ids.length,
  limit: 20,
  cursor: '',
  has_more: false,
});

const cached = (id: string): Article => ({
  id,
  url: `https://example.com/${id}`,
  title: `Cached Item ${id}`,
  tags: [],
  isRead: false,
  isFavorite: false,
  list: 'feed',
  addedAt: 1,
});

describe('FeedScreen', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    mockedFeedCache.save.mockResolvedValue(undefined);
    mockedFeedCache.load.mockResolvedValue([]);
  });

  afterEach(() => jest.restoreAllMocks());

  it('fetches only the feed list and renders it', async () => {
    mockedReadService.listUserContents.mockResolvedValue(page(['a', 'b']));

    render(<FeedScreen />);

    expect(await screen.findByText('Feed Item a')).toBeTruthy();
    expect(screen.getByText('Feed Item b')).toBeTruthy();
    expect(mockedReadService.listUserContents).toHaveBeenCalledWith(
      expect.objectContaining({ list: 'feed' }),
    );
  });

  it('caches the fetched page for a later stale render', async () => {
    mockedReadService.listUserContents.mockResolvedValue(page(['a']));

    render(<FeedScreen />);
    await screen.findByText('Feed Item a');

    await waitFor(() => expect(mockedFeedCache.save).toHaveBeenCalled());
    const saved = mockedFeedCache.save.mock.calls[0][0];
    expect(saved.map((a) => a.id)).toEqual(['a']);
  });

  it('never touches the offline Reads store', async () => {
    mockedReadService.listUserContents.mockResolvedValue(page(['a']));

    render(<FeedScreen />);
    await screen.findByText('Feed Item a');

    expect(mockedArticleStore.upsertMany).not.toHaveBeenCalled();
    expect(mockedArticleStore.listRecent).not.toHaveBeenCalled();
  });

  it('shows the cached page with a stale banner when the refresh fails', async () => {
    mockedFeedCache.load.mockResolvedValue([cached('old')]);
    mockedReadService.listUserContents.mockRejectedValue(new Error('offline'));

    render(<FeedScreen />);

    expect(await screen.findByText('Cached Item old')).toBeTruthy();
    expect(await screen.findByText('Showing cached data — pull to refresh')).toBeTruthy();
  });

  it('shows an error with Retry when the refresh fails and nothing is cached', async () => {
    mockedReadService.listUserContents.mockRejectedValue(new Error('offline'));

    render(<FeedScreen />);

    expect(
      await screen.findByText("Couldn't load your Feed. Check your connection and try again."),
    ).toBeTruthy();

    mockedReadService.listUserContents.mockResolvedValue(page(['a']));
    fireEvent.press(screen.getByText('Retry'));
    expect(await screen.findByText('Feed Item a')).toBeTruthy();
  });

  it('shows an empty state for an empty Feed', async () => {
    mockedReadService.listUserContents.mockResolvedValue(page([]));

    render(<FeedScreen />);

    expect(await screen.findByText('Your Feed is empty')).toBeTruthy();
  });

  it('opens an item in the reader, and drops it from the list when the reader reports it left the Feed', async () => {
    mockedReadService.listUserContents.mockResolvedValue(page(['a', 'b']));

    render(<FeedScreen />);
    fireEvent.press(await screen.findByText('Feed Item a'));

    expect(mockNavigate).toHaveBeenCalledWith(
      'ArticleDetail',
      expect.objectContaining({
        article: expect.objectContaining({ id: 'a', list: 'feed' }),
        currentIndex: 0,
      }),
    );

    const { onArchived } = mockNavigate.mock.calls[0][1];
    act(() => onArchived('a'));

    await waitFor(() => expect(screen.queryByText('Feed Item a')).toBeNull());
    expect(screen.getByText('Feed Item b')).toBeTruthy();
    expect(mockedFeedCache.save).toHaveBeenLastCalledWith(
      expect.arrayContaining([expect.objectContaining({ id: 'b' })]),
    );
  });
});
