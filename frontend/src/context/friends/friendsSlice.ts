import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import { listFriendRequests, listFriends } from '../../api/social';
import { signOut } from '../session/sessionSlice';
import type { Friend, FriendRequest } from '../../types';

interface SessionTokenState {
  session: { token: string | null };
}

interface FriendsData {
  friends: Friend[];
  incoming: FriendRequest[];
  outgoing: FriendRequest[];
}

export const fetchFriends = createAsyncThunk<
  FriendsData,
  void,
  { state: SessionTokenState; rejectValue: string }
>('friends/fetch', async (_arg, { getState, rejectWithValue }) => {
  const token = getState().session.token;
  if (!token) {
    return { friends: [], incoming: [], outgoing: [] };
  }
  try {
    const [friends, requests] = await Promise.all([listFriends(token), listFriendRequests(token)]);
    return { friends, ...requests };
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface FriendsState extends FriendsData {
  loading: boolean;
  error: string | null;
}

const initialState: FriendsState = {
  friends: [],
  incoming: [],
  outgoing: [],
  loading: false,
  error: null,
};

const friendsSlice = createSlice({
  name: 'friends',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(signOut, () => initialState)
      .addCase(fetchFriends.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchFriends.fulfilled, (state, action) => {
        state.loading = false;
        state.friends = action.payload.friends;
        state.incoming = action.payload.incoming;
        state.outgoing = action.payload.outgoing;
      })
      .addCase(fetchFriends.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? null;
      });
  },
});

export const friendsReducer = friendsSlice.reducer;
