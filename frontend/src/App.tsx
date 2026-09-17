import { useEffect } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';

import { AppLayout } from './components/layout/AppLayout';
import { useAppDispatch } from './context';
import { fetchProfiles } from './context/session/sessionSlice';
import { BoardPage } from './pages/BoardPage';
import { StatusesPage } from './pages/StatusesPage';

export default function App() {
  const dispatch = useAppDispatch();

  useEffect(() => {
    void dispatch(fetchProfiles());
  }, [dispatch]);

  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route path="/" element={<BoardPage />} />
        <Route path="/statuses" element={<StatusesPage />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
