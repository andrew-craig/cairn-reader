// Web Explore service. Ports the subset of apps/mobile/src/services/explore.ts
// needed so far — the user's aggregate vote counts shown in the sidebar's You
// section. Built on AuthService.fetchWithAuth, mirroring mobile. Later tasks
// extend this with recommendations, voting, and shown-reporting.
import { getServerUrl } from '@cairn/shared';
import { AuthService } from './auth';

interface UserVotesResponse {
  votes: Array<{ vote_type: 'upvote' | 'downvote' }>;
}

export class ExploreService {
  /** Aggregate the current user's up/down vote counts. */
  static async getUserVoteStats(): Promise<{ upvotes: number; downvotes: number }> {
    // Fetch all votes with a high limit to get complete counts (mirrors mobile).
    const response = await AuthService.fetchWithAuth(
      `${getServerUrl()}/api/v1/explore/user/votes?limit=10000&offset=0`,
    );
    const result = await response.json();

    if (!response.ok) {
      throw new Error(result.message || result.error || 'Failed to get vote stats');
    }

    const data: UserVotesResponse = result.data;
    const upvotes = data.votes.filter((v) => v.vote_type === 'upvote').length;
    const downvotes = data.votes.filter((v) => v.vote_type === 'downvote').length;
    return { upvotes, downvotes };
  }
}
