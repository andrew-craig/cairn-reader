// Read Service API Types
// Based on OpenAPI spec in services/read/api/openapi.yaml

export interface ContentResponse {
  id: string;
  content_hash: string;
  cleaned_html: string;
  original_url: string;
  canonical_url?: string;
  title: string;
  author?: string;
  published_at?: string;
  description?: string;
  excerpt?: string;
  site_name?: string;
  favicon_url?: string;
  lead_image_url?: string;
  word_count?: number;
  created_at: string;
  updated_at: string;
}

export type ContentStatus = 'unread' | 'reading' | 'completed' | 'archived';

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
  offset: number;
  next_cursor?: string;
}

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
  offset?: number;
}

export interface ListContentsParams {
  status?: ContentStatus;
  is_favorite?: boolean;
  limit?: number;
  offset?: number;
}
