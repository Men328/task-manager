import { useCallback } from 'react';

import i18n from '../../i18n';
import type { CreateEventInput, UpdateEventInput } from '../../types';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchEvents,
  removeEvent,
  saveEvent,
  selectEventError,
  selectEventLoading,
  selectEvents,
} from './eventSlice';

export type NewEventInput = Omit<CreateEventInput, 'profileId'>;

export function useEvent() {
  const dispatch = useAppDispatch();
  const events = useAppSelector(selectEvents);
  const loading = useAppSelector(selectEventLoading);
  const error = useAppSelector(selectEventError);
  const profileId = useAppSelector((state) => state.session.profileId);

  const refresh = useCallback(() => {
    if (profileId) {
      void dispatch(fetchEvents({ profileId }));
    }
  }, [dispatch, profileId]);

  const requireProfile = () => {
    if (!profileId) {
      throw new Error(i18n.t('errors.noProfile'));
    }
    return profileId;
  };

  return {
    events,
    loading,
    error,
    profileId,
    refresh,
    create: async (input: NewEventInput) => {
      const pid = requireProfile();
      await dispatch(saveEvent({ profileId: pid, input: { ...input, profileId: pid } })).unwrap();
    },
    update: async (id: string, input: UpdateEventInput) => {
      await dispatch(saveEvent({ profileId: requireProfile(), id, input })).unwrap();
    },
    remove: async (id: string) => {
      await dispatch(removeEvent({ profileId: requireProfile(), id })).unwrap();
    },
  };
}
