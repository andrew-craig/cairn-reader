// Web Read service. Ports the subset of apps/mobile/src/services/read.ts needed
// so far — the reading list (listUserContents + transformToArticle) and the
// sidebar You-section counts (bookmarks total and subscription split). Built on
// AuthService.fetchWithAuth (proactive refresh + 401 retry), mirroring mobile.
// Later tasks extend this with search and add-link methods.
import {
  type Article,
  type AddURLRequest,
  type AddURLResponse,
  type DetectURLResponse,
  type DiscoverFeedResponse,
  type ListContentsParams,
  type SearchParams,
  type UpdateUserContentRequest,
  type UserContentResponse,
  type UserContentDetailResponse,
  type UserContentsListResponse,
  type UnifiedSubscriptionsResponse,
  getServerUrl,
} from '@cairn/shared';
import { AuthService } from './auth';

const PAGE_SIZE_DEFAULT = 20;

/** Reading-list page size, mirroring mobile's PAGE_SIZE. */
export const PAGE_SIZE = 20;

export class ReadService {
  /**
   * Transform a backend UserContentResponse (summary) into the UI Article shape.
   * Summary responses omit cleaned_html, so the returned Article has
   * content=undefined; use getUserContent/transformDetailToArticle to fill in
   * the HTML body. Mirrors apps/mobile/src/services/read.ts.
   */
  static transformToArticle(userContent: UserContentResponse): Article {
    const content = userContent.content;

    if (!content) {
      // Fallback when the list response omits the nested content.
      return {
        id: userContent.content_id,
        url: '',
        title: 'Unknown Article',
        description: '',
        tags: [],
        isRead: userContent.status === 'completed',
        isFavorite: userContent.is_favorite,
        addedAt: new Date(userContent.added_at).getTime(),
        scrollPosition: userContent.scroll_position || undefined,
      };
    }

    return {
      id: content.id,
      url: content.original_url,
      title: content.title,
      description: content.description,
      // cleaned_html is not present in summary responses — loaded on demand.
      content: undefined,
      imageUrl: content.image_urls?.[0],
      author: content.author,
      publishedDate: content.published_at,
      readingTime: content.word_count ? Math.ceil(content.word_count / 200) : undefined,
      tags: [],
      isRead: userContent.status === 'completed',
      isFavorite: userContent.is_favorite,
      addedAt: new Date(userContent.added_at).getTime(),
      readAt:
        userContent.status === 'completed'
          ? new Date(userContent.updated_at).getTime()
          : undefined,
      scrollPosition: userContent.scroll_position || undefined,
    };
  }

  /**
   * Transform a full UserContentDetailResponse (with cleaned_html) into an
   * Article. Mirrors apps/mobile/src/services/read.ts.
   */
  static transformDetailToArticle(userContent: UserContentDetailResponse): Article {
    const content = userContent.content;

    if (!content) {
      return {
        id: userContent.content_id,
        url: '',
        title: 'Unknown Article',
        description: '',
        tags: [],
        isRead: userContent.status === 'completed',
        isFavorite: userContent.is_favorite,
        addedAt: new Date(userContent.added_at).getTime(),
        scrollPosition: userContent.scroll_position || undefined,
      };
    }

    return {
      id: content.id,
      url: content.original_url,
      title: content.title,
      description: content.description,
      content: content.cleaned_html,
      imageUrl: content.image_urls?.[0],
      author: content.author,
      publishedDate: content.published_at,
      readingTime: content.word_count ? Math.ceil(content.word_count / 200) : undefined,
      tags: [],
      isRead: userContent.status === 'completed',
      isFavorite: userContent.is_favorite,
      addedAt: new Date(userContent.added_at).getTime(),
      readAt:
        userContent.status === 'completed'
          ? new Date(userContent.updated_at).getTime()
          : undefined,
      scrollPosition: userContent.scroll_position || undefined,
    };
  }

  /** List the current user's saved content (cursor-paginated). */
  static async listUserContents(
    params?: ListContentsParams,
  ): Promise<UserContentsListResponse> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const queryParams = new URLSearchParams();
    if (params?.status) queryParams.append('status', params.status);
    if (params?.is_favorite !== undefined) {
      queryParams.append('is_favorite', params.is_favorite.toString());
    }
    if (params?.limit) queryParams.append('limit', params.limit.toString());
    if (params?.cursor) queryParams.append('cursor', params.cursor);

    const query = queryParams.toString();
    const url = `${getServerUrl()}/api/v1/content/user/${userId}${query ? `?${query}` : ''}`;

