import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { isAuthenticated } from './auth';
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

// Landing redirect: authenticated users go to their reading list, everyone else
// to login (mirrors the mobile RootNavigator's authenticated/unauthenticated split).
function RootRedirect() {
  return <Navigate to={isAuthenticated() ? '/read' : '/login'} replace />;
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<RootRedirect />} />
        <Route path="/login" element={<Login />} />
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
        <Route path="*" element={<RootRedirect />} />
      </Routes>
    </BrowserRouter>
  );
}
