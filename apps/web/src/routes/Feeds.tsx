import { useCallback, useEffect, useState } from 'react';
import type { UnifiedSubscription } from '@cairn/shared';
import { ReadService } from '../services/read';
import { decodeEntities } from '../utils/decodeEntities';
import './Read.css';
import './Newsletters.css';

// Feeds screen (/you/feeds): the user's RSS feed subscriptions.
// Mirrors mobile's FeedsScreen + SubscriptionListScreen — filters to non-email
// types, shows title/URL, and offers per-row unsubscribe (optimistic removal).
export default function Feeds() {
  const [subscriptions, setSubscriptions] = useState<UnifiedSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  // Track per-row unsubscribe in-progress to prevent double-clicks.
  const [unsubscribing, setUnsubscribing] = useState<Set<string>>(new Set());

  const load = useCallback(async () => {
    setError(null);
    try {
      const response = await ReadService.listAllSubscriptions();
      setSubscriptions(response.subscriptions.filter((s) => s.type !== 'email'));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load feeds.');
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    load().finally(() => setLoading(false));
  }, [load]);

  const handleUnsubscribe = useCallback(async (sub: UnifiedSubscription) => {
    if (!sub.rss_data?.feed_id) return;
    if (!window.confirm(`Unsubscribe from ${sub.title}?`)) return;
    const feedId = sub.rss_data.feed_id;

    // Optimistic: remove immediately, restore on error.
    setSubscriptions((prev) => prev.filter((s) => s.id !== sub.id));
    setUnsubscribing((prev) => new Set(prev).add(sub.id));
    try {
      await ReadService.unsubscribeFromRSSFeed(feedId);
    } catch {
      // Restore the removed item in its original position.
      setSubscriptions((prev) => {
        const next = [...prev];
        next.push(sub);
        return next;
      });
    } finally {
      setUnsubscribing((prev) => {
        const next = new Set(prev);
        next.delete(sub.id);
        return next;
      });
    }
  }, []);

  return (
    <div className="read">
      <header className="read__header">
        <h1 className="sr-only">Feeds</h1>
      </header>

      {loading ? (
        <p className="read__status">Loading feeds…</p>
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
      ) : subscriptions.length === 0 ? (
        <p className="read__status">No feed subscriptions yet. Add a feed URL from the reading list to get started.</p>
      ) : (
        <ul className="read__list feeds__list">
          {subscriptions.map((sub) => (
            <li key={sub.id} className="feeds__item">
              <div className="feeds__info">
                <span className="feeds__title">{decodeEntities(sub.title)}</span>
                {sub.rss_data?.feed_url && (
                  <span className="feeds__url">{sub.rss_data.feed_url}</span>
                )}
              </div>
              <button
                type="button"
                className="feeds__unsubscribe"
                onClick={() => handleUnsubscribe(sub)}
                disabled={unsubscribing.has(sub.id) || !sub.rss_data?.feed_id}
              >
                {unsubscribing.has(sub.id) ? 'Removing…' : 'Unsubscribe'}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
