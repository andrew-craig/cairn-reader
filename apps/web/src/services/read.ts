// Web Read service. Ports the subset of apps/mobile/src/services/read.ts needed
// so far — the reading list (listUserContents + transformToArticle) and the
// sidebar You-section counts (bookmarks total and subscription split). Built on
// AuthService.fetchWithAuth (proactive refresh + 401 retry), mirroring mobile.
// Later tasks extend this with search and add-link methods.
import {
  getServerUrl,
  type Article,
  type ListContentsParams,
  type UpdateUserContentRequest,
  type UserContentResponse,
  type UserContentsListResponse,
  type UnifiedSubscriptionsResponse,
} from '@cairn/shared';
import { AuthService } from './auth';

const PAGE_SIZE_DEFAULT = 20;

/** Reading-list page size, mirroring mobile's PAGE_SIZE. */
export const PAGE_SIZE = 20;

export class ReadService {
  /**
   * Transform a backend UserContentResponse into the UI Article shape.
   * Mirrors apps/mobile/src/services/read.ts, including the reading-time
   * estimate from word_count (undefined until the backend emits word_count).
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
   * Fetch a single saved article by content id, as an Article.
   *
   * The Read Service exposes no protected per-user get-by-id route — the user
   * routes are list, search, PATCH and DELETE only (see
   * services/read/content/internal/api/router.go). So this pages through the
   * user's list to find the matching item, which also carries the per-user
   * metadata (status, favorite, scroll position) the reader needs. Used only
   * as a fallback when the reader is opened without navigation state (direct
   * URL / hard refresh).
   */
  static async getUserContent(contentId: string): Promise<Article> {
    let cursor: string | undefined;
    do {
      const page = await this.listUserContents({ limit: PAGE_SIZE, cursor });
      const match = page.contents.find((c) => c.content_id === contentId);
      if (match) {
        return this.transformToArticle(match);
      }
      cursor = page.has_more ? page.cursor : undefined;
    } while (cursor);

    throw new Error('Article not found');
  }
}
