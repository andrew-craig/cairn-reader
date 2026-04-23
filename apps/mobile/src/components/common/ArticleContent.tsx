import React, { useCallback, useMemo, useRef } from 'react';
import {
  View,
  Text,
  StyleSheet,
  ScrollView,
  Linking,
  Alert,
  ActionSheetIOS,
  Platform,
  useWindowDimensions,
  NativeSyntheticEvent,
  NativeScrollEvent,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import RenderHTML, { CustomTextualRenderer } from 'react-native-render-html';
import * as Haptics from 'expo-haptics';
import * as Clipboard from 'expo-clipboard';
import { Article } from '../../types';
import { formatDate, extractDomain } from '../../utils';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily, Layout } from '../../constants';
import { ReadService } from '../../services/read';

interface ArticleContentProps {
  article: Article;
  colors: typeof Colors.light;
  onScrollPositionChange?: (position: number) => void;
  initialScrollPosition?: number;
}

export const ArticleContent: React.FC<ArticleContentProps> = ({
  article,
  colors,
  onScrollPositionChange,
  initialScrollPosition,
}) => {
  const { width } = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const scrollViewRef = useRef<ScrollView>(null);
  const hasRestoredPosition = useRef(false);

  const handleLinkAction = useCallback(async (actionIndex: number, href: string) => {
    switch (actionIndex) {
      case 0:
        try {
          await ReadService.addContentToUser({ url: href, source_type: 'web' });
          await Haptics.notificationAsync(Haptics.NotificationFeedbackType.Success);
          Alert.alert('Saved', 'Link saved to your reading list.');
        } catch {
          Alert.alert('Error', 'Failed to save link. Please try again.');
        }
        break;
      case 1:
        Linking.openURL(href);
        break;
      case 2:
        await Clipboard.setStringAsync(href);
        break;
    }
  }, []);

  const handleLinkLongPress = useCallback(async (href: string) => {
    await Haptics.impactAsync(Haptics.ImpactFeedbackStyle.Medium);

    const options = ['Save to Reading List', 'Open in Browser', 'Copy Link', 'Cancel'];
    const cancelButtonIndex = 3;

    if (Platform.OS === 'ios') {
      ActionSheetIOS.showActionSheetWithOptions(
        { options, cancelButtonIndex, title: extractDomain(href) },
        (index) => handleLinkAction(index, href)
      );
    } else {
      Alert.alert(
        extractDomain(href),
        undefined,
        [
          { text: 'Save to Reading List', onPress: () => handleLinkAction(0, href) },
          { text: 'Open in Browser', onPress: () => handleLinkAction(1, href) },
          { text: 'Copy Link', onPress: () => handleLinkAction(2, href) },
          { text: 'Cancel', style: 'cancel' },
        ]
      );
    }
  }, [handleLinkAction]);

  const renderers = useMemo<{ a: CustomTextualRenderer }>(() => ({
    a: function AnchorRenderer({ tnode, TDefaultRenderer, textProps, ...props }) {
      const href = tnode.attributes.href;
      return (
        <TDefaultRenderer
          tnode={tnode}
          textProps={{
            ...textProps,
            onLongPress: href ? () => handleLinkLongPress(href) : undefined,
          }}
          {...props}
        />
      );
    },
  }), [handleLinkLongPress]);

  const handleScroll = useCallback((event: NativeSyntheticEvent<NativeScrollEvent>) => {
    onScrollPositionChange?.(Math.round(event.nativeEvent.contentOffset.y));
  }, [onScrollPositionChange]);

  const handleContentSizeChange = useCallback(() => {
    if (!hasRestoredPosition.current && initialScrollPosition && initialScrollPosition > 0) {
      scrollViewRef.current?.scrollTo({ y: initialScrollPosition, animated: false });
      hasRestoredPosition.current = true;
    }
  }, [initialScrollPosition]);

  // Calculate padding to account for safe areas and floating action menu
  const topPadding = insets.top + Spacing.md;
  const bottomPadding = Layout.bottomActionMenuHeight + insets.bottom + Spacing.md;

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
      ref={scrollViewRef}
      style={styles.scrollView}
      contentContainerStyle={{ paddingTop: topPadding, paddingBottom: bottomPadding }}
      showsVerticalScrollIndicator={false}
      onScroll={handleScroll}
      scrollEventThrottle={500}
      onContentSizeChange={handleContentSizeChange}
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
              renderers={renderers}
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
