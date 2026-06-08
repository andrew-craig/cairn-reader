import React, { useCallback, useRef } from 'react';
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
  LayoutChangeEvent,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import RenderHTML from 'react-native-render-html';
import * as Haptics from 'expo-haptics';
import * as Clipboard from 'expo-clipboard';
import { Article } from '../../types';
import { formatDate, extractDomain } from '../../utils';
import { Colors, Spacing, FontSizes, BorderRadius, FontFamily, Layout } from '../../constants';
import { ReadService } from '../../services/read';

export interface ScrollProgressInfo {
  offsetY: number;
  contentHeight: number;
  layoutHeight: number;
}

/**
 * Whether a saved scroll offset can actually be reached given the currently
 * measured content and viewport heights. HTML content (especially emails) lays
 * out progressively, so early size measurements understate the final height and
 * a scrollTo to the target would be clamped short. We only consider restoration
 * final once the content has grown tall enough to reach the target.
 */
export const isScrollTargetReachable = (
  contentHeight: number,
  layoutHeight: number,
  target: number
): boolean => layoutHeight > 0 && contentHeight - layoutHeight >= target;

interface ArticleContentProps {
  article: Article;
  colors: typeof Colors.light;
  onScrollProgress?: (info: ScrollProgressInfo) => void;
  initialScrollPosition?: number;
}

export const ArticleContent: React.FC<ArticleContentProps> = ({
  article,
  colors,
  onScrollProgress,
  initialScrollPosition,
}) => {
  const { width } = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const scrollViewRef = useRef<ScrollView>(null);
  const hasRestoredPosition = useRef(false);
  const offsetYRef = useRef(0);
  const contentHeightRef = useRef(0);
  const layoutHeightRef = useRef(0);

  const emitProgress = useCallback(() => {
    if (!onScrollProgress) return;
    if (contentHeightRef.current <= 0 || layoutHeightRef.current <= 0) return;
    onScrollProgress({
      offsetY: offsetYRef.current,
      contentHeight: contentHeightRef.current,
      layoutHeight: layoutHeightRef.current,
    });
  }, [onScrollProgress]);

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

  const handleLinkPress = useCallback(async (href: string) => {
    if (!href) return;
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

  const renderersProps = { a: { onPress: (_e: unknown, href: string) => handleLinkPress(href) } };

  const handleScroll = useCallback((event: NativeSyntheticEvent<NativeScrollEvent>) => {
    const { contentOffset, contentSize, layoutMeasurement } = event.nativeEvent;
    offsetYRef.current = Math.round(contentOffset.y);
    contentHeightRef.current = contentSize.height;
    layoutHeightRef.current = layoutMeasurement.height;
    emitProgress();
  }, [emitProgress]);

  const handleContentSizeChange = useCallback((_w: number, contentHeight: number) => {
    contentHeightRef.current = contentHeight;
    // Re-apply the saved position on every size change while the content keeps
    // growing, latching only once the target is actually reachable. A single
    // early scrollTo would be clamped near the top for progressively-rendered
    // HTML (e.g. emails) and never retried.
    if (!hasRestoredPosition.current && initialScrollPosition && initialScrollPosition > 0 && initialScrollPosition <= 1) {
      const targetY = Math.round(initialScrollPosition * contentHeight);
      scrollViewRef.current?.scrollTo({ y: targetY, animated: false });
      if (isScrollTargetReachable(contentHeight, layoutHeightRef.current, targetY)) {
        hasRestoredPosition.current = true;
      }
    }
    emitProgress();
  }, [initialScrollPosition, emitProgress]);

  // Once the user takes control by dragging, stop trying to restore so we never
  // yank them away from where they scrolled.
  const handleScrollBeginDrag = useCallback(() => {
    hasRestoredPosition.current = true;
  }, []);

  const handleScrollViewLayout = useCallback((event: LayoutChangeEvent) => {
    layoutHeightRef.current = event.nativeEvent.layout.height;
    emitProgress();
  }, [emitProgress]);

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
      scrollEventThrottle={100}
      onScrollBeginDrag={handleScrollBeginDrag}
      onContentSizeChange={handleContentSizeChange}
      onLayout={handleScrollViewLayout}
    >
      <View style={styles.content}>
        {/* Article Header */}
        <View style={styles.header}>
          <Text
            style={[styles.title, { color: colors.text }]}
            onPress={() => Linking.openURL(article.url).catch(() => {})}
          >
            {article.title}
          </Text>
          <Text style={[styles.publishedOn, { color: colors.textSecondary }]}>
            {article.url.startsWith('email://') ? (
              <Text style={[styles.publisherLink, { color: colors.textSecondary }]}>
                {article.author || 'Email'}
              </Text>
            ) : (
              <Text
                style={[styles.publisherLink, { color: colors.textSecondary }]}
                onPress={async () => {
                  try {
                    await Linking.openURL(new URL(article.url).origin);
                  } catch {
                    Linking.openURL(article.url).catch(() => {});
                  }
                }}
              >
                {article.author || extractDomain(article.url)}
              </Text>
            )}
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
              renderersProps={renderersProps}
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
