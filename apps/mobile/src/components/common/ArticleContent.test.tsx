import React from 'react';
import { render, screen } from '@testing-library/react-native';
import { ArticleContent, isScrollTargetReachable } from './ArticleContent';
import { Colors } from '../../constants';
import { Article } from '../../types';

describe('isScrollTargetReachable', () => {
  it('is false while the content is still shorter than the saved fraction target', () => {
    // fraction 0.8 on contentHeight 800 → target 640, max scroll 800-600=200 < 640
    expect(isScrollTargetReachable(800, 600, 0.8)).toBe(false);
  });

  it('is true once the content has grown tall enough to reach the fraction', () => {
    // fraction 0.5 on contentHeight 2600 → target 1300, max scroll 2600-600=2000 >= 1300
    expect(isScrollTargetReachable(2600, 600, 0.5)).toBe(true);
  });

  it('is false before the viewport has been measured', () => {
    expect(isScrollTargetReachable(5000, 0, 0.5)).toBe(false);
  });

  it('treats a fraction at the exact max scroll boundary as reachable', () => {
    // fraction 0.5 on contentHeight 1200 → target 600, max scroll 1200-600=600 >= 600
    expect(isScrollTargetReachable(1200, 600, 0.5)).toBe(true);
    // fraction 0.51 → target 612, max scroll 600 < 612
    expect(isScrollTargetReachable(1200, 600, 0.51)).toBe(false);
  });

  it('handles zero fraction', () => {
    expect(isScrollTargetReachable(2000, 600, 0)).toBe(true);
  });
});

describe('ArticleContent byline date', () => {
  const DAY_MS = 24 * 60 * 60 * 1000;
  const baseArticle: Article = {
    id: 'a1',
    url: 'https://example.com/article',
    title: 'September newsletter',
    content: '<p>Body</p>',
    tags: [],
    isRead: false,
    isFavorite: false,
    list: 'reads',
    addedAt: Date.now(),
  };

  it('shows the published date rather than the date the article was added', () => {
    const publishedDate = new Date(Date.now() - 10 * DAY_MS).toISOString();
    render(<ArticleContent article={{ ...baseArticle, publishedDate }} colors={Colors.light} />);
    expect(screen.getByText(/1 week ago/)).toBeTruthy();
    expect(screen.queryByText(/Today/)).toBeNull();
  });

  it('omits the date when the article has no published date', () => {
    render(<ArticleContent article={baseArticle} colors={Colors.light} />);
    expect(screen.queryByText(/Today/)).toBeNull();
  });
});
