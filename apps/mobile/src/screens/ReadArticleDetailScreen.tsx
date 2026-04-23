import React, { useState } from 'react';
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
