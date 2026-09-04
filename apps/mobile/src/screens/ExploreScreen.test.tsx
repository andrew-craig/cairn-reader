import React from 'react';
import { ActivityIndicator, FlatList } from 'react-native';
import { render, screen, fireEvent, act } from '@testing-library/react-native';
import { ExploreScreen } from './ExploreScreen';
import { ExploreService } from '../services';
import { StorageService } from '../services/storage';
import { Article } from '../types';

jest.mock('../services', () => ({
  ExploreService: {
    getRecommendations: jest.fn(),
    markShown: jest.fn(),
  },
}));

jest.mock('../services/storage', () => ({
  StorageService: {
    getExploreCache: jest.fn(),
    saveExploreCache: jest.fn(),
  },
}));

jest.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({ logout: jest.fn() }),
}));

jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({
    navigate: jest.fn(),
    addListener: jest.fn(() => () => {}),
  }),
}));

const mockedExploreService = ExploreService as jest.Mocked<typeof ExploreService>;
const mockedStorageService = StorageService as jest.Mocked<typeof StorageService>;

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((res) => {
    resolve = res;
  });
  return { promise, resolve };
}

const article = (id: string): Article => ({
  id,
  url: `https://example.com/${id}`,
  title: `Article ${id}`,
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: Date.now(),
});

describe('ExploreScreen retry/refresh spinner', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockedStorageService.getExploreCache.mockResolvedValue(null);
    mockedExploreService.markShown.mockResolvedValue(undefined);
  });

  it('shows the full-screen spinner when retrying from an empty/error state', async () => {
    // Short page (< RECOMMENDATION_PAGE_SIZE) so no cache and no articles on
    // screen when the load fails.
    mockedExploreService.getRecommendations.mockRejectedValueOnce(new Error('network down'));

    render(<ExploreScreen />);

    const retryButton = await screen.findByLabelText('Retry loading');

    const retry = deferred<Article[]>();
    mockedExploreService.getRecommendations.mockReturnValueOnce(retry.promise);

    fireEvent.press(retryButton);

    expect(screen.UNSAFE_queryAllByType(ActivityIndicator).length).toBeGreaterThan(0);

    await act(async () => {
      retry.resolve([article('a1')]);
    });
  });

  it('does not show the full-screen spinner when pulling to refresh with articles already on screen', async () => {
    mockedExploreService.getRecommendations.mockResolvedValueOnce([article('a1')]);

    render(<ExploreScreen />);

    await screen.findByText('Article a1');

    const refresh = deferred<Article[]>();
    mockedExploreService.getRecommendations.mockReturnValueOnce(refresh.promise);

    act(() => {
      screen.UNSAFE_getByType(FlatList).props.onRefresh();
    });

    expect(screen.UNSAFE_queryAllByType(ActivityIndicator).length).toBe(0);

    await act(async () => {
      refresh.resolve([article('a1')]);
    });
  });
});
