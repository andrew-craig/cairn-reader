import { Outlet } from 'react-router-dom';
import Sidebar from './Sidebar';
import './AppLayout.css';

// The authenticated chrome: a persistent left sidebar with the main content
// rendered alongside it. Replaces mobile's bottom tab bar (CustomTabBar).
export default function AppLayout() {
  return (
    <div className="app-shell">
      <Sidebar />
      <main className="app-main">
        <div className="app-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
}
