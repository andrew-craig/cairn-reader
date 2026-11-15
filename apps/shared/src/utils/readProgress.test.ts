import { describe, it, expect } from 'vitest';
import type { Article } from '../types';
import { isPartiallyRead } from './readProgress';

const base: Article = {
  id: 'a1',
  url: 'https://example.com',
  title: 'Title',
  tags: [],
  isRead: false,
  isFavorite: false,
  list: 'reads',
  addedAt: 0,
};

describe('isPartiallyRead', () => {
  it('is false with no saved progress', () => {
    expect(isPartiallyRead(base)).toBe(false);
    expect(isPartiallyRead({ ...base, scrollPosition: 0 })).toBe(false);
  });

  it('is true with progress between 0 and 1 from either field', () => {
    expect(isPartiallyRead({ ...base, scrollPosition: 0.4 })).toBe(true);
    expect(isPartiallyRead({ ...base, scrollFraction: 0.4 })).toBe(true);
  });

  it('prefers the local scrollFraction over the server scrollPosition', () => {
    expect(isPartiallyRead({ ...base, scrollFraction: 0, scrollPosition: 0.4 })).toBe(false);
  });

  it('ignores legacy pixel offsets', () => {
    expect(isPartiallyRead({ ...base, scrollPosition: 350 })).toBe(false);
  });

  it('is false once the article is read', () => {
    expect(isPartiallyRead({ ...base, isRead: true, scrollPosition: 0.5 })).toBe(false);
  });
});
