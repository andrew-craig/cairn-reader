// Read Service API Types
// Based on OpenAPI spec in services/read/api/openapi.yaml

interface ContentResponse {
  id: string;
  content_hash: string;
  cleaned_html: string;
  original_url: string;
  canonical_url?: string;
  title: string;
  author?: string;
  published_at?: string;
  description?: string;
  image_urls?: string[];
  source_type: string;
  word_count?: number;
  created_at: string;
  updated_at: string;
}

type ContentStatus = 'unread' | 'reading' | 'completed' | 'archived';

export interface UserContentResponse {
  id: string;
  user_id: string;
  content_id: string;
  status: ContentStatus;
  scroll_position: number;
  is_favorite: boolean;
  added_at: string;
  updated_at: string;
  content?: ContentResponse;
}

export interface UserContentsListResponse {
  contents: UserContentResponse[];
  total_count: number;
  limit: number;
  cursor: string;
  has_more: boolean;
}

// URL Detection Types
type URLType = 'feed' | 'page' | 'unknown';

export interface DetectURLResponse {
  url: string;
  type: URLType;
  title: string | null;
}

// Feed Discovery Types
interface DiscoveredFeed {
  url: string;
  title: string;
}

export interface DiscoverFeedResponse {
  feeds: DiscoveredFeed[];
}

export interface AddURLRequest {
  url: string;
  type?: URLType;
  title?: string;
}

interface AddFeedResponse {
  type: 'feed';
  feed_id: string;
  subscription: {
    id: string;
    user_id: string;
    feed_id: string;
    feed_url: string;
    title: string;
    subscribed_at: string;
  };
}

interface AddPageResponse {
  type: 'page';
  content: UserContentResponse;
}

export type AddURLResponse = AddFeedResponse | AddPageResponse;

// Legacy: Direct content addition (requires pre-created content)
export interface AddContentToUserRequest {
  url: string;
  html?: string;
  source_type?: 'rss' | 'manual' | 'web';
}

export interface UpdateUserContentRequest {
  status?: ContentStatus;
  scroll_position?: number;
  is_favorite?: boolean;
  notes?: string;
}

export interface SearchParams {
  q: string;
  limit?: number;
  cursor?: string;
}

export interface ListContentsParams {
  status?: ContentStatus;
  is_favorite?: boolean;
  limit?: number;
  cursor?: string;
}

// Feed Subscription Types (Legacy - kept for backward compatibility)
interface FeedSubscriptionResponse {
  subscription_id: string;
  feed_id: string;
  feed_url: string;
  feed_title: string;
  feed_status: string;
  polling_tier: string;
  last_fetched_at?: string;
  subscribed_at: string;
}

export interface ListFeedSubscriptionsResponse {
  subscriptions: FeedSubscriptionResponse[];
  count: number;
}

// Unified Subscription Types
type SubscriptionType = 'rss' | 'social' | 'email';

interface RSSSubscriptionData {
  feed_id: string;
  feed_url: string;
  site_url?: string;
  polling_tier?: string;
  last_fetched_at?: string;
}

interface SocialSubscriptionData {
  platform: string;
  handle: string;
}

interface EmailSubscriptionData {
  email_address: string;
  filter_rules?: string;
}

export interface UnifiedSubscription {
  id: string;
  type: SubscriptionType;
  title: string;
  description?: string;
  subscribed_at: string;

  // Type-specific data (only one will be populated based on type)
  rss_data?: RSSSubscriptionData;
  social_data?: SocialSubscriptionData;
  email_data?: EmailSubscriptionData;
}

export interface UnifiedSubscriptionsResponse {
  subscriptions: UnifiedSubscription[];
  total_count: number;
}
