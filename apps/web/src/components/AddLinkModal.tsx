import { useEffect, useRef, useState } from 'react';
import type { DetectURLResponse, DiscoverFeedResponse } from '@cairn/shared';
import { ReadService } from '../services/read';
import { decodeEntities } from '../utils/decodeEntities';
import './AddLinkModal.css';

const FOCUSABLE_SELECTORS =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function useFocusTrap(containerRef: React.RefObject<HTMLElement | null>) {
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return;
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Tab') return;
      const focusable = Array.from(container.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTORS));
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    container.addEventListener('keydown', handleKeyDown);
    return () => container.removeEventListener('keydown', handleKeyDown);
  }, [containerRef]);
}

interface AddLinkModalProps {
  onClose: () => void;
  onSuccess?: () => void;
  closing?: boolean;
}

function isValidUrl(urlString: string): boolean {
  const trimmed = urlString.trim();
  if (!trimmed) return false;
  const urlToTest = /^https?:\/\//.test(trimmed) ? trimmed : `https://${trimmed}`;
  try {
    const parsed = new URL(urlToTest);
    // Reject single-label hosts (e.g. "not-a-url", "asdf") that have no TLD —
    // they pass URL parsing but will never resolve publicly.
    return parsed.hostname.includes('.');
  } catch {
    return false;
  }
}

function normalizeUrl(urlString: string): string {
  const trimmed = urlString.trim();
  return /^https?:\/\//.test(trimmed) ? trimmed : `https://${trimmed}`;
}

