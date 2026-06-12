import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { ReadService } from '../services/read';
import { sanitizeArticleHtml } from '../utils/sanitize';
import './ReadArticle.css';

// Scrolled this far through the article (fraction of scrollable height) marks it
// completed. Mirrors mobile's COMPLETED_PROGRESS_THRESHOLD.
const COMPLETED_THRESHOLD = 0.95;
// Debounce persisting scroll position while the user is actively scrolling.
const SCROLL_SAVE_DEBOUNCE_MS = 500;

// Navigation state passed from the reading list (Read.tsx): the list and the
// position of the opened article, so the reader can offer prev/next without a
// refetch. Absent on direct URL load / hard refresh.
interface ReaderNavState {
  articles?: Article[];
  index?: number;
}

function formatPublishedDate(value?: string): string | null {
  if (!value) return null;
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

// The article reader (/read/:id): full sanitized content, scroll-progress
// persistence and reader actions. Mirrors mobile's ReadArticleDetailScreen,
// adapted to DOM scroll events and react-router navigation.
export default function ReadArticle() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const navState = (location.state as ReaderNavState | null) ?? null;

  const articles = navState?.articles;
  const index = navState?.index ?? -1;
  const hasList = Array.isArray(articles) && index >= 0;

  // The article from nav state is the source of truth when present; otherwise we
  // fetch it (direct load). Re-derive on id change so prev/next swaps the article.
  const initialArticle = hasList ? articles[index] : undefined;
  const [article, setArticle] = useState<Article | undefined>(initialArticle);
  const [loading, setLoading] = useState(!initialArticle);
  const [error, setError] = useState<string | null>(null);

  const scrollRef = useRef<HTMLDivElement>(null);
  // Scroll fraction in [0,1] — offsetY / contentHeight, matching mobile and the
  // backend's NUMERIC(5,4) scroll_position column (commit c67a17d).
  const scrollFractionRef = useRef(0);
  const hasScrolledRef = useRef(false);
  const hasMarkedCompletedRef = useRef(false);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const prevIndex = hasList ? index - 1 : -1;
  const nextIndex = hasList ? index + 1 : -1;
  const hasPrev = hasList && prevIndex >= 0;
  const hasNext = hasList && nextIndex < (articles?.length ?? 0);

  // When opened without nav state, fetch the single article (paging the list,
  // since the backend has no per-user get-by-id route — see ReadService).
  useEffect(() => {
    if (initialArticle || !id) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    ReadService.getUserContent(id)
      .then((fetched) => {
        if (!cancelled) setArticle(fetched);
      })
      .catch((e) => {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Failed to load this article.');
        }
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id, initialArticle]);

  // Reset per-article progress refs and seed them from the article's saved state
  // whenever the displayed article changes (open, prev/next, or fetch resolve).
  useEffect(() => {
    if (!article) return;
    scrollFractionRef.current = article.scrollPosition ?? 0;
    hasScrolledRef.current = false;
    hasMarkedCompletedRef.current = article.isRead;
  }, [article]);

  // On open, mark as reading if not already read. Mirrors mobile.
  const articleId = article?.id;
  const articleIsRead = article?.isRead ?? false;
  useEffect(() => {
    if (!articleId || articleIsRead) return;
    ReadService.updateUserContent(articleId, { status: 'reading' }).catch((err) =>
      console.error('Failed to mark article reading:', err),
    );
  }, [articleId, articleIsRead]);

  const sanitizedHtml = useMemo(
    () => (article?.content ? sanitizeArticleHtml(article.content) : ''),
    [article?.content],
  );

  // Restore the saved scroll position once the sanitized content is in the DOM.
  // scrollTop = fraction * (scrollHeight - clientHeight), the inverse of how the
  // fraction is computed on scroll.
  useEffect(() => {
    const el = scrollRef.current;
    if (!el || !article) return;
    const fraction = article.scrollPosition ?? 0;
    if (fraction > 0) {
      el.scrollTop = fraction * (el.scrollHeight - el.clientHeight);
    }
  }, [sanitizedHtml, article]);

  const markCompleted = useCallback((contentId: string) => {
    if (hasMarkedCompletedRef.current) return;
    hasMarkedCompletedRef.current = true;
    ReadService.updateUserContent(contentId, { status: 'completed' }).catch((err) =>
      console.error('Failed to mark article completed:', err),
    );
    setArticle((prev) => (prev ? { ...prev, isRead: true } : prev));
  }, []);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el || !articleId) return;
    const scrollable = el.scrollHeight - el.clientHeight;
    const fraction = scrollable > 0 ? el.scrollTop / scrollable : 0;
    scrollFractionRef.current = fraction;
    hasScrolledRef.current = true;

    if (!hasMarkedCompletedRef.current && fraction >= COMPLETED_THRESHOLD) {
      markCompleted(articleId);
    }

    // Debounced persistence so we PATCH at most every ~500ms during a scroll.
    if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
    saveTimerRef.current = setTimeout(() => {
      ReadService.updateUserContent(articleId, {
        scroll_position: scrollFractionRef.current,
      }).catch((err) => console.error('Failed to save scroll position:', err));
    }, SCROLL_SAVE_DEBOUNCE_MS);
  }, [articleId, markCompleted]);

  // On unmount (or article change), flush the final scroll position if the user
  // scrolled, cancelling any pending debounced save. Mirrors mobile's cleanup.
  useEffect(() => {
    return () => {
      if (saveTimerRef.current) clearTimeout(saveTimerRef.current);
      if (!hasScrolledRef.current || !articleId) return;
      ReadService.updateUserContent(articleId, {
        scroll_position: scrollFractionRef.current,
      }).catch((err) => console.error('Failed to save scroll position:', err));
    };
  }, [articleId]);

  const goToAdjacent = useCallback(
    (targetIndex: number) => {
      if (!articles) return;
      const target = articles[targetIndex];
      navigate(`/read/${target.id}`, {
        state: { articles, index: targetIndex },
        replace: true,
      });
      // Swap the displayed article in place (route stays on the reader).
      setArticle(target);
      scrollRef.current?.scrollTo({ top: 0 });
    },
    [articles, navigate],
  );

  const handleToggleFavorite = useCallback(() => {
    if (!article) return;
    const newIsFavorite = !article.isFavorite;
    setArticle({ ...article, isFavorite: newIsFavorite });
    ReadService.updateUserContent(article.id, { is_favorite: newIsFavorite }).catch(
      (err) => {
        console.error('Failed to toggle favorite:', err);
        // Roll back the optimistic icon update on failure.
        setArticle((prev) => (prev ? { ...prev, isFavorite: !newIsFavorite } : prev));
      },
    );
  }, [article]);

  const handleArchive = useCallback(async () => {
    if (!article) return;
    try {
      await ReadService.updateUserContent(article.id, { status: 'archived' });
      navigate('/read');
    } catch (err) {
      console.error('Failed to archive article:', err);
    }
  }, [article, navigate]);

  const handleDelete = useCallback(async () => {
    if (!article) return;
    try {
      await ReadService.deleteUserContent(article.id);
      navigate('/read');
    } catch (err) {
      console.error('Failed to delete article:', err);
    }
  }, [article, navigate]);

  const handleOpenOriginal = useCallback(() => {
    if (article?.url) {
      window.open(article.url, '_blank', 'noopener,noreferrer');
    }
  }, [article?.url]);

  if (loading) {
    return <p className="reader__status">Loading article…</p>;
  }
  if (error || !article) {
    return (
      <div className="reader__status">
        <p className="reader__error">{error ?? 'Article not found.'}</p>
        <button type="button" className="reader__action" onClick={() => navigate('/read')}>
          Back to reading list
        </button>
      </div>
    );
  }

  const publishedDate = formatPublishedDate(article.publishedDate);

  return (
    <div className="reader" ref={scrollRef} onScroll={handleScroll}>
      <div className="reader__toolbar">
        <button type="button" className="reader__action" onClick={() => navigate('/read')}>
          Back
        </button>
        <div className="reader__toolbar-group">
          <button
            type="button"
            className="reader__action"
            onClick={() => goToAdjacent(prevIndex)}
            disabled={!hasPrev}
          >
            Previous
          </button>
          <button
            type="button"
            className="reader__action"
            onClick={() => goToAdjacent(nextIndex)}
            disabled={!hasNext}
          >
            Next
          </button>
          <button
            type="button"
            className={`reader__action${article.isFavorite ? ' reader__action--active' : ''}`}
            onClick={handleToggleFavorite}
            aria-pressed={article.isFavorite}
          >
            {article.isFavorite ? '★ Favorited' : '☆ Favorite'}
          </button>
          <button type="button" className="reader__action" onClick={handleArchive}>
            Archive
          </button>
          <button type="button" className="reader__action" onClick={handleDelete}>
            Delete
          </button>
          <button type="button" className="reader__action" onClick={handleOpenOriginal}>
            Open original
          </button>
        </div>
      </div>

      <article className="reader__article">
        <header className="reader__header">
          <h1 className="reader__title">{article.title}</h1>
          <p className="reader__meta">
            {article.author && <span>{article.author}</span>}
            {publishedDate && <span>{publishedDate}</span>}
            {article.readingTime && <span>{article.readingTime} min read</span>}
          </p>
        </header>
        {sanitizedHtml ? (
          <div
            className="reader__body"
            // Content is sanitized with DOMPurify (utils/sanitize) before injection.
            dangerouslySetInnerHTML={{ __html: sanitizedHtml }}
          />
        ) : (
          <p className="reader__status">No content available for this article.</p>
        )}
      </article>
    </div>
  );
}
