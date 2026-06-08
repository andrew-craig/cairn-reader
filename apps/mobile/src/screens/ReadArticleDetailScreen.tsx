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
  const { article: initialArticle, articles = [], currentIndex = -1 } = route.params;
  const [article, setArticle] = useState(initialArticle);
  const scrollPositionRef = useRef(initialArticle.scrollPosition ?? 0);
  const hasScrolledRef = useRef(false);
  const hasMarkedCompletedRef = useRef(initialArticle.isRead);

  const hasNextArticle = currentIndex >= 0 && currentIndex < articles.length - 1;

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
    setArticle((prev) => ({ ...prev, isRead: true, readAt }));
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
    if (initialArticle.isRead) return;
    ReadService.updateUserContent(initialArticle.id, { status: 'reading' }).catch(
      (err) => console.error('Failed to mark article reading:', err)
    );
  }, [initialArticle.id, initialArticle.isRead]);

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
    try {
      const newIsFavorite = !article.isFavorite;
      // Update both local storage and backend
      await StorageService.updateArticle(article.id, {
        isFavorite: newIsFavorite,
      });
      try {
        await ReadService.updateUserContent(article.id, {
          is_favorite: newIsFavorite,
        });
      } catch (backendError) {
        console.error('Failed to sync favorite status to backend:', backendError);
        // Continue anyway - local update was successful
      }
      setArticle({ ...article, isFavorite: newIsFavorite });
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
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
            active: article.isFavorite,
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
