import React from 'react';
import { Alert } from 'react-native';
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react-native';
import { ReadArticleDetailScreen } from './ReadArticleDetailScreen';
import { ArticleStore, ArticleMutations } from '../services';
import { useNetworkStatus } from '../hooks/useNetworkStatus';
import { Article } from '../types';
import { HttpError } from '../utils/errors';

// task_ebf1 tech-lead review follow-up: two behaviors specific to the
// ArticleMutations facade wiring, not covered by
// ReadArticleDetailScreen.test.tsx (which mocks BottomActionMenu away
// entirely and never exercises these handlers).
//
// 1. handleToggleFavorite: ArticleMutations.setFavorite writes the store
//    before it (maybe) rethrows, so a definitive-failure rollback must undo
//    the store write too, not just the optimistic UI — otherwise this
//    screen and BookmarksScreen (listFavorites()) disagree until the next
//    list sync.
// 2. handleArchive: navigation must not wait on the backend delete — only a
//    genuine (non-NetworkError) failure should surface, via Alert, after
//    navigation has already happened.

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
  ArticleMutations: {
    markCompleted: jest.fn(),
    markReading: jest.fn(),
    saveScrollPosition: jest.fn(),
    setFavorite: jest.fn(),
    archive: jest.fn(),
  },
}));

jest.mock('../hooks/useNetworkStatus');

// Unlike ReadArticleDetailScreen.test.tsx, BottomActionMenu renders real
// pressable text nodes here (keyed by label) so these tests can fire the
// Favorite/Archive actions directly.
jest.mock('../components/common', () => {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { Text } = require('react-native');
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const ReactModule = require('react');
  return {
    ArticleContent: ({ article }: { article: { content?: string } }) =>
      ReactModule.createElement(Text, null, article.content ?? 'NO CONTENT'),
    BottomActionMenu: ({ actions }: { actions: { label: string; onPress: () => void; active?: boolean }[] }) =>
      ReactModule.createElement(
        ReactModule.Fragment,
        null,
        ...actions.map((action: { label: string; onPress: () => void; active?: boolean }) =>
          ReactModule.createElement(
            Text,
            { key: action.label, onPress: action.onPress },
            `${action.label}:${action.active ? 'on' : 'off'}`,
          ),
        ),
      ),
  };
});

let mockRouteParams: Record<string, unknown>;
const mockGoBack = jest.fn();
jest.mock('@react-navigation/native', () => ({
  useNavigation: () => ({ goBack: mockGoBack, replace: jest.fn() }),
  useRoute: () => ({ params: mockRouteParams }),
}));

const mockedArticleStore = ArticleStore as jest.Mocked<typeof ArticleStore>;
const mockedArticleMutations = ArticleMutations as jest.Mocked<typeof ArticleMutations>;
const mockedUseNetworkStatus = useNetworkStatus as jest.Mock;

const baseArticle: Article = {
  id: 'a1',
  url: 'https://example.com/a1',
  title: 'Article One',
  tags: [],
  isRead: true, // skip the "mark as reading" effect — irrelevant here
  isFavorite: false,
  addedAt: Date.now(),
  content: '<p>Body</p>',
};

describe('ReadArticleDetailScreen mutation call sites', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    mockRouteParams = { article: baseArticle, onArchived: jest.fn() };
    mockedUseNetworkStatus.mockReturnValue({ isOffline: false });
    mockedArticleStore.updateUserState.mockResolvedValue(undefined);
    mockedArticleStore.remove.mockResolvedValue(undefined);
  });

  describe('handleToggleFavorite', () => {
    it('rolls back both the UI and the store on a definitive failure', async () => {
      mockedArticleMutations.setFavorite.mockRejectedValue(new HttpError(422, 'nope'));

      render(<ReadArticleDetailScreen />);

      fireEvent.press(screen.getByText('Favorite:off'));

      // Optimistic update first.
      expect(await screen.findByText('Favorite:on')).toBeTruthy();

      // Rejection rolls the UI back...
      await waitFor(() => expect(screen.getByText('Favorite:off')).toBeTruthy());

      // ...and must roll the store back too — setFavorite(targetId, true)
      // already wrote isFavorite: true to the store before rethrowing.
      expect(mockedArticleStore.updateUserState).toHaveBeenCalledWith('a1', { isFavorite: false });
    });

    it('does not touch the store when the mutation succeeds', async () => {
      mockedArticleMutations.setFavorite.mockResolvedValue(undefined);

      render(<ReadArticleDetailScreen />);
      fireEvent.press(screen.getByText('Favorite:off'));

      await waitFor(() => expect(mockedArticleMutations.setFavorite).toHaveBeenCalled());
      expect(mockedArticleStore.updateUserState).not.toHaveBeenCalled();
    });
  });

  describe('handleArchive', () => {
    it('navigates back immediately without waiting on the backend delete', async () => {
      let resolveArchive: () => void;
      mockedArticleMutations.archive.mockReturnValue(
        new Promise((resolve) => {
          resolveArchive = resolve;
        }),
      );

      render(<ReadArticleDetailScreen />);
      fireEvent.press(screen.getByText('Archive:off'));

      // Navigation and onArchived happen synchronously, before the backend
      // promise has any chance to settle.
      expect(mockRouteParams.onArchived).toHaveBeenCalledWith('a1');
      expect(mockGoBack).toHaveBeenCalledTimes(1);

      await act(async () => {
        resolveArchive!();
        await Promise.resolve();
      });
    });

    it('surfaces a definitive failure via Alert after navigation, instead of swallowing it', async () => {
      const alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
      const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => {});
      mockedArticleMutations.archive.mockRejectedValue(new HttpError(403, 'forbidden'));

      render(<ReadArticleDetailScreen />);
      fireEvent.press(screen.getByText('Archive:off'));

      expect(mockGoBack).toHaveBeenCalledTimes(1);

      await waitFor(() =>
        expect(alertSpy).toHaveBeenCalledWith('Error', 'Failed to archive article'),
      );

      alertSpy.mockRestore();
      consoleErrorSpy.mockRestore();
    });

    it('does not alert when the backend delete succeeds', async () => {
      const alertSpy = jest.spyOn(Alert, 'alert').mockImplementation(() => {});
      mockedArticleMutations.archive.mockResolvedValue(undefined);

      render(<ReadArticleDetailScreen />);
      fireEvent.press(screen.getByText('Archive:off'));

      await waitFor(() => expect(mockedArticleMutations.archive).toHaveBeenCalledWith('a1'));
      expect(alertSpy).not.toHaveBeenCalled();

      alertSpy.mockRestore();
    });
  });
});
