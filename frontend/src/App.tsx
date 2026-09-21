import { useEffect } from 'react';
import { Navigate, Outlet, Route, Routes } from 'react-router-dom';

import { AppLayout } from './components/layout/AppLayout';
import { useAppDispatch, useAppSelector } from './context';
import { restoreSession } from './context/session/sessionSlice';
import { readToken } from './lib/session';
import { AuthCallbackPage } from './pages/AuthCallbackPage';
import { BoardPage } from './pages/BoardPage';
import { LoginPage } from './pages/LoginPage';
import { StatusesPage } from './pages/StatusesPage';

const CALLBACK_PATH = '/auth/callback';

function RequireAuth() {
  const authenticated = useAppSelector((state) => state.session.token !== null);

  if (!authenticated) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
}

export default function App() {
  const dispatch = useAppDispatch();

  useEffect(() => {
    if (window.location.pathname === CALLBACK_PATH) {
      return;
    }
    if (readToken()) {
      void dispatch(restoreSession());
    }
  }, [dispatch]);

  return (
    <Routes>
      <Route path="/login" element={<LoginPage />} />
      <Route path={CALLBACK_PATH} element={<AuthCallbackPage />} />

      <Route element={<RequireAuth />}>
        <Route element={<AppLayout />}>
          <Route path="/" element={<BoardPage />} />
          <Route path="/statuses" element={<StatusesPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Route>
    </Routes>
  );
}
