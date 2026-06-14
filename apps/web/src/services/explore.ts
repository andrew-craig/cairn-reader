// Web Explore service. Extends the initial stub (getUserVoteStats) to cover the
// full feature set needed by the Explore tab: recommendations, shown-reporting,
// voting, and reading. Mirrors apps/mobile/src/services/explore.ts, substituting
// AuthService.fetchWithAuth for the mobile fetch wrapper and @cairn/shared's
// getServerUrl for the mobile config.
import { type Article } from '@cairn/shared';
import { getServerUrl } from '../config/api';
import { AuthService } from './auth';

// Internal backend shapes — not in @cairn/shared (explore-only concepts).
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

interface RecommendationsResponse {
  user_id: string;
  recommendations: BackendArticle[];
  count: number;
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

/** Article from the explore feed with the user's vote attached (Votes screen). */
export interface VotedArticleWithType extends Article {
  voteType: 'upvote' | 'downvote';
}

export class ExploreService {
  /** Fetch one page of personalised recommendations (offset pagination). */
  static async getRecommendations(offset = 0): Promise<Article[]> {
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/recommendation?offset=${offset}`,
    );
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to get recommendations');
    }

    const data: RecommendationsResponse = result?.data ?? { recommendations: [] };
    return (data.recommendations ?? []).map((a) => this.transformArticle(a));
  }

  /**
   * Best-effort batched "shown" reporting. Fires and forgets — telemetry loss
   * on network failure is acceptable (mirrors mobile's markShown).
   */
  static async markShown(articleIds: string[]): Promise<void> {
    if (articleIds.length === 0) return;
    try {
      await AuthService.fetchWithAuth(`${getServerUrl()}/api/v1/explore/shown`, {
        method: 'POST',
        body: JSON.stringify({ article_ids: articleIds }),
      });
    } catch (err) {
      console.error('Failed to mark articles as shown:', err);
    }
  }

  /** Mark an explore article as read. POST /api/v1/explore/article/{id}/read. */
  static async markAsRead(articleId: string): Promise<void> {
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/article/${articleId}/read`,
      { method: 'POST' },
    );
    if (!response.ok) {
      const result = await response.json().catch(() => ({}));
      throw new Error(result.message || result.error || 'Failed to mark article as read');
    }
  }

  /** Cast an up- or down-vote on an explore article. */
  static async voteOnArticle(
    articleId: string,
    voteType: 'upvote' | 'downvote',
  ): Promise<void> {
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/article/${articleId}/vote`,
      { method: 'POST', body: JSON.stringify({ vote_type: voteType }) },
    );
    if (!response.ok) {
      const result = await response.json().catch(() => ({}));
      throw new Error(result.message || result.error || 'Failed to vote on article');
    }
  }

  /** Remove the current user's vote from an explore article. */
  static async removeVote(articleId: string): Promise<void> {
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/article/${articleId}/vote`,
      { method: 'DELETE' },
    );
    if (!response.ok) {
      const result = await response.json().catch(() => ({}));
      throw new Error(result.message || result.error || 'Failed to remove vote');
    }
  }

  /** Get aggregate vote counts + the current user's vote for an article. */
  static async getVoteCounts(
    articleId: string,
  ): Promise<{ upvotes: number; downvotes: number; user_vote?: string }> {
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/article/${articleId}/vote`,
    );
    const result = await response.json();
    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to get vote counts');
    }
    return result.data;
  }

  /**
   * List articles the user has voted on, with their vote type attached.
   * Consumed by the Votes screen (later task).
   */
  static async getUserVotedArticles(
    limit = 20,
    offset = 0,
  ): Promise<VotedArticleWithType[]> {
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/user/votes?limit=${limit}&offset=${offset}`,
    );
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to get voted articles');
    }

    const data: UserVotesResponse = result?.data ?? { votes: [] };
    return (data.votes ?? []).map((v) => ({
      ...this.transformArticle(v.article),
      voteType: v.vote_type,
    }));
  }

  /** Aggregate the current user's up/down vote counts (sidebar You section). */
  static async getUserVoteStats(): Promise<{ upvotes: number; downvotes: number }> {
    // Fetch all votes with a high limit to get complete counts (mirrors mobile).
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/user/votes?limit=10000&offset=0`,
    );
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to get vote stats');
    }

    const data: UserVotesResponse = result?.data ?? { votes: [] };
    const votes = Array.isArray(data.votes) ? data.votes : [];
    const upvotes = votes.filter((v) => v.vote_type === 'upvote').length;
    const downvotes = votes.filter((v) => v.vote_type === 'downvote').length;
    return { upvotes, downvotes };
  }

  /** Map a backend explore article to the shared Article UI shape. */
  private static transformArticle(a: BackendArticle): Article {
    return {
      id: a.id,
      url: a.link,
      title: a.title,
      description: a.description || undefined,
      content: a.content || undefined,
      author: a.author || a.feed_title || undefined,
      publishedDate: a.published || undefined,
      tags: a.categories ?? [],
      isRead: false,
      isFavorite: false,
      addedAt: new Date(a.created_at).getTime(),
      imageUrl: this.extractImageUrl(a.content || a.description),
    };
  }

  /** Pull the first <img> src out of an HTML or Markdown string. */
  private static extractImageUrl(content: string): string | undefined {
    if (!content) return undefined;
    const imgMatch = content.match(/<img[^>]+src=["']([^"']+)["']/i);
    if (imgMatch) return imgMatch[1];
    const mdMatch = content.match(/!\[.*?\]\(([^)]+)\)/);
    if (mdMatch) return mdMatch[1];
    return undefined;
  }
}
