import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import AppLayout from './components/AppLayout';
import ErrorBoundary from './components/ErrorBoundary';
import Login from './routes/Login';
import Read from './routes/Read';
import ReadArticle from './routes/ReadArticle';
import Explore from './routes/Explore';
import ExploreArticle from './routes/ExploreArticle';
import You from './routes/You';
import Account from './routes/Account';
import Feeds from './routes/Feeds';
import Newsletters from './routes/Newsletters';
import Bookmarks from './routes/Bookmarks';
import Votes from './routes/Votes';
import About from './routes/About';

// Landing redirect: authenticated users go to their reading list, everyone else
// to login (mirrors the mobile RootNavigator's authenticated/unauthenticated split).
function RootRedirect() {
  const { isAuthenticated, isLoading } = useAuth();
  // Wait for the session check before redirecting, otherwise an authenticated
  // user landing on "/" is sent to /login (isAuthenticated is false until the
  // check resolves) and stranded there.
  if (isLoading) {
    return null;
  }
  return <Navigate to={isAuthenticated ? '/read' : '/login'} replace />;
}

// Auth guard: holds rendering until the session check resolves, then redirects
// unauthenticated users to /login. Authenticated routes render via <Outlet />.
function RequireAuth() {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading) {
    return null;
  }
  return isAuthenticated ? <Outlet /> : <Navigate to="/login" replace />;
}

export default function App() {
  return (
    <BrowserRouter>
      <ErrorBoundary>
        <AuthProvider>
          <Routes>
            <Route path="/" element={<RootRedirect />} />
            <Route path="/login" element={<Login />} />
            <Route element={<RequireAuth />}>
              <Route element={<AppLayout />}>
                <Route path="/read" element={<Read />} />
                <Route path="/read/:id" element={<ReadArticle />} />
                <Route path="/explore" element={<Explore />} />
                <Route path="/explore/:id" element={<ExploreArticle />} />
                <Route path="/you" element={<You />} />
                <Route path="/you/account" element={<Account />} />
                <Route path="/you/feeds" element={<Feeds />} />
                <Route path="/you/newsletters" element={<Newsletters />} />
                <Route path="/you/bookmarks" element={<Bookmarks />} />
                <Route path="/you/votes" element={<Votes />} />
                <Route path="/you/about" element={<About />} />
              </Route>
            </Route>
            <Route path="*" element={<RootRedirect />} />
          </Routes>
        </AuthProvider>
      </ErrorBoundary>
    </BrowserRouter>
  );
}
