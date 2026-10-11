import React from 'react';
import { render, screen, waitFor } from '@testing-library/react-native';
import { BookmarksScreen } from './BookmarksScreen';
import { ReadService } from '../services/read';
import { ArticleStore } from '../services/articleStore';
import { Article } from '../types';

// task_a8a4: BookmarksScreen had no local cache at all before this change —
// a network failure blanked the list to the error state even when the
// favorite was already known locally. It must now render from the store
// first, and a subsequent network failure must not blank that list.

jest.mock('../services/read', () => ({
  ReadService: {
    listUserContents: jest.fn(),
    transformToArticle: jest.requireActual('../services/read').ReadService.transformToArticle,
  },
}));

jest.mock('../services/articleStore', () => ({
  ArticleStore: {
    listFavorites: jest.fn(),
    upsertMany: jest.fn(),
  },
}));

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn() }),
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;

const article = (id: string): Article => ({
  id,
  url: `https://example.com/${id}`,
  title: `Favorite ${id}`,
  tags: [],
  isRead: false,
  isFavorite: true,
  list: 'reads' as const,
  addedAt: Date.now(),
});

describe('BookmarksScreen offline-first render', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedArticleStore.upsertMany.mockResolvedValue(undefined);
  });

  it('renders stored favorites while the network call is still pending', async () => {
    mockedArticleStore.listFavorites.mockResolvedValue([article('fav-1')]);
    mockedReadService.listUserContents.mockReturnValue(new Promise(() => {}));

    render(<BookmarksScreen />);

    expect(await screen.findByText('Favorite fav-1')).toBeTruthy();
  });

  it('does not blank the list when the network refresh fails', async () => {
    mockedArticleStore.listFavorites.mockResolvedValue([article('fav-1')]);
    mockedReadService.listUserContents.mockRejectedValue(new Error('network down'));

    render(<BookmarksScreen />);

    await screen.findByText('Favorite fav-1');

    await waitFor(() => expect(mockedReadService.listUserContents).toHaveBeenCalled());

    // The failure happened after the store already primed the list — it
    // must remain on screen instead of falling back to the error state.
    expect(screen.queryByText("Couldn't load your bookmarks. Check your connection and try again.")).toBeNull();
    expect(screen.getByText('Favorite fav-1')).toBeTruthy();
  });

  it('keeps favorited Feed items out of the offline store', async () => {
    mockedArticleStore.listFavorites.mockResolvedValue([]);
    mockedReadService.listUserContents.mockResolvedValue({
      contents: [
        userContent('reads-1', 'reads'),
        userContent('feed-1', 'feed'),
      ],
      total_count: 2,
      limit: 20,
      cursor: '',
      has_more: false,
    });

    render(<BookmarksScreen />);

    // Both are listed...
    expect(await screen.findByText('Item reads-1')).toBeTruthy();
    expect(screen.getByText('Item feed-1')).toBeTruthy();
    // ...but only the Reads one is cached for offline.
    await waitFor(() => expect(mockedArticleStore.upsertMany).toHaveBeenCalled());
    const stored = mockedArticleStore.upsertMany.mock.calls[0][0];
    expect(stored.map((a) => a.id)).toEqual(['reads-1']);
  });
});

function userContent(id: string, list: 'feed' | 'reads') {
  return {
    id: `uc-${id}`,
    user_id: 'u',
    content_id: id,
    status: 'unread' as const,
    list,
    scroll_position: 0,
    is_favorite: true,
    added_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
    content: {
      id,
      content_hash: id,
      original_url: `https://example.com/${id}`,
      title: `Item ${id}`,
      source_type: 'rss',
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    },
  };
}
