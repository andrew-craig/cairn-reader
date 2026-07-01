export interface Article {
  id: string;
  url: string;
  title: string;
  description?: string;
  content?: string; // Cleaned HTML content from readability extraction
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
