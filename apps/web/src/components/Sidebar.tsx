import { useCallback, useEffect, useState } from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { ReadService } from '../services/read';
import { ExploreService } from '../services/explore';

// Counts shown beside the You sub-items. `null` means "not loaded yet" so the
// badge can stay hidden until the first successful fetch.
interface YouCounts {
  feeds: number | null;
  newsletters: number | null;
  bookmarks: number | null;
  votes: number | null;
}

const EMPTY_COUNTS: YouCounts = {
  feeds: null,
  newsletters: null,
  bookmarks: null,
  votes: null,
};

// The You sub-menu, mirroring the mobile YouScreen menu order. `countKey` picks
// which fetched count (if any) renders as a badge beside the item.
const YOU_ITEMS: Array<{ to: string; label: string; countKey?: keyof YouCounts }> = [
  { to: '/you/account', label: 'Account' },
  { to: '/you/feeds', label: 'Feeds', countKey: 'feeds' },
  { to: '/you/newsletters', label: 'Newsletters', countKey: 'newsletters' },
  { to: '/you/bookmarks', label: 'Bookmarks', countKey: 'bookmarks' },
  { to: '/you/votes', label: 'Votes', countKey: 'votes' },
];

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
    const [voteResult, subscriptionsResult, bookmarksResult] = await Promise.allSettled([
      ExploreService.getUserVoteStats(),
      ReadService.listAllSubscriptions(),
      ReadService.listUserContents({ limit: 1 }),
    ]);

    setCounts((prev) => {
      const next = { ...prev };
      if (voteResult.status === 'fulfilled') {
        next.votes = voteResult.value.upvotes + voteResult.value.downvotes;
      } else {
        console.error('Error fetching vote stats:', voteResult.reason);
      }
      if (subscriptionsResult.status === 'fulfilled') {
        const subs = subscriptionsResult.value?.subscriptions ?? [];
        next.feeds = subs.filter((s) => s.type !== 'email').length;
        next.newsletters = subs.filter((s) => s.type === 'email').length;
      } else {
        console.error('Error fetching subscriptions:', subscriptionsResult.reason);
      }
      if (bookmarksResult.status === 'fulfilled') {
        next.bookmarks = bookmarksResult.value.total_count;
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
          <NavLink to="/read" className="sidebar__item" data-label="R">
            Read
          </NavLink>
        </li>
        <li>
          <NavLink to="/explore" className="sidebar__item" data-label="E">
            Explore
          </NavLink>
        </li>
        <li>
          <button
            type="button"
            className={`sidebar__item sidebar__item--button${onYouRoute ? ' sidebar__item--active' : ''}`}
            data-label="Y"
            aria-expanded={youExpanded}
            onClick={() => setYouExpanded((open) => !open)}
          >
            <span>You</span>
            <span className={`sidebar__chevron${youExpanded ? ' sidebar__chevron--open' : ''}`} aria-hidden>
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
