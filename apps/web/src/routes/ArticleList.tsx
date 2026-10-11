import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Article, ContentList } from '@cairn/shared';
import { PAGE_SIZE, ReadService } from '../services/read';
import ArticleRow from '../components/ArticleRow';
import AddLinkModal from '../components/AddLinkModal';
import SearchModal from '../components/SearchModal';
import './Read.css';

const COPY: Record<ContentList, { search: string; loading: string; empty: string; error: string }> = {
  reads: {
    search: 'Search your reads',
    loading: 'Loading your reads…',
    empty: 'No saved articles yet.',
    error: 'Failed to load your reads.',
  },
  feed: {
    search: 'Search your feed',
    loading: 'Loading your feed…',
    empty: 'Your feed is empty. Choose Feed for an RSS source in You → RSS.',
    error: 'Failed to load your feed.',
  },
};

// One of the two lists (/feed, /reads): the user's content for that list,
// cursor-paginated from the Read Service with infinite scroll. Mirrors mobile's
// FeedScreen/ReadsScreen — it reloads from the first page on (re)mount, so
// archive/favorite changes made in the reader are reflected on return (mobile
// uses useFocusEffect for the same). Add link is Reads-only: pages always save
// to Reads, and the modal asks Feed or Reads when it detects an RSS feed.
// Callers key this component by `list` so switching lists remounts it.
export default function ArticleList({ list }: { list: ContentList }) {
  const copy = COPY[list];
  const navigate = useNavigate();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true); // first page
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [showAddLink, setShowAddLink] = useState(false);
  const [closingAddLink, setClosingAddLink] = useState(false);
  const [showSearch, setShowSearch] = useState(false);
  const [closingSearch, setClosingSearch] = useState(false);

  const handleCloseAddLink = useCallback(() => {
    setClosingAddLink(true);
    setTimeout(() => {
      setShowAddLink(false);
      setClosingAddLink(false);
    }, 150);
  }, []);

  const handleCloseSearch = useCallback(() => {
    setClosingSearch(true);
    setTimeout(() => {
      setShowSearch(false);
      setClosingSearch(false);
    }, 150);
  }, []);

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
        list,
        limit: PAGE_SIZE,
        cursor: reset ? undefined : cursorRef.current || undefined,
      });
      const page = response.contents.map(ReadService.transformToArticle);
      setArticles((prev) => (reset ? page : [...prev, ...page]));
      cursorRef.current = response.cursor;
      hasMoreRef.current = response.has_more;
      setHasMore(response.has_more);
    } catch (e) {
      setError(e instanceof Error ? e.message : copy.error);
    } finally {
      inFlightRef.current = false;
    }
  }, [list, copy.error]);

  // Initial load on mount (and remount when returning from the reader).
  useEffect(() => {
    setLoading(true);
    fetchPage(true).finally(() => setLoading(false));
  }, [fetchPage]);

  const handleRefresh = useCallback(() => {
    if (inFlightRef.current) return;
    // Keep the current list visible; fetchPage(true) replaces it with the first
    // page once it loads, avoiding an empty-state flash mid-refresh.
    void fetchPage(true);
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
  // `articles` is a dep so each page append rebinds the observer: a fresh
  // observer reports the current intersection state, re-arming the check when
  // the sentinel stays visible (short content / tall viewport) — IO otherwise
  // only fires on intersection *changes*, which would stall the scroll.
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
    // `articles` is intentional even though the body doesn't read it: changing it
    // must rebind the observer (see comment above). eslint can't see that intent.
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [handleLoadMore, articles],
  );

  const handleSelect = useCallback(
    (article: Article) => {
      const index = articles.findIndex((a) => a.id === article.id);
      // Pass the list + index so the reader can offer prev/next without a refetch.
      navigate(`/${list}/${article.id}`, { state: { articles, index } });
    },
    [articles, list, navigate],
  );

  return (
    <div className="read">
      <header className="read__header">
        <button
          type="button"
          className="read__search-bar"
          onClick={() => setShowSearch(true)}
          aria-label={copy.search}
        >
          {copy.search}
        </button>
        {list === 'reads' && (
          <div className="read__actions">
            <button
              type="button"
              className="read__add-link"
              onClick={() => setShowAddLink(true)}
            >
              Add link
            </button>
          </div>
        )}
      </header>

      {showAddLink && (
        <AddLinkModal
          onClose={handleCloseAddLink}
          onSuccess={handleRefresh}
          closing={closingAddLink}
        />
      )}

      {showSearch && <SearchModal list={list} onClose={handleCloseSearch} closing={closingSearch} />}

      {loading ? (
        <p className="read__status">{copy.loading}</p>
      ) : error ? (
        <div className="read__status">
          <p className="read__error">{error}</p>
          <button type="button" className="read__refresh" onClick={handleRefresh}>
            Try again
          </button>
        </div>
      ) : articles.length === 0 ? (
        <p className="read__status">{copy.empty}</p>
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
