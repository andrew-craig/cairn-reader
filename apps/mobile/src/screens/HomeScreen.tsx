import React, { useState, useCallback } from 'react';
import {
  View,
  FlatList,
  useColorScheme,
  RefreshControl,
} from 'react-native';
import { useNavigation, useFocusEffect } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { Article, RootStackParamList } from '../types';
import { ArticleCard, EmptyState } from '../components/common';
import { StorageService } from '../services';
import { Colors, GlobalStyles } from '../constants';

type HomeScreenNavigationProp = StackNavigationProp<RootStackParamList>;

export const HomeScreen: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const navigation = useNavigation<HomeScreenNavigationProp>();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const loadArticles = async () => {
    try {
      const loadedArticles = await StorageService.getArticles();
      const unreadArticles = loadedArticles.filter(a => !a.isRead);
      setArticles(unreadArticles);
    } catch (error) {
      console.error('Failed to load articles:', error);
    }
  };

  useFocusEffect(
    useCallback(() => {
      loadArticles();
    }, [])
  );

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadArticles();
    setRefreshing(false);
  };

  const handleToggleFavorite = async (article: Article) => {
    try {
      await StorageService.updateArticle(article.id, {
        isFavorite: !article.isFavorite,
      });
      await loadArticles();
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
    }
  };

  const handleArticlePress = (article: Article) => {
    navigation.navigate('ArticleDetail', { article });
  };

  return (
    <View style={[GlobalStyles.container, { backgroundColor: colors.background }]}>
      <FlatList
        data={articles}
        keyExtractor={item => item.id}
        renderItem={({ item }) => (
          <ArticleCard
            article={item}
            onPress={() => handleArticlePress(item)}
            onToggleFavorite={() => handleToggleFavorite(item)}
          />
        )}
        ListEmptyComponent={
          <EmptyState
            icon="book-outline"
            title="No articles yet"
            message="Add your first article to get started with ReadItLater"
          />
        }
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            onRefresh={handleRefresh}
            tintColor={colors.primary}
          />
        }
        contentContainerStyle={
          articles.length === 0 ? GlobalStyles.container : undefined
        }
      />
    </View>
  );
};
