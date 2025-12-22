export interface Article {
  id: string;
  url: string;
  title: string;
  description?: string;
  imageUrl?: string;
  author?: string;
  publishedDate?: string;
  readingTime?: number;
  tags: string[];
  isRead: boolean;
  isFavorite: boolean;
  addedAt: number;
  readAt?: number;
  notes?: string;
}

export interface ArticleMetadata {
  title: string;
  description?: string;
  imageUrl?: string;
  author?: string;
  publishedDate?: string;
  readingTime?: number;
}

export type SortOption = 'recent' | 'oldest' | 'title' | 'unread';
export type FilterOption = 'all' | 'unread' | 'read' | 'favorites';
