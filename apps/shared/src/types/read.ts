// Read Service API Types
// Based on OpenAPI spec in services/read/api/openapi.yaml

// Summary content returned in list/search responses (no cleaned_html)
interface ContentSummaryResponse {
  id: string;
  content_hash: string;
  original_url: string;
  canonical_url?: string;
  title: string;
  author?: string;
  published_at?: string;
  description?: string;
  image_urls?: string[];
  word_count?: number;
  source_type: string;
  created_at: string;
  updated_at: string;
}

// Full content returned in detail/create responses (includes cleaned_html)
export interface ContentDetailResponse {
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
  word_count?: number;
  source_type: string;
  created_at: string;
  updated_at: string;
}

type ContentStatus = 'unread' | 'reading' | 'completed' | 'archived';

// Which list an item lives in: the skimmable Feed, or Reads (read or triage).
export type ContentList = 'feed' | 'reads';

// List/search response — content is a summary (no cleaned_html)
export interface UserContentResponse {
  id: string;
  user_id: string;
  content_id: string;
  status: ContentStatus;
  list: ContentList;
  scroll_position: number;
  is_favorite: boolean;
  added_at: string;
  updated_at: string;
  content?: ContentSummaryResponse;
}

// Detail response — content includes cleaned_html
export interface UserContentDetailResponse {
  id: string;
  user_id: string;
  content_id: string;
  status: ContentStatus;
  list: ContentList;
  scroll_position: number;
  is_favorite: boolean;
  added_at: string;
  updated_at: string;
  content?: ContentDetailResponse;
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
  /** Feeds only: where the feed's new items land (backend default: reads). */
  list?: ContentList;
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
    list: ContentList;
    subscribed_at: string;
  };
}

interface AddPageResponse {
  type: 'page';
  content: UserContentDetailResponse;
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
  /** Move the item to this list (e.g. 'reads' for Save to Reads). */
  list?: ContentList;
}

export interface SearchParams {
  q: string;
  list?: ContentList;
  limit?: number;
  cursor?: string;
}

export interface ListContentsParams {
  status?: ContentStatus;
  list?: ContentList;
  is_favorite?: boolean;
  limit?: number;
  cursor?: string;
}

// Params for the count-only endpoint (GET .../count) — no pagination fields,
// since it returns a single number rather than a page of results.
export type CountContentsParams = Pick<ListContentsParams, 'status' | 'is_favorite' | 'list'>;

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
  /** Where this source's new items are delivered. */
  list: ContentList;

  // Type-specific data (only one will be populated based on type)
  rss_data?: RSSSubscriptionData;
  social_data?: SocialSubscriptionData;
  email_data?: EmailSubscriptionData;
}

export interface UnifiedSubscriptionsResponse {
  subscriptions: UnifiedSubscription[];
  total_count: number;
  /** Sources that failed to load; their subscriptions are missing from the list. */
  failed_sources: Array<'rss' | 'email'>;
}

/** Source types that can be routed to a list (the `{type}` in the list endpoint). */
export type SourceRouteType = 'rss' | 'email';

export interface SetSourceListResponse {
  type: SourceRouteType;
  key: string;
  list: ContentList;
}
