import { useState } from 'react';
import type { ContentList, SourceRouteType } from '@cairn/shared';
import { ReadService } from '../services/read';
import './SourceListToggle.css';

interface SourceListToggleProps {
  type: SourceRouteType;
  /** Feed ID for 'rss', subscription (sender) ID for 'email'. */
  sourceKey: string;
  list: ContentList;
  label: string;
}

// Per-source Feed/Reads destination. Optimistic: flips immediately and rolls
// back with a visible error if the request fails. Only new items follow the
// change (the screens say so in their footer).
export default function SourceListToggle({ type, sourceKey, list, label }: SourceListToggleProps) {
  const [current, setCurrent] = useState<ContentList>(list);
  const [error, setError] = useState(false);

  const choose = async (next: ContentList) => {
    if (next === current) return;
    const previous = current;
    setCurrent(next);
    setError(false);
    try {
      await ReadService.setSourceList(type, sourceKey, next);
    } catch (err) {
      console.error('Failed to update source list:', err);
      setCurrent(previous);
      setError(true);
    }
  };

  return (
    <div className="source-list-toggle">
      <div className="source-list-toggle__group" role="group" aria-label={`Destination for ${label}`}>
        {(['feed', 'reads'] as const).map((choice) => (
          <button
            key={choice}
            type="button"
            className={`source-list-toggle__option${current === choice ? ' source-list-toggle__option--active' : ''}`}
            aria-pressed={current === choice}
            onClick={() => choose(choice)}
          >
            {choice === 'feed' ? 'Feed' : 'Reads'}
          </button>
        ))}
      </div>
      {error && (
        <span className="source-list-toggle__error" role="alert">
          Couldn't update. Try again.
        </span>
      )}
    </div>
  );
}
