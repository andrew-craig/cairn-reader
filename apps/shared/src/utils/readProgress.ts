import type { Article } from '../types';

// Values above 1 are legacy pixel offsets, not fractions, so they carry no
// usable progress.
const toFraction = (v?: number): number | undefined =>
  v !== undefined && v <= 1 ? v : undefined;

/**
 * True when the user has started an article but not finished it: some scroll
 * progress is saved (the local `scrollFraction` first, then the server's
 * `scrollPosition`) and the article is not yet marked read.
 */
export function isPartiallyRead(article: Article): boolean {
  if (article.isRead) return false;
  const progress = toFraction(article.scrollFraction) ?? toFraction(article.scrollPosition) ?? 0;
  return progress > 0;
}
