import { useEffect, useState } from 'react';
import { SystemService } from '../services/system';
import './About.css';

export default function About() {
  const [version, setVersion] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    let active = true;
    SystemService.getServerVersion()
      .then((v) => {
        if (active) setVersion(v);
      })
      .catch((err) => {
        console.error('Error fetching server version:', err);
        if (active) setError(true);
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  // The cloud backend may not report a version; degrade gracefully rather than
  // showing an empty value.
  const versionValue = loading ? 'Loading…' : error ? 'Unavailable' : version ?? 'Unknown';

  return (
    <div className="about-page">
      <div className="about-row">
        <span className="about-label">App version</span>
        <span className="about-value">{__APP_VERSION__}</span>
      </div>
      <div className="about-row">
        <span className="about-label">Server version</span>
        <span className="about-value">{versionValue}</span>
      </div>
    </div>
  );
}