    const response = await AuthService.fetchWithAuth(url);
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to list user contents');
    }

    // Cursor-based pagination: data is the array, pagination carries the totals.
    const contents = Array.isArray(result.data) ? result.data : [];
    const pagination = result.pagination || {};
    return {
      contents,
      total_count: pagination.total || 0,
      limit: pagination.limit || params?.limit || PAGE_SIZE_DEFAULT,
      cursor: pagination.cursor || '',
      has_more: pagination.has_more === true,
    };
  }

  /** Search the current user's saved content. Mirrors mobile's searchUserContents. */
  static async searchUserContents(
    params: SearchParams,
  ): Promise<UserContentsListResponse> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const queryParams = new URLSearchParams();
    queryParams.append('q', params.q);
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.cursor) queryParams.append('cursor', params.cursor);

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/search?${queryParams.toString()}`;

    const response = await AuthService.fetchWithAuth(url);
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to search contents');
    }

    const contents = Array.isArray(result.data) ? result.data : [];
    const pagination = result.pagination || {};
    return {
      contents,
      total_count: pagination.total || 0,
      limit: pagination.limit || params.limit || PAGE_SIZE_DEFAULT,
      cursor: pagination.cursor || '',
      has_more: pagination.has_more === true,
    };
  }

  /** List all subscriptions across sources (RSS, social, email). */
  static async listAllSubscriptions(): Promise<UnifiedSubscriptionsResponse> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/subscriptions`;
    const response = await AuthService.fetchWithAuth(url);
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result?.message || result?.error || 'Failed to list subscriptions');
    }

    return result?.data;
  }

  /**
   * Update user content metadata (status, favorite, scroll position).
   * PATCH /api/v1/content/user/{userId}/{contentId}. Mirrors mobile.
   */
  static async updateUserContent(
    contentId: string,
    updates: UpdateUserContentRequest,
  ): Promise<UserContentResponse> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/${contentId}`;
    const response = await AuthService.fetchWithAuth(url, {
      method: 'PATCH',
      body: JSON.stringify(updates),
    });
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to update content');
    }

    return result.data;
  }

  /**
   * Remove content from the user's reading list.
   * DELETE /api/v1/content/user/{userId}/{contentId}. Mirrors mobile.
   */
  static async deleteUserContent(contentId: string): Promise<void> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/${contentId}`;
    const response = await AuthService.fetchWithAuth(url, { method: 'DELETE' });

    if (!response.ok) {
      const error = await response.text();
      throw new Error(`Failed to delete content: ${error}`);
    }
  }

  /**
   * Detect whether a URL is a feed or page. Non-authenticated; 10 s timeout.
   * On timeout or network error returns { url, type: 'unknown', title: null }
   * matching mobile behavior.
   */
  static async detectURL(url: string): Promise<DetectURLResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10000);
    try {
      const response = await fetch(`${getServerUrl()}/api/v1/content/detect`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
        signal: controller.signal,
      });
      clearTimeout(timeoutId);
      const result = await response.json();
      if (!response.ok) {
        throw new Error(result.message || result.error || 'Detection failed');
      }
      return result.data as DetectURLResponse;
    } catch {
      clearTimeout(timeoutId);
      return { url, type: 'unknown', title: null };
    }
  }

  /**
   * Discover RSS/Atom feeds associated with a URL. Non-authenticated; 15 s timeout.
   * On error/timeout returns { feeds: [] } matching mobile behavior.
   */
  static async discoverFeed(url: string): Promise<DiscoverFeedResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 15000);
    try {
      const response = await fetch(`${getServerUrl()}/api/v1/content/discover-feed`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url }),
        signal: controller.signal,
      });
      clearTimeout(timeoutId);
      const result = await response.json();
      if (!response.ok) {
        return { feeds: [] };
      }
      return (result.data as DiscoverFeedResponse) ?? { feeds: [] };
    } catch {
      clearTimeout(timeoutId);
      return { feeds: [] };
    }
  }

  /**
   * Add a URL (page or feed) to the current user's reading list / subscriptions.
   * POST /api/v1/content/user/{userId}. Mirrors mobile.
   */
  static async addURL(request: AddURLRequest): Promise<AddURLResponse> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }
    const url = `${getServerUrl()}/api/v1/content/user/${userId}`;
    const response = await AuthService.fetchWithAuth(url, {
      method: 'POST',
      body: JSON.stringify(request),
    });
    const result = await response.json();
    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to add URL');
    }
    return result.data as AddURLResponse;
  }

  /**
   * Unsubscribe the current user from an RSS feed.
   * DELETE /api/v1/content/user/{userId}/subscriptions/rss/{feedId}.
   * Ported from apps/mobile/src/services/read.ts. Existing articles in the
   * reading list are preserved; only future deliveries are stopped.
   */
  static async unsubscribeFromRSSFeed(feedId: string): Promise<void> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/subscriptions/rss/${feedId}`;
    const response = await AuthService.fetchWithAuth(url, { method: 'DELETE' });

    if (!response.ok) {
      const result = await response.json().catch(() => ({}));
      throw new Error(result.message || result.error || 'Failed to unsubscribe');
    }
  }

  /**
   * Get or create the current user's newsletter ingest email address.
   * POST /api/v1/source/email/user/{userId}/address (idempotent — returns the
   * same address on repeated calls). Ported from mobile.
   */
  static async getOrCreateEmailAddress(): Promise<string> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/source/email/user/${userId}/address`;
    const response = await AuthService.fetchWithAuth(url, { method: 'POST' });
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to get email address');
    }

    return result.data.email_address as string;
  }

  /**
   * Fetch a single saved article by content id, as an Article, including the
   * full article body (cleaned_html).
   *
   * GET /api/v1/content/user/{userId}/{contentId}. Unlike the list endpoint —
   * whose nested content is a summary that omits cleaned_html — this detail
   * route returns the full content together with the per-user metadata (status,
   * favorite, scroll position) the reader needs. Mirrors mobile's getContentById.
   */
  static async getUserContent(contentId: string): Promise<Article> {
    const userId = await AuthService.getUserId();
    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/${contentId}`;
    const response = await AuthService.fetchWithAuth(url);
    const result = await response.json().catch(() => ({}));

    if (!response.ok || !result?.data) {
      throw new Error(result?.message || result?.error || 'Failed to fetch article');
    }

    return this.transformDetailToArticle(result.data as UserContentDetailResponse);
  }
}
