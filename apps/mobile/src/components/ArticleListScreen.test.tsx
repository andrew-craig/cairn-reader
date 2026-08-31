import React from 'react';
import { ActivityIndicator } from 'react-native';
import { render, screen } from '@testing-library/react-native';
import { ArticleListScreen } from './ArticleListScreen';
import { Article } from '../types';

const article: Article = {
  id: 'a1',
  url: 'https://example.com/a',
  title: 'Already on screen',
  tags: [],
  isRead: false,
  isFavorite: false,
  addedAt: Date.now(),
};

describe('ArticleListScreen loading behavior', () => {
  it('shows the full-screen spinner only on the first load (no articles yet)', () => {
    render(<ArticleListScreen title="Read" articles={[]} loading />);

    expect(screen.UNSAFE_queryAllByType(ActivityIndicator).length).toBeGreaterThan(0);
    expect(screen.queryByText('Already on screen')).toBeNull();
  });

  it('keeps the populated list visible while reloading (pull-to-refresh)', () => {
    // loading=true (loadReadArticles(reset=true) sets it) but articles are
    // already on screen — the list must stay, not be replaced by a spinner.
    render(
      <ArticleListScreen title="Read" articles={[article]} loading refreshing onRefresh={() => {}} />,
    );

    expect(screen.getByText('Already on screen')).toBeTruthy();
  });
});
