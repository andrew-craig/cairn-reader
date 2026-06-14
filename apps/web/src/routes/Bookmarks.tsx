import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { ReadService } from '../services/read';
import ArticleRow from '../components/ArticleRow';
import './Read.css';

// Bookmarks page size: single-page fetch, no infinite scroll (mirrors
// mobile's BookmarksScreen which uses a 20-item page but the web version
// keeps it simpler with a 50-item ceiling to avoid pagination complexity).
const BOOKMARKS_LIMIT = 50;

// The bookmarks screen (/you/bookmarks): saved articles the user has marked as
// favourite. Same row layout as /read, reloads on mount so changes made in the
// reader are reflected on return (matching mobile's useFocusEffect pattern).
export default function Bookmarks() {
  const navigate = useNavigate();
  const [articles, setArticles] = useState<Article[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const response = await ReadService.listUserContents({
        is_favorite: true,
        limit: BOOKMARKS_LIMIT,
      });
      setArticles(response.contents.map(ReadService.transformToArticle));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load bookmarks.');
    }
  }, []);

  // Reload on every mount so favouriting/unfavouriting in the reader is
  // reflected when the user navigates back here.
  useEffect(() => {
    setLoading(true);
    load().finally(() => setLoading(false));
  }, [load]);

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
        <h1 className="sr-only">Bookmarks</h1>
      </header>

      {loading ? (
        <p className="read__status">Loading bookmarks…</p>
      ) : error ? (
        <div className="read__status">
          <p className="read__error">{error}</p>
          <button
            type="button"
            className="read__refresh"
            onClick={() => {
              setLoading(true);
              load().finally(() => setLoading(false));
            }}
          >
            Try again
          </button>
        </div>
      ) : articles.length === 0 ? (
        <p className="read__status">No bookmarked articles yet. Favourite an article while reading to see it here.</p>
      ) : (
        <ul className="read__list">
          {articles.map((article) => (
            <ArticleRow key={article.id} article={article} onSelect={handleSelect} />
          ))}
        </ul>
      )}
    </div>
  );
}
