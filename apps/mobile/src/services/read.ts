import { AuthService } from './auth';
import { Article } from '../types';
import {
  UserContentsListResponse,
  UserContentResponse,
  AddContentToUserRequest,
  UpdateUserContentRequest,
  SearchParams,
  ListContentsParams,
} from '../types/read';
import { API_CONFIG } from '../config/api';

const READ_SERVICE_BASE_URL = API_CONFIG.READ_SERVICE_URL;

export class ReadService {
  private static async fetchWithAuth(
    url: string,
    options: RequestInit = {}
  ): Promise<Response> {
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

    // Handle 401 Unauthorized - try to refresh token
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
      if (params?.offset) queryParams.append('offset', params.offset.toString());

      const url = `${READ_SERVICE_BASE_URL}/api/v1/content/user/${userId}${
        queryParams.toString() ? `?${queryParams.toString()}` : ''
      }`;

      const response = await this.fetchWithAuth(url);

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to list user contents: ${error}`);
      }

      return await response.json();
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
      if (params.offset) queryParams.append('offset', params.offset.toString());

      const url = `${READ_SERVICE_BASE_URL}/api/v1/content/user/${userId}/search?${queryParams.toString()}`;

      const response = await this.fetchWithAuth(url);

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to search contents: ${error}`);
      }

      return await response.json();
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
        `${READ_SERVICE_BASE_URL}/api/v1/content/user/${userId}`,
        {
          method: 'POST',
          body: JSON.stringify(request),
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to add content: ${error}`);
      }

      return await response.json();
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
        `${READ_SERVICE_BASE_URL}/api/v1/content/user/${userId}/${contentId}`,
        {
          method: 'PATCH',
          body: JSON.stringify(updates),
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to update content: ${error}`);
      }

      return await response.json();
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
        `${READ_SERVICE_BASE_URL}/api/v1/content/user/${userId}/${contentId}`,
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
   * Transform backend UserContentResponse to mobile Article format
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
      };
    }

    return {
      id: content.id,
      url: content.original_url,
      title: content.title,
      description: content.description || content.excerpt,
      imageUrl: content.lead_image_url,
      author: content.author || content.site_name,
      publishedDate: content.published_at,
      readingTime: content.word_count ? Math.ceil(content.word_count / 200) : undefined,
      tags: [],
      isRead: userContent.status === 'completed',
      isFavorite: userContent.is_favorite,
      addedAt: new Date(userContent.added_at).getTime(),
      readAt: userContent.status === 'completed' ? new Date(userContent.updated_at).getTime() : undefined,
    };
  }
}
