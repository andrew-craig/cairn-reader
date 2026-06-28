import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { ExploreService } from '../services/explore';
import { ReadService } from '../services/read';
import { sanitizeArticleHtml } from '../utils/sanitize';
import { decodeEntities } from '../utils/decodeEntities';
import FloatingActionBar from '../components/FloatingActionBar';
import './ReadArticle.css';

// An explore article in the reader may carry the user's existing vote when it
// arrives from the Votes screen (a VotedArticleWithType); the Explore feed
// passes a plain Article (voteType undefined).
type ReaderArticle = Article & { voteType?: 'upvote' | 'downvote' };

// Navigation state passed from the Explore feed or the Votes screen. The single
// article is always present; articles/index allow the reader to offer "next".
interface ExploreNavState {
  article?: ReaderArticle;
  articles?: ReaderArticle[];
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

  const [article, setArticle] = useState<ReaderArticle | undefined>(initialArticle);

  // Vote state: seeded from the article's voteType when it arrives from the
  // Votes screen, otherwise null (the Explore feed doesn't prefetch vote status).
  // Optimistic updates take over on the first vote interaction.
  const [vote, setVote] = useState<'upvote' | 'downvote' | null>(
    initialArticle?.voteType ?? null,
  );
  const [isSaved, setIsSaved] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Tracks the article id in scope so async vote callbacks don't mutate state
  // for a different article after a "next" navigation.
  const articleIdRef = useRef<string | undefined>(initialArticle?.id);

  const hasNext = hasList && index + 1 < (articles?.length ?? 0);

  // Reset per-article UI state whenever the displayed article changes. isSaving
  // is reset too so a save left in flight during a "next" navigation doesn't
  // strand the new article's button in the "Saving…" state.
  useEffect(() => {
    if (!article) return;
    articleIdRef.current = article.id;
    setVote(article.voteType ?? null);
    setIsSaved(false);
    setIsSaving(false);
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
  // State writes are guarded by the article id captured at call time so a save
  // that resolves after a "next" navigation can't mark a different article saved.
  const handleSave = useCallback(async () => {
    if (!article || isSaving || isSaved) return;
    const targetId = article.id;
    setIsSaving(true);
    setSaveError(null);
    try {
      await ReadService.addURL({ url: article.url });
      if (articleIdRef.current === targetId) setIsSaved(true);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to save article.';
      if (articleIdRef.current === targetId) {
        // 409 Conflict means it was already saved — reflect that.
        if (msg.includes('409') || msg.toLowerCase().includes('already')) {
          setIsSaved(true);
        } else {
          setSaveError(msg);
        }
      }
    } finally {
      if (articleIdRef.current === targetId) setIsSaving(false);
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
      {saveError && (
        <p className="reader__status reader__error" role="alert">
          {saveError}
        </p>
      )}

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
        ) : (
          <p className="reader__status">No content available for this article.</p>
        )}
      </article>
      <FloatingActionBar
        actions={[
          { icon: 'return', label: 'Back', onClick: () => navigate('/explore') },
          { icon: 'thumbs-up', label: 'Upvote', active: isUp, onClick: () => handleVote('upvote') },
          { icon: 'thumbs-down', label: 'Downvote', active: isDown, onClick: () => handleVote('downvote') },
          { icon: 'next-article', label: 'Next', disabled: !hasNext, onClick: handleNext },
          { icon: 'save', label: 'Save to reading list', active: isSaved, disabled: isSaving || isSaved, onClick: handleSave },
          { icon: 'open-original', label: 'Open original', onClick: handleOpenOriginal },
        ]}
      />
    </div>
  );
}
