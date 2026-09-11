import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { ReadArticleDetailScreen } from './ReadArticleDetailScreen';
import { ArticleStore, ReadService } from '../services';
import { useNetworkStatus } from '../hooks/useNetworkStatus';
import { Article } from '../types';
import type { UserContentResponse, UserContentDetailResponse } from '@cairn/shared';

// task_a8a4: opening a previously-read article must render its cached body
// (decision 4: opportunistic body caching) without waiting on the network
// getContentById call to resolve.
// task_c55c: a stored body whose hash matches the fresh route-param hash
// skips the network call entirely, and offline with nothing stored shows an
// explicit "Not available offline" state instead of a blank body.

jest.mock('../services', () => ({
  ArticleStore: {
    getById: jest.fn(),
    saveBody: jest.fn(),
    updateUserState: jest.fn(),
    remove: jest.fn(),
  },
  ReadService: {
    getContentById: jest.fn(),
    updateUserContent: jest.fn(),
    transformDetailToArticle: jest.fn(),
  },
}));

jest.mock('../hooks/useNetworkStatus');

jest.mock('../components/common', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  return {
    ArticleContent: ({ article }: { article: { content?: string } }) =>
      // eslint-disable-next-line @typescript-eslint/no-require-imports
      require('react').createElement(Text, null, article.content ?? 'NO CONTENT'),
    BottomActionMenu: () => null,
  };
});

let mockRouteParams: Record<string, unknown>;
jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ goBack: jest.fn(), replace: jest.fn() }),
  useRoute: () => ({ params: mockRouteParams }),
}));

const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;
const mockedReadService = ReadService as jest.Mocked<typeof ReadService>;
const mockedUseNetworkStatus = useNetworkStatus as jest.Mock;

const summaryArticle: Article = {
  id: 'a1',
  url: 'https://example.com/a1',
  title: 'Article One',
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: Date.now(),
  content: undefined,
};

describe('ReadArticleDetailScreen offline body cache', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRouteParams = { article: summaryArticle };
    mockedArticleStore.updateUserState.mockResolvedValue(undefined);
    mockedArticleStore.saveBody.mockResolvedValue(undefined);
    mockedReadService.updateUserContent.mockResolvedValue({} as UserContentResponse);
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
  });

  it('renders a stored body without waiting for getContentById to resolve', async () => {
    mockedArticleStore.getById.mockResolvedValue({ ...summaryArticle, content: '<p>Cached</p>' });
    // Never resolves during this test.
    mockedReadService.getContentById.mockReturnValue(new Promise(() => {}));

    render(<ReadArticleDetailScreen />);

    expect(await screen.findByText('<p>Cached</p>')).toBeTruthy();
    expect(mockedReadService.getContentById).toHaveBeenCalled();
  });

  it('renders a stored body with no network call when the stored hash matches the route-param hash', async () => {
    mockRouteParams = { article: { ...summaryArticle, contentHash: 'hash-1' } };
    mockedArticleStore.getById.mockResolvedValue({
      ...summaryArticle,
      content: '<p>Current</p>',
      contentHash: 'hash-1',
    });

    render(<ReadArticleDetailScreen />);

    expect(await screen.findByText('<p>Current</p>')).toBeTruthy();
    expect(mockedReadService.getContentById).not.toHaveBeenCalled();
  });

  it('still fetches when the stored hash differs from the route-param hash', async () => {
    mockRouteParams = { article: { ...summaryArticle, contentHash: 'hash-2' } };
    mockedArticleStore.getById.mockResolvedValue({
      ...summaryArticle,
      content: '<p>Stale</p>',
      contentHash: 'hash-1',
    });
    mockedReadService.getContentById.mockReturnValue(new Promise(() => {}));

    render(<ReadArticleDetailScreen />);

    // The stale stored body still renders immediately...
    expect(await screen.findByText('<p>Stale</p>')).toBeTruthy();
    // ...but the hash mismatch means a refresh was still kicked off.
    expect(mockedReadService.getContentById).toHaveBeenCalled();
  });

  it('shows "Not available offline" when offline with no stored body', async () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: true });
    mockedArticleStore.getById.mockResolvedValue(null);

    render(<ReadArticleDetailScreen />);

    expect(await screen.findByText('Not available offline')).toBeTruthy();
    expect(mockedReadService.getContentById).not.toHaveBeenCalled();
  });

  it('renders the stored body offline instead of the unavailable state when one is cached', async () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: true });
    mockedArticleStore.getById.mockResolvedValue({ ...summaryArticle, content: '<p>Cached</p>' });

    render(<ReadArticleDetailScreen />);

    expect(await screen.findByText('<p>Cached</p>')).toBeTruthy();
    expect(screen.queryByText('Not available offline')).toBeNull();
    expect(mockedReadService.getContentById).not.toHaveBeenCalled();
  });

  // task_06e5: regaining connectivity while this screen is open used to
  // leave it stuck on a blank ArticleContent — the render guard
  // (`!article.content && isOffline`) stopped matching once isOffline went
  // false, but nothing re-triggered the fetch that would give it content.
  it('fetches and renders content once connectivity returns, never falling through to a blank body', async () => {
    mockedUseNetworkStatus.mockReturnValue({ isOffline: true });
    mockedArticleStore.getById.mockResolvedValue(null);

    const { rerender } = render(<ReadArticleDetailScreen />);

    expect(await screen.findByText('Not available offline')).toBeTruthy();
    expect(mockedReadService.getContentById).not.toHaveBeenCalled();

    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    mockedReadService.getContentById.mockResolvedValue(
      { content_id: 'a1' } as unknown as UserContentDetailResponse,
    );
    mockedReadService.transformDetailToArticle.mockReturnValue({
      ...summaryArticle,
      content: '<p>Fresh</p>',
    });

    rerender(<ReadArticleDetailScreen />);

    expect(await screen.findByText('<p>Fresh</p>')).toBeTruthy();
    expect(screen.queryByText('Not available offline')).toBeNull();
    expect(screen.queryByText('NO CONTENT')).toBeNull();
    expect(mockedReadService.getContentById).toHaveBeenCalled();
  });
});
