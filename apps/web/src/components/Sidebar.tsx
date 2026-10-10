import { useCallback, useEffect, useState } from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { ReadService } from '../services/read';

// Counts shown beside the You sub-items. `null` means "not loaded yet" so the
// badge can stay hidden until the first successful fetch.
interface YouCounts {
  feeds: number | null;
  newsletters: number | null;
  bookmarks: number | null;
}

const EMPTY_COUNTS: YouCounts = {
  feeds: null,
  newsletters: null,
  bookmarks: null,
};

// The You sub-menu, mirroring the mobile YouScreen menu order. `countKey` picks
// which fetched count (if any) renders as a badge beside the item.
const YOU_ITEMS: Array<{ to: string; label: string; countKey?: keyof YouCounts }> = [
  { to: '/you/account', label: 'Account' },
  { to: '/you/feeds', label: 'Feeds', countKey: 'feeds' },
  { to: '/you/newsletters', label: 'Newsletters', countKey: 'newsletters' },
  { to: '/you/bookmarks', label: 'Bookmarks', countKey: 'bookmarks' },
  { to: '/you/about', label: 'About' },
];

// Outline SVG icons ported from apps/mobile/src/components/icons/
// viewBox "0 0 24 24", stroke="currentColor", fill="none"

function ReadIcon() {
  return (
    <svg
      aria-hidden="true"
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M12 6.042A8.967 8.967 0 0 0 6 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 0 1 6 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 0 1 6-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0 0 18 18a8.967 8.967 0 0 0-6 2.292m0-14.25v14.25" />
    </svg>
  );
}

function ProfileIcon() {
  return (
    <svg
      aria-hidden="true"
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M17.982 18.725A7.488 7.488 0 0 0 12 15.75a7.488 7.488 0 0 0-5.982 2.975m11.963 0a9 9 0 1 0-11.963 0m11.963 0A8.966 8.966 0 0 1 12 21a8.966 8.966 0 0 1-5.982-2.275M15 9.75a3 3 0 1 1-6 0 3 3 0 0 1 6 0Z" />
    </svg>
  );
}

export default function Sidebar() {
  const { logout } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  const onYouRoute = location.pathname.startsWith('/you');
  // Auto-expand when landing on a You route so the active sub-item is visible.
  const [youExpanded, setYouExpanded] = useState(onYouRoute);
  const [counts, setCounts] = useState<YouCounts>(EMPTY_COUNTS);

  // Fetch the You counts in parallel with independent failure tolerance, so one
  // broken endpoint doesn't hide the others (mirrors mobile YouScreen).
  const refreshCounts = useCallback(async () => {
    const [subscriptionsResult, bookmarksResult] = await Promise.allSettled([
      ReadService.listAllSubscriptions(),
      ReadService.countUserContents({ is_favorite: true }),
    ]);

    setCounts((prev) => {
      const next = { ...prev };
      if (subscriptionsResult.status === 'fulfilled') {
        const subs = subscriptionsResult.value?.subscriptions ?? [];
        next.feeds = subs.filter((s) => s.type !== 'email').length;
        next.newsletters = subs.filter((s) => s.type === 'email').length;
      } else {
        console.error('Error fetching subscriptions:', subscriptionsResult.reason);
      }
      if (bookmarksResult.status === 'fulfilled') {
        next.bookmarks = bookmarksResult.value;
      } else {
        console.error('Error fetching bookmarks:', bookmarksResult.reason);
      }
      return next;
    });
  }, []);

  // Refresh counts every time the You section is (re-)expanded.
  useEffect(() => {
    if (youExpanded) {
      void refreshCounts();
    }
  }, [youExpanded, refreshCounts]);

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Error logging out:', error);
    }
    navigate('/login', { replace: true });
  };

  return (
    <nav className="sidebar" aria-label="Primary">
      <div className="sidebar__brand">Cairn</div>

      <ul className="sidebar__nav">
        <li>
          <NavLink to="/read" className="sidebar__item" aria-label="Read">
            <span className="sidebar__item-leading">
              <span className="sidebar__icon"><ReadIcon /></span>
              <span>Read</span>
            </span>
          </NavLink>
        </li>
        <li>
          <button
            type="button"
            className={`sidebar__item sidebar__item--button${onYouRoute ? ' sidebar__item--active' : ''}`}
            aria-expanded={youExpanded}
            aria-label="You"
            onClick={() => setYouExpanded((open) => !open)}
          >
            <span className="sidebar__item-leading">
              <span className="sidebar__icon"><ProfileIcon /></span>
              <span>You</span>
            </span>
            <span className={`sidebar__chevron${youExpanded ? ' sidebar__chevron--open' : ''}`} aria-hidden="true">
              ›
            </span>
          </button>

          {youExpanded && (
            <ul className="sidebar__subnav">
              {YOU_ITEMS.map(({ to, label, countKey }) => {
                const count = countKey ? counts[countKey] : null;
                return (
                  <li key={to}>
                    <NavLink to={to} className="sidebar__item sidebar__item--sub" end>
                      <span>{label}</span>
                      {count !== null && <span className="sidebar__badge">{count}</span>}
                    </NavLink>
                  </li>
                );
              })}
              <li>
                <button
                  type="button"
                  className="sidebar__item sidebar__item--sub sidebar__item--button"
                  onClick={handleLogout}
                >
                  Log out
                </button>
              </li>
            </ul>
          )}
        </li>
      </ul>
    </nav>
  );
}
