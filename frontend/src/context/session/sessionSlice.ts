import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { getErrorCode, getErrorMessage } from '../../api/client';
import { createProfile, getCurrentProfile, listProfiles } from '../../api/identity';
import i18n from '../../i18n';
import { clearToken, readToken, writeToken } from '../../lib/session';
import type { CreateProfileInput, Profile } from '../../types';

export const fetchProfiles = createAsyncThunk<Profile[], void, { rejectValue: string }>(
  'session/fetchProfiles',
  async (_arg, { rejectWithValue }) => {
    try {
      return await listProfiles();
    } catch (cause) {
      return rejectWithValue(getErrorMessage(cause));
    }
  },
);

export const createProfileAndRefresh = createAsyncThunk<
  void,
  CreateProfileInput,
  { rejectValue: string }
>('session/createProfile', async (input, { dispatch, rejectWithValue }) => {
  try {
    await createProfile(input);
    await dispatch(fetchProfiles()).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const restoreSession = createAsyncThunk<
  Profile,
  void,
  { rejectValue: string; state: { session: SessionState } }
>('session/restore', async (_arg, { getState, rejectWithValue }) => {
  const token = getState().session.token;
  if (!token) {
    return rejectWithValue('anonymous');
  }
  try {
    return await getCurrentProfile(token);
  } catch (cause) {
    return rejectWithValue(getErrorCode(cause) ?? getErrorMessage(cause));
  }
});

export interface SessionState {
  profiles: Profile[];
  profileId: string | null;
  query: string;
  token: string | null;
  loading: boolean;
  error: string | null;
}

const initialState: SessionState = {
  profiles: [],
  profileId: null,
  query: '',
  token: readToken(),
  loading: true,
  error: null,
};

const sessionSlice = createSlice({
  name: 'session',
  initialState,
  reducers: {
    selectProfile(state, action: PayloadAction<string>) {
      state.profileId = action.payload;
    },
    setQuery(state, action: PayloadAction<string>) {
      state.query = action.payload;
    },
    setToken(state, action: PayloadAction<string>) {
      state.token = action.payload;
      state.loading = true;
      state.error = null;
      writeToken(action.payload);
    },
    signOut(state) {
      state.token = null;
      state.profiles = [];
      state.profileId = null;
      state.query = '';
      state.loading = false;
      state.error = null;
      clearToken();
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchProfiles.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchProfiles.fulfilled, (state, action) => {
        state.loading = false;
        state.profiles = action.payload;
        const current = state.profileId;
        if (!current || !action.payload.some((profile) => profile.id === current)) {
          state.profileId = action.payload[0]?.id ?? null;
        }
      })
      .addCase(fetchProfiles.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadProfiles');
      })
      .addCase(restoreSession.pending, (state) => {
        state.loading = true;
      })
      .addCase(restoreSession.fulfilled, (state, action) => {
        state.loading = false;
        state.error = null;
        state.profiles = [action.payload];
        state.profileId = action.payload.id;
      })
      .addCase(restoreSession.rejected, (state) => {
        state.loading = false;
        state.token = null;
        state.profiles = [];
        state.profileId = null;
        state.error = null;
        clearToken();
      });
  },
});

export const { selectProfile, setQuery, setToken, signOut } = sessionSlice.actions;
export const sessionReducer = sessionSlice.reducer;
