import React from 'react';
import { ActivityIndicator, FlatList } from 'react-native';
import { render, screen, fireEvent } from '@testing-library/react-native';
import { ArticleListScreen } from './ArticleListScreen';
import { Article } from '../types';

const article: Article = {
  id: 'a1',
  url: 'https://example.com/a',
  title: 'Already on screen',
  tags: [],
  isRead: false,
  isFavorite: false,
  list: 'reads' as const,
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

describe('ArticleListScreen error state', () => {
  it('shows the error message and a Retry control when a load fails with nothing to show', () => {
    const onRetry = jest.fn();
    render(
      <ArticleListScreen
        title="Read"
        articles={[]}
        loading={false}
        error="Couldn't load your reading list."
        onRetry={onRetry}
      />,
    );

    expect(screen.getByText("Couldn't load your reading list.")).toBeTruthy();
    fireEvent.press(screen.getByLabelText('Retry loading'));
    expect(onRetry).toHaveBeenCalledTimes(1);

    // It must NOT look like an empty account.
    expect(screen.queryByText('No saved articles yet')).toBeNull();
  });

  it('keeps showing the list (not the error state) when articles are present', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[article]}
        loading={false}
        error="stale/network error"
        onRetry={() => {}}
      />,
    );

    expect(screen.getByText('Already on screen')).toBeTruthy();
    expect(screen.queryByLabelText('Retry loading')).toBeNull();
  });

  it('shows the first-load spinner (not the error) while a retry is in flight', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[]}
        loading
        error="previous failure"
        onRetry={() => {}}
      />,
    );
    expect(screen.queryByText('previous failure')).toBeNull();
  });

  it('does not show the stale banner alongside the error state when there is nothing cached', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[]}
        loading={false}
        error="Couldn't load your reading list."
        onRetry={() => {}}
        staleMessage="Showing cached data — pull to refresh"
      />,
    );

    expect(screen.getByText("Couldn't load your reading list.")).toBeTruthy();
    expect(screen.queryByText('Showing cached data — pull to refresh')).toBeNull();
  });

  it('still shows the stale banner when there are cached articles to show', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[article]}
        loading={false}
        staleMessage="Showing cached data — pull to refresh"
      />,
    );

    expect(screen.getByText('Showing cached data — pull to refresh')).toBeTruthy();
  });

  it('suppresses the stale banner while the search field is open', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[article]}
        loading={false}
        staleMessage="Showing cached data — pull to refresh"
        search={{
          open: true,
          query: 'whatever',
          onChangeQuery: () => {},
          onOpen: () => {},
          onClose: () => {},
          placeholder: 'Search your reads',
        }}
      />,
    );

    expect(screen.queryByText('Showing cached data — pull to refresh')).toBeNull();
    expect(screen.getByDisplayValue('whatever')).toBeTruthy();
  });

  it('shows empty search results (not the error state) while the search field is open', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[]}
        loading={false}
        error="Couldn't load your reading list."
        onRetry={() => {}}
        search={{
          open: true,
          query: 'whatever',
          onChangeQuery: () => {},
          onOpen: () => {},
          onClose: () => {},
          placeholder: 'Search your reads',
        }}
        emptyMessage="No matching articles"
      />,
    );

    expect(screen.queryByText("Couldn't load your reading list.")).toBeNull();
    expect(screen.queryByLabelText('Retry loading')).toBeNull();
    expect(screen.getByText('No matching articles')).toBeTruthy();
  });

  it('keeps the header (and search field) mounted while loading with the search open', () => {
    render(
      <ArticleListScreen
        title="Read"
        articles={[]}
        loading
        search={{
          open: true,
          query: 'whatever',
          onChangeQuery: () => {},
          onOpen: () => {},
          onClose: () => {},
          placeholder: 'Search your reads',
        }}
      />,
    );

    expect(screen.getByDisplayValue('whatever')).toBeTruthy();
  });
});

describe('ArticleListScreen keyboard taps', () => {
  it('lets taps on header controls through while the search keyboard is up', () => {
    // The header (with the search field) is inside the FlatList; the default
    // ('never') makes the first tap on close search only dismiss the keyboard.
    render(<ArticleListScreen title="Read" articles={[article]} loading={false} />);

    expect(screen.UNSAFE_getByType(FlatList).props.keyboardShouldPersistTaps).toBe('handled');
  });
});
