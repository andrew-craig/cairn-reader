import { ReadService } from './read';
import { UserContentResponse } from '../types/read';

describe('ReadService.transformToArticle', () => {
  const baseUserContent: UserContentResponse = {
    id: 'uc-1',
    user_id: 'user-1',
    content_id: 'content-1',
    status: 'unread',
    scroll_position: 0,
    is_favorite: false,
    added_at: '2025-01-15T10:00:00Z',
    updated_at: '2025-01-15T10:00:00Z',
    content: {
      id: 'content-1',
      content_hash: 'abc123',
      cleaned_html: '<p>Article body</p>',
      original_url: 'https://example.com/article',
      title: 'Test Article',
      author: 'John Doe',
      published_at: '2025-01-10T08:00:00Z',
      description: 'A test description',
      excerpt: 'A test excerpt',
      site_name: 'Example',
      lead_image_url: 'https://example.com/image.jpg',
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
    expect(article.content).toBe('<p>Article body</p>');
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

  it('uses description, falls back to excerpt', () => {
    const article = ReadService.transformToArticle(baseUserContent);
    expect(article.description).toBe('A test description');

    const noDesc = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, description: undefined },
    };
    const fallback = ReadService.transformToArticle(noDesc);
    expect(fallback.description).toBe('A test excerpt');
  });

  it('uses author, falls back to site_name', () => {
    const noAuthor = {
      ...baseUserContent,
      content: { ...baseUserContent.content!, author: undefined },
    };
    const article = ReadService.transformToArticle(noAuthor);
    expect(article.author).toBe('Example');
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
