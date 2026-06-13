import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { ExploreService } from '../services/explore';
import { ReadService } from '../services/read';
import { sanitizeArticleHtml } from '../utils/sanitize';
import './ReadArticle.css';

// Navigation state passed from the Explore feed. The single article is always
// present; articles/index allow the reader to offer "next" without a refetch.
interface ExploreNavState {
  article?: Article;
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

// The Explore article reader (/explore/:id): sanitized full content from the
// explore feed, with voting, mark-as-read, and save-to-reading-list. Mirrors
// ReadArticle.tsx in layout and patterns; adapted for the explore action set.
export default function ExploreArticle() {
  useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const navState = (location.state as ExploreNavState | null) ?? null;

  // Article from nav state is the source of truth; no per-article fetch
  // endpoint exists on the explore service, so on direct load we show an error.
  const initialArticle = navState?.article;
  const articles = navState?.articles;
  const index = navState?.index ?? -1;
  const hasList = Array.isArray(articles) && index >= 0;

  const [article, setArticle] = useState<Article | undefined>(initialArticle);

  // Vote state: seeded to null (unknown) — we don't prefetch vote status from
  // the feed. On first vote interaction the optimistic update takes over.
  const [vote, setVote] = useState<'upvote' | 'downvote' | null>(null);
  const [isSaved, setIsSaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Tracks the article id in scope so async vote callbacks don't mutate state
  // for a different article after a "next" navigation.
  const articleIdRef = useRef<string | undefined>(initialArticle?.id);

  const hasNext = hasList && index + 1 < (articles?.length ?? 0);

  // Reset per-article UI state whenever the displayed article changes.
  useEffect(() => {
    if (!article) return;
    articleIdRef.current = article.id;
    setVote(null);
    setIsSaved(false);
    setSaveError(null);
  }, [article]);

  // Mark article as read on open. Best-effort — fire and forget.
  const articleId = article?.id;
  useEffect(() => {
    if (!articleId) return;
    ExploreService.markAsRead(articleId).catch((err) =>
      console.error('Failed to mark explore article as read:', err),
    );
  }, [articleId]);

  const sanitizedHtml = useMemo(
    () => (article?.content ? sanitizeArticleHtml(article.content) : ''),
    [article?.content],
  );

  // Navigate to the next article in the feed list, swapping in place (replace).
  const handleNext = useCallback(() => {
    if (!articles || !hasNext) return;
    const nextIndex = index + 1;
    const next = articles[nextIndex];
    navigate(`/explore/${next.id}`, {
      state: { article: next, articles, index: nextIndex },
      replace: true,
    });
    setArticle(next);
  }, [articles, hasNext, index, navigate]);

  // Optimistic voting: flip local state immediately; revert on API failure.
  const handleVote = useCallback(
    (type: 'upvote' | 'downvote') => {
      if (!article) return;
      const targetId = article.id;
      const previous = vote;
      const next = vote === type ? null : type;
      setVote(next);

      const action =
        next === null
          ? ExploreService.removeVote(targetId)
          : ExploreService.voteOnArticle(targetId, type);

      action.catch((err) => {
        console.error('Failed to update vote:', err);
        if (articleIdRef.current === targetId) setVote(previous);
      });
    },
    [article, vote],
  );

  // Save to reading list via ReadService.addURL. One-way (no unsave here).
  const handleSave = useCallback(async () => {
    if (!article || isSaving || isSaved) return;
    setIsSaving(true);
    setSaveError(null);
    try {
      await ReadService.addURL({ url: article.url });
      setIsSaved(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to save article.';
      // 409 Conflict means it was already saved — reflect that.
      if (msg.includes('409') || msg.toLowerCase().includes('already')) {
        setIsSaved(true);
      } else {
        setSaveError(msg);
      }
    } finally {
      setIsSaving(false);
    }
  }, [article, isSaving, isSaved]);

  const handleOpenOriginal = useCallback(() => {
    if (article?.url) window.open(article.url, '_blank', 'noopener,noreferrer');
  }, [article?.url]);

  if (!article) {
    return (
      <div className="reader__status">
        <p className="reader__error">
          Article not found. Open it from the{' '}
          <button type="button" className="reader__action" onClick={() => navigate('/explore')}>
            Explore feed
          </button>
          .
        </p>
      </div>
    );
  }

  const isUp = vote === 'upvote';
  const isDown = vote === 'downvote';
  const publishedDate = formatPublishedDate(article.publishedDate);

  return (
    <div className="reader">
      <div className="reader__toolbar">
        <button type="button" className="reader__action" onClick={() => navigate('/explore')}>
          Back
        </button>
        <div className="reader__toolbar-group">
          {hasNext && (
            <button type="button" className="reader__action" onClick={handleNext}>
              Next
            </button>
          )}
          <button
            type="button"
            className={`reader__action${isUp ? ' reader__action--active' : ''}`}
            onClick={() => handleVote('upvote')}
            aria-pressed={isUp}
          >
            {isUp ? '▲ Upvoted' : '▲ Upvote'}
          </button>
          <button
            type="button"
            className={`reader__action${isDown ? ' reader__action--active' : ''}`}
            onClick={() => handleVote('downvote')}
            aria-pressed={isDown}
          >
            {isDown ? '▼ Downvoted' : '▼ Downvote'}
          </button>
          <button
            type="button"
            className={`reader__action${isSaved ? ' reader__action--active' : ''}`}
            onClick={handleSave}
            disabled={isSaving || isSaved}
          >
            {isSaved ? '✓ Saved' : isSaving ? 'Saving…' : 'Save to reading list'}
          </button>
          <button type="button" className="reader__action" onClick={handleOpenOriginal}>
            Open original
          </button>
        </div>
      </div>

      {saveError && (
        <p className="reader__status reader__error" role="alert">
          {saveError}
        </p>
      )}

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
