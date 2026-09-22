import type { UpdateWorkspaceInput } from '../../types';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  createWorkspaceAndSelect,
  fetchWorkspaces,
  selectSelectedWorkspace,
  selectSelectedWorkspaceId,
  selectWorkspace,
  selectWorkspaceError,
  selectWorkspaceLoading,
  selectWorkspaceSaving,
  selectWorkspaces,
  updateWorkspaceAndStore,
} from './workspaceSlice';

export function useWorkspace() {
  const dispatch = useAppDispatch();
  const workspaces = useAppSelector(selectWorkspaces);
  const selected = useAppSelector(selectSelectedWorkspace);
  const selectedId = useAppSelector(selectSelectedWorkspaceId);
  const loading = useAppSelector(selectWorkspaceLoading);
  const saving = useAppSelector(selectWorkspaceSaving);
  const error = useAppSelector(selectWorkspaceError);

  return {
    workspaces,
    selected,
    selectedId,
    loading,
    saving,
    error,
    refresh: (profileId: string) => {
      void dispatch(fetchWorkspaces(profileId));
    },
    select: (id: string) => dispatch(selectWorkspace(id)),
    create: async (profileId: string, name: string, description?: string) => {
      return dispatch(createWorkspaceAndSelect({ profileId, name, description })).unwrap();
    },
    update: async (id: string, input: UpdateWorkspaceInput) => {
      return dispatch(updateWorkspaceAndStore({ id, input })).unwrap();
    },
  };
}
