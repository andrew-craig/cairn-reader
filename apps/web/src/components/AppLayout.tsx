import { Outlet, useLocation } from 'react-router-dom';
import Sidebar from './Sidebar';
import './AppLayout.css';

// The authenticated chrome: a persistent left sidebar with the main content
// rendered alongside it. Replaces mobile's bottom tab bar (CustomTabBar).
export default function AppLayout() {
  const location = useLocation();
  // The search + quick-actions bar belongs to the reading list (mobile's top
  // quick-actions row). Only shown on /read; search/add land in later tasks.
  const showReadToolbar = location.pathname === '/read';

  return (
    <div className="app-shell">
      <Sidebar />
      <main className="app-main">
        {showReadToolbar && (
          <div className="read-toolbar">
            <input
              type="search"
              className="read-toolbar__search"
              placeholder="Search your reading list"
              aria-label="Search your reading list"
            />
          </div>
        )}
        <div className="app-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
