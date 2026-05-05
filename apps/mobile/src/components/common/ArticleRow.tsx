import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  TouchableOpacity,
  Image,
  useColorScheme,
} from 'react-native';
import { Article } from '../../types';
import { Colors, FontFamily } from '../../constants';
import { ThumbsUpIcon, ThumbsDownIcon } from '../icons';

// Vote indicator colors
const VOTE_COLORS = {
  upvote: '#219653', // Green
  downvote: '#E67E22', // Orange
};

interface ArticleRowProps {
  article: Article;
  onPress: () => void;
  voteType?: 'upvote' | 'downvote';
}

export const ArticleRow: React.FC<ArticleRowProps> = ({ article, onPress, voteType }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const renderRightContent = () => {
    if (voteType) {
      const iconColor = VOTE_COLORS[voteType];
      return (
        <View style={styles.voteIndicator}>
          {voteType === 'upvote' ? (
            <ThumbsUpIcon size={24} color={iconColor} />
          ) : (
            <ThumbsDownIcon size={24} color={iconColor} />
          )}
        </View>
      );
    }

    if (article.imageUrl) {
      return (
        <View style={styles.imageFrame}>
          <Image
            source={{ uri: article.imageUrl }}
            style={styles.image}
            resizeMode="cover"
          />
        </View>
      );
    }

    return null;
  };

  return (
    <TouchableOpacity
      style={[styles.container, { borderBottomColor: colors.border }]}
      onPress={onPress}
      activeOpacity={0.7}
    >
      <View style={styles.textCluster}>
        <Text
          style={[styles.title, { color: colors.text }]}
          numberOfLines={2}
        >
          {article.title}
        </Text>
        <Text
          style={[styles.author, { color: colors.textSecondary }]}
          numberOfLines={1}
        >
          {article.author || 'Unknown'}
        </Text>
      </View>
      {renderRightContent()}
    </TouchableOpacity>
  );
};

const styles = StyleSheet.create({
  container: {
    flexDirection: 'row',
    gap: 24,
    paddingHorizontal: 16,
    paddingVertical: 24,
    borderBottomWidth: 1,
    alignItems: 'center',
  },
  textCluster: {
    flex: 1,
    flexDirection: 'column',
    gap: 4,
    justifyContent: 'center',
  },
  title: {
    fontSize: 16,
    fontFamily: FontFamily.defaultBold,
    lineHeight: 20,
  },
  author: {
    fontSize: 14,
    fontFamily: FontFamily.default,
  },
  imageFrame: {
    width: 48,
    height: 48,
    borderRadius: 6,
    overflow: 'hidden',
    flexShrink: 0,
  },
  image: {
    width: '100%',
    height: '100%',
  },
  voteIndicator: {
    justifyContent: 'center',
    alignItems: 'center',
    paddingLeft: 8,
  },
});
