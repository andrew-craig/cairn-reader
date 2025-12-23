import React, { useState, useEffect } from 'react';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { Article } from '../types';

export const ExploreScreen: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    loadExploreArticles();
  }, []);

  const loadExploreArticles = async () => {
    try {
      // TODO: Replace with actual API call to explore service
      // For now, using mock data
      const mockArticles: Article[] = [
        {
          id: '1',
          title: "Agenda: Word's largest wooden structure to be burned as chips...",
          author: 'Dezeen',
          url: 'https://example.com/article1',
          imageUrl: 'https://via.placeholder.com/64',
          tags: [],
          isRead: false,
          isFavorite: false,
          addedAt: Date.now(),
        },
        {
          id: '2',
          title: 'TBM 396: So You Want To Define "The Problem"?',
          author: 'John Cutler from The Beautiful Mess',
          url: 'https://example.com/article2',
          imageUrl: 'https://via.placeholder.com/64',
          tags: [],
          isRead: false,
          isFavorite: false,
          addedAt: Date.now(),
        },
        {
          id: '3',
          title: 'TBM 396: So You Want To Define "The Problem"?',
          author: 'John Cutler from The Beautiful Mess',
          url: 'https://example.com/article3',
          imageUrl: 'https://via.placeholder.com/64',
          tags: [],
          isRead: false,
          isFavorite: false,
          addedAt: Date.now(),
        },
        {
          id: '4',
          title: 'TBM 396: So You Want To Define "The Problem"?',
          author: 'John Cutler from The Beautiful Mess',
          url: 'https://example.com/article4',
          imageUrl: 'https://via.placeholder.com/64',
          tags: [],
          isRead: false,
          isFavorite: false,
          addedAt: Date.now(),
        },
        {
          id: '5',
          title: 'TBM 396: So You Want To Define "The Problem"?',
          author: 'John Cutler from The Beautiful Mess',
          url: 'https://example.com/article5',
          imageUrl: 'https://via.placeholder.com/64',
          tags: [],
          isRead: false,
          isFavorite: false,
          addedAt: Date.now(),
        },
      ];
      setArticles(mockArticles);
    } catch (error) {
      console.error('Error loading explore articles:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = () => {
    setRefreshing(true);
    loadExploreArticles();
  };

  const handleArticlePress = (article: Article) => {
    // TODO: Navigate to article detail screen
    console.log('Article pressed:', article.id);
  };

  const handleSearchPress = () => {
    // TODO: Navigate to search screen
    console.log('Search pressed');
  };

  const headerActions = (
    <IconButton icon="search-outline" onPress={handleSearchPress} />
  );

  return (
    <ArticleListScreen
      title="Explore"
      articles={articles}
      loading={loading}
      headerActions={headerActions}
      onArticlePress={handleArticlePress}
      onRefresh={handleRefresh}
      refreshing={refreshing}
      emptyMessage="No articles available"
    />
  );
};
