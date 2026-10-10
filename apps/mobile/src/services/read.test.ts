import { ReadService } from './read';
import { AuthService } from './auth';
import { UserContentResponse } from '@cairn/shared';

jest.mock('./auth', () => ({
  AuthService: {
    getUserId: jest.fn(),
    fetchWithAuth: jest.fn(),
    fetchWithAuthAndRetry: jest.fn(),
  },
}));
jest.mock('@cairn/shared', () => ({
  ...jest.requireActual('@cairn/shared'),
  getServerUrl: () => 'https://api.test',
}));

const mockedAuth = AuthService as jest.Mocked<typeof AuthService>;

const okResponse = (body: unknown) =>
  ({ ok: true, status: 200, json: async () => body, text: async () => JSON.stringify(body) }) as unknown as Response;
const errorResponse = (status: number, body: unknown) =>
  ({ ok: false, status, json: async () => body, text: async () => JSON.stringify(body) }) as unknown as Response;

describe('ReadService.transformToArticle', () => {
  const baseUserContent: UserContentResponse = {
    id: 'uc-1',
    user_id: 'user-1',
    content_id: 'content-1',
    status: 'unread',
    list: 'reads',
    scroll_position: 0,
    is_favorite: false,
    added_at: '2025-01-15T10:00:00Z',
    updated_at: '2025-01-15T10:00:00Z',
    content: {
      id: 'content-1',
      content_hash: 'abc123',
      original_url: 'https://example.com/article',
      title: 'Test Article',
      author: 'John Doe',
      published_at: '2025-01-10T08:00:00Z',
      description: 'A test description',
      image_urls: ['https://example.com/image.jpg'],
      source_type: 'rss',
      word_count: 1000,
      created_at: '2025-01-15T09:00:00Z',
      updated_at: '2025-01-15T09:00:00Z',
    },
  };

  it('transforms basic fields correctly', () => {
    const article = ReadService.transformToArticle(baseUserContent);

    expect(article.id).toBe('content-1');
    expect(article.url).toBe('https://example.com/article');
    expect(article.title).toBe('Test Article');
    expect(article.author).toBe('John Doe');
    expect(article.imageUrl).toBe('https://example.com/image.jpg');
    expect(article.publishedDate).toBe('2025-01-10T08:00:00Z');
    expect(article.content).toBeUndefined();
  });

  it('calculates reading time from word count (200 wpm)', () => {
    const article = ReadService.transformToArticle(baseUserContent);
    // 1000 words / 200 wpm = 5 minutes
    expect(article.readingTime).toBe(5);
  });

  it('returns undefined readingTime when word_count is missing', () => {
    const noWordCount = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, word_count: undefined },
    };
    const article = ReadService.transformToArticle(noWordCount);
    expect(article.readingTime).toBeUndefined();
  });

  it('maps status "completed" to isRead true', () => {
    const completed = { ...baseUserContent, status: 'completed' as const };
    const article = ReadService.transformToArticle(completed);
    expect(article.isRead).toBe(true);
    expect(article.readAt).toBeDefined();
  });

  it('maps status "unread" to isRead false', () => {
    const article = ReadService.transformToArticle(baseUserContent);
    expect(article.isRead).toBe(false);
    expect(article.readAt).toBeUndefined();
  });

  it('maps is_favorite correctly', () => {
    const favorited = { ...baseUserContent, is_favorite: true };
    const article = ReadService.transformToArticle(favorited);
    expect(article.isFavorite).toBe(true);
  });

  it('converts added_at to numeric timestamp', () => {
    const article = ReadService.transformToArticle(baseUserContent);
    expect(typeof article.addedAt).toBe('number');
    expect(article.addedAt).toBe(new Date('2025-01-15T10:00:00Z').getTime());
  });

  it('maps description from content.description', () => {
    const article = ReadService.transformToArticle(baseUserContent);
    expect(article.description).toBe('A test description');

    const noDesc = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, description: undefined },
    };
    const missing = ReadService.transformToArticle(noDesc);
    expect(missing.description).toBeUndefined();
  });

  it('maps imageUrl from the first image_urls entry', () => {
    const article = ReadService.transformToArticle(baseUserContent);
    expect(article.imageUrl).toBe('https://example.com/image.jpg');

    const noImages = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, image_urls: undefined },
    };
    const missing = ReadService.transformToArticle(noImages);
    expect(missing.imageUrl).toBeUndefined();
  });

  it('leaves author undefined when content.author is absent', () => {
    const noAuthor = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, author: undefined },
    };
    const article = ReadService.transformToArticle(noAuthor);
    expect(article.author).toBeUndefined();
  });

  it('handles missing content (fallback path)', () => {
    const noContent: UserContentResponse = {
      ...baseUserContent,
      content: undefined,
    };
    const article = ReadService.transformToArticle(noContent);

    expect(article.id).toBe('content-1');
    expect(article.title).toBe('Unknown Article');
    expect(article.url).toBe('');
    expect(article.tags).toEqual([]);
  });

  it('rounds up reading time for partial minutes', () => {
    const oddWordCount = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, word_count: 201 },
    };
    const article = ReadService.transformToArticle(oddWordCount);
    // 201 / 200 = 1.005 -> ceil -> 2
    expect(article.readingTime).toBe(2);
  });
});

