import { useCallback } from 'react';

import i18n from '../../i18n';
import type {
  BacklogStatus,
  CreateBacklogInput,
  UpdateBacklogInput,
} from '../../types';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchBacklogs,
  removeBacklog,
  saveBacklog,
  selectBacklogError,
  selectBacklogFilter,
  selectBacklogItems,
  selectBacklogLoading,
} from './backlogSlice';

export type NewBacklogInput = Omit<CreateBacklogInput, 'profileId'>;

export function useBacklog() {
  const dispatch = useAppDispatch();
  const items = useAppSelector(selectBacklogItems);
  const filter = useAppSelector(selectBacklogFilter);
  const loading = useAppSelector(selectBacklogLoading);
  const error = useAppSelector(selectBacklogError);
  const profileId = useAppSelector((state) => state.session.profileId);

  const applyFilter = useCallback(
    (next: { status?: BacklogStatus; category?: string }) => {
      if (profileId) {
        void dispatch(fetchBacklogs({ profileId, ...next }));
      }
    },
    [dispatch, profileId],
  );

  const refresh = useCallback(() => {
    if (profileId) {
      void dispatch(fetchBacklogs({ profileId, ...filter }));
    }
  }, [dispatch, profileId, filter]);

  const requireProfile = () => {
    if (!profileId) {
      throw new Error(i18n.t('errors.noProfile'));
    }
    return profileId;
  };

  return {
    items,
    filter,
    loading,
    error,
    profileId,
    applyFilter,
    refresh,
    create: async (input: NewBacklogInput) => {
      const pid = requireProfile();
      await dispatch(saveBacklog({ profileId: pid, input: { ...input, profileId: pid } })).unwrap();
    },
    update: async (id: string, input: UpdateBacklogInput) => {
      await dispatch(saveBacklog({ profileId: requireProfile(), id, input })).unwrap();
    },
    remove: async (id: string) => {
      await dispatch(removeBacklog({ profileId: requireProfile(), id })).unwrap();
    },
  };
}
