import { NavLink, useLocation } from 'react-router-dom';
import './BottomNav.css';

// Mobile bottom navigation bar — shown only at <768px.
// Mirrors the mobile app's CustomTabBar: Feed, Reads, You tabs.
export default function BottomNav() {
  const location = useLocation();
  const onYouRoute = location.pathname.startsWith('/you');

  return (
    <nav className="bottom-nav" aria-label="Primary">
      <NavLink to="/feed" className={({ isActive }) => `bottom-nav__item${isActive ? ' bottom-nav__item--active' : ''}`}>
        <span className="bottom-nav__icon" aria-hidden="true">
          {/* RSS/feed icon */}
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M4 11a9 9 0 0 1 9 9" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M4 4a16 16 0 0 1 16 16" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <circle cx="5" cy="19" r="1" fill="currentColor" stroke="currentColor" strokeWidth="2"/>
          </svg>
        </span>
        <span className="bottom-nav__label">Feed</span>
      </NavLink>

      <NavLink to="/reads" className={({ isActive }) => `bottom-nav__item${isActive ? ' bottom-nav__item--active' : ''}`}>
        <span className="bottom-nav__icon" aria-hidden="true">
          {/* Book/read icon */}
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </span>
        <span className="bottom-nav__label">Reads</span>
      </NavLink>

      <NavLink
        to="/you"
        className={`bottom-nav__item${onYouRoute ? ' bottom-nav__item--active' : ''}`}
      >
        <span className="bottom-nav__icon" aria-hidden="true">
          {/* Profile/you icon */}
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <circle cx="12" cy="7" r="4" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </span>
        <span className="bottom-nav__label">You</span>
      </NavLink>
    </nav>
  );
}
