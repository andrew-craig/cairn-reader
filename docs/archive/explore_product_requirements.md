# Cairn Explore - Product Requirements (Archived)

> **Archived.** The Explore tab and its backend were removed in #425 (Oct 2026). This content was moved out of `docs/product_requirements.md` unchanged, for historical reference. It does not describe shipped behaviour. See also [explore_service_requirements.md](explore_service_requirements.md) and [explore_web_app_requirements.md](explore_web_app_requirements.md).

## Product Overview (Explore)

Explore: Discover curated articles from across the web. It was one of two core experiences, alongside Read (now the Feed and Reads lists).

## Product Requirements - Content Discovery (Explore)

#### Article Curation
- Content will ffocus on independent blogs, personal websites, and thoughtful writing
  - Articles will initially be sourced from [Kagi Small Web Text collection](https://github.com/kagisearch/smallweb/blob/main/smallweb.txt)
- Content will be removed after 90 days

#### Article Recommendations
- Users receive personalized article recommendations
- Recommendations balance quality content with exploration. Initial split is:
  - 80% high-quality articles based on community engagement
  - 20% exploratory article to discover new content
- Fresh content added continuously (new articles fetched every minute)
- Content sourced from curated collection of independent blogs and websites

#### Engagement & Feedback
- **Upvote/Downvote**: Users can vote on articles to improve recommendations
- **Vote Management**:
  - Change vote from upvote to downvote (or vice versa)
  - Remove vote entirely
  - One vote per article per user
- Voting influences future recommendations for all users

## User Workflows (Explore)

### New User Onboarding
2. Browse recommended articles (Explore)
3. Vote on articles to personalize recommendations

### Daily Reading Routine
1. Check Explore for new recommendations
2. Read and vote on interesting articles

## Success Metrics (Explore)

- Upvote/downvote ratio

## Design Principles (Explore)

7. **Community-Driven**: Upvotes/downvotes improve recommendations for everyone

## Future Enhancements (Explore)

- Popular articles from community
- Trending content alerts

## Open Questions (Explore)

1. Should users be able to see which articles they've already voted on?
4. Should there be categories or topics for Explore content?
