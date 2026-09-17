import { useMemo } from 'react';

import type { CreateProfileInput } from '../../types';
import { useAppDispatch, useAppSelector } from '../hooks';
import { createProfileAndRefresh, fetchProfiles, selectProfile, setQuery } from './sessionSlice';

export function useSession() {
  const dispatch = useAppDispatch();
  const profiles = useAppSelector((state) => state.session.profiles);
  const profileId = useAppSelector((state) => state.session.profileId);
  const query = useAppSelector((state) => state.session.query);
  const loading = useAppSelector((state) => state.session.loading);
  const error = useAppSelector((state) => state.session.error);

  const profile = useMemo(
    () => profiles.find((item) => item.id === profileId) ?? null,
    [profiles, profileId],
  );

  return {
    profiles,
    profile,
    profileId,
    query,
    loading,
    error,
    selectProfile: (id: string) => dispatch(selectProfile(id)),
    setQuery: (value: string) => dispatch(setQuery(value)),
    refresh: () => {
      void dispatch(fetchProfiles());
    },
    createProfile: async (input: CreateProfileInput) => {
      await dispatch(createProfileAndRefresh(input)).unwrap();
    },
  };
}
