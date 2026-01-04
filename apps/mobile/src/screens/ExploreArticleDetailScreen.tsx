import React, { useState } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  useColorScheme,
  TouchableOpacity,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { Ionicons } from '@expo/vector-icons';
import { RootStackParamList } from '../types';
import { Colors, Spacing, FontSizes } from '../constants';
import { ExploreService } from '../services';

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
      <ScrollView
        style={styles.scrollView}
        contentContainerStyle={styles.scrollContent}
      >
        <Text style={[styles.title, { color: colors.text }]}>
          {article.title}
        </Text>

        {article.description && (
          <Text style={[styles.content, { color: colors.text }]}>
            {article.description}
          </Text>
        )}
      </ScrollView>

      <View style={[styles.bottomBar, { backgroundColor: colors.card, borderTopColor: colors.border }]}>
        <TouchableOpacity style={styles.bottomButton} onPress={handleBack}>
          <Ionicons name="arrow-back" size={28} color={colors.text} />
        </TouchableOpacity>

        <TouchableOpacity style={styles.bottomButton} onPress={handleSave}>
          <Ionicons
            name={isSaved ? 'bookmark' : 'bookmark-outline'}
            size={28}
            color={isSaved ? colors.primary : colors.text}
          />
        </TouchableOpacity>

        <TouchableOpacity style={styles.bottomButton} onPress={handleUpvote}>
          <Ionicons
            name={hasUpvoted ? 'thumbs-up' : 'thumbs-up-outline'}
            size={28}
            color={hasUpvoted ? colors.success : colors.text}
          />
        </TouchableOpacity>

        <TouchableOpacity style={styles.bottomButton} onPress={handleDownvote}>
          <Ionicons
            name={hasDownvoted ? 'thumbs-down' : 'thumbs-down-outline'}
            size={28}
            color={hasDownvoted ? colors.error : colors.text}
          />
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
  scrollContent: {
    padding: Spacing.lg,
    paddingBottom: Spacing.xl,
  },
  title: {
    fontSize: FontSizes.xxl,
    fontWeight: 'bold',
    marginBottom: Spacing.lg,
    lineHeight: 32,
  },
  content: {
    fontSize: FontSizes.md,
    lineHeight: 24,
  },
  bottomBar: {
    flexDirection: 'row',
    justifyContent: 'space-around',
    alignItems: 'center',
    paddingVertical: Spacing.md,
    paddingHorizontal: Spacing.lg,
    borderTopWidth: 1,
    paddingBottom: Spacing.lg,
  },
  bottomButton: {
    padding: Spacing.sm,
  },
});
