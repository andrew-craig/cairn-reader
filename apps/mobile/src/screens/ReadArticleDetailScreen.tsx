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
import { ArticleStore, ReadService, ArticleMutations } from '../services';
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

  useEffect(() => {
    // Gate on the currently displayed article's content, not the initial
    // route param: on the first run these are the same thing (`article`
    // state is seeded from `initialArticle`), but this effect also re-runs
    // when connectivity returns (see the `isOffline` dep below), and by
    // then `article` may already carry a body a background prefetch wrote
    // to the store while this screen was open. Gating on "still missing"
    // rather than the frozen initial value is what stops a completed load
    // from being restarted by later connectivity flapping.
    if (article.content) return;
    // Past this point there is definitely no content to show yet, and we're
    // about to hit the store and possibly the network for it. Show the
    // spinner rather than leaving contentLoading at whatever it already was
    // — on a fresh mount that's already true, but on a reconnect-triggered
    // re-run (the `isOffline` dep below) it's false, left over from the
    // earlier "Not available offline" render, which would otherwise fall
    // through to a blank ArticleContent for the duration of this fetch. The
    // offline-with-nothing-stored branch further down still wins once the
    // store lookup resolves, since it explicitly sets this back to false.
    setContentLoading(true);
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
      // instead of hanging on the spinner. Once connectivity returns, the
      // `isOffline` dep below re-runs this effect and retries.
      if (isOffline) {
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
  // `article` is deliberately excluded: it's read for the "still missing"
  // gate above via closure, which is enough since a dep change (id or
  // isOffline) always re-renders before this effect body runs; adding it
  // would just make this effect re-run once more (a no-op past the gate)
  // every time it sets `article` itself.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialArticle.id, isOffline]);

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
  // Throttled while actively scrolling; the unmount flush below covers the
  // final position on the way out. Both go through the same facade call
  // (store write, then a backend attempt queued on NetworkError).
  const throttledSaveRef = useRef(
    throttle((articleId: string, fraction: number) => {
      ArticleMutations.saveScrollPosition(articleId, fraction).catch(
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
    ArticleMutations.markCompleted(articleId, readAt).catch(
      (err) => console.error('Failed to mark article completed:', err)
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
    ArticleMutations.markReading(article.id).catch(
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
      ArticleMutations.saveScrollPosition(articleId, fraction).catch(
        (err) => console.error('Failed to save scroll position:', err)
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
      await ArticleMutations.setFavorite(targetId, newIsFavorite);
    } catch (error) {
      console.error('Failed to toggle favorite:', error);
      // Roll back the optimistic update on failure, but only if the same
      // article is still displayed — otherwise we'd flip the wrong article.
      // ArticleMutations.setFavorite already wrote the store before it
      // rethrew (only NetworkError is absorbed there), so the store needs
      // the same rollback as the UI or BookmarksScreen (listFavorites())
      // would disagree with what this screen now shows.
      if (articleIdRef.current === targetId) {
        setIsFavorite(!newIsFavorite);
        ArticleStore.updateUserState(targetId, { isFavorite: !newIsFavorite }).catch(
          (storeError) => console.error('Failed to roll back favorite locally:', storeError)
        );
      }
    }
  };

  const handleArchive = () => {
    const targetId = article.id;
    onArchived?.(targetId);
    navigation.goBack();
    // Deliberately not awaited: offline is fast (ArticleMutations.archive
    // queues it on NetworkError), but waiting on a slow-but-online DELETE
    // would freeze the archive button with no spinner. A real failure still
    // surfaces via the alert below — it just does so after navigation
    // instead of blocking it.
    ArticleMutations.archive(targetId).catch((error) => {
      console.error('Failed to archive article:', error);
      Alert.alert('Error', 'Failed to archive article');
    });
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
