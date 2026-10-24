import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Article, ContentList } from '@cairn/shared';
import { ReadService } from '../services/read';
import ArticleRow from './ArticleRow';

interface SearchResultsProps {
  list: ContentList;
  query: string;
}

// Search results for one list (Feed or Reads), shown in place of the list while
// the header search field holds a query. Debounces the query and calls
// ReadService.searchUserContents. Results render in the same ArticleRow layout
// as the list itself.
export default function SearchResults({ list, query }: SearchResultsProps) {
  const navigate = useNavigate();
  const [results, setResults] = useState<Article[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searched, setSearched] = useState(false);

  // Debounced search: fires 300 ms after the user stops typing.
  useEffect(() => {
    const trimmed = query.trim();
    if (!trimmed) {
      setResults([]);
      setSearched(false);
      setError(null);
      return;
    }

    let cancelled = false;
    const timerId = setTimeout(async () => {
      setLoading(true);
      setError(null);
      try {
        const response = await ReadService.searchUserContents({ q: trimmed, list });
        if (!cancelled) {
          setResults(response.contents.map(ReadService.transformToArticle));
          setSearched(true);
        }
      } catch (e) {
        if (!cancelled) {
          setError(e instanceof Error ? e.message : 'Search failed.');
          setSearched(true);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    }, 300);

    return () => {
      cancelled = true;
      clearTimeout(timerId);
    };
  }, [query, list]);

  const handleSelect = useCallback(
    (article: Article) => {
      const index = results.findIndex((a) => a.id === article.id);
      navigate(`/${list}/${article.id}`, { state: { articles: results, index } });
    },
    [results, list, navigate],
  );

  if (loading || !searched) return <p className="read__status">Searching…</p>;
  if (error) return <p className="read__status read__error">{error}</p>;
  if (results.length === 0) return <p className="read__status">No results.</p>;
  return (
    <ul className="read__list">
      {results.map((article) => (
        <ArticleRow key={article.id} article={article} onSelect={handleSelect} />
      ))}
    </ul>
  );
}
