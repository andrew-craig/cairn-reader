export interface Article {
  id: string;
  url: string;
  title: string;
  description?: string;
  content?: string; // Cleaned HTML content from readability extraction
  contentHash?: string; // content.content_hash from the Read service, used to diff cached bodies
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
  scrollPosition?: number;
  scrollFraction?: number;
}
