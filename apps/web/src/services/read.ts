// Web Read service. Ports the subset of apps/mobile/src/services/read.ts needed
// so far — the count queries the sidebar's You section reads (bookmarks total and
// subscription split). Built on AuthService.fetchWithAuth (proactive refresh +
// 401 retry), mirroring mobile. Later tasks extend this with the full reading-list,
// search, and add-link methods.
import {
  getServerUrl,
  type ListContentsParams,
  type UserContentsListResponse,
  type UnifiedSubscriptionsResponse,
} from '@cairn/shared';
import { AuthService } from './auth';

const PAGE_SIZE_DEFAULT = 20;

export class ReadService {
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
