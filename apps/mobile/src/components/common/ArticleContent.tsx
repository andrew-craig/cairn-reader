import React from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Linking,
  useWindowDimensions,
} from 'react-native';
import RenderHTML from 'react-native-render-html';
import { Article } from '../../types';
import { formatDate, extractDomain } from '../../utils';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily } from '../../constants';

interface ArticleContentProps {
  article: Article;
  colors: typeof Colors.light;
}

export const ArticleContent: React.FC<ArticleContentProps> = ({
  article,
  colors,
}) => {
  const { width } = useWindowDimensions();

  // HTML rendering configuration
  const tagsStyles = {
    body: {
      color: colors.text,
      fontSize: FontSizes.md,
      lineHeight: 24,
    },
    p: {
      marginBottom: Spacing.md,
      color: colors.text,
    },
    a: {
      color: colors.primary,
      textDecorationLine: 'underline' as const,
    },
    h1: {
      fontSize: FontSizes.xl,
      fontWeight: 'bold' as const,
      marginTop: Spacing.lg,
      marginBottom: Spacing.md,
      color: colors.text,
    },
    h2: {
      fontSize: FontSizes.lg,
      fontWeight: 'bold' as const,
      marginTop: Spacing.lg,
      marginBottom: Spacing.sm,
      color: colors.text,
    },
    h3: {
      fontSize: FontSizes.md,
      fontWeight: 'bold' as const,
      marginTop: Spacing.md,
      marginBottom: Spacing.sm,
      color: colors.text,
    },
    blockquote: {
      borderLeftWidth: 4,
      borderLeftColor: colors.border,
      paddingLeft: Spacing.md,
      marginLeft: 0,
      marginVertical: Spacing.md,
      fontStyle: 'italic' as const,
      color: colors.textSecondary,
    },
    pre: {
      backgroundColor: colors.card,
      padding: Spacing.md,
      borderRadius: BorderRadius.sm,
      marginVertical: Spacing.md,
    },
    code: {
      backgroundColor: colors.card,
      paddingHorizontal: Spacing.xs,
      fontFamily: 'monospace',
      fontSize: FontSizes.sm,
    },
    img: {
      marginVertical: Spacing.md,
    },
    ul: {
      marginBottom: Spacing.md,
    },
    ol: {
      marginBottom: Spacing.md,
    },
    li: {
      marginBottom: Spacing.xs,
    },
  };

  return (
    <ScrollView
      style={styles.scrollView}
      contentContainerStyle={{ paddingTop: Spacing.md, paddingBottom: 100 }}
      showsVerticalScrollIndicator={false}
    >
      <View style={styles.content}>
        {/* Article Header */}
        <View style={styles.header}>
          <Text style={[styles.title, { color: colors.text }]}>
            {article.title}
          </Text>
          <Text style={[styles.publishedOn, { color: colors.textSecondary }]}>
            Published on{' '}
            <Text
              style={[styles.publisherLink, { color: colors.textSecondary }]}
              onPress={() => Linking.openURL(article.url)}
            >
              {extractDomain(article.url)}
            </Text>
            {article.addedAt && ` | ${formatDate(article.addedAt)}`}
          </Text>
        </View>

        {/* Article Body */}
        <View style={styles.textFrame}>
          {article.content ? (
            <RenderHTML
              contentWidth={width - (Spacing.md * 2)}
              source={{ html: article.content }}
              tagsStyles={tagsStyles}
              defaultTextProps={{
                selectable: true,
              }}
            />
          ) : article.description ? (
            <Text style={[styles.bodyText, { color: colors.text }]}>
              {article.description}
            </Text>
          ) : null}
        </View>
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  scrollView: {
    flex: 1,
  },
  content: {
    flex: 1,
  },
  header: {
    paddingTop: Spacing.sm,
    paddingHorizontal: Spacing.md,
    paddingBottom: Spacing.md,
    gap: 10,
  },
  title: {
    fontSize: 24,
    fontFamily: FontFamily.defaultSemiBold,
    lineHeight: 32,
  },
  publishedOn: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    lineHeight: 22,
  },
  publisherLink: {
    textDecorationLine: 'underline',
  },
  textFrame: {
    paddingHorizontal: Spacing.md,
    paddingVertical: 0,
  },
  bodyText: {
    fontSize: FontSizes.md,
    fontFamily: FontFamily.default,
    lineHeight: 22.4,
  },
});
