import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { ExploreService, type VotedArticleWithType } from '../services/explore';
import './Votes.css';

// Page size: fetch one page of 50 voted articles (offset-paginated, no
// infinite scroll — mirrors mobile's VotesScreen single-page simplicity).
const VOTES_LIMIT = 50;

// A single voted-article row: title, source, vote badge. Clicking navigates to
// the explore reader with the article in router state (ExploreArticle has no
// fetch-by-id fallback, so the article must be passed via state).
interface VoteRowProps {
  voted: VotedArticleWithType;
  onSelect: (voted: VotedArticleWithType) => void;
}

function VoteRow({ voted, onSelect }: VoteRowProps) {
  const isUp = voted.voteType === 'upvote';
  return (
    <li className="vote-row">
      <button
        type="button"
        className="vote-row__button"
        onClick={() => onSelect(voted)}
      >
        <div className="vote-row__text">
          <h2 className="vote-row__title">{voted.title}</h2>
          <p className="vote-row__meta">{voted.author || 'Unknown'}</p>
          {voted.description && (
            <p className="vote-row__excerpt">{voted.description}</p>
          )}
        </div>
        <span
          className={`vote-row__badge${isUp ? ' vote-row__badge--up' : ' vote-row__badge--down'}`}
          aria-label={isUp ? 'Upvoted' : 'Downvoted'}
        >
          {isUp ? '▲' : '▼'}
        </span>
      </button>
    </li>
  );
}

// The votes screen (/you/votes): articles the user has upvoted or downvoted in
// the Explore feed. Grouped into upvotes then downvotes, matching mobile's
// VotesScreen rendering order. Reloads on mount so recent votes are reflected.
export default function Votes() {
  const navigate = useNavigate();
  const [voted, setVoted] = useState<VotedArticleWithType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const articles = await ExploreService.getUserVotedArticles(VOTES_LIMIT, 0);
      setVoted(articles);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load votes.');
    }
  }, []);

  // Reload on every mount so votes cast in the Explore reader appear here.
  useEffect(() => {
    setLoading(true);
    load().finally(() => setLoading(false));
  }, [load]);

  const handleSelect = useCallback(
    (article: VotedArticleWithType) => {
      // ExploreArticle reads location.state.article; no fetch-by-id exists on
      // the explore service, so we must pass the full article via router state.
      navigate(`/explore/${article.id}`, { state: { article } });
    },
    [navigate],
  );

  const upvotes = voted.filter((a) => a.voteType === 'upvote');
  const downvotes = voted.filter((a) => a.voteType === 'downvote');

  return (
    <div className="votes">
      <header className="votes__header">
        <h1 className="votes__title">Votes</h1>
      </header>

      {loading ? (
        <p className="votes__status">Loading votes…</p>
      ) : error ? (
        <div className="votes__status">
          <p className="votes__error">{error}</p>
          <button
            type="button"
            className="votes__refresh"
            onClick={() => {
              setLoading(true);
              load().finally(() => setLoading(false));
            }}
          >
            Try again
          </button>
        </div>
      ) : voted.length === 0 ? (
        <p className="votes__status">No voted articles yet. Vote on recommendations in the Explore feed to see them here.</p>
      ) : (
        <>
          {upvotes.length > 0 && (
            <section>
              <h2 className="votes__section-heading">Upvoted</h2>
              <ul className="votes__list">
                {upvotes.map((a) => (
                  <VoteRow key={a.id} voted={a} onSelect={handleSelect} />
                ))}
              </ul>
            </section>
          )}
          {downvotes.length > 0 && (
            <section>
              <h2 className="votes__section-heading">Downvoted</h2>
              <ul className="votes__list">
                {downvotes.map((a) => (
                  <VoteRow key={a.id} voted={a} onSelect={handleSelect} />
                ))}
              </ul>
            </section>
          )}
        </>
      )}
    </div>
  );
}
