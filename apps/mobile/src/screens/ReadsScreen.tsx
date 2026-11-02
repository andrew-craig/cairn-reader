import React, { useState, useCallback, useMemo, useRef } from 'react';
import { Alert, LayoutAnimation } from 'react-native';
import { useFocusEffect, useNavigation } from '@react-navigation/native';
import { StackNavigationProp } from '@react-navigation/stack';
import { ArticleListScreen } from '../components/ArticleListScreen';
import { IconButton } from '../components/common/IconButton';
import { ArticleRow } from '../components/common/ArticleRow';
import { SwipeAction } from '../components/common/SwipeAction';
import { ArchiveIcon } from '../components/icons';
import { AddLinkModal } from '../components/AddLinkModal';
import { Article, RootStackParamList } from '../types';
import { ReadService } from '../services/read';
import { ArticleStore } from '../services/articleStore';
import { ArticleMutations } from '../services/articleMutations';
import { SyncTrigger } from '../services/syncTrigger';
import { readsInvalidatedSince } from '../services/readsInvalidation';
import { useCursorArticleList, PAGE_SIZE } from '../hooks/useCursorArticleList';

// Minimum ms between background refetches triggered by tab focus.
const FOCUS_REFETCH_TTL_MS = 30_000;
// Cap on stored articles read for the initial/offline render — the store is a
// read-through cache for the first render, not a paginated query engine.
const STORED_ARTICLES_LIMIT = 100;

type ReadsScreenNavigationProp = StackNavigationProp<RootStackParamList, 'MainTabs'>;

const idsOf = (articles: Article[]) => new Set(articles.map((a) => a.id));