export default function AddLinkModal({ onClose, onSuccess, closing = false }: AddLinkModalProps) {
  const [url, setUrl] = useState('');
  const [loading, setLoading] = useState(false);
  const [detecting, setDetecting] = useState(false);
  const [discovering, setDiscovering] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [detectionResult, setDetectionResult] = useState<DetectURLResponse | null>(null);
  // Multiple feeds discovered: show inline picker
  const [feedChoices, setFeedChoices] = useState<DiscoverFeedResponse['feeds']>([]);
  const inputRef = useRef<HTMLInputElement>(null);
  const modalRef = useRef<HTMLDivElement>(null);
  // Capture the element that had focus when the modal opened so we can restore
  // it on close — required by ARIA modal best practices.
  const triggerRef = useRef<HTMLElement | null>(null);

  useFocusTrap(modalRef);

  // Focus the input on mount; restore focus to the trigger on unmount.
  useEffect(() => {
    if (document.activeElement instanceof HTMLElement) {
      triggerRef.current = document.activeElement;
    }
    inputRef.current?.focus();
    return () => {
      triggerRef.current?.focus();
    };
  }, []);

  // Debounced URL detection when the URL changes
  useEffect(() => {
    if (!url.trim() || !isValidUrl(url)) {
      setDetectionResult(null);
      return;
    }
    const normalizedUrl = normalizeUrl(url);
    let cancelled = false;

    const runDetect = async () => {
      setDetecting(true);
      try {
        const result = await ReadService.detectURL(normalizedUrl);
        if (!cancelled) setDetectionResult(result);
      } catch {
        if (!cancelled) setDetectionResult({ url: normalizedUrl, type: 'unknown', title: null });
      } finally {
        if (!cancelled) setDetecting(false);
      }
    };

    // Small debounce so rapid keystrokes don't fire many requests
    const timer = setTimeout(runDetect, 400);
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [url]);

  const handleUrlChange = (value: string) => {
    setUrl(value);
    if (error) setError(null);
    setFeedChoices([]);
  };

  const handleSubmit = async () => {
    if (!isValidUrl(url)) {
      setError('Please enter a valid URL');
      return;
    }
    setError(null);
    setLoading(true);
    try {
      const normalizedUrl = normalizeUrl(url);
      const response = await ReadService.addURL({
        url: normalizedUrl,
        type: detectionResult?.type,
        title: detectionResult?.title ?? undefined,
      });
      // Close and refresh
      onClose();
      if (onSuccess) onSuccess();
      // Brief success: the list will refresh, which is the key behavior
      if (response.type === 'feed') {
        // Feed subscription confirmed by the refreshed /you/feeds count
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : '';
      if (msg.includes('already subscribed')) {
        setError('Already subscribed to this feed');
      } else if (msg.includes('Failed to subscribe')) {
        setError('Failed to subscribe to feed');
      } else if (msg.includes('Failed to add')) {
        setError('Failed to add article');
      } else {
        // Don't surface raw backend errors (e.g. Go dial/DNS messages) to the user.
        console.error('Add link failed:', err);
        setError("Couldn't add that link — check the URL and try again.");
      }
    } finally {
      setLoading(false);
    }
  };

  const handleFindFeed = async () => {
    if (!isValidUrl(url)) {
      setError('Please enter a valid URL');
      return;
    }
    setError(null);
    setFeedChoices([]);
    setDiscovering(true);
    try {
      const normalizedUrl = normalizeUrl(url);
      const { feeds } = await ReadService.discoverFeed(normalizedUrl);
      if (feeds.length === 0) {
        setError('No RSS feed found for this site');
        return;
      }
      if (feeds.length === 1) {
        // Single feed: fill input; detection effect will re-run as 'feed'
        setUrl(feeds[0].url);
        return;
      }
      // Multiple feeds: show inline picker
      setFeedChoices(feeds);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setDiscovering(false);
    }
  };

  const handlePickFeed = (feedUrl: string) => {
    setUrl(feedUrl);
    setFeedChoices([]);
  };

  const handleClose = () => {
    if (!loading && !detecting && !discovering) {
      setUrl('');
      setError(null);
      setDetectionResult(null);
      setFeedChoices([]);
      onClose();
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') handleClose();
    if (e.key === 'Enter') handleSubmit();
  };

  const getSubmitLabel = () => {
    if (loading) return 'Adding…';
    if (detecting) return 'Detecting…';
    if (detectionResult?.type === 'feed') return 'Add Feed';
    return 'Add';
  };

  const isBusy = loading || detecting || discovering;

  return (
    // Overlay backdrop
    <div className={`add-link-overlay${closing ? ' add-link-overlay--closing' : ''}`} role="dialog" aria-modal="true" aria-label="Add link">
      <div className="add-link-modal" ref={modalRef} onKeyDown={handleKeyDown}>
        <div className="add-link-modal__body">
          <input
            ref={inputRef}
            type="url"
            className={`add-link-modal__input${error ? ' add-link-modal__input--error' : ''}`}
            placeholder="Add link"
            aria-label="URL to add"
            value={url}
            onChange={(e) => handleUrlChange(e.target.value)}
            disabled={loading}
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
          />
          {error && <p className="add-link-modal__error">{error}</p>}

          {feedChoices.length > 0 && (
            <ul className="add-link-modal__feed-list">
              {feedChoices.map((feed) => (
                <li key={feed.url}>
                  <button
                    type="button"
                    className="add-link-modal__feed-item"
                    onClick={() => handlePickFeed(feed.url)}
                  >
                    {feed.title ? decodeEntities(feed.title) : feed.url}
                  </button>
                </li>
              ))}
            </ul>
          )}

          <div className="add-link-modal__actions">
            <button
              type="button"
              className="add-link-modal__btn add-link-modal__btn--primary"
              onClick={handleSubmit}
              disabled={isBusy || !url.trim()}
            >
              {getSubmitLabel()}
            </button>

            {detectionResult?.type !== 'feed' && (
              <button
                type="button"
                className="add-link-modal__btn add-link-modal__btn--secondary"
                onClick={handleFindFeed}
                disabled={isBusy || !url.trim()}
              >
                {discovering ? 'Finding…' : 'Find feed'}
              </button>
            )}

            <button
              type="button"
              className="add-link-modal__btn add-link-modal__btn--cancel"
              onClick={handleClose}
              disabled={isBusy}
            >
              Cancel
            </button>
          </div>
        </div>
      </div>
      {/* Click outside to close */}
      <div className="add-link-overlay__backdrop" onClick={handleClose} aria-hidden="true" />
    </div>
  );
}
