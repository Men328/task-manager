import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import { createProfile, listProfiles } from '../../api/identity';
import i18n from '../../i18n';
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

export interface SessionState {
  profiles: Profile[];
  profileId: string | null;
  query: string;
  loading: boolean;
  error: string | null;
}

const initialState: SessionState = {
  profiles: [],
  profileId: null,
  query: '',
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
      });
  },
});

export const { selectProfile, setQuery } = sessionSlice.actions;
export const sessionReducer = sessionSlice.reducer;
