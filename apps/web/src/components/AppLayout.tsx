import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import BottomNav from './BottomNav';
import './AppLayout.css';

// The authenticated chrome: a persistent left sidebar with the main content
// rendered alongside it. On mobile (<768px) the sidebar is hidden and replaced
// by a bottom navigation bar (BottomNav). Replaces mobile's CustomTabBar.
export default function AppLayout() {
  return (
    <div className="app-shell">
      <div className="app-shell__inner">
        <Sidebar />
        <main className="app-main">
          <div className="app-content">
            <Outlet />
          </div>
        </main>
      </div>
      <BottomNav />
    </div>
  );
}