describe('ReadService list routing', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.spyOn(console, 'error').mockImplementation(() => {});
    mockedAuth.getUserId.mockResolvedValue('user-1');
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  describe('listUserContents / searchUserContents / countUserContents', () => {
    it('sends the list filter when given', async () => {
      mockedAuth.fetchWithAuthAndRetry.mockResolvedValue(okResponse({ data: [], pagination: {} }));
      await ReadService.listUserContents({ list: 'feed', limit: 20 });
      const url = new URL(mockedAuth.fetchWithAuthAndRetry.mock.calls[0][0] as string);
      expect(url.pathname).toBe('/api/v1/content/user/user-1');
      expect(url.searchParams.get('list')).toBe('feed');
    });

    it('omits the list filter when not given', async () => {
      mockedAuth.fetchWithAuthAndRetry.mockResolvedValue(okResponse({ data: [], pagination: {} }));
      await ReadService.listUserContents({ limit: 20 });
      const url = new URL(mockedAuth.fetchWithAuthAndRetry.mock.calls[0][0] as string);
      expect(url.searchParams.has('list')).toBe(false);
    });

    it('scopes search to a list', async () => {
      mockedAuth.fetchWithAuthAndRetry.mockResolvedValue(okResponse({ data: [], pagination: {} }));
      await ReadService.searchUserContents({ q: 'go', list: 'reads' });
      const url = new URL(mockedAuth.fetchWithAuthAndRetry.mock.calls[0][0] as string);
      expect(url.searchParams.get('q')).toBe('go');
      expect(url.searchParams.get('list')).toBe('reads');
    });

    it('scopes count to a list', async () => {
      mockedAuth.fetchWithAuthAndRetry.mockResolvedValue(okResponse({ data: { count: 3 } }));
      await expect(ReadService.countUserContents({ list: 'feed' })).resolves.toBe(3);
      const url = new URL(mockedAuth.fetchWithAuthAndRetry.mock.calls[0][0] as string);
      expect(url.searchParams.get('list')).toBe('feed');
    });
  });

  describe('setSourceList', () => {
    it('PUTs the list for an RSS feed', async () => {
      mockedAuth.fetchWithAuth.mockResolvedValue(
        okResponse({ data: { type: 'rss', key: 'feed-1', list: 'feed' } }),
      );
      await ReadService.setSourceList('rss', 'feed-1', 'feed');
      const [url, init] = mockedAuth.fetchWithAuth.mock.calls[0];
      expect(url).toBe('https://api.test/api/v1/content/user/user-1/subscriptions/rss/feed-1/list');
      expect(init).toMatchObject({ method: 'PUT', body: JSON.stringify({ list: 'feed' }) });
    });

    it('PUTs the list for an email sender', async () => {
      mockedAuth.fetchWithAuth.mockResolvedValue(
        okResponse({ data: { type: 'email', key: 'sender-1', list: 'reads' } }),
      );
      await ReadService.setSourceList('email', 'sender-1', 'reads');
      expect(mockedAuth.fetchWithAuth.mock.calls[0][0]).toBe(
        'https://api.test/api/v1/content/user/user-1/subscriptions/email/sender-1/list',
      );
    });

    it('throws with the server message on failure', async () => {
      mockedAuth.fetchWithAuth.mockResolvedValue(errorResponse(404, { message: 'not subscribed' }));
      await expect(ReadService.setSourceList('rss', 'feed-1', 'feed')).rejects.toThrow('not subscribed');
    });

    it('throws when not authenticated', async () => {
      mockedAuth.getUserId.mockResolvedValue(null);
      await expect(ReadService.setSourceList('rss', 'feed-1', 'feed')).rejects.toThrow('Not authenticated');
    });
  });

  describe('moveToReads', () => {
    it('PATCHes list=reads on the item', async () => {
      mockedAuth.fetchWithAuth.mockResolvedValue(okResponse({ data: { id: 'uc-1', list: 'reads' } }));
      await ReadService.moveToReads('content-1');
      const [url, init] = mockedAuth.fetchWithAuth.mock.calls[0];
      expect(url).toBe('https://api.test/api/v1/content/user/user-1/content-1');
      expect(init).toMatchObject({ method: 'PATCH', body: JSON.stringify({ list: 'reads' }) });
    });
  });

  describe('addURL', () => {
    it('forwards the chosen list for a feed', async () => {
      mockedAuth.fetchWithAuth.mockResolvedValue(okResponse({ data: { type: 'feed' } }));
      await ReadService.addURL({ url: 'https://example.com/rss', type: 'feed', list: 'feed' });
      const init = mockedAuth.fetchWithAuth.mock.calls[0][1] as RequestInit;
      expect(JSON.parse(init.body as string)).toMatchObject({ type: 'feed', list: 'feed' });
    });
  });
});