export const ReadsScreen: React.FC = () => {
  const navigation = useNavigation<ReadsScreenNavigationProp>();
  const [modalVisible, setModalVisible] = useState(false);
  const [isStale, setIsStale] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Timestamp of the last successful network fetch (null = never fetched)
  const lastFetchedAtRef = useRef<number | null>(null);

  // Saves queued offline (ArticleMutations.saveToReads) are listed above the
  // server's page until it includes them. `queued` is snapshotted just before
  // each first-page fetch, so a save that drains after that fetch (the server
  // page lacks it) stays listed until the next one. `pendingIds` drives the
  // "waiting to save" indicator and is re-read after every outbox drain.
  const [queued, setQueued] = useState<Article[]>([]);
  const [pendingIds, setPendingIds] = useState<Set<string>>(() => new Set());
  const queuedSnapshotRef = useRef<Article[]>([]);

  const fetchPage = useCallback(async (cursor: string | undefined) => {
    if (cursor === undefined) queuedSnapshotRef.current = await ArticleStore.listPendingSaves();
    return ReadService.listUserContents({ limit: PAGE_SIZE, cursor, list: 'reads' });
  }, []);

  const onResetLoaded = useCallback((next: Article[]) => {
    setQueued(queuedSnapshotRef.current);
    // SyncTrigger.run() drains the outbox and then runs prefetch, in that
    // fixed order (task_ebf1) — prefetch reads the post-sync store state
    // (upsertMany already cleared any body whose hash changed), so it must
    // run after upsertMany resolves, not alongside it. Routing pull-to-
    // refresh through the same entry point as the reconnect/foreground
    // triggers means the outbox gets a drain on every path, not only those.
    void ArticleStore.upsertMany(next)
      .then(() => SyncTrigger.run())
      .then(() => ArticleStore.listPendingSaves())
      .then((pending) => setPendingIds(idsOf(pending)))
      .catch((err) => console.error('Failed to sync articles to local store:', err));
    lastFetchedAtRef.current = Date.now();
    setIsStale(false);
    setError(null);
  }, []);

  const onLoadError = useCallback((reset: boolean) => {
    if (reset) {
      // Network failure after showing stale data — mark stale rather than
      // blocking with an alert; the user can still read cached content. With
      // nothing to show, ArticleListScreen surfaces `error` + a Retry button
      // instead (and hides the stale banner in that case).
      setIsStale(true);
      setError("Couldn't load your reading list. Check your connection and try again.");
    } else {
      Alert.alert('Error', 'Failed to load articles. Please try again.', [{ text: 'OK' }]);
    }
  }, []);

  const {
    articles,
    setArticles,
    loading,
    setLoading,
    refreshing,
    loadingMore,
    searchOpen,
    query,
    setQuery,
    searchQuery,
    openSearch,
    closeSearch,
    load,
    handleRefresh,
    handleLoadMore,
  } = useCursorArticleList({ fetchPage, onResetLoaded, onLoadError, list: 'reads' });

  // Mirror the current list so the stable archive callback (also handed to the
  // reader) computes the next list from the latest articles.
  const articlesRef = useRef<Article[]>(articles);
  articlesRef.current = articles;

  // On focus: show cached stale data immediately, then refetch in the
  // background if the TTL has expired. Skip while a search is active.
  useFocusEffect(
    useCallback(() => {
      if (searchQuery) return;

      const now = Date.now();
      const ttlExpired =
        lastFetchedAtRef.current === null ||
        now - lastFetchedAtRef.current > FOCUS_REFETCH_TTL_MS ||
        readsInvalidatedSince(lastFetchedAtRef.current);

      if (!ttlExpired) return;

      Promise.all([
        ArticleStore.listRecent(STORED_ARTICLES_LIMIT),
        ArticleStore.listPendingSaves(),
      ]).then(([stored, pending]) => {
        setQueued(pending);
        setPendingIds(idsOf(pending));
        if (stored.length > 0) {
          setArticles(stored);
          setLoading(false);
          setIsStale(true);
        }
        void load(true);
      });
    }, [searchQuery, load, setArticles, setLoading]),
  );

  const handleArticleArchived = useCallback(
    (articleId: string) => {
      const next = articlesRef.current.filter((a) => a.id !== articleId);
      // The store row is removed by ArticleMutations.archive, which every
      // caller runs alongside this list update.
      setArticles(next);
      setQueued((prev) => prev.filter((a) => a.id !== articleId));
    },
    [setArticles],
  );

  // Search results stand alone; otherwise queued saves the page lacks go on top.
  const listed = useMemo(() => {
    if (searchQuery) return articles;
    const onPage = idsOf(articles);
    return [...queued.filter((a) => !onPage.has(a.id)), ...articles];
  }, [searchQuery, queued, articles]);

  const handleSwipeArchive = (articleId: string) => {
    // Collapse the gap the swiped-away row leaves behind.
    LayoutAnimation.configureNext(LayoutAnimation.Presets.easeInEaseOut);
    handleArticleArchived(articleId);
    ArticleMutations.archive(articleId).catch((error) => {
      console.error('Failed to archive article:', error);
      Alert.alert('Error', 'Failed to archive article');
    });
  };

  const handleArticlePress = (article: Article) => {
    const currentIndex = listed.findIndex((a) => a.id === article.id);
    navigation.navigate('ArticleDetail', {
      article,
      articles: listed,
      currentIndex,
      onArchived: handleArticleArchived,
    });
  };

  const headerActions = (
    <IconButton icon="add-outline" onPress={() => setModalVisible(true)} accessibilityLabel="Add link" />
  );

  return (
    <>
      <ArticleListScreen
        title="Reads"
        articles={listed}
        loading={loading}
        headerActions={headerActions}
        renderItem={(article) => (
          <SwipeAction direction="left" icon={ArchiveIcon} color="success" onAction={() => handleSwipeArchive(article.id)}>
            <ArticleRow
              article={article}
              onPress={() => handleArticlePress(article)}
              pending={pendingIds.has(article.id)}
            />
          </SwipeAction>
        )}
        onRefresh={handleRefresh}
        refreshing={refreshing}
        emptyMessage={searchQuery ? 'No matching articles' : 'No saved articles yet'}
        onEndReached={handleLoadMore}
        loadingMore={loadingMore}
        search={{
          open: searchOpen,
          query,
          onChangeQuery: setQuery,
          onOpen: openSearch,
          onClose: closeSearch,
          placeholder: 'Search your reads',
        }}
        staleMessage={isStale ? 'Showing cached data — pull to refresh' : undefined}
        error={error}
        onRetry={() => load(true)}
      />
      <AddLinkModal
        visible={modalVisible}
        onClose={() => setModalVisible(false)}
        onSuccess={handleRefresh}
      />
    </>
  );
};
