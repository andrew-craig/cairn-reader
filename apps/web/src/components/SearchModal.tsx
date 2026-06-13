import { useCallback, useEffect, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import type { Article } from '@cairn/shared';
import { ReadService } from '../services/read';
import ArticleRow from './ArticleRow';
import './SearchModal.css';

interface SearchModalProps {
  onClose: () => void;
}

// Search overlay for the reading list. Debounces the query and calls
// ReadService.searchUserContents. Results render in the same ArticleRow
// layout as the main reading list.
export default function SearchModal({ onClose }: SearchModalProps) {
  const navigate = useNavigate();
  const inputRef = useRef<HTMLInputElement>(null);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<Article[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searched, setSearched] = useState(false);

  // Focus the input on mount.
  useEffect(() => {
    inputRef.current?.focus();
  }, []);

  // Escape key closes the modal.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [onClose]);

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
        const response = await ReadService.searchUserContents({ q: trimmed });
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
  }, [query]);

  const handleSelect = useCallback(
    (article: Article) => {
      const index = results.findIndex((a) => a.id === article.id);
      navigate(`/read/${article.id}`, { state: { articles: results, index } });
      onClose();
    },
    [results, navigate, onClose],
  );

  const handleClear = () => {
    setQuery('');
    setResults([]);
    setSearched(false);
    setError(null);
    inputRef.current?.focus();
  };

  return (
    <div className="search-overlay" role="dialog" aria-modal="true" aria-label="Search your reading list">
      <div className="search-modal">
        <div className="search-modal__header">
          <div className="search-modal__input-wrap">
            <input
              ref={inputRef}
              type="search"
              className="search-modal__input"
              placeholder="Search your reading list"
              aria-label="Search your reading list"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              autoComplete="off"
              spellCheck={false}
            />
            {query && (
              <button
                type="button"
                className="search-modal__clear"
                onClick={handleClear}
                aria-label="Clear search"
              >
                ✕
              </button>
            )}
          </div>
          <button
            type="button"
            className="search-modal__close"
            onClick={onClose}
            aria-label="Close search"
          >
            Cancel
          </button>
        </div>

        <div className="search-modal__body">
          {loading ? (
            <p className="search-modal__status">Searching…</p>
          ) : error ? (
            <p className="search-modal__status search-modal__status--error">{error}</p>
          ) : searched && results.length === 0 ? (
            <p className="search-modal__status">No results.</p>
          ) : results.length > 0 ? (
            <ul className="search-modal__results">
              {results.map((article) => (
                <ArticleRow key={article.id} article={article} onSelect={handleSelect} />
              ))}
            </ul>
          ) : (
            <p className="search-modal__status">Type to search your saved articles.</p>
          )}
        </div>
      </div>
      <div
        className="search-overlay__backdrop"
        onClick={onClose}
        aria-hidden="true"
      />
    </div>
  );
}
