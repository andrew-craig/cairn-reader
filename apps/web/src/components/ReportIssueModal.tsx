import { useEffect, useRef, useState } from 'react';
import { FEEDBACK_CATEGORIES, ReadService, type FeedbackCategory } from '../services/read';
import { useFocusTrap } from '../hooks/useFocusTrap';
import './AddLinkModal.css';
import './ReportIssueModal.css';

const COMMENT_MAX_LENGTH = 2000;
// How long the confirmation stays visible before the dialog closes.
const SENT_CLOSE_DELAY_MS = 1200;

const REASON_LABELS: Record<FeedbackCategory, string> = {
  body_missing_content: 'Part of the article is missing',
  body_extra_content: "Contains things that aren't part of the article (ads, menus, related links)",
  images: 'Images are missing or broken',
  formatting: 'Formatting is wrong (lists, tables, code, layout)',
  links: 'Links are missing or broken',
  title: 'Title is wrong',
  author: 'Author is wrong',
  description: 'Summary is wrong',
  lead_image: 'Header image is wrong',
  other: 'Something else',
};

interface ReportIssueModalProps {
  contentId: string;
  onClose: () => void;
  closing?: boolean;
}

// "Report an issue with the article" dialog. Same modal mechanics as AddLinkModal.
export default function ReportIssueModal({ contentId, onClose, closing = false }: ReportIssueModalProps) {
  const [selected, setSelected] = useState<FeedbackCategory[]>([]);
  const [comment, setComment] = useState('');
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const modalRef = useRef<HTMLDivElement>(null);
  const firstReasonRef = useRef<HTMLInputElement>(null);
  const triggerRef = useRef<HTMLElement | null>(null);
  // Set once the user edits, so a late-arriving existing report can't overwrite their input.
  const touchedRef = useRef(false);

  useFocusTrap(modalRef);

  // Focus the first reason on mount; restore focus to the trigger on unmount.
  useEffect(() => {
    if (document.activeElement instanceof HTMLElement) {
      triggerRef.current = document.activeElement;
    }
    firstReasonRef.current?.focus();
    return () => {
      triggerRef.current?.focus();
    };
  }, []);

  // Prefill from the user's existing report, if any.
  useEffect(() => {
    let cancelled = false;
    ReadService.getArticleFeedback(contentId)
      .then((existing) => {
        if (cancelled || touchedRef.current || !existing) return;
        setSelected(existing.categories);
        setComment(existing.comment ?? '');
      })
      .catch((err) => console.error('Failed to load existing report:', err));
    return () => {
      cancelled = true;
    };
  }, [contentId]);

  // Close shortly after the confirmation appears.
  useEffect(() => {
    if (!sent) return;
    const timer = setTimeout(onClose, SENT_CLOSE_DELAY_MS);
    return () => clearTimeout(timer);
  }, [sent, onClose]);

  const toggle = (category: FeedbackCategory) => {
    touchedRef.current = true;
    setSelected((prev) =>
      prev.includes(category) ? prev.filter((c) => c !== category) : [...prev, category],
    );
  };

  const handleSubmit = async () => {
    setError(null);
    setSending(true);
    try {
      await ReadService.submitArticleFeedback(contentId, {
        // Keep the list's canonical order regardless of click order.
        categories: FEEDBACK_CATEGORIES.filter((c) => selected.includes(c)),
        comment: comment.trim() || undefined,
      });
      setSent(true);
    } catch (err) {
      console.error('Report failed:', err);
      setError("Couldn't send your report — please try again.");
    } finally {
      setSending(false);
    }
  };

  const handleClose = () => {
    if (sending) return;
    onClose();
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') handleClose();
  };

  return (
    <div
      className={`add-link-overlay${closing ? ' add-link-overlay--closing' : ''}`}
      role="dialog"
      aria-modal="true"
      aria-label="Report an issue"
    >
      <div className="add-link-modal" ref={modalRef} onKeyDown={handleKeyDown}>
        <div className="add-link-modal__body">
          {sent ? (
            <p className="report-modal__sent" role="status">
              Thanks — your report was sent.
            </p>
          ) : (
            <>
              <h2 className="report-modal__title">Report an issue</h2>
              <fieldset className="report-modal__reasons" disabled={sending}>
                <legend className="report-modal__legend">What went wrong?</legend>
                {FEEDBACK_CATEGORIES.map((category, i) => (
                  <label key={category} className="report-modal__reason">
                    <input
                      ref={i === 0 ? firstReasonRef : undefined}
                      type="checkbox"
                      checked={selected.includes(category)}
                      onChange={() => toggle(category)}
                    />
                    {REASON_LABELS[category]}
                  </label>
                ))}
              </fieldset>
              <textarea
                className="add-link-modal__input report-modal__comment"
                aria-label="Details (optional)"
                placeholder="Details (optional)"
                maxLength={COMMENT_MAX_LENGTH}
                rows={4}
                value={comment}
                onChange={(e) => {
                  touchedRef.current = true;
                  setComment(e.target.value);
                }}
                disabled={sending}
              />
              {error && <p className="add-link-modal__error">{error}</p>}
              <div className="add-link-modal__actions">
                <button
                  type="button"
                  className="add-link-modal__btn add-link-modal__btn--primary"
                  onClick={handleSubmit}
                  disabled={sending || selected.length === 0}
                >
                  {sending ? 'Sending…' : 'Submit'}
                </button>
                <button
                  type="button"
                  className="add-link-modal__btn add-link-modal__btn--cancel"
                  onClick={handleClose}
                  disabled={sending}
                >
                  Cancel
                </button>
              </div>
            </>
          )}
        </div>
      </div>
      {/* Click outside to close */}
      <div className="add-link-overlay__backdrop" onClick={handleClose} aria-hidden="true" />
    </div>
  );
}
