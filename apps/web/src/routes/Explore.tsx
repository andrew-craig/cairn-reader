import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { ExploreService } from '../services/explore';
import { decodeEntities } from '../utils/decodeEntities';
import './Explore.css';

// Page size matched to the backend's recommendationPageSize. A page shorter
// than this signals end-of-feed, matching the mobile pattern.
const RECOMMENDATION_PAGE_SIZE = 10;

// Shown-event batching: collect visible article IDs locally and POST in
// batches. Flush triggers: threshold reached, debounce timer, or unmount.
const SHOWN_BATCH_FLUSH_THRESHOLD = 10;
const SHOWN_BATCH_DEBOUNCE_MS = 3000;

// Explore recommendation card: title, source, excerpt, optional image.
// Clicking the card body navigates to the reader.
interface ExploreCardProps {
  article: Article;
  onSelect: (article: Article) => void;
  observeRef: (id: string, node: HTMLLIElement | null) => void;
}

function ExploreCard({ article, onSelect, observeRef }: ExploreCardProps) {
  return (
    <li className="explore-card" ref={(node) => observeRef(article.id, node)}>
      <button
        type="button"
        className="explore-card__button"
        onClick={() => onSelect(article)}
      >
        <div className="explore-card__text">
          <h2 className="explore-card__title">{decodeEntities(article.title)}</h2>
          <p className="explore-card__meta">{article.author || 'Unknown'}</p>
          {article.description && (
            <p className="explore-card__excerpt">{article.description}</p>
          )}
        </div>
        {article.imageUrl && (
          <div className="explore-card__image-frame">
            <img
              className="explore-card__image"
              src={article.imageUrl}
              alt=""
              loading="lazy"
            />
          </div>
        )}
      </button>
    </li>
  );
}

