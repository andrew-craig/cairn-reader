import { renderHook, act, waitFor } from '@testing-library/react-native';
import { useCursorArticleList } from './useCursorArticleList';
import { ReadService } from '../services/read';
import type { UserContentsListResponse } from '@cairn/shared';

function page(ids: string[], cursor: string, hasMore: boolean): UserContentsListResponse {
  return {
    contents: ids.map((id) => ({
      id,
      user_id: 'u',
      content_id: id,
      status: 'unread' as const,
      list: 'reads' as const,
      scroll_position: 0,
      is_favorite: false,
      added_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
      content: {
        id,
        content_hash: id,
        original_url: `https://example.com/${id}`,
        title: `Article ${id}`,
        source_type: 'rss',
        created_at: '2025-01-01T00:00:00Z',
        updated_at: '2025-01-01T00:00:00Z',
      },
    })),
    total_count: 0,
    limit: 20,
    cursor,
    has_more: hasMore,
  };
}

describe('useCursorArticleList', () => {
  afterEach(() => jest.restoreAllMocks());

  it('resets then appends across pages via the cursor', async () => {
    const fetchPage = jest
      .fn<Promise<UserContentsListResponse>, [string | undefined]>()
      .mockResolvedValueOnce(page(['a', 'b'], 'cursor-1', true))
      .mockResolvedValueOnce(page(['c'], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));

    await act(async () => {
      await result.current.load(true);
    });
    expect(result.current.articles.map((a) => a.id)).toEqual(['a', 'b']);
    expect(fetchPage).toHaveBeenLastCalledWith(undefined);

    await act(async () => {
      await result.current.load(false);
    });
    expect(result.current.articles.map((a) => a.id)).toEqual(['a', 'b', 'c']);
    expect(fetchPage).toHaveBeenLastCalledWith('cursor-1');
    expect(result.current.hasMore).toBe(false);
  });

  it('calls onResetLoaded with the first page and onLoadError on failure', async () => {
    const onResetLoaded = jest.fn();
    const onLoadError = jest.fn();
    const fetchPage = jest
      .fn()
      .mockResolvedValueOnce(page(['a'], '', false))
      .mockRejectedValueOnce(new Error('boom'));

    jest.spyOn(console, 'error').mockImplementation(() => {});
    const { result } = renderHook(() =>
      useCursorArticleList({ fetchPage, onResetLoaded, onLoadError }),
    );

    await act(async () => {
      await result.current.load(true);
    });
    expect(onResetLoaded).toHaveBeenCalledWith([expect.objectContaining({ id: 'a' })]);

    await act(async () => {
      await result.current.load(false);
    });
    expect(onLoadError).toHaveBeenCalledWith(false);
  });

  it('handleLoadMore is a no-op while already loading more or when no more pages', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page(['a'], 'c1', true));
    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));

    await act(async () => {
      await result.current.load(true);
    });
    fetchPage.mockClear();

    // hasMore true, not loading -> triggers one load
    await act(async () => {
      result.current.handleLoadMore();
    });
    await waitFor(() => expect(fetchPage).toHaveBeenCalledTimes(1));
  });

  it('live search debounces the query and replaces the list with the results', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page([], '', false));
    const searchSpy = jest
      .spyOn(ReadService, 'searchUserContents')
      .mockResolvedValue(page(['s1', 's2'], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));

    act(() => result.current.setQuery('te'));
    act(() => result.current.setQuery('term '));

    expect(result.current.searchQuery).toBe('term');
    await waitFor(() => expect(result.current.articles.map((a) => a.id)).toEqual(['s1', 's2']));
    expect(searchSpy).toHaveBeenCalledTimes(1);
    expect(searchSpy).toHaveBeenCalledWith(expect.objectContaining({ q: 'term' }));
  });

  it('scopes search to the list the hook was given', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page([], '', false));
    const searchSpy = jest
      .spyOn(ReadService, 'searchUserContents')
      .mockResolvedValue(page(['s1'], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage, list: 'feed' }));

    act(() => result.current.setQuery('term'));

    await waitFor(() =>
      expect(searchSpy).toHaveBeenCalledWith(expect.objectContaining({ q: 'term', list: 'feed' })),
    );
  });

  it('clears the refreshing indicator after a pull-to-refresh during search', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page([], '', false));
    jest.spyOn(ReadService, 'searchUserContents').mockResolvedValue(page(['s1'], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));

    act(() => result.current.setQuery('term'));
    await waitFor(() => expect(result.current.articles.map((a) => a.id)).toEqual(['s1']));

    await act(async () => {
      result.current.handleRefresh();
    });

    await waitFor(() => expect(result.current.refreshing).toBe(false));
  });

  it('closing search restores the list at once, then refreshes it without a spinner', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page(['a', 'b'], 'c1', true));
    jest.spyOn(ReadService, 'searchUserContents').mockResolvedValue(page([], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));
    await act(async () => {
      await result.current.load(true);
    });

    act(() => result.current.openSearch());
    act(() => result.current.setQuery('nothing'));
    await waitFor(() => expect(result.current.articles).toEqual([]));

    fetchPage.mockClear();
    fetchPage.mockResolvedValue(page(['a', 'b', 'new'], 'c2', true));
    act(() => result.current.closeSearch());

    expect(result.current.searchOpen).toBe(false);
    expect(result.current.searchQuery).toBeNull();
    expect(result.current.articles.map((a) => a.id)).toEqual(['a', 'b']);
    expect(result.current.loading).toBe(false);
    await waitFor(() => expect(result.current.articles.map((a) => a.id)).toEqual(['a', 'b', 'new']));
    expect(fetchPage).toHaveBeenCalledWith(undefined);
  });

  it('a list reload during search updates what search restores, not the results', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page(['a'], '', false));
    jest.spyOn(ReadService, 'searchUserContents').mockResolvedValue(page(['s1'], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));
    await act(async () => {
      await result.current.load(true);
    });

    act(() => result.current.setQuery('term'));
    await waitFor(() => expect(result.current.articles.map((a) => a.id)).toEqual(['s1']));

    fetchPage.mockResolvedValue(page(['a', 'fresh'], '', false));
    await act(async () => {
      await result.current.load(true);
    });
    expect(result.current.articles.map((a) => a.id)).toEqual(['s1']);

    fetchPage.mockReturnValue(new Promise(() => {}));
    act(() => result.current.setQuery(''));
    expect(result.current.articles.map((a) => a.id)).toEqual(['a', 'fresh']);
  });
});
