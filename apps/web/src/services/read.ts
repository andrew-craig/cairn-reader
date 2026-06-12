// Web Read service. Ports the subset of apps/mobile/src/services/read.ts needed
// so far — the reading list (listUserContents + transformToArticle) and the
// sidebar You-section counts (bookmarks total and subscription split). Built on
// AuthService.fetchWithAuth (proactive refresh + 401 retry), mirroring mobile.
// Later tasks extend this with search and add-link methods.
import {
  getServerUrl,
  type Article,
  type ListContentsParams,
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
   * Ported verbatim from apps/mobile/src/services/read.ts so the two apps map
   * content identically (reading time = ceil(word_count / 200)).
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
      description: content.description || content.excerpt,
      content: content.cleaned_html,
      imageUrl: content.lead_image_url,
      author: content.author || content.site_name,
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
}
