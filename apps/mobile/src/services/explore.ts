import { AuthService } from './auth';
import { Article } from '../types';
import { getServerUrl } from '@cairn/shared';

interface RecommendationsResponse {
  user_id: string;
  recommendations: BackendArticle[];
  count: number;
}

interface BackendArticle {
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

interface VoteRequest {
  vote_type: 'upvote' | 'downvote';
}

interface VotedArticle {
  article: BackendArticle;
  vote_type: 'upvote' | 'downvote';
}

interface UserVotesResponse {
  user_id: string;
  votes: VotedArticle[];
  count: number;
}

interface SearchResponse {
  articles: BackendArticle[];
  count: number;
  pagination: {
    limit: number;
    offset: number;
    has_more: boolean;
  };
}

export interface VotedArticleWithType extends Article {
  voteType: 'upvote' | 'downvote';
}

export class ExploreService {
  static async getRecommendations(offset = 0): Promise<Article[]> {
    try {
      const response = await AuthService.fetchWithAuthAndRetry(
        `${getServerUrl()}/api/v1/explore/recommendation?offset=${offset}`
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

  // Best-effort batched "shown" reporting. Lost on network failure (no retry)
  // — acceptable telemetry loss given the dedup row will be re-attempted on
  // the next batch that includes the same ID after a refresh.
  static async markShown(articleIds: string[]): Promise<void> {
    if (articleIds.length === 0) return;
    try {
      const response = await AuthService.fetchWithAuth(
        `${getServerUrl()}/api/v1/explore/shown`,
        {
          method: 'POST',
          body: JSON.stringify({ article_ids: articleIds }),
        }
      );

      if (!response.ok) {
        const error = await response.text();
        console.error('Failed to mark articles as shown:', error);
      }
    } catch (error) {
      console.error('Error marking articles as shown:', error);
    }
  }

  static async markAsRead(articleId: string): Promise<void> {
    try {
      const response = await AuthService.fetchWithAuth(
        `${getServerUrl()}/api/v1/explore/article/${articleId}/read`,
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
      const response = await AuthService.fetchWithAuth(
        `${getServerUrl()}/api/v1/explore/article/${articleId}/vote`,
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
      const response = await AuthService.fetchWithAuth(
        `${getServerUrl()}/api/v1/explore/article/${articleId}/vote`,
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
      const response = await AuthService.fetchWithAuth(
        `${getServerUrl()}/api/v1/explore/article/${articleId}/vote`
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

  static async getUserVotedArticles(
    limit: number = 20,
    offset: number = 0
  ): Promise<VotedArticleWithType[]> {
    try {
      const response = await AuthService.fetchWithAuth(
        `${getServerUrl()}/api/v1/explore/user/votes?limit=${limit}&offset=${offset}`
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to get voted articles');
      }

      const data: UserVotesResponse = result.data;

      // Transform backend articles to mobile Article format with vote type
      return data.votes.map((votedArticle) => ({
        ...this.transformArticle(votedArticle.article),
        voteType: votedArticle.vote_type,
      }));
    } catch (error) {
      console.error('Error fetching user voted articles:', error);
      throw error;
    }
  }

  static async getUserVoteStats(): Promise<{ upvotes: number; downvotes: number }> {
    try {
      const response = await AuthService.fetchWithAuthAndRetry(
        `${getServerUrl()}/api/v1/explore/user/vote-stats`
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to get vote stats');
      }

      const data = result.data as { upvotes: number; downvotes: number };
      return { upvotes: data.upvotes, downvotes: data.downvotes };
    } catch (error) {
      console.error('Error fetching user vote stats:', error);
      throw error;
    }
  }

  static async searchArticles(q: string, limit = 20, offset = 0): Promise<Article[]> {
    try {
      const params = new URLSearchParams({ q, limit: String(limit), offset: String(offset) });
      const response = await AuthService.fetchWithAuthAndRetry(
        `${getServerUrl()}/api/v1/explore/search?${params.toString()}`
      );

      const result = await response.json();

      if (!response.ok) {
        throw new Error(result.message || result.error || 'Failed to search articles');
      }

      const data: SearchResponse = result.data;

      return data.articles.map((article) => this.transformArticle(article));
    } catch (error) {
      console.error('Error searching articles:', error);
      throw error;
    }
  }

  private static transformArticle(backendArticle: BackendArticle): Article {
    return {
      id: backendArticle.id,
      url: backendArticle.link,
      title: backendArticle.title,
      description: backendArticle.description || undefined,
      content: backendArticle.content || undefined,
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
