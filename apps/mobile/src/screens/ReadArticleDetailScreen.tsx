import React, { useState, useRef, useCallback, useEffect } from 'react';
import {
  View,
  Text,
  ActivityIndicator,
  StyleSheet,
  useColorScheme,
  Alert,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { throttle } from '@cairn/shared';
import { Article, RootStackParamList } from '../types';
import { ArticleStore, ReadService } from '../services';
import { Colors, GlobalStyles } from '../constants';
import { useNetworkStatus } from '../hooks/useNetworkStatus';
import { ArticleContent, BottomActionMenu } from '../components/common';
import type { ScrollProgressInfo } from '../components/common/ArticleContent';

const COMPLETED_PROGRESS_THRESHOLD = 0.95;
// Throttle persisting scroll position while the user is actively scrolling,
// so progress survives the app being closed mid-read instead of only being
// saved on navigation.
const SCROLL_SAVE_THROTTLE_MS = 1000;

const toFraction = (v?: number): number | undefined =>
  v !== undefined && v <= 1 ? v : undefined;

type ReadArticleDetailRouteProp = RouteProp<RootStackParamList, 'ArticleDetail'>;
type ReadArticleDetailNavigationProp = StackNavigationProp<RootStackParamList, 'ArticleDetail'>;

export const ReadArticleDetailScreen: React.FC = () => {
  const route = useRoute<ReadArticleDetailRouteProp>();
  const navigation = useNavigation<ReadArticleDetailNavigationProp>();
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;
  const { article: initialArticle, articles = [], currentIndex = -1, onArchived } = route.params;

  // When arriving from a list screen the article may not have cleaned_html yet
  // (list responses are summaries). We lazy-load the full content on mount.
  const [article, setArticle] = useState<Article>(initialArticle);
  const [contentLoading, setContentLoading] = useState(!initialArticle.content);
  const { isOffline } = useNetworkStatus();
  // Read inside the effect below without adding `isOffline` to its deps —
  // this mirrors the file's existing pattern (see articleIdRef etc.) of
  // reading current values through a ref so connectivity flapping mid-read
  // can't restart the content-loading effect.
  const isOfflineRef = useRef(isOffline);
  isOfflineRef.current = isOffline;

  useEffect(() => {
    if (initialArticle.content) return; // already have the HTML
    let cancelled = false;

    // Resolve the article by id from the store rather than trusting route
    // params alone: the store carries the body and the hash it was cached
    // against, while route params carry the fresh list metadata (including
    // the fresh hash) — merge the two.
    ArticleStore.getById(initialArticle.id).then((stored) => {
      if (cancelled) return;
      const hasStoredBody = Boolean(stored?.content);

      if (hasStoredBody && stored) {
        setArticle((current) =>
          current.content
            ? current
            : { ...current, content: stored.content, contentHash: stored.contentHash },
        );
        setContentLoading(false);
      }

      // A stored body whose hash matches the fresh route-param hash is
      // already current — skip the network fetch entirely. That's the point
      // of the hash diff: today's screen (pre-task_c55c) fetched on every
      // open regardless.
      const hashMatches =
        hasStoredBody &&
        stored?.contentHash !== undefined &&
        stored.contentHash === initialArticle.contentHash;
      if (hashMatches) return;

      // Never fetch while offline — it can only fail. With nothing stored,
      // stop loading so the "Not available offline" state below can render
      // instead of hanging on the spinner.
      if (isOfflineRef.current) {
        if (!hasStoredBody) setContentLoading(false);
        return;
      }

      ReadService.getContentById(initialArticle.id)
        .then((detail) => {
          if (cancelled) return;
          const updated = ReadService.transformDetailToArticle(detail);
          setArticle(updated);
          setContentLoading(false);
          if (updated.content) {
            void ArticleStore.saveBody(initialArticle.id, updated.content);
          }
        })
        .catch((err) => {
          if (cancelled) return;
          console.error('Failed to load article content:', err);
          setContentLoading(false);
        });
    });

    return () => { cancelled = true; };
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialArticle.id]);

  // Mutable UI state tracked separately from the article object so async updates
  // are scoped to the displayed article. Seeded from the article and resynced
  // when the displayed article changes (see effect below). isRead drives the
  // completion guard; isFavorite is reflected in the action menu.
  const [isFavorite, setIsFavorite] = useState(article.isFavorite);
  const scrollFractionRef = useRef(toFraction(article.scrollFraction) ?? toFraction(article.scrollPosition) ?? 0);
  const hasScrolledRef = useRef(false);
  const hasMarkedCompletedRef = useRef(article.isRead);
  // Tracks the currently displayed article id so async callbacks can detect a
  // swap (next article) and avoid mutating UI state for a different article.
  const articleIdRef = useRef(article.id);
  // Backend-only throttled save; local AsyncStorage persistence stays on the
  // unmount flush below since it rewrites the whole articles array.
  const throttledSaveRef = useRef(
    throttle((articleId: string, fraction: number) => {
      ReadService.updateUserContent(articleId, { scroll_position: fraction }).catch(
        (err) => console.error('Failed to save scroll position:', err)
      );
    }, SCROLL_SAVE_THROTTLE_MS),
  );

  const hasNextArticle = currentIndex >= 0 && currentIndex < articles.length - 1;

  // Reset per-article progress refs and reseed the mutable UI state whenever the
  // displayed article changes (next article reuses this screen via replace()).
  useEffect(() => {
    articleIdRef.current = article.id;
    scrollFractionRef.current = toFraction(article.scrollFraction) ?? toFraction(article.scrollPosition) ?? 0;
    hasScrolledRef.current = false;
    hasMarkedCompletedRef.current = article.isRead;
    setIsFavorite(article.isFavorite);
  }, [article]);

  const markCompleted = useCallback((articleId: string) => {
    if (hasMarkedCompletedRef.current) return;
    hasMarkedCompletedRef.current = true;
    const readAt = Date.now();
    ReadService.updateUserContent(articleId, { status: 'completed' }).catch(
      (err) => console.error('Failed to mark article completed:', err)
    );
    ArticleStore.updateUserState(articleId, { isRead: true, readAt }).catch(
      (err) => console.error('Failed to persist completed locally:', err)
    );
  }, []);

  const handleScrollProgress = useCallback((info: ScrollProgressInfo) => {
    // Emits fired before the saved position has been restored report a stale
    // offsetY of 0. Skip updating the refs so the unmount-flush safety net
    // can't send that stale value either — not just the throttled save.
    // Completion marking stays unconditional: it also covers articles short
    // enough to never scroll, which never leave the restoring state.
    if (!info.isRestoring) {
      scrollFractionRef.current = info.fraction;
      hasScrolledRef.current = true;
    }

    if (!hasMarkedCompletedRef.current && info.contentHeight > 0) {
      const progress = (info.offsetY + info.layoutHeight) / info.contentHeight;
      if (progress >= COMPLETED_PROGRESS_THRESHOLD) {
        markCompleted(article.id);
      }
    }

    if (!info.isRestoring) {
      throttledSaveRef.current(article.id, info.fraction);
    }
  }, [article.id, markCompleted]);

  useEffect(() => {
    if (article.isRead) return;
    ReadService.updateUserContent(article.id, { status: 'reading' }).catch(
      (err) => console.error('Failed to mark article reading:', err)
    );
  }, [article.id, article.isRead]);

  useEffect(() => {
    const articleId = article.id;
    const throttledSave = throttledSaveRef.current;
    return () => {
      throttledSave.cancel();
      if (!hasScrolledRef.current) return;
      const fraction = scrollFractionRef.current;
      ReadService.updateUserContent(articleId, { scroll_position: fraction }).catch(
        (err) => console.error('Failed to save scroll position:', err)
      );
      ArticleStore.updateUserState(articleId, { scrollFraction: fraction }).catch(
        (err) => console.error('Failed to save scroll position locally:', err)
      );
    };
  }, [article.id]);

  const handleBack = () => {
    navigation.goBack();
  };

  const handleNextArticle = () => {
    if (!hasNextArticle) return;
    const nextIndex = currentIndex + 1;
    navigation.replace('ArticleDetail', {
      article: articles[nextIndex],
      articles,
      currentIndex: nextIndex,
      onArchived,
    });
  };

  const handleToggleFavorite = async () => {
    const targetId = article.id;
    const newIsFavorite = !isFavorite;
    // Optimistically reflect the new state in the action menu.
    setIsFavorite(newIsFavorite);
    try {
      // Update both local storage and backend
      await ArticleStore.updateUserState(targetId, {
        isFavorite: newIsFavorite,
      });
      try {
        await ReadService.updateUserContent(targetId, {
          is_favorite: newIsFavorite,
        });
      } catch (backendError) {
        console.error('Failed to sync favorite status to backend:', backendError);
        // Continue anyway - local update was successful
      }
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
      // Roll back the optimistic update on failure, but only if the same
      // article is still displayed — otherwise we'd flip the wrong article.
      if (articleIdRef.current === targetId) setIsFavorite(!newIsFavorite);
    }
  };

  const handleArchive = async () => {
    try {
      await ArticleStore.remove(article.id);
      onArchived?.(article.id);
      // Backend delete runs in the background so a slow/offline network
      // doesn't block navigation; failures are logged only (see above).
      ReadService.deleteUserContent(article.id).catch((backendError) => {
        console.error('Failed to archive article in backend:', backendError);
      });
      navigation.goBack();
    } catch (error) {
      console.error('Failed to archive article:', error);
      Alert.alert('Error', 'Failed to archive article');
    }
  };

  if (contentLoading) {
    return (
      <View style={[styles.container, styles.centered, { backgroundColor: colors.background }]}>
        <ActivityIndicator size="large" color={colors.primary} />
      </View>
    );
  }

  // Offline with nothing cached: a blank article body is misleading, so say
  // so explicitly instead.
  if (!article.content && isOffline) {
    return (
      <View style={[styles.container, styles.centered, { backgroundColor: colors.background }]}>
        <View style={GlobalStyles.emptyContainer}>
          <Text style={[GlobalStyles.emptyText, { color: colors.textSecondary }]}>
            Not available offline
          </Text>
        </View>
      </View>
    );
  }

  return (
    <View style={[styles.container, { backgroundColor: colors.background }]}>
      <ArticleContent
        article={article}
        colors={colors}
        onScrollProgress={handleScrollProgress}
        initialScrollFraction={toFraction(article.scrollFraction) ?? toFraction(article.scrollPosition)}
      />

      <BottomActionMenu
        actions={[
          {
            icon: 'return',
            label: 'Back',
            onPress: handleBack,
          },
          {
            icon: 'next-article',
            label: 'Next',
            onPress: handleNextArticle,
            disabled: !hasNextArticle,
          },
          {
            icon: 'bookmark',
            label: 'Favorite',
            onPress: handleToggleFavorite,
            active: isFavorite,
          },
          {
            icon: 'archive',
            label: 'Archive',
            onPress: handleArchive,
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
  centered: {
    alignItems: 'center',
    justifyContent: 'center',
  },
});
