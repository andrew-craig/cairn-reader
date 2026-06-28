import { useEffect, useState } from 'react';
import { useNavigate, NavLink, Navigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './You.css';

// YOU_ITEMS mirrors Sidebar.tsx so both surfaces stay in sync.
const YOU_ITEMS = [
  { to: '/you/account', label: 'Account' },
  { to: '/you/feeds', label: 'Feeds' },
  { to: '/you/newsletters', label: 'Newsletters' },
  { to: '/you/bookmarks', label: 'Bookmarks' },
  { to: '/you/votes', label: 'Votes' },
  { to: '/you/about', label: 'About' },
];

// The You index route (/you). On desktop/tablet (>=768px) the sidebar is
// visible and shows the sub-menu, so /you is redundant — redirect to the first
// sub-page. On mobile (<768px) the sidebar is hidden; /you acts as the hub.
export default function You() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  // SSR-safe: window may not exist. Initialise from window if available, then
  // keep in sync with resize so hot-reloading and orientation changes work.
  const [isDesktop, setIsDesktop] = useState(
    typeof window !== 'undefined' ? window.matchMedia('(min-width: 768px)').matches : false,
  );

  useEffect(() => {
    if (typeof window === 'undefined') return;
    const mq = window.matchMedia('(min-width: 768px)');
    setIsDesktop(mq.matches);
    const handler = (e: MediaQueryListEvent) => setIsDesktop(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Error logging out:', error);
    }
    navigate('/login', { replace: true });
  };

  if (isDesktop) {
    return <Navigate to="/you/account" replace />;
  }

  return (
    <div className="you-page">
      <h1 className="you-page__title sr-only">You</h1>

      <ul className="you-page__list">
        {YOU_ITEMS.map(({ to, label }) => (
          <li key={to} className="you-page__item">
            <NavLink to={to} className="you-page__link">
              {label}
            </NavLink>
          </li>
        ))}
        <li className="you-page__item">
          <button type="button" className="you-page__link you-page__link--logout" onClick={handleLogout}>
            Log out
          </button>
        </li>
      </ul>
    </div>
  );
}
