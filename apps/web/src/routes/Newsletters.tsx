import { useCallback, useEffect, useState } from 'react';
import type { UnifiedSubscription } from '@cairn/shared';
import { ReadService } from '../services/read';
import './Read.css';
import './Newsletters.css';

// Newsletters screen (/you/newsletters): the user's email ingest address and
// email-type subscriptions. Mirrors mobile's NewslettersScreen.
export default function Newsletters() {
  const [emailAddress, setEmailAddress] = useState<string | null>(null);
  const [subscriptions, setSubscriptions] = useState<UnifiedSubscription[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [address, response] = await Promise.all([
        ReadService.getOrCreateEmailAddress(),
        ReadService.listAllSubscriptions(),
      ]);
      setEmailAddress(address);
      setSubscriptions(response.subscriptions.filter((s) => s.type === 'email'));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load newsletter info.');
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    load().finally(() => setLoading(false));
  }, [load]);

  const handleCopy = useCallback(() => {
    if (!emailAddress) return;
    navigator.clipboard.writeText(emailAddress).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    });
  }, [emailAddress]);

  return (
    <div className="read">
      <header className="read__header">
        <h1 style={{ margin: 0, fontFamily: 'var(--font-heading)', fontSize: 'var(--font-size-xl)', fontWeight: 600 }}>
          Newsletters
        </h1>
      </header>

      {loading ? (
        <p className="read__status">Loading…</p>
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
      ) : (
        <>
          <div className="newsletters__address-card">
            <p className="newsletters__label">Send or forward emails to</p>
            <div className="newsletters__address-row">
              <span className="newsletters__address">{emailAddress}</span>
              <button
                type="button"
                className="newsletters__copy"
                onClick={handleCopy}
                aria-label="Copy email address"
              >
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
            <p className="newsletters__hint">
              New subscriptions will be automatically added to your reading list.
            </p>
          </div>

          {subscriptions.length === 0 ? (
            <p className="read__status">No newsletter subscriptions yet. Forward an email to your address above to add one.</p>
          ) : (
            <ul className="read__list feeds__list">
              {subscriptions.map((sub) => (
                <li key={sub.id} className="feeds__item">
                  <div className="feeds__info">
                    <span className="feeds__title">{sub.title}</span>
                    {sub.email_data?.email_address && (
                      <span className="feeds__url">{sub.email_data.email_address}</span>
                    )}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
