import React, { useState, useRef, useCallback, useEffect } from 'react';
import {
  View,
  StyleSheet,
  useColorScheme,
  Alert,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { RootStackParamList } from '../types';
import { StorageService, ReadService } from '../services';
import { Colors } from '../constants';
import { ArticleContent, BottomActionMenu } from '../components/common';

type ReadArticleDetailRouteProp = RouteProp<RootStackParamList, 'ArticleDetail'>;

export const ReadArticleDetailScreen: React.FC = () => {
  const route = useRoute<ReadArticleDetailRouteProp>();
  const navigation = useNavigation();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const [article, setArticle] = useState(route.params.article);
  const scrollPositionRef = useRef(route.params.article.scrollPosition ?? 0);
  const hasScrolledRef = useRef(false);

  const handleScrollPositionChange = useCallback((position: number) => {
    scrollPositionRef.current = position;
    hasScrolledRef.current = true;
  }, []);

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

  const handleArchive = () => {
    Alert.alert(
      'Archive Article',
      'Are you sure you want to archive this article?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Archive',
          style: 'default',
          onPress: async () => {
            try {
              // Delete from local storage (archiving = deleting)
              await StorageService.deleteArticle(article.id);
              // Also delete from backend
              try {
                await ReadService.deleteUserContent(article.id);
              } catch (backendError) {
                console.error('Failed to archive article in backend:', backendError);
                // Continue anyway - local deletion was successful
              }
              navigation.goBack();
            } catch (error) {
              console.error('Failed to archive article:', error);
              Alert.alert('Error', 'Failed to archive article');
            }
          },
        },
      ]
    );
  };

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ArticleContent
        article={article}
        colors={colors}
        onScrollPositionChange={handleScrollPositionChange}
        initialScrollPosition={article.scrollPosition}
      />

      <BottomActionMenu
        actions={[
          {
            icon: 'return',
            onPress: handleBack,
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
