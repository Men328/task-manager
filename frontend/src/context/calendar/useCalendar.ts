import { useCallback } from 'react';

import i18n from '../../i18n';
import type { CreateScheduleInput, UpdateScheduleInput } from '../../types';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchSchedules,
  removeSchedule,
  saveSchedule,
  selectCalendarError,
  selectCalendarLoading,
  selectCalendarRange,
  selectSchedules,
} from './calendarSlice';

export type NewScheduleInput = Omit<CreateScheduleInput, 'profileId'>;

export function useCalendar() {
  const dispatch = useAppDispatch();
  const schedules = useAppSelector(selectSchedules);
  const range = useAppSelector(selectCalendarRange);
  const loading = useAppSelector(selectCalendarLoading);
  const error = useAppSelector(selectCalendarError);
  const profileId = useAppSelector((state) => state.session.profileId);

  const loadRange = useCallback(
    (from: string, to: string) => {
      if (profileId) {
        void dispatch(fetchSchedules({ profileId, from, to }));
      }
    },
    [dispatch, profileId],
  );

  const refresh = useCallback(() => {
    if (profileId) {
      void dispatch(fetchSchedules({ profileId, from: range.from, to: range.to }));
    }
  }, [dispatch, profileId, range.from, range.to]);

  const requireProfile = () => {
    if (!profileId) {
      throw new Error(i18n.t('errors.noProfile'));
    }
    return profileId;
  };

  return {
    schedules,
    loading,
    error,
    profileId,
    loadRange,
    refresh,
    create: async (input: NewScheduleInput) => {
      const pid = requireProfile();
      await dispatch(saveSchedule({ profileId: pid, input: { ...input, profileId: pid } })).unwrap();
    },
    update: async (id: string, input: UpdateScheduleInput) => {
      await dispatch(saveSchedule({ profileId: requireProfile(), id, input })).unwrap();
    },
    remove: async (id: string) => {
      await dispatch(removeSchedule({ profileId: requireProfile(), id })).unwrap();
    },
  };
}
