import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  TouchableOpacity,
  Image,
  Linking,
} from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { Article } from '../../types';
import { formatDate, extractDomain } from '../../utils';
import { Colors, Spacing, FontSizes, BorderRadius } from '../../constants';

interface ArticleContentProps {
  article: Article;
  colors: typeof Colors.light;
  onOpenInBrowser?: () => void;
  showMetadata?: boolean;
}

export const ArticleContent: React.FC<ArticleContentProps> = ({
  article,
  colors,
  onOpenInBrowser,
  showMetadata = true,
}) => {
  const insets = useSafeAreaInsets();

  const handleOpenInBrowser = async () => {
    if (onOpenInBrowser) {
      onOpenInBrowser();
    } else {
      try {
        await WebBrowser.openBrowserAsync(article.url);
      } catch (error) {
        console.error('Failed to open browser:', error);
      }
    }
  };

  return (
    <ScrollView style={styles.scrollView} contentContainerStyle={{ paddingTop: insets.top }}>
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
        <Text style={[styles.publishedOn, { color: colors.textSecondary }]}>
          Published on{' '}
          <Text
            style={[styles.publisherLink, { color: colors.primary }]}
            onPress={() => Linking.openURL(article.url)}
          >
            {extractDomain(article.url)}
          </Text>
        </Text>
        {article.description && (
          <Text style={[styles.description, { color: colors.textSecondary }]}>
            {article.description}
          </Text>
        )}
        {showMetadata && (
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
        )}
        <TouchableOpacity
          style={[styles.openButton, { backgroundColor: colors.primary }]}
          onPress={handleOpenInBrowser}
        >
          <Text style={styles.openButtonText}>Open in Browser</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
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
    marginBottom: Spacing.sm,
  },
  publishedOn: {
    fontSize: FontSizes.sm,
    marginBottom: Spacing.md,
  },
  publisherLink: {
    textDecorationLine: 'underline',
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
});
