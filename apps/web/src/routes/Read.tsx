import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { PAGE_SIZE, ReadService } from '../services/read';
import ArticleRow from '../components/ArticleRow';
import './Read.css';

// The reading list (/read): the user's saved content, cursor-paginated from the
// Read Service with infinite scroll. Mirrors mobile's ReadScreen — it reloads
// from the first page on (re)mount, so archive/favorite changes made in the
// reader are reflected on return (mobile uses useFocusEffect for the same).
export default function Read() {
  const navigate = useNavigate();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true); // first page
  const [loadingMore, setLoadingMore] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Pagination cursor + guards kept in refs so the IntersectionObserver
  // callback reads current values without re-subscribing on every page.
  const cursorRef = useRef('');
  const hasMoreRef = useRef(true);
  const inFlightRef = useRef(false);
  const observerRef = useRef<IntersectionObserver | null>(null);

  // Fetch one page. reset=true starts from the first page (cursor cleared);
  // otherwise it appends the next page using the stored cursor.
  const fetchPage = useCallback(async (reset: boolean) => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setError(null);
    try {
      const response = await ReadService.listUserContents({
        limit: PAGE_SIZE,
        cursor: reset ? undefined : cursorRef.current || undefined,
      });
      const page = response.contents.map(ReadService.transformToArticle);
      setArticles((prev) => (reset ? page : [...prev, ...page]));
      cursorRef.current = response.cursor;
      hasMoreRef.current = response.has_more;
      setHasMore(response.has_more);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load your reading list.');
    } finally {
      inFlightRef.current = false;
    }
  }, []);

  // Initial load on mount (and remount when returning from the reader).
  useEffect(() => {
    setLoading(true);
    fetchPage(true).finally(() => setLoading(false));
  }, [fetchPage]);

  const handleRefresh = useCallback(() => {
    if (inFlightRef.current) return;
    setRefreshing(true);
    // Keep the current list visible; fetchPage(true) replaces it with the first
    // page once it loads, avoiding an empty-state flash mid-refresh.
    fetchPage(true).finally(() => setRefreshing(false));
  }, [fetchPage]);

  const handleLoadMore = useCallback(() => {
    if (inFlightRef.current || !hasMoreRef.current) return;
    setLoadingMore(true);
    fetchPage(false).finally(() => setLoadingMore(false));
  }, [fetchPage]);

  // Infinite scroll: a callback ref binds the IntersectionObserver as the
  // sentinel mounts and disconnects it when it unmounts. (A plain ref + effect
  // would miss empty→populated transitions, e.g. after a refresh or retry,
  // since updating ref.current neither re-renders nor re-runs the effect.)
  const sentinelRef = useCallback(
    (node: HTMLDivElement | null) => {
      observerRef.current?.disconnect();
      observerRef.current = null;
      if (node) {
        const observer = new IntersectionObserver(
          (entries) => {
            if (entries[0].isIntersecting) handleLoadMore();
          },
          { rootMargin: '200px' },
        );
        observer.observe(node);
        observerRef.current = observer;
      }
    },
    [handleLoadMore],
  );

  const handleSelect = useCallback(
    (article: Article) => {
      const index = articles.findIndex((a) => a.id === article.id);
      // Pass the list + index so the reader can offer prev/next without a refetch.
      navigate(`/read/${article.id}`, { state: { articles, index } });
    },
    [articles, navigate],
  );

  return (
    <div className="read">
      <header className="read__header">
        <h1 className="read__title">Read</h1>
        <button
          type="button"
          className="read__refresh"
          onClick={handleRefresh}
          disabled={refreshing || loading}
        >
          {refreshing ? 'Refreshing…' : 'Refresh'}
        </button>
      </header>

      {loading ? (
        <p className="read__status">Loading your reading list…</p>
      ) : error ? (
        <div className="read__status">
          <p className="read__error">{error}</p>
          <button type="button" className="read__refresh" onClick={handleRefresh}>
            Try again
          </button>
        </div>
      ) : articles.length === 0 ? (
        <p className="read__status">No saved articles yet.</p>
      ) : (
        <>
          <ul className="read__list">
            {articles.map((article) => (
              <ArticleRow key={article.id} article={article} onSelect={handleSelect} />
            ))}
          </ul>
          <div ref={sentinelRef} className="read__sentinel" aria-hidden="true" />
          {loadingMore && <p className="read__status">Loading more…</p>}
          {!hasMore && <p className="read__status">You've reached the end.</p>}
        </>
      )}
    </div>
  );
}
