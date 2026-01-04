import { AuthService } from './auth';
import { Article } from '../types';
import { API_CONFIG } from '../config/api';

const RECOMMENDER_BASE_URL = API_CONFIG.RECOMMENDER_SERVICE_URL;

export interface RecommendationsResponse {
  user_id: string;
  recommendations: BackendArticle[];
  count: number;
}

export interface BackendArticle {
  id: string;
  title: string;
  link: string;
  description: string;
  content: string;
  author: string;
  published: string;
  feed_url: string;
  feed_title: string;
  categories: string[];
  upvotes: number;
  downvotes: number;
  recommends: number;
  deleted: boolean;
  created_at: string;
  updated_at: string;
}

export interface VoteRequest {
  vote_type: 'upvote' | 'downvote';
}

export class ExploreService {
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

  static async getRecommendations(): Promise<Article[]> {
    try {
      const userId = await AuthService.getUserId();

      if (!userId) {
        throw new Error('Not authenticated');
      }

      const response = await this.fetchWithAuth(
        `${RECOMMENDER_BASE_URL}/api/v1/explore/recommendation/${userId}`
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to get recommendations');
      }

      const data: RecommendationsResponse = result.data;

      // Transform backend articles to mobile Article format
      return data.recommendations.map((article) => this.transformArticle(article));
    } catch (error) {
      console.error('Error fetching recommendations:', error);
      throw error;
    }
  }

  static async markAsRead(articleId: string): Promise<void> {
    try {
      const response = await this.fetchWithAuth(
        `${RECOMMENDER_BASE_URL}/api/v1/explore/article/${articleId}/read`,
        {
          method: 'POST',
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to mark article as read: ${error}`);
      }
    } catch (error) {
      console.error('Error marking article as read:', error);
      throw error;
    }
  }

  static async voteOnArticle(
    articleId: string,
    voteType: 'upvote' | 'downvote'
  ): Promise<void> {
    try {
      const response = await this.fetchWithAuth(
        `${RECOMMENDER_BASE_URL}/api/v1/explore/article/${articleId}/vote`,
        {
          method: 'POST',
          body: JSON.stringify({ vote_type: voteType } as VoteRequest),
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to vote on article: ${error}`);
      }
    } catch (error) {
      console.error('Error voting on article:', error);
      throw error;
    }
  }

  static async upvoteArticle(articleId: string): Promise<void> {
    return this.voteOnArticle(articleId, 'upvote');
  }

  static async downvoteArticle(articleId: string): Promise<void> {
    return this.voteOnArticle(articleId, 'downvote');
  }

  static async removeVote(articleId: string): Promise<void> {
    try {
      const response = await this.fetchWithAuth(
        `${RECOMMENDER_BASE_URL}/api/v1/explore/article/${articleId}/vote`,
        {
          method: 'DELETE',
        }
      );

      if (!response.ok) {
        const error = await response.text();
        throw new Error(`Failed to remove vote: ${error}`);
      }
    } catch (error) {
      console.error('Error removing vote:', error);
      throw error;
    }
  }

  static async getVoteCounts(
    articleId: string
  ): Promise<{ upvotes: number; downvotes: number; user_vote?: string }> {
    try {
      const response = await this.fetchWithAuth(
        `${RECOMMENDER_BASE_URL}/api/v1/explore/article/${articleId}/vote`
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to get vote counts');
      }

      return result.data;
    } catch (error) {
      console.error('Error getting vote counts:', error);
      throw error;
    }
  }

  private static transformArticle(backendArticle: BackendArticle): Article {
    return {
      id: backendArticle.id,
      url: backendArticle.link,
      title: backendArticle.title,
      description: backendArticle.description || backendArticle.content,
      author: backendArticle.author || backendArticle.feed_title,
      publishedDate: backendArticle.published,
      tags: backendArticle.categories || [],
      isRead: false, // Will be managed locally or fetched from backend
      isFavorite: false, // Will be managed locally
      addedAt: new Date(backendArticle.created_at).getTime(),
      // Extract image from content if available
      imageUrl: this.extractImageUrl(backendArticle.content || backendArticle.description),
    };
  }

  private static extractImageUrl(content: string): string | undefined {
    if (!content) return undefined;

    // Try to find img tags
    const imgMatch = content.match(/<img[^>]+src=["']([^"']+)["']/i);
    if (imgMatch) {
      return imgMatch[1];
    }

    // Try to find image URLs in markdown format
    const markdownMatch = content.match(/!\[.*?\]\(([^)]+)\)/);
    if (markdownMatch) {
      return markdownMatch[1];
    }

    return undefined;
  }
}
