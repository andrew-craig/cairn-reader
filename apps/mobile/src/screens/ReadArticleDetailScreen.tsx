import React, { useState, useRef, useCallback, useEffect } from 'react';
import {
  View,
  StyleSheet,
  useColorScheme,
  Alert,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { RootStackParamList } from '../types';
import { StorageService, ReadService } from '../services';
import { Colors } from '../constants';
import { ArticleContent, BottomActionMenu } from '../components/common';
import type { ScrollProgressInfo } from '../components/common/ArticleContent';

const COMPLETED_PROGRESS_THRESHOLD = 0.95;

type ReadArticleDetailRouteProp = RouteProp<RootStackParamList, 'ArticleDetail'>;
type ReadArticleDetailNavigationProp = StackNavigationProp<RootStackParamList, 'ArticleDetail'>;

export const ReadArticleDetailScreen: React.FC = () => {
  const route = useRoute<ReadArticleDetailRouteProp>();
  const navigation = useNavigation<ReadArticleDetailNavigationProp>();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const { article, articles = [], currentIndex = -1 } = route.params;

  // Mutable UI state tracked separately from the article object so async updates
  // are scoped to the displayed article. Seeded from the article and resynced
  // when the displayed article changes (see effect below). isRead drives the
  // completion guard; isFavorite is reflected in the action menu.
  const [isFavorite, setIsFavorite] = useState(article.isFavorite);
  const scrollPositionRef = useRef(article.scrollPosition ?? 0);
  const hasScrolledRef = useRef(false);
  const hasMarkedCompletedRef = useRef(article.isRead);
  // Tracks the currently displayed article id so async callbacks can detect a
  // swap (next article) and avoid mutating UI state for a different article.
  const articleIdRef = useRef(article.id);

  const hasNextArticle = currentIndex >= 0 && currentIndex < articles.length - 1;

  // Reset per-article progress refs and reseed the mutable UI state whenever the
  // displayed article changes (next article reuses this screen via replace()).
  useEffect(() => {
    articleIdRef.current = article.id;
    scrollPositionRef.current = article.scrollPosition ?? 0;
    hasScrolledRef.current = false;
    hasMarkedCompletedRef.current = article.isRead;
    setIsFavorite(article.isFavorite);
  }, [article]);

  const markCompleted = useCallback((articleId: string) => {
    if (hasMarkedCompletedRef.current) return;
    hasMarkedCompletedRef.current = true;
    const readAt = Date.now();
    ReadService.updateUserContent(articleId, { status: 'completed' }).catch(
      (err) => console.error('Failed to mark article completed:', err)
    );
    StorageService.updateArticle(articleId, { isRead: true, readAt }).catch(
      (err) => console.error('Failed to persist completed locally:', err)
    );
  }, []);

  const handleScrollProgress = useCallback((info: ScrollProgressInfo) => {
    scrollPositionRef.current = info.contentHeight > 0
      ? info.offsetY / info.contentHeight
      : 0;
    hasScrolledRef.current = true;

    if (hasMarkedCompletedRef.current || info.contentHeight <= 0) return;
    const progress = (info.offsetY + info.layoutHeight) / info.contentHeight;
    if (progress >= COMPLETED_PROGRESS_THRESHOLD) {
      markCompleted(article.id);
    }
  }, [article.id, markCompleted]);

  useEffect(() => {
    if (article.isRead) return;
    ReadService.updateUserContent(article.id, { status: 'reading' }).catch(
      (err) => console.error('Failed to mark article reading:', err)
    );
  }, [article.id, article.isRead]);

  useEffect(() => {
    const articleId = article.id;
    return () => {
      if (!hasScrolledRef.current) return;
      const position = scrollPositionRef.current;
      ReadService.updateUserContent(articleId, { scroll_position: position }).catch(
        (err) => console.error('Failed to save scroll position:', err)
      );
      StorageService.updateArticle(articleId, { scrollPosition: position }).catch(
        (err) => console.error('Failed to save scroll position locally:', err)
      );
    };
  }, [article.id]);

  const handleBack = () => {
    navigation.goBack();
  };

  const handleNextArticle = () => {
    if (!hasNextArticle) return;
    const nextIndex = currentIndex + 1;
    navigation.replace('ArticleDetail', {
      article: articles[nextIndex],
      articles,
      currentIndex: nextIndex,
    });
  };

  const handleToggleFavorite = async () => {
    const targetId = article.id;
    const newIsFavorite = !isFavorite;
    // Optimistically reflect the new state in the action menu.
    setIsFavorite(newIsFavorite);
    try {
      // Update both local storage and backend
      await StorageService.updateArticle(targetId, {
        isFavorite: newIsFavorite,
      });
      try {
        await ReadService.updateUserContent(targetId, {
          is_favorite: newIsFavorite,
        });
      } catch (backendError) {
        console.error('Failed to sync favorite status to backend:', backendError);
        // Continue anyway - local update was successful
      }
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
      // Roll back the optimistic update on failure, but only if the same
      // article is still displayed — otherwise we'd flip the wrong article.
      if (articleIdRef.current === targetId) setIsFavorite(!newIsFavorite);
    }
  };

  const handleArchive = async () => {
    try {
      await StorageService.deleteArticle(article.id);
      try {
        await ReadService.deleteUserContent(article.id);
      } catch (backendError) {
        console.error('Failed to archive article in backend:', backendError);
      }
      navigation.goBack();
    } catch (error) {
      console.error('Failed to archive article:', error);
      Alert.alert('Error', 'Failed to archive article');
    }
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ArticleContent
        article={article}
        colors={colors}
        onScrollProgress={handleScrollProgress}
        initialScrollPosition={article.scrollPosition}
      />

      <BottomActionMenu
        actions={[
          {
            icon: 'return',
            onPress: handleBack,
          },
          {
            icon: 'next-article',
            onPress: handleNextArticle,
            disabled: !hasNextArticle,
          },
          {
            icon: 'bookmark',
            onPress: handleToggleFavorite,
            active: isFavorite,
          },
          {
            icon: 'archive',
            onPress: handleArchive,
          },
        ]}
      />
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
});
