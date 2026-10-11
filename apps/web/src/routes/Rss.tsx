import { useCallback, useEffect, useState } from 'react';
import type { UnifiedSubscription } from '@cairn/shared';
import { ReadService } from '../services/read';
import SourceListToggle from '../components/SourceListToggle';
import { decodeEntities } from '../utils/decodeEntities';
import './Read.css';
import './Newsletters.css';

// RSS screen (/you/rss): the user's RSS feed subscriptions.
// Mirrors mobile's RssScreen + SubscriptionListScreen — filters to non-email
// types, shows title/URL, and offers per-row unsubscribe (optimistic removal).
export default function Rss() {
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
      setError(e instanceof Error ? e.message : 'Failed to load RSS feeds.');
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    load().finally(() => setLoading(false));
  }, [load]);

  const handleUnsubscribe = useCallback(async (sub: UnifiedSubscription) => {
    if (!sub.rss_data?.feed_id) return;
    if (!window.confirm(`Unsubscribe from ${decodeEntities(sub.title)}?`)) return;
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
        <h1 className="sr-only">RSS</h1>
      </header>

      {loading ? (
        <p className="read__status">Loading RSS feeds…</p>
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
        <p className="read__status">No RSS subscriptions yet. Add a feed URL from Reads to get started.</p>
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
              {sub.rss_data?.feed_id && (
                <SourceListToggle
                  type="rss"
                  sourceKey={sub.rss_data.feed_id}
                  list={sub.list}
                  label={decodeEntities(sub.title)}
                />
              )}
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
      {subscriptions.length > 0 && (
        <p className="read__status">Feed or Reads applies to new items only.</p>
      )}
    </div>
  );
}
