import { useEffect } from 'react';

import { useAppDispatch, useAppSelector } from '../hooks';
import { fetchProjects } from './projectSlice';

export type ProjectGate =
  | { status: 'ready' }
  | { status: 'loading' }
  | { status: 'empty' }
  | { status: 'error'; message: string; retry: () => void };

/**
 * Nạp project khi đã có profile và chặn render trang cho tới khi projects của profile này đã tải và currentProjectId đã được kiểm tra.
 * Không có profile thì không chặn: các trang tự hiển thị trạng thái "chưa có hồ sơ".
 */
export function useProjectGate(): ProjectGate {
  const dispatch = useAppDispatch();
  const profileId = useAppSelector((state) => state.session.profileId);
  const currentId = useAppSelector((state) => state.project.currentId);
  const loadedProfileId = useAppSelector((state) => state.project.loadedProfileId);
  const error = useAppSelector((state) => state.project.error);

  useEffect(() => {
    if (profileId) {
      void dispatch(fetchProjects(profileId));
    }
  }, [dispatch, profileId]);

  if (!profileId) {
    return { status: 'ready' };
  }
  if (loadedProfileId === profileId) {
    return currentId ? { status: 'ready' } : { status: 'empty' };
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
