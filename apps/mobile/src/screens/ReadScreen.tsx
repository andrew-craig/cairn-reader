import React, { useState, useEffect } from 'react';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { Article } from '../types';

export const ReadScreen: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);

  useEffect(() => {
    loadReadArticles();
  }, []);

  const loadReadArticles = async () => {
    try {
      // TODO: Replace with actual storage/API call
      // For now, using mock data
      const mockArticles: Article[] = [
        {
          id: '1',
          title: "Agenda: Word's largest wooden structure to be burned as chips...",
          author: 'Dezeen',
          url: 'https://example.com/article1',
          imageUrl: 'https://via.placeholder.com/64',
          tags: [],
          isRead: true,
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
          isRead: true,
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
          isRead: true,
          isFavorite: false,
          addedAt: Date.now(),
        },
      ];
      setArticles(mockArticles);
    } catch (error) {
      console.error('Error loading read articles:', error);
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  };

  const handleRefresh = () => {
    setRefreshing(true);
    loadReadArticles();
  };

  const handleArticlePress = (article: Article) => {
    // TODO: Navigate to article detail screen
    console.log('Article pressed:', article.id);
  };

  const handleAddPress = () => {
    // TODO: Navigate to add article screen or show add modal
    console.log('Add pressed');
  };

  const handleSearchPress = () => {
    // TODO: Navigate to search screen
    console.log('Search pressed');
  };

  const headerActions = (
    <>
      <IconButton icon="add-outline" onPress={handleAddPress} />
      <IconButton icon="search-outline" onPress={handleSearchPress} />
    </>
  );

  return (
    <ArticleListScreen
      title="Read"
      articles={articles}
      loading={loading}
      headerActions={headerActions}
      onArticlePress={handleArticlePress}
      onRefresh={handleRefresh}
      refreshing={refreshing}
      emptyMessage="No saved articles yet"
    />
  );
};
