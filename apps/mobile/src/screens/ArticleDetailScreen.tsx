import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  useColorScheme,
  TouchableOpacity,
  Alert,
  Image,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { Ionicons } from '@expo/vector-icons';
import * as WebBrowser from 'expo-web-browser';
import * as Sharing from 'expo-sharing';
import { RootStackParamList } from '../types';
import { StorageService } from '../services';
import { formatDate, extractDomain } from '../utils';
import { Colors, Spacing, FontSizes, BorderRadius } from '../constants';

type ArticleDetailRouteProp = RouteProp<RootStackParamList, 'ArticleDetail'>;

export const ArticleDetailScreen: React.FC = () => {
  const route = useRoute<ArticleDetailRouteProp>();
  const navigation = useNavigation();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const [article, setArticle] = useState(route.params.article);

  const handleOpenInBrowser = async () => {
    try {
      await WebBrowser.openBrowserAsync(article.url);
      if (!article.isRead) {
        await StorageService.updateArticle(article.id, {
          isRead: true,
          readAt: Date.now(),
        });
        setArticle({ ...article, isRead: true, readAt: Date.now() });
      }
    } catch (error) {
      console.error('Failed to open browser:', error);
    }
  };

  const handleToggleFavorite = async () => {
    try {
      await StorageService.updateArticle(article.id, {
        isFavorite: !article.isFavorite,
      });
      setArticle({ ...article, isFavorite: !article.isFavorite });
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
    }
  };

  const handleToggleRead = async () => {
    try {
      const newIsRead = !article.isRead;
      await StorageService.updateArticle(article.id, {
        isRead: newIsRead,
        readAt: newIsRead ? Date.now() : undefined,
      });
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
              await StorageService.deleteArticle(article.id);
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
      <ScrollView style={styles.scrollView}>
        {article.imageUrl && (
          <Image
            source={{ uri: article.imageUrl }}
            style={styles.image}
            resizeMode="cover"
          />
        )}
        <View style={styles.content}>
          <Text style={[styles.title, { color: colors.text }]}>
            {article.title}
          </Text>
          {article.description && (
            <Text style={[styles.description, { color: colors.textSecondary }]}>
              {article.description}
            </Text>
          )}
          <View style={styles.metadata}>
            <Text style={[styles.metadataText, { color: colors.textSecondary }]}>
              {extractDomain(article.url)}
            </Text>
            <Text style={[styles.metadataText, { color: colors.textSecondary }]}>
              Added {formatDate(article.addedAt)}
            </Text>
            {article.isRead && article.readAt && (
              <Text style={[styles.metadataText, { color: colors.textSecondary }]}>
                Read {formatDate(article.readAt)}
              </Text>
            )}
          </View>
          <TouchableOpacity
            style={[styles.openButton, { backgroundColor: colors.primary }]}
            onPress={handleOpenInBrowser}
          >
            <Text style={styles.openButtonText}>Open in Browser</Text>
          </TouchableOpacity>
        </View>
      </ScrollView>

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
  scrollView: {
    flex: 1,
  },
  image: {
    width: '100%',
    height: 250,
  },
  content: {
    padding: Spacing.md,
  },
  title: {
    fontSize: FontSizes.xxl,
    fontWeight: 'bold',
    marginBottom: Spacing.md,
  },
  description: {
    fontSize: FontSizes.md,
    lineHeight: 24,
    marginBottom: Spacing.md,
  },
  metadata: {
    marginBottom: Spacing.lg,
  },
  metadataText: {
    fontSize: FontSizes.sm,
    marginBottom: Spacing.xs,
  },
  openButton: {
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.lg,
    borderRadius: BorderRadius.md,
    alignItems: 'center',
  },
  openButtonText: {
    color: '#FFFFFF',
    fontSize: FontSizes.md,
    fontWeight: '600',
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
