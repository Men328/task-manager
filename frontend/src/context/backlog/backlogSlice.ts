import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  createBacklog as createBacklogApi,
  deleteBacklog as deleteBacklogApi,
  listBacklogs,
  updateBacklog as updateBacklogApi,
} from '../../api/backlog';
import i18n from '../../i18n';
import type {
  Backlog,
  BacklogStatus,
  CreateBacklogInput,
  UpdateBacklogInput,
} from '../../types';
import type { RootState } from '../store';

export interface BacklogFilter {
  status?: BacklogStatus;
  category?: string;
}

interface FetchBacklogsArgs extends BacklogFilter {
  profileId: string;
}

export const fetchBacklogs = createAsyncThunk<
  Backlog[],
  FetchBacklogsArgs,
  { rejectValue: string }
>('backlog/fetchBacklogs', async (params, { rejectWithValue }) => {
  try {
    return await listBacklogs({
      profileId: params.profileId,
      status: params.status,
      category: params.category,
    });
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const saveBacklog = createAsyncThunk<
  void,
  { profileId: string; id?: string; input: CreateBacklogInput | UpdateBacklogInput },
  { state: RootState; rejectValue: string }
>('backlog/saveBacklog', async ({ profileId, id, input }, { dispatch, getState, rejectWithValue }) => {
  try {
    if (id) {
      await updateBacklogApi(id, input as UpdateBacklogInput);
    } else {
      await createBacklogApi({ ...(input as CreateBacklogInput), profileId });
    }
    const { filter } = getState().backlog;
    await dispatch(fetchBacklogs({ profileId, ...filter })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const removeBacklog = createAsyncThunk<
  void,
  { profileId: string; id: string },
  { state: RootState; rejectValue: string }
>('backlog/removeBacklog', async ({ profileId, id }, { dispatch, getState, rejectWithValue }) => {
  try {
    await deleteBacklogApi(id);
    const { filter } = getState().backlog;
    await dispatch(fetchBacklogs({ profileId, ...filter })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface BacklogState {
  items: Backlog[];
  filter: BacklogFilter;
  loading: boolean;
  error: string | null;
}

const initialState: BacklogState = {
  items: [],
  filter: {},
  loading: false,
  error: null,
};

const backlogSlice = createSlice({
  name: 'backlog',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchBacklogs.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchBacklogs.fulfilled, (state, action) => {
        state.loading = false;
        state.items = action.payload;
        state.filter = { status: action.meta.arg.status, category: action.meta.arg.category };
      })
      .addCase(fetchBacklogs.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadBacklog');
      });
  },
});

export const backlogReducer = backlogSlice.reducer;

export const selectBacklogItems = (state: RootState) => state.backlog.items;
export const selectBacklogFilter = (state: RootState) => state.backlog.filter;
export const selectBacklogLoading = (state: RootState) => state.backlog.loading;
export const selectBacklogError = (state: RootState) => state.backlog.error;
