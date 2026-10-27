import { useCallback } from 'react';

import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchProjects,
  selectActiveProjects,
  selectCurrentProject,
  selectCurrentProjectId,
  selectProjectError,
  selectProjectLoading,
  selectProjects,
  setCurrentProject,
  setProjectArchived,
  saveProject,
} from './projectSlice';
import type { CreateProjectInput, UpdateProjectInput } from '../../types';

/** Thunk reject value là chuỗi đã localize; ném lại thành Error để UI dùng getErrorMessage. */
async function unwrapError<T>(promise: { unwrap: () => Promise<T> }): Promise<T> {
  try {
    return await promise.unwrap();
  } catch (cause) {
    throw typeof cause === 'string' ? new Error(cause) : cause;
  }
}

export function useProject() {
  const dispatch = useAppDispatch();
  const projects = useAppSelector(selectProjects);
  const activeProjects = useAppSelector(selectActiveProjects);
  const currentProject = useAppSelector(selectCurrentProject);
  const currentProjectId = useAppSelector(selectCurrentProjectId);
  const loading = useAppSelector(selectProjectLoading);
  const error = useAppSelector(selectProjectError);
  const profileId = useAppSelector((state) => state.session.profileId);

  const setCurrent = useCallback((id: string) => dispatch(setCurrentProject(id)), [dispatch]);

  const reload = useCallback(() => {
    if (profileId) {
      void dispatch(fetchProjects(profileId));
    }
  }, [dispatch, profileId]);

  const save = useCallback(
    async (input: CreateProjectInput | UpdateProjectInput, id?: string) => {
      if (!profileId) {
        throw new Error('no profile');
      }
      return unwrapError(dispatch(saveProject({ profileId, id, input })));
    },
    [dispatch, profileId],
  );

  const setArchived = useCallback(
    async (id: string, archived: boolean) => {
      await unwrapError(dispatch(setProjectArchived({ id, archived })));
    },
    [dispatch],
  );

  return {
    projects,
    activeProjects,
    currentProject,
    currentProjectId,
    loading,
    error,
    setCurrent,
    reload,
    save,
    setArchived,
  };
}
