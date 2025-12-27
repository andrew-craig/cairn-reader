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

interface ArticleRowProps {
  article: Article;
  onPress: () => void;
}

export const ArticleRow: React.FC<ArticleRowProps> = ({ article, onPress }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

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
      {article.imageUrl && (
        <View style={styles.imageFrame}>
          <Image
            source={{ uri: article.imageUrl }}
            style={styles.image}
            resizeMode="cover"
          />
        </View>
      )}
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
    alignItems: 'flex-start',
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
    width: 64,
    height: 64,
    borderRadius: 6,
    overflow: 'hidden',
  },
  image: {
    width: '100%',
    height: '100%',
  },
});
