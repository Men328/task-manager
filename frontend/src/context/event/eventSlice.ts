import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  createEvent as createEventApi,
  deleteEvent as deleteEventApi,
  listEvents,
  updateEvent as updateEventApi,
} from '../../api/event';
import i18n from '../../i18n';
import type { CreateEventInput, Event, UpdateEventInput } from '../../types';
import type { RootState } from '../store';

export const fetchEvents = createAsyncThunk<
  Event[],
  { profileId: string },
  { rejectValue: string }
>('event/fetchEvents', async (params, { rejectWithValue }) => {
  try {
    return await listEvents({ profileId: params.profileId });
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const saveEvent = createAsyncThunk<
  void,
  { profileId: string; id?: string; input: CreateEventInput | UpdateEventInput },
  { state: RootState; rejectValue: string }
>('event/saveEvent', async ({ profileId, id, input }, { dispatch, rejectWithValue }) => {
  try {
    if (id) {
      await updateEventApi(id, input as UpdateEventInput);
    } else {
      await createEventApi({ ...(input as CreateEventInput), profileId });
    }
    await dispatch(fetchEvents({ profileId })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const removeEvent = createAsyncThunk<
  void,
  { profileId: string; id: string },
  { state: RootState; rejectValue: string }
>('event/removeEvent', async ({ profileId, id }, { dispatch, rejectWithValue }) => {
  try {
    await deleteEventApi(id);
    await dispatch(fetchEvents({ profileId })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface EventState {
  events: Event[];
  loading: boolean;
  error: string | null;
}

const initialState: EventState = {
  events: [],
  loading: false,
  error: null,
};

const eventSlice = createSlice({
  name: 'event',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchEvents.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchEvents.fulfilled, (state, action) => {
        state.loading = false;
        state.events = action.payload;
      })
      .addCase(fetchEvents.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadEvents');
      });
  },
});

export const eventReducer = eventSlice.reducer;

export const selectEvents = (state: RootState) => state.event.events;
export const selectEventLoading = (state: RootState) => state.event.loading;
export const selectEventError = (state: RootState) => state.event.error;
