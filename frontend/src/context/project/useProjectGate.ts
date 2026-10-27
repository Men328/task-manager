import { useEffect } from 'react';

import { useAppDispatch, useAppSelector } from '../hooks';
import { fetchProjects } from './projectSlice';

export type ProjectGate =
  | { status: 'ready' }
  | { status: 'loading' }
  | { status: 'error'; message: string; retry: () => void };

/**
 * Nạp project khi đã có profile và chặn render trang cho tới khi có currentProjectId.
 * Không có profile thì không chặn: các trang tự hiển thị trạng thái "chưa có hồ sơ".
 */
export function useProjectGate(): ProjectGate {
  const dispatch = useAppDispatch();
  const profileId = useAppSelector((state) => state.session.profileId);
  const currentId = useAppSelector((state) => state.project.currentId);
  const error = useAppSelector((state) => state.project.error);

  useEffect(() => {
    if (profileId) {
      void dispatch(fetchProjects(profileId));
    }
  }, [dispatch, profileId]);

  if (!profileId || currentId) {
    return { status: 'ready' };
  }
  if (error) {
    return {
      status: 'error',
      message: error,
      retry: () => void dispatch(fetchProjects(profileId)),
    };
  }
  return { status: 'loading' };
}
