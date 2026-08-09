import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { throttle, type Article } from '@cairn/shared';
import { ReadService } from '../services/read';
import { sanitizeArticleHtml } from '../utils/sanitize';
import { decodeEntities } from '../utils/decodeEntities';
import { formatPublishedDate } from '../utils/helpers';
import FloatingActionBar from '../components/FloatingActionBar';
import './ReadArticle.css';

// Scrolled this far through the article (fraction of scrollable height) marks it
// completed. Mirrors mobile's COMPLETED_PROGRESS_THRESHOLD.
const COMPLETED_THRESHOLD = 0.95;
// Throttle persisting scroll position while the user is actively scrolling,
// so progress survives the app being closed mid-read instead of only being
// saved on navigation.
const SCROLL_SAVE_THROTTLE_MS = 1000;

// Navigation state passed from the reading list (Read.tsx): the list and the
// position of the opened article, so the reader can offer prev/next without a
// refetch. Absent on direct URL load / hard refresh.
interface ReaderNavState {
  articles?: Article[];
  index?: number;
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

  // The article from nav state seeds an instant placeholder (title/metadata),
  // but it is a list *summary* that omits the article body — so we always fetch
  // the full detail by id below. Re-derive on id change so prev/next swaps it.
  const initialArticle = hasList ? articles[index] : undefined;
  const [article, setArticle] = useState<Article | undefined>(initialArticle);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Mutable UI state tracked separately from the article object so async updates
  // are scoped to the displayed article. Seeded from the article and resynced
  // when the displayed article changes (see effect below). Completion (isRead)
  // is tracked via hasMarkedCompletedRef rather than state because nothing
  // renders it — only the favorite toggle is reflected in the toolbar.
  const [isFavorite, setIsFavorite] = useState(initialArticle?.isFavorite ?? false);

  const scrollRef = useRef<HTMLDivElement>(null);
  // Scroll fraction in [0,1] — offsetY / contentHeight, matching mobile and the
  // backend's NUMERIC(5,4) scroll_position column.
  const scrollFractionRef = useRef(0);
  const hasScrolledRef = useRef(false);
  const hasMarkedCompletedRef = useRef(false);
  const throttledSaveRef = useRef(
    throttle((contentId: string, fraction: number) => {
      ReadService.updateUserContent(contentId, { scroll_position: fraction })
        // Clear the dirty flag so the unmount/switch cleanup doesn't re-send the
        // same position; a later scroll sets it true again.
        .then(() => {
          hasScrolledRef.current = false;
        })
        .catch((err) => console.error('Failed to save scroll position:', err));
    }, SCROLL_SAVE_THROTTLE_MS),
  );
  // Tracks the currently displayed article id so async callbacks can detect a
  // swap (prev/next) and avoid mutating UI state for a different article.
  const articleIdRef = useRef<string | undefined>(initialArticle?.id);

  const nextIndex = hasList ? index + 1 : -1;
  const hasNext = hasList && nextIndex < (articles?.length ?? 0);

  // Fetch the full article detail (including the body) by id. The list endpoint
  // returns only a summary without cleaned_html, so the nav-state article seeds
  // the placeholder while this loads the content; a direct load fetches it
  // outright. Runs on id change so prev/next loads the swapped article's body.
  useEffect(() => {
    if (!id) return;
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
  }, [id]);

  // Reset per-article progress refs and seed the mutable UI state from the
  // article's saved state whenever the displayed article changes (open,
  // prev/next, or fetch resolve).
  useEffect(() => {
    if (!article) return;
    articleIdRef.current = article.id;
    scrollFractionRef.current = article.scrollPosition ?? 0;
    hasScrolledRef.current = false;
    hasMarkedCompletedRef.current = article.isRead;
    setIsFavorite(article.isFavorite);
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
    // Always set scrollTop (0 for unread/new articles) so navigating to an
    // article doesn't inherit the previous one's scroll position.
    const fraction = article.scrollPosition ?? 0;
    el.scrollTop = fraction * (el.scrollHeight - el.clientHeight);
  }, [sanitizedHtml, article]);

  const markCompleted = useCallback((contentId: string) => {
    if (hasMarkedCompletedRef.current) return;
    hasMarkedCompletedRef.current = true;
    ReadService.updateUserContent(contentId, { status: 'completed' }).catch((err) =>
      console.error('Failed to mark article completed:', err),
    );
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

    // Throttled persistence so we PATCH at most once per second during a
    // scroll, guaranteeing progress is saved even if the app closes mid-read.
    throttledSaveRef.current(articleId, fraction);
  }, [articleId, markCompleted]);

  // On unmount (or article change), flush the final scroll position if the user
  // scrolled, cancelling any pending throttled save. Mirrors mobile's cleanup.
  useEffect(() => {
    const throttledSave = throttledSaveRef.current;
    return () => {
      throttledSave.cancel();
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
      // Swap the displayed article in place (route stays on the reader). The
      // scroll-restore effect resets scrollTop for the new article; scrolling
      // here synchronously would fire handleScroll against the old article id
      // and flush a 0 scroll position, wiping its saved progress.
      setArticle(target);
    },
    [articles, navigate],
  );

  const handleToggleFavorite = useCallback(() => {
    if (!article) return;
    const targetId = article.id;
    const newIsFavorite = !isFavorite;
    setIsFavorite(newIsFavorite);
    ReadService.updateUserContent(targetId, { is_favorite: newIsFavorite }).catch(
      (err) => {
        console.error('Failed to toggle favorite:', err);
        // Roll back the optimistic icon update on failure, but only if the same
        // article is still displayed — otherwise we'd flip the wrong article.
        if (articleIdRef.current === targetId) setIsFavorite(!newIsFavorite);
      },
    );
  }, [article, isFavorite]);

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

  // Full-screen states only when there is nothing to display yet (direct load).
  // With a nav-state placeholder we keep the title visible and surface load /
  // error states inside the body instead.
  if (loading && !article) {
    return <p className="reader__status">Loading article…</p>;
  }
  if (!article) {
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
      <article className="reader__article">
        <header className="reader__header">
          <h1 className="reader__title">{decodeEntities(article.title)}</h1>
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
        ) : loading ? (
          <p className="reader__status">Loading article…</p>
        ) : (
          <p className="reader__status">{error ?? 'No content available for this article.'}</p>
        )}
      </article>
      <FloatingActionBar
        actions={[
          { icon: 'return', label: 'Back', onClick: () => navigate('/read') },
          { icon: 'bookmark', label: 'Favorite', active: isFavorite, onClick: handleToggleFavorite },
          { icon: 'archive', label: 'Archive', onClick: handleArchive },
          { icon: 'next-article', label: 'Next', disabled: !hasNext, onClick: () => goToAdjacent(nextIndex) },
          { icon: 'delete', label: 'Delete', onClick: handleDelete },
          { icon: 'open-original', label: 'Open original', onClick: handleOpenOriginal },
        ]}
      />
    </div>
  );
}
