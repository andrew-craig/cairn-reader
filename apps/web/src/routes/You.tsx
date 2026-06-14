import { useNavigate, NavLink } from 'react-router-dom';
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

// The You index route (/you). On desktop/tablet this is never reached because
// the sidebar auto-expands the You sub-menu and routes land on sub-pages.
// On mobile (<768px) the sidebar is hidden, so /you acts as the "You" hub page
// showing all sub-destinations — mirroring the mobile app's YouScreen.
export default function You() {
  const { logout } = useAuth();
  const navigate = useNavigate();

  const handleLogout = async () => {
    try {
      await logout();
    } catch (error) {
      console.error('Error logging out:', error);
    }
    navigate('/login', { replace: true });
  };

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
