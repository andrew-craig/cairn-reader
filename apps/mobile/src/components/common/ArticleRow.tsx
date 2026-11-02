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
import { formatPublishedDate } from '@cairn/shared';

interface ArticleRowProps {
  article: Article;
  onPress: () => void;
  /** A Save to Reads still queued offline: shows a "waiting to save" note. */
  pending?: boolean;
}

export const ArticleRow: React.FC<ArticleRowProps> = ({ article, onPress, pending = false }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  const publishedDateLabel = formatPublishedDate(article.publishedDate);

  const renderRightContent = () => {
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
          style={[
            styles.title,
            article.isRead
              ? { color: colors.textSecondary, fontFamily: FontFamily.default }
              : { color: colors.text },
          ]}
          numberOfLines={2}
        >
          {article.title}
        </Text>
        <Text
          style={[styles.author, { color: colors.textSecondary }]}
          numberOfLines={1}
        >
          {article.author || 'Unknown'}
          {publishedDateLabel && ` | ${publishedDateLabel}`}
        </Text>
        {pending && (
          <Text style={[styles.pending, { color: colors.info }]} numberOfLines={1}>
            Waiting to save — will sync when online
          </Text>
        )}
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
  pending: {
    fontSize: 12,
    fontFamily: FontFamily.defaultMedium,
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
});
