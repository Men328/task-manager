import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  createSchedule as createScheduleApi,
  deleteSchedule as deleteScheduleApi,
  listSchedules,
  updateSchedule as updateScheduleApi,
} from '../../api/calendar';
import i18n from '../../i18n';
import type { CreateScheduleInput, Schedule, UpdateScheduleInput } from '../../types';
import type { RootState } from '../store';

export interface DateRange {
  from?: string;
  to?: string;
}

interface FetchSchedulesArgs {
  profileId: string;
  from?: string;
  to?: string;
}

export const fetchSchedules = createAsyncThunk<
  Schedule[],
  FetchSchedulesArgs,
  { rejectValue: string }
>('calendar/fetchSchedules', async (params, { rejectWithValue }) => {
  try {
    return await listSchedules({
      profileId: params.profileId,
      from: params.from,
      to: params.to,
    });
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const saveSchedule = createAsyncThunk<
  void,
  { profileId: string; id?: string; input: CreateScheduleInput | UpdateScheduleInput },
  { state: RootState; rejectValue: string }
>('calendar/saveSchedule', async ({ profileId, id, input }, { dispatch, getState, rejectWithValue }) => {
  try {
    if (id) {
      await updateScheduleApi(id, input as UpdateScheduleInput);
    } else {
      await createScheduleApi({ ...(input as CreateScheduleInput), profileId });
    }
    const { range } = getState().calendar;
    await dispatch(fetchSchedules({ profileId, from: range.from, to: range.to })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const removeSchedule = createAsyncThunk<
  void,
  { profileId: string; id: string },
  { state: RootState; rejectValue: string }
>('calendar/removeSchedule', async ({ profileId, id }, { dispatch, getState, rejectWithValue }) => {
  try {
    await deleteScheduleApi(id);
    const { range } = getState().calendar;
    await dispatch(fetchSchedules({ profileId, from: range.from, to: range.to })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface CalendarState {
  schedules: Schedule[];
  range: DateRange;
  loading: boolean;
  error: string | null;
}

const initialState: CalendarState = {
  schedules: [],
  range: {},
  loading: false,
  error: null,
};

const calendarSlice = createSlice({
  name: 'calendar',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchSchedules.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchSchedules.fulfilled, (state, action) => {
        state.loading = false;
        state.schedules = action.payload;
        state.range = { from: action.meta.arg.from, to: action.meta.arg.to };
      })
      .addCase(fetchSchedules.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadCalendar');
      });
  },
});

export const calendarReducer = calendarSlice.reducer;

export const selectSchedules = (state: RootState) => state.calendar.schedules;
export const selectCalendarRange = (state: RootState) => state.calendar.range;
export const selectCalendarLoading = (state: RootState) => state.calendar.loading;
export const selectCalendarError = (state: RootState) => state.calendar.error;
