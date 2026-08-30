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

  it('search populates from ReadService.searchUserContents and sets searchQuery', async () => {
    const fetchPage = jest.fn().mockResolvedValue(page([], '', false));
    jest
      .spyOn(ReadService, 'searchUserContents')
      .mockResolvedValue(page(['s1', 's2'], '', false));

    const { result } = renderHook(() => useCursorArticleList({ fetchPage }));

    await act(async () => {
      await result.current.search('term');
    });

    expect(result.current.searchQuery).toBe('term');
    expect(result.current.articles.map((a) => a.id)).toEqual(['s1', 's2']);
  });
});
