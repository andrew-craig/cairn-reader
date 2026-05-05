import React, { useState } from 'react';
import {
  View,
  StyleSheet,
  useColorScheme,
  Alert,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { RootStackParamList } from '../types';
import { Colors } from '../constants';
import { ExploreService, ReadService } from '../services';
import { ArticleContent, BottomActionMenu } from '../components/common';

type ExploreArticleDetailRouteProp = RouteProp<RootStackParamList, 'ExploreArticleDetail'>;
type ExploreArticleDetailNavigationProp = StackNavigationProp<RootStackParamList, 'ExploreArticleDetail'>;

export const ExploreArticleDetailScreen: React.FC = () => {
  const route = useRoute<ExploreArticleDetailRouteProp>();
  const navigation = useNavigation<ExploreArticleDetailNavigationProp>();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const { article, articles = [], currentIndex = -1 } = route.params;

  const hasNextArticle = currentIndex >= 0 && currentIndex < articles.length - 1;

  const [isSaved, setIsSaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [savedContentId, setSavedContentId] = useState<string | null>(null);
  const [hasUpvoted, setHasUpvoted] = useState(false);
  const [hasDownvoted, setHasDownvoted] = useState(false);

  const handleBack = () => {
    navigation.goBack();
  };

  const handleNextArticle = async () => {
    if (!hasNextArticle) return;
    const nextIndex = currentIndex + 1;
    const nextArticle = articles[nextIndex];
    navigation.replace('ExploreArticleDetail', {
      article: nextArticle,
      articles,
      currentIndex: nextIndex,
    });
    try {
      await ExploreService.markAsRead(nextArticle.id);
    } catch (error) {
      console.error('Error marking article as read:', error);
    }
  };

  const handleSave = async () => {
    if (isSaving) return;
    setIsSaving(true);

    try {
      if (isSaved && savedContentId) {
        // Remove from Read list
        await ReadService.deleteUserContent(savedContentId);
        setIsSaved(false);
        setSavedContentId(null);
      } else {
        // Save to Read list
        const response = await ReadService.addContentToUser({
          url: article.url,
          source_type: 'web',
        });
        setIsSaved(true);
        setSavedContentId(response.content_id);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      if (message.includes('409') || message.toLowerCase().includes('already')) {
        // Already saved — just reflect the state
        setIsSaved(true);
      } else {
        Alert.alert('Error', isSaved ? 'Failed to remove article.' : 'Failed to save article.');
      }
      console.error('Failed to save/unsave article:', error);
    } finally {
      setIsSaving(false);
    }
  };

  const handleUpvote = async () => {
    try {
      if (hasUpvoted) {
        // Remove upvote
        await ExploreService.removeVote(article.id);
        setHasUpvoted(false);
      } else {
        // Add upvote (remove downvote if present)
        await ExploreService.upvoteArticle(article.id);
        setHasUpvoted(true);
        setHasDownvoted(false);
      }
    } catch (error) {
      console.error('Failed to upvote article:', error);
    }
  };

  const handleDownvote = async () => {
    try {
      if (hasDownvoted) {
        // Remove downvote
        await ExploreService.removeVote(article.id);
        setHasDownvoted(false);
      } else {
        // Add downvote (remove upvote if present)
        await ExploreService.downvoteArticle(article.id);
        setHasDownvoted(true);
        setHasUpvoted(false);
      }
    } catch (error) {
      console.error('Failed to downvote article:', error);
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
            icon: 'next-article',
            onPress: handleNextArticle,
            disabled: !hasNextArticle,
          },
          {
            icon: 'bookmark',
            onPress: handleSave,
            active: isSaved,
          },
          {
            icon: 'thumbs-up',
            onPress: handleUpvote,
            active: hasUpvoted,
          },
          {
            icon: 'thumbs-down',
            onPress: handleDownvote,
            active: hasDownvoted,
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
