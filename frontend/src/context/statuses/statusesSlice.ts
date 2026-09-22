import { createAsyncThunk, createSelector, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import { listStatuses, listTasks, listTransitions } from '../../api/task';
import i18n from '../../i18n';
import type { StatusTransition, Task, TaskStatus } from '../../types';
import { seedDefaultStatuses } from '../seed';
import type { RootState } from '../store';

interface StatusesPageData {
  statuses: TaskStatus[];
  transitions: StatusTransition[];
  tasks: Task[];
}

interface FetchStatusesPageArgs {
  profileId: string;
  workspaceId: string | null;
}

export const fetchStatusesPage = createAsyncThunk<
  StatusesPageData,
  FetchStatusesPageArgs,
  { rejectValue: string }
>('statusesPage/fetch', async ({ profileId, workspaceId }, { rejectWithValue }) => {
  try {
    const [statuses, transitions, tasks] = await Promise.all([
      listStatuses(profileId),
      listTransitions(profileId),
      workspaceId ? listTasks({ profileId, workspaceId, includeSubtasks: true }) : Promise.resolve([]),
    ]);
    return { statuses, transitions, tasks };
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const seedStatusesPage = createAsyncThunk<
  void,
  FetchStatusesPageArgs,
  { rejectValue: string }
>('statusesPage/seedStatuses', async ({ profileId, workspaceId }, { dispatch, rejectWithValue }) => {
  try {
    await seedDefaultStatuses(profileId);
    await dispatch(fetchStatusesPage({ profileId, workspaceId })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface StatusesPageState {
  statuses: TaskStatus[];
  transitions: StatusTransition[];
  tasks: Task[];
  loading: boolean;
  error: string | null;
}

const initialState: StatusesPageState = {
  statuses: [],
  transitions: [],
  tasks: [],
  loading: false,
  error: null,
};

const statusesSlice = createSlice({
  name: 'statusesPage',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchStatusesPage.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchStatusesPage.fulfilled, (state, action) => {
        state.loading = false;
        state.statuses = action.payload.statuses;
        state.transitions = action.payload.transitions;
        state.tasks = action.payload.tasks;
      })
      .addCase(fetchStatusesPage.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadStatuses');
      });
  },
});

export const statusesPageReducer = statusesSlice.reducer;

export const selectPageStatuses = (state: RootState) => state.statusesPage.statuses;
export const selectPageTransitions = (state: RootState) => state.statusesPage.transitions;
export const selectPageTasks = (state: RootState) => state.statusesPage.tasks;
export const selectPageLoading = (state: RootState) => state.statusesPage.loading;
export const selectPageError = (state: RootState) => state.statusesPage.error;

export const selectPageStatusById = createSelector([selectPageStatuses], (statuses) => {
  const map = new Map<string, TaskStatus>();
  statuses.forEach((status) => map.set(status.id, status));
  return map;
});
