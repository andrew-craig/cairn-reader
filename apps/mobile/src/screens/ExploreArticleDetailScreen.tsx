import React, { useState } from 'react';
import {
  View,
  StyleSheet,
  useColorScheme,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { RootStackParamList } from '../types';
import { Colors } from '../constants';
import { ExploreService } from '../services';
import { ArticleContent, BottomActionMenu } from '../components/common';

type ExploreArticleDetailRouteProp = RouteProp<RootStackParamList, 'ExploreArticleDetail'>;

export const ExploreArticleDetailScreen: React.FC = () => {
  const route = useRoute<ExploreArticleDetailRouteProp>();
  const navigation = useNavigation();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const article = route.params.article;

  const [isSaved, setIsSaved] = useState(false);
  const [hasUpvoted, setHasUpvoted] = useState(false);
  const [hasDownvoted, setHasDownvoted] = useState(false);

  const handleBack = () => {
    navigation.goBack();
  };

  const handleSave = async () => {
    try {
      // TODO: Implement save to Read list functionality
      setIsSaved(!isSaved);
      console.log('Save article:', article.id);
    } catch (error) {
      console.error('Failed to save article:', error);
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
