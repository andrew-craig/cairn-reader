import React, { useState, useCallback } from 'react';
import {
  View,
  FlatList,
  StyleSheet,
  useColorScheme,
  RefreshControl,
} from 'react-native';
import { useNavigation, useFocusEffect } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { Article, RootStackParamList } from '../types';
import { ArticleCard, EmptyState } from '../components/common';
import { StorageService } from '../services';
import { Colors } from '../constants';

type ArchiveScreenNavigationProp = StackNavigationProp<RootStackParamList>;

export const ArchiveScreen: React.FC = () => {
  const [articles, setArticles] = useState<Article[]>([]);
  const [refreshing, setRefreshing] = useState(false);
  const navigation = useNavigation<ArchiveScreenNavigationProp>();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const loadArticles = async () => {
    try {
      const loadedArticles = await StorageService.getArticles();
      const readArticles = loadedArticles.filter(a => a.isRead);
      setArticles(readArticles);
    } catch (error) {
      console.error('Failed to load archive:', error);
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
    <View style={[styles.container, { backgroundColor: colors.background }]}>
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
            icon="archive-outline"
            title="No archived articles"
            message="Articles you've read will appear here"
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
          articles.length === 0 ? styles.emptyContainer : undefined
        }
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  emptyContainer: {
    flex: 1,
  },
});
