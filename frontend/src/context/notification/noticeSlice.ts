import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  countUnreadNotices,
  listNotices,
  markAllNoticesRead,
  markNoticesRead as markNoticesReadApi,
} from '../../api/notification';
import i18n from '../../i18n';
import type { Notice } from '../../types';

interface SessionTokenState {
  session: { token: string | null };
}

export const fetchNotices = createAsyncThunk<
  Notice[],
  void,
  { state: SessionTokenState; rejectValue: string }
>('notices/fetch', async (_arg, { getState, rejectWithValue }) => {
  const token = getState().session.token;
  if (!token) {
    return [];
  }
  try {
    return await listNotices(token, { limit: 20 });
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const fetchUnreadCount = createAsyncThunk<number, void, { state: SessionTokenState }>(
  'notices/fetchUnreadCount',
  async (_arg, { getState }) => {
    const token = getState().session.token;
    if (!token) {
      return 0;
    }
    return countUnreadNotices(token);
  },
);

export const markNoticesRead = createAsyncThunk<
  number,
  string[],
  { state: SessionTokenState; rejectValue: string }
>('notices/markRead', async (ids, { getState, rejectWithValue }) => {
  const token = getState().session.token;
  if (!token) {
    return 0;
  }
  try {
    return await markNoticesReadApi(token, ids);
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const markAllNoticesReadThunk = createAsyncThunk<
  number,
  void,
  { state: SessionTokenState; rejectValue: string }
>('notices/markAllRead', async (_arg, { getState, rejectWithValue }) => {
  const token = getState().session.token;
  if (!token) {
    return 0;
  }
  try {
    return await markAllNoticesRead(token);
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface NoticeState {
  items: Notice[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
}

const initialState: NoticeState = {
  items: [],
  unreadCount: 0,
  loading: false,
  error: null,
};

const noticeSlice = createSlice({
  name: 'notices',
  initialState,
  reducers: {
    clearNotices(state) {
      state.items = [];
      state.unreadCount = 0;
      state.loading = false;
      state.error = null;
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchNotices.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchNotices.fulfilled, (state, action) => {
        state.loading = false;
        state.items = action.payload;
      })
      .addCase(fetchNotices.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? i18n.t('notice.loadError');
      })
      .addCase(fetchUnreadCount.fulfilled, (state, action) => {
        state.unreadCount = action.payload;
      })
      .addCase(markNoticesRead.fulfilled, (state, action) => {
        const ids = action.meta.arg;
        const markAll = ids.includes('*');
        state.items = state.items.map((notice) =>
          markAll || ids.includes(notice.id) ? { ...notice, isRead: true } : notice,
        );
        if (markAll) {
          state.unreadCount = 0;
        } else {
          state.unreadCount = Math.max(0, state.unreadCount - action.payload);
        }
      })
      .addCase(markAllNoticesReadThunk.fulfilled, (state) => {
        state.items = state.items.map((notice) => ({ ...notice, isRead: true }));
        state.unreadCount = 0;
        state.error = null;
      })
      .addCase(markAllNoticesReadThunk.rejected, (state, action) => {
        state.error = action.payload ?? i18n.t('notice.loadError');
      });
  },
});

export const { clearNotices } = noticeSlice.actions;
export const noticeReducer = noticeSlice.reducer;
