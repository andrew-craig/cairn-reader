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
});
