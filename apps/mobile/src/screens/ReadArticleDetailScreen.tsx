import React, { useState } from 'react';
import {
  View,
  StyleSheet,
  useColorScheme,
  TouchableOpacity,
  Alert,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { Ionicons } from '@expo/vector-icons';
import * as WebBrowser from 'expo-web-browser';
import * as Sharing from 'expo-sharing';
import { RootStackParamList } from '../types';
import { StorageService, ReadService } from '../services';
import { Colors, Spacing } from '../constants';
import { ArticleContent } from '../components/common';

type ReadArticleDetailRouteProp = RouteProp<RootStackParamList, 'ArticleDetail'>;

export const ReadArticleDetailScreen: React.FC = () => {
  const route = useRoute<ReadArticleDetailRouteProp>();
  const navigation = useNavigation();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const [article, setArticle] = useState(route.params.article);

  const handleOpenInBrowser = async () => {
    try {
      await WebBrowser.openBrowserAsync(article.url);
      if (!article.isRead) {
        // Update both local storage and backend
        await StorageService.updateArticle(article.id, {
          isRead: true,
          readAt: Date.now(),
        });
        try {
          await ReadService.updateUserContent(article.id, {
            status: 'completed',
          });
        } catch (backendError) {
          console.error('Failed to sync read status to backend:', backendError);
          // Continue anyway - local update was successful
        }
        setArticle({ ...article, isRead: true, readAt: Date.now() });
      }
    } catch (error) {
      console.error('Failed to open browser:', error);
    }
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

  const handleToggleRead = async () => {
    try {
      const newIsRead = !article.isRead;
      // Update both local storage and backend
      await StorageService.updateArticle(article.id, {
        isRead: newIsRead,
        readAt: newIsRead ? Date.now() : undefined,
      });
      try {
        await ReadService.updateUserContent(article.id, {
          status: newIsRead ? 'completed' : 'unread',
        });
      } catch (backendError) {
        console.error('Failed to sync read status to backend:', backendError);
        // Continue anyway - local update was successful
      }
      setArticle({
        ...article,
        isRead: newIsRead,
        readAt: newIsRead ? Date.now() : undefined,
      });
    } catch (error) {
      console.error('Failed to toggle read status:', error);
    }
  };

  const handleShare = async () => {
    try {
      const isAvailable = await Sharing.isAvailableAsync();
      if (isAvailable) {
        await Sharing.shareAsync(article.url);
      }
    } catch (error) {
      console.error('Failed to share:', error);
    }
  };

  const handleDelete = () => {
    Alert.alert(
      'Delete Article',
      'Are you sure you want to delete this article?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Delete',
          style: 'destructive',
          onPress: async () => {
            try {
              // Delete from local storage
              await StorageService.deleteArticle(article.id);
              // Also delete from backend
              try {
                await ReadService.deleteUserContent(article.id);
              } catch (backendError) {
                console.error('Failed to delete article from backend:', backendError);
                // Continue anyway - local deletion was successful
              }
              navigation.goBack();
            } catch (error) {
              console.error('Failed to delete article:', error);
              Alert.alert('Error', 'Failed to delete article');
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
        onOpenInBrowser={handleOpenInBrowser}
        showMetadata={true}
      />

      <View style={[styles.actionBar, { backgroundColor: colors.card, borderTopColor: colors.border }]}>
        <TouchableOpacity style={styles.actionButton} onPress={handleToggleFavorite}>
          <Ionicons
            name={article.isFavorite ? 'heart' : 'heart-outline'}
            size={24}
            color={article.isFavorite ? colors.error : colors.text}
          />
        </TouchableOpacity>
        <TouchableOpacity style={styles.actionButton} onPress={handleToggleRead}>
          <Ionicons
            name={article.isRead ? 'checkmark-circle' : 'checkmark-circle-outline'}
            size={24}
            color={article.isRead ? colors.success : colors.text}
          />
        </TouchableOpacity>
        <TouchableOpacity style={styles.actionButton} onPress={handleShare}>
          <Ionicons name="share-outline" size={24} color={colors.text} />
        </TouchableOpacity>
        <TouchableOpacity style={styles.actionButton} onPress={handleDelete}>
          <Ionicons name="trash-outline" size={24} color={colors.error} />
        </TouchableOpacity>
      </View>
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
  },
  actionBar: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    paddingVertical: Spacing.md,
    borderTopWidth: 1,
  },
  actionButton: {
    padding: Spacing.sm,
  },
});
