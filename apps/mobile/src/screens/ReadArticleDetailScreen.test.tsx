import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { ReadArticleDetailScreen } from './ReadArticleDetailScreen';
import { ArticleStore, ReadService } from '../services';
import { Article } from '../types';
import type { UserContentResponse } from '@cairn/shared';

// task_a8a4: opening a previously-read article must render its cached body
// (decision 4: opportunistic body caching) without waiting on the network
// getContentById call to resolve.

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
  });

  it('renders a stored body without waiting for getContentById to resolve', async () => {
    mockedArticleStore.getById.mockResolvedValue({ ...summaryArticle, content: '<p>Cached</p>' });
    // Never resolves during this test.
    mockedReadService.getContentById.mockReturnValue(new Promise(() => {}));

    render(<ReadArticleDetailScreen />);

    expect(await screen.findByText('<p>Cached</p>')).toBeTruthy();
    expect(mockedReadService.getContentById).toHaveBeenCalled();
  });
});
