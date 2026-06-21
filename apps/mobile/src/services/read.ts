import { AuthService } from './auth';
import { Article } from '../types';
import {
  UserContentsListResponse,
  UserContentResponse,
  UserContentDetailResponse,
  ContentDetailResponse,
  AddContentToUserRequest,
  UpdateUserContentRequest,
  SearchParams,
  ListContentsParams,
  DetectURLResponse,
  DiscoverFeedResponse,
  AddURLRequest,
  AddURLResponse,
  ListFeedSubscriptionsResponse,
  UnifiedSubscriptionsResponse,
} from '../types/read';
import { getServerUrl } from '../config/api';
import { withRetry } from '../utils/retry';

const PAGE_SIZE_DEFAULT = 20;

export class ReadService {
  private static async fetchWithAuth(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
    // Proactively check and refresh token if expired before making request
    const isValid = await AuthService.ensureValidToken();
    if (!isValid) {
      throw new Error('Session expired. Please log in again.');
    }

    const accessToken = await AuthService.getAccessToken();

    if (!accessToken) {
      throw new Error('Not authenticated');
    }

    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${accessToken}`,
        ...options.headers,
      },
    });

    // Handle 401 Unauthorized - try to refresh token (fallback for edge cases)
    if (response.status === 401) {
      try {
        await AuthService.refreshAccessToken();
        const newAccessToken = await AuthService.getAccessToken();

        // Retry the request with new token
        const retryResponse = await fetch(url, {
          ...options,
          headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${newAccessToken}`,
            ...options.headers,
          },
        });

        return retryResponse;
      } catch (refreshError) {
        // If refresh fails, clear tokens and throw
        await AuthService.clearTokens();
        throw new Error('Session expired. Please log in again.');
      }
    }

    return response;
  }

  /**
   * Like fetchWithAuth but also retries on 5xx / network errors.
   * 4xx responses are returned as-is (callers handle error status).
   */
  private static async fetchWithAuthAndRetry(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
    return withRetry(async (signal) => {
      const response = await this.fetchWithAuth(url, { ...options, signal });
      // Throw on 5xx so withRetry can retry; let 4xx pass through to caller
      if (response.status >= 500) {
        throw new Error(`Server error ${response.status}`);
      }
      return response;
    });
  }

  /**
   * List user's saved content
   */
  static async listUserContents(
    params?: ListContentsParams
  ): Promise<UserContentsListResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      // Build query parameters
      const queryParams = new URLSearchParams();
      if (params?.status) queryParams.append('status', params.status);
      if (params?.is_favorite !== undefined) {
        queryParams.append('is_favorite', params.is_favorite.toString());
      }
      if (params?.limit) queryParams.append('limit', params.limit.toString());
      if (params?.cursor) queryParams.append('cursor', params.cursor);

      const url = `${getServerUrl()}/api/v1/content/user/${userId}${
        queryParams.toString() ? `?${queryParams.toString()}` : ''
      }`;

      const response = await this.fetchWithAuthAndRetry(url);

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to list user contents');
      }

      // The backend returns a paginated response with cursor-based pagination:
      // - result.data is the array of contents
      // - result.pagination contains cursor (next page), has_more, limit
      const contents = Array.isArray(result.data) ? result.data : [];
      const pagination = result.pagination || {};
      return {
        contents: contents,
        total_count: pagination.total || 0,
        limit: pagination.limit || params?.limit || PAGE_SIZE_DEFAULT,
        cursor: pagination.cursor || '',
        has_more: pagination.has_more === true,
      };
    } catch (error) {
      console.error('Error listing user contents:', error);
      throw error;
    }
  }

  /**
   * Search user's content
   */
  static async searchUserContents(
    params: SearchParams
  ): Promise<UserContentsListResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const queryParams = new URLSearchParams();
      queryParams.append('q', params.q);
      if (params.limit) queryParams.append('limit', params.limit.toString());
      if (params.cursor) queryParams.append('cursor', params.cursor);

      const url = `${getServerUrl()}/api/v1/content/user/${userId}/search?${queryParams.toString()}`;

      const response = await this.fetchWithAuthAndRetry(url);

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to search contents');
      }

      const contents = Array.isArray(result.data) ? result.data : [];
      const pagination = result.pagination || {};
      return {
        contents: contents,
        total_count: pagination.total || 0,
        limit: pagination.limit || params.limit || PAGE_SIZE_DEFAULT,
        cursor: pagination.cursor || '',
        has_more: pagination.has_more === true,
      };
    } catch (error) {
      console.error('Error searching contents:', error);
      throw error;
    }
  }

  /**
   * Add content to user's reading list
   */
  static async addContentToUser(
    request: AddContentToUserRequest
  ): Promise<UserContentResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const response = await this.fetchWithAuth(
        `${getServerUrl()}/api/v1/content/user/${userId}`,
        {
          method: 'POST',
          body: JSON.stringify(request),
        }
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to add content');
      }

      return result.data;
    } catch (error) {
      console.error('Error adding content:', error);
      throw error;
    }
  }

  /**
   * Update user content metadata (status, favorite, scroll position)
   */
  static async updateUserContent(
    contentId: string,
    updates: UpdateUserContentRequest
  ): Promise<UserContentResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const response = await this.fetchWithAuth(
        `${getServerUrl()}/api/v1/content/user/${userId}/${contentId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(updates),
        }
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to update content');
      }

      return result.data;
    } catch (error) {
      console.error('Error updating content:', error);
      throw error;
    }
  }

  /**
   * Delete content from user's reading list
   */
  static async deleteUserContent(contentId: string): Promise<void> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const response = await this.fetchWithAuth(
        `${getServerUrl()}/api/v1/content/user/${userId}/${contentId}`,
        {
          method: 'DELETE',
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to delete content: ${error}`);
      }
    } catch (error) {
      console.error('Error deleting content:', error);
      throw error;
    }
  }

  /**
   * Detect URL type (feed or page)
   * Non-blocking with 10s timeout
   */
  static async detectURL(url: string): Promise<DetectURLResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 10000); // 10s timeout

    try {
      const response = await fetch(
        `${getServerUrl()}/api/v1/content/detect`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url }),
          signal: controller.signal,
        }
      );

      clearTimeout(timeoutId);

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Detection failed');
      }

      return result.data;
    } catch (error) {
      clearTimeout(timeoutId);

      // On timeout or error, return unknown
      return {
        url,
        type: 'unknown',
        title: null,
      };
    }
  }

  /**
   * Discover RSS/Atom feeds associated with a page URL.
   * Returns an array of discovered feeds (may be empty).
   *
   * No authentication required (matches `/detect` pattern).
   * Uses a 15s timeout matching the backend discovery timeout.
   * On error, returns { feeds: [] } rather than throwing.
   */
  static async discoverFeed(url: string): Promise<DiscoverFeedResponse> {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 15000); // 15s timeout

    try {
      const response = await fetch(
        `${getServerUrl()}/api/v1/content/discover-feed`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ url }),
          signal: controller.signal,
        }
      );

      clearTimeout(timeoutId);

      const result = await response.json();

      if (!response.ok) {
        return { feeds: [] };
      }

      return result.data ?? { feeds: [] };
    } catch (error) {
      clearTimeout(timeoutId);
      // On timeout or network error, return empty feeds
      return { feeds: [] };
    }
  }

  /**
   * Add URL to user's content (handles both feeds and pages)
   */
  static async addURL(request: AddURLRequest): Promise<AddURLResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const response = await this.fetchWithAuth(
        `${getServerUrl()}/api/v1/content/user/${userId}`,
        {
          method: 'POST',
          body: JSON.stringify(request),
        }
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to add URL');
      }

      return result.data;
    } catch (error) {
      console.error('Error adding URL:', error);
      throw error;
    }
  }

  /**
   * List all user's subscriptions (unified across all sources: RSS, social, email, etc.)
   * This is the new aggregated endpoint that returns subscriptions from all sources.
   */
  static async listAllSubscriptions(): Promise<UnifiedSubscriptionsResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      // Use the new aggregated endpoint in Content Service
      const url = `${getServerUrl()}/api/v1/content/user/${userId}/subscriptions`;
      console.log('Fetching subscriptions from:', url);

      const response = await this.fetchWithAuthAndRetry(url);

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to list subscriptions');
      }

      return result.data;
    } catch (error) {
      console.error('Error listing subscriptions:', error);
      throw error;
    }
  }

  /**
   * Unsubscribe the current user from an RSS feed.
   * Existing articles already in the user's reading list are preserved;
   * only future deliveries from this feed are stopped.
   */
  static async unsubscribeFromRSSFeed(feedId: string): Promise<void> {
    const userId = await AuthService.getUserId();

    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/subscriptions/rss/${feedId}`;

    const response = await this.fetchWithAuth(url, { method: 'DELETE' });

    if (!response.ok) {
      const text = await response.text();
      throw new Error(`Failed to unsubscribe (HTTP ${response.status}): ${text.substring(0, 200)}`);
    }
  }

  /**
   * List user's feed subscriptions (RSS only)
   * @deprecated Use listAllSubscriptions() for unified subscriptions across all sources
   */
  static async listFeedSubscriptions(): Promise<ListFeedSubscriptionsResponse> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const url = `${getServerUrl()}/api/v1/source/rss/user/${userId}/subscription`;
      console.log('Fetching feed subscriptions from:', url);

      const response = await this.fetchWithAuth(url);

      console.log('Feed subscriptions response status:', response.status);
      const responseText = await response.text();
      console.log('Feed subscriptions response (first 300 chars):', responseText.substring(0, 300));

      if (!response.ok) {
        throw new Error(`Failed to list feed subscriptions (HTTP ${response.status}): ${responseText.substring(0, 100)}`);
      }

      let result;
      try {
        result = JSON.parse(responseText);
      } catch (parseError) {
        throw new Error(`Invalid JSON response from ${url}. Response: ${responseText.substring(0, 200)}`);
      }

      return result.data;
    } catch (error) {
      console.error('Error listing feed subscriptions:', error);
      throw error;
    }
  }

  /**
   * Get or create the current user's newsletter email address.
   * The domain is determined by the backend's EMAIL_DOMAIN configuration.
   */
  static async getOrCreateEmailAddress(): Promise<string> {
    const userId = await AuthService.getUserId();

    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/source/email/user/${userId}/address`;

    const response = await this.fetchWithAuth(url, { method: 'POST' });

    if (!response.ok) {
      let message = 'Failed to get email address';
      try {
        const errJson = await response.json();
        message = errJson.message || errJson.error || message;
      } catch {}
      throw new Error(message);
    }

    const result = await response.json();
    return result.data.email_address;
  }

  /**
   * Fetch a single user content item with full detail (including cleaned_html).
   * Use this when opening an article for reading if the list response did not
   * include cleaned_html.
   */
  static async getContentById(contentId: string): Promise<UserContentDetailResponse> {
    const userId = await AuthService.getUserId();

    if (!userId) {
      throw new Error('Not authenticated');
    }

    const url = `${getServerUrl()}/api/v1/content/user/${userId}/${contentId}`;
    const response = await this.fetchWithAuth(url);

    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to fetch content detail');
    }

    return result.data;
  }

  /**
   * Transform backend UserContentResponse (summary) to mobile Article format.
   * The returned Article will have content=undefined when the response is a summary.
   * Call getContentById and use transformDetailToArticle to fill in the HTML body.
   */
  static transformToArticle(userContent: UserContentResponse): Article {
    const content = userContent.content;

    if (!content) {
      // Fallback if content is not included
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
      // cleaned_html is not present in summary responses — will be loaded on demand
      content: undefined,
      imageUrl: content.image_urls?.[0],
      author: content.author,
      publishedDate: content.published_at,
      readingTime: content.word_count ? Math.ceil(content.word_count / 200) : undefined,
      tags: [],
      isRead: userContent.status === 'completed',
      isFavorite: userContent.is_favorite,
      addedAt: new Date(userContent.added_at).getTime(),
      readAt: userContent.status === 'completed' ? new Date(userContent.updated_at).getTime() : undefined,
      scrollPosition: userContent.scroll_position || undefined,
    };
  }

  /**
   * Transform a full UserContentDetailResponse (with cleaned_html) to Article.
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
      readAt: userContent.status === 'completed' ? new Date(userContent.updated_at).getTime() : undefined,
      scrollPosition: userContent.scroll_position || undefined,
    };
  }
}
