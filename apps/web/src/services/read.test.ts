import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthService } from '@cairn/shared';
import type {
  UserContentDetailResponse,
  UserContentResponse,
} from '@cairn/shared';
import { ReadService } from './read';

// task_de6d: transformToArticle and transformDetailToArticle were near-identical
// mappers differing only in whether `content` (the cleaned HTML body) is set.
// transformDetailToArticle now composes on transformToArticle; these tests pin
// that the detail mapping still equals the summary mapping plus the HTML body.

const summary: UserContentResponse = {
  id: 'uc-1',
  user_id: 'user-1',
  content_id: 'content-1',
  status: 'completed',
  list: 'reads',
  scroll_position: 12,
  is_favorite: true,
  added_at: '2025-01-15T10:00:00Z',
  updated_at: '2025-01-16T10:00:00Z',
  content: {
    id: 'content-1',
    content_hash: 'abc123',
    original_url: 'https://example.com/article',
    title: 'Test Article',
    author: 'John Doe',
    published_at: '2025-01-10T08:00:00Z',
    description: 'A description',
    image_urls: ['https://example.com/a.jpg', 'https://example.com/b.jpg'],
    source_type: 'rss',
    word_count: 401,
    created_at: '2025-01-15T09:00:00Z',
    updated_at: '2025-01-15T09:00:00Z',
  },
};

const detail: UserContentDetailResponse = {
  ...summary,
  content: { ...summary.content!, cleaned_html: '<p>body</p>' },
};

describe('ReadService.transformToArticle', () => {
  it('maps the summary fields and leaves content undefined', () => {
    const article = ReadService.transformToArticle(summary);
    expect(article).toMatchObject({
      id: 'content-1',
      url: 'https://example.com/article',
      title: 'Test Article',
      description: 'A description',
      author: 'John Doe',
      imageUrl: 'https://example.com/a.jpg',
      publishedDate: '2025-01-10T08:00:00Z',
      readingTime: 3, // ceil(401 / 200)
      isRead: true,
      isFavorite: true,
      scrollPosition: 12,
    });
    expect(article.content).toBeUndefined();
    expect(article.addedAt).toBe(new Date('2025-01-15T10:00:00Z').getTime());
    expect(article.readAt).toBe(new Date('2025-01-16T10:00:00Z').getTime());
  });

  it('uses the fallback shape when the nested content is missing', () => {
    const article = ReadService.transformToArticle({ ...summary, content: undefined });
    expect(article).toMatchObject({
      id: 'content-1',
      url: '',
      title: 'Unknown Article',
      description: '',
      tags: [],
    });
  });
});

describe('ReadService.transformDetailToArticle', () => {
  it('equals the summary mapping plus the cleaned HTML body', () => {
    const base = ReadService.transformToArticle(summary);
    const full = ReadService.transformDetailToArticle(detail);

    expect(full).toEqual({ ...base, content: '<p>body</p>' });
  });

  it('keeps the fallback shape (no content set) when nested content is missing', () => {
    const full = ReadService.transformDetailToArticle({
      ...detail,
      content: undefined,
    });
    expect(full.content).toBeUndefined();
    expect(full.title).toBe('Unknown Article');
  });
});

// Feed/Reads routing (task_f4ac): list filter on read calls, moveToReads, setSourceList.
describe('ReadService list routing', () => {
  const fetchWithAuth = vi.fn();

  beforeEach(() => {
    fetchWithAuth.mockReset();
    vi.spyOn(AuthService, 'getUserId').mockResolvedValue('user-1');
    vi.spyOn(AuthService, 'fetchWithAuth').mockImplementation(fetchWithAuth);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const ok = (data: unknown, extra: object = {}) => ({
    ok: true,
    json: async () => ({ data, ...extra }),
  });

  it('sends list when listing, counting and searching', async () => {
    fetchWithAuth.mockResolvedValue(ok([], { pagination: {} }));
    await ReadService.listUserContents({ list: 'feed' });
    await ReadService.searchUserContents({ q: 'x', list: 'feed' });
    fetchWithAuth.mockResolvedValue(ok({ count: 3 }));
    await ReadService.countUserContents({ list: 'reads' });

    const urls = fetchWithAuth.mock.calls.map((c) => c[0] as string);
    expect(urls[0]).toContain('?list=feed');
    expect(urls[1]).toContain('/search?q=x&list=feed');
    expect(urls[2]).toContain('/count?list=reads');
  });

  it('omits list when not given', async () => {
    fetchWithAuth.mockResolvedValue(ok([], { pagination: {} }));
    await ReadService.listUserContents();
    expect(fetchWithAuth.mock.calls[0][0]).not.toContain('list=');
  });

  it('moveToReads patches the item to the reads list', async () => {
    fetchWithAuth.mockResolvedValue(ok({ id: 'uc-1' }));
    await ReadService.moveToReads('uc-1');
    const [url, init] = fetchWithAuth.mock.calls[0];
    expect(url).toContain('/content/user/user-1/uc-1');
    expect(init).toMatchObject({ method: 'PATCH', body: JSON.stringify({ list: 'reads' }) });
  });

  it('setSourceList PUTs the list to the source route', async () => {
    fetchWithAuth.mockResolvedValue(ok({ type: 'rss', key: 'feed-1', list: 'feed' }));
    const result = await ReadService.setSourceList('rss', 'feed-1', 'feed');
    const [url, init] = fetchWithAuth.mock.calls[0];
    expect(url).toContain('/content/user/user-1/subscriptions/rss/feed-1/list');
    expect(init).toMatchObject({ method: 'PUT', body: JSON.stringify({ list: 'feed' }) });
    expect(result.list).toBe('feed');
  });

  it('setSourceList surfaces the backend error message', async () => {
    fetchWithAuth.mockResolvedValue({ ok: false, json: async () => ({ message: 'nope' }) });
    await expect(ReadService.setSourceList('email', 's-1', 'reads')).rejects.toThrow('nope');
  });
});
