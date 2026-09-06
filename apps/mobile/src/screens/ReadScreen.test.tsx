import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { ReadScreen } from './ReadScreen';
import { ReadService } from '../services/read';
import { ArticleStore } from '../services/articleStore';
import { Article } from '../types';

// task_a8a4: ReadScreen must render articles already in the local SQLite
// store immediately on focus, before the network page for the same list has
// resolved — the store is the read-through cache for the initial render.

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

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ navigate: jest.fn(), goBack: jest.fn() }),
  // Minimal stand-in: run the focus callback on mount / when it changes.
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  useFocusEffect: (cb: () => void) => require('react').useEffect(() => cb(), [cb]),
}));

const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;

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
  });

  it('renders stored articles before the network call resolves', async () => {
    mockedArticleStore.listRecent.mockResolvedValue([article('stored-1')]);
    // Never resolves during this test.
    mockedReadService.listUserContents.mockReturnValue(new Promise(() => {}));

    render(<ReadScreen />);

    expect(await screen.findByText('Stored Article stored-1')).toBeTruthy();
    expect(mockedReadService.listUserContents).toHaveBeenCalled();
  });
});