// The Explore feed (/explore): personalised recommendations from the Explore
// Service, with infinite-scroll offset pagination, batched shown-reporting,
// and per-card up/down voting with optimistic UI. Mirrors mobile's ExploreScreen.
export default function Explore() {
  const navigate = useNavigate();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Offset + guards in refs so the IntersectionObserver callback sees current
  // values without re-subscribing on each state update.
  const offsetRef = useRef(0);
  const hasMoreRef = useRef(true);
  const inFlightRef = useRef(false);
  const observerRef = useRef<IntersectionObserver | null>(null);

  // Shown-event batching (mirrors mobile's markShown logic).
  const shownInSessionRef = useRef<Set<string>>(new Set());
  const shownQueueRef = useRef<string[]>([]);
  const flushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // IntersectionObserver entries keyed by article id so we can report shown.
  const cardObserverRef = useRef<IntersectionObserver | null>(null);
  const cardNodeMapRef = useRef<Map<string, Element>>(new Map());

  const flushShown = useCallback(() => {
    if (flushTimerRef.current) {
      clearTimeout(flushTimerRef.current);
      flushTimerRef.current = null;
    }
    if (shownQueueRef.current.length === 0) return;
    const batch = shownQueueRef.current.slice();
    shownQueueRef.current = [];
    void ExploreService.markShown(batch);
  }, []);

  // Flush shown queue on unmount.
  useEffect(() => {
    return () => {
      flushShown();
      if (flushTimerRef.current) clearTimeout(flushTimerRef.current);
      cardObserverRef.current?.disconnect();
    };
  }, [flushShown]);

  // Register/unregister a card node with the visibility observer. Called as a
  // callback ref from each ExploreCard so it runs on mount and unmount.
  const observeCard = useCallback((articleId: string, node: HTMLLIElement | null) => {
    if (node) {
      cardNodeMapRef.current.set(articleId, node);
      cardObserverRef.current?.observe(node);
    } else {
      const old = cardNodeMapRef.current.get(articleId);
      if (old) {
        cardObserverRef.current?.unobserve(old);
        cardNodeMapRef.current.delete(articleId);
      }
    }
  }, []);

  // Set up the card-visibility observer once on mount.
  useEffect(() => {
    cardObserverRef.current = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (!entry.isIntersecting) continue;
          // Find the article id for this DOM element.
          for (const [id, el] of cardNodeMapRef.current.entries()) {
            if (el !== entry.target) continue;
            if (!shownInSessionRef.current.has(id)) {
              shownInSessionRef.current.add(id);
              shownQueueRef.current.push(id);
            }
            break;
          }
        }
        if (shownQueueRef.current.length >= SHOWN_BATCH_FLUSH_THRESHOLD) {
          flushShown();
        } else if (shownQueueRef.current.length > 0 && !flushTimerRef.current) {
          flushTimerRef.current = setTimeout(flushShown, SHOWN_BATCH_DEBOUNCE_MS);
        }
      },
      { threshold: 0.5 },
    );

    // Observe any nodes registered before the observer was ready.
    for (const el of cardNodeMapRef.current.values()) {
      cardObserverRef.current.observe(el);
    }

    return () => {
      cardObserverRef.current?.disconnect();
      cardObserverRef.current = null;
    };
  }, [flushShown]);

  const fetchPage = useCallback(async (reset: boolean) => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setError(null);
    try {
      const offset = reset ? 0 : offsetRef.current;
      const page = await ExploreService.getRecommendations(offset);
      offsetRef.current = offset + page.length;
      const reachedEnd = page.length < RECOMMENDATION_PAGE_SIZE;
      hasMoreRef.current = !reachedEnd;
      setHasMore(!reachedEnd);

      if (reset) {
        setArticles(page);
        shownInSessionRef.current = new Set();
        shownQueueRef.current = [];
      } else {
        setArticles((prev) => {
          // De-duplicate in case the eligible pool shrank mid-scroll.
          const existingIds = new Set(prev.map((a) => a.id));
          const unique = page.filter((a) => !existingIds.has(a.id));
          return [...prev, ...unique];
        });
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load recommendations.');
    } finally {
      inFlightRef.current = false;
    }
  }, []);

  // Initial load on mount (and remount).
  useEffect(() => {
    setLoading(true);
    fetchPage(true).finally(() => setLoading(false));
  }, [fetchPage]);

  const handleRefresh = useCallback(() => {
    if (inFlightRef.current) return;
    void fetchPage(true);
  }, [fetchPage]);

  const handleLoadMore = useCallback(() => {
    if (inFlightRef.current || !hasMoreRef.current) return;
    setLoadingMore(true);
    fetchPage(false).finally(() => setLoadingMore(false));
  }, [fetchPage]);

  // Sentinel IntersectionObserver for infinite scroll. Same callback-ref pattern
  // as Read.tsx: re-binds on each article append so the observer re-fires if the
  // sentinel is still visible after the new page is rendered.
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
    // `articles` intentionally included so the sentinel re-binds after each
    // page append, matching Read.tsx's pattern (see comment there).
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [handleLoadMore, articles],
  );

  // Navigate to the article reader, passing the article via router state so the
  // reader can display content without a refetch (same pattern as Read.tsx).
  const handleSelect = useCallback(
    (article: Article) => {
      const index = articles.findIndex((a) => a.id === article.id);
      navigate(`/explore/${article.id}`, { state: { article, articles, index } });
    },
    [articles, navigate],
  );

  return (
    <div className="explore">
      <h1 className="explore__title sr-only">Explore</h1>

      {loading ? (
        <p className="explore__status">Loading recommendations…</p>
      ) : error ? (
        <div className="explore__status">
          <p className="explore__error">{error}</p>
          <button type="button" className="explore__refresh" onClick={handleRefresh}>
            Try again
          </button>
        </div>
      ) : articles.length === 0 ? (
        <p className="explore__status">No recommendations available.</p>
      ) : (
        <>
          <ul className="explore__list">
            {articles.map((article) => (
              <ExploreCard
                key={article.id}
                article={article}
                onSelect={handleSelect}
                observeRef={observeCard}
              />
            ))}
          </ul>
          <div ref={sentinelRef} className="explore__sentinel" aria-hidden="true" />
          {loadingMore && <p className="explore__status">Loading more…</p>}
          {!hasMore && <p className="explore__status">You've reached the end.</p>}
        </>
      )}
    </div>
  );
}
