import { createAsyncThunk, createSelector, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import { createWorkspace, listWorkspaces, updateWorkspace } from '../../api/workspace';
import { clearWorkspaceId, readWorkspaceId, uniqueWorkspaceSlug, writeWorkspaceId } from '../../lib/workspace';
import type { UpdateWorkspaceInput, Workspace } from '../../types';
import type { RootState } from '../store';

export const fetchWorkspaces = createAsyncThunk<Workspace[], string, { rejectValue: string }>(
  'workspace/fetchWorkspaces',
  async (profileId, { rejectWithValue }) => {
    try {
      return await listWorkspaces(profileId);
    } catch (cause) {
      return rejectWithValue(getErrorMessage(cause));
    }
  },
);

export const createWorkspaceAndSelect = createAsyncThunk<
  Workspace,
  { profileId: string; name: string; description?: string },
  { rejectValue: string }
>('workspace/createWorkspace', async ({ profileId, name, description }, { rejectWithValue }) => {
  try {
    return await createWorkspace({
      ownerProfileId: profileId,
      name,
      slug: uniqueWorkspaceSlug(name),
      description,
    });
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const updateWorkspaceAndStore = createAsyncThunk<
  Workspace,
  { id: string; input: UpdateWorkspaceInput },
  { rejectValue: string }
>('workspace/updateWorkspace', async ({ id, input }, { rejectWithValue }) => {
  try {
    return await updateWorkspace(id, input);
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface WorkspaceState {
  items: Workspace[];
  selectedId: string | null;
  loading: boolean;
  saving: boolean;
  error: string | null;
}

const initialState: WorkspaceState = {
  items: [],
  selectedId: readWorkspaceId(),
  loading: false,
  saving: false,
  error: null,
};

function reconcileSelection(state: WorkspaceState): void {
  const current = state.selectedId;
  if (current && state.items.some((workspace) => workspace.id === current)) {
    return;
  }
  const fallback = state.items.find((workspace) => workspace.isDefault) ?? state.items[0] ?? null;
  state.selectedId = fallback?.id ?? null;
  if (fallback) {
    writeWorkspaceId(fallback.id);
  } else {
    clearWorkspaceId();
  }
}

const workspaceSlice = createSlice({
  name: 'workspace',
  initialState,
  reducers: {
    selectWorkspace(state, action: PayloadAction<string>) {
      state.selectedId = action.payload;
      writeWorkspaceId(action.payload);
    },
    clearWorkspaceSelection(state) {
      state.selectedId = null;
      clearWorkspaceId();
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchWorkspaces.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchWorkspaces.fulfilled, (state, action) => {
        state.loading = false;
        state.items = action.payload;
        reconcileSelection(state);
      })
      .addCase(fetchWorkspaces.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? null;
      })
      .addCase(createWorkspaceAndSelect.pending, (state) => {
        state.saving = true;
        state.error = null;
      })
      .addCase(createWorkspaceAndSelect.fulfilled, (state, action) => {
        state.saving = false;
        state.items = [...state.items, action.payload];
        state.selectedId = action.payload.id;
        writeWorkspaceId(action.payload.id);
      })
      .addCase(createWorkspaceAndSelect.rejected, (state, action) => {
        state.saving = false;
        state.error = action.payload ?? action.error.message ?? null;
      })
      .addCase(updateWorkspaceAndStore.pending, (state) => {
        state.saving = true;
        state.error = null;
      })
      .addCase(updateWorkspaceAndStore.fulfilled, (state, action) => {
        state.saving = false;
        state.items = state.items.map((workspace) =>
          workspace.id === action.payload.id ? action.payload : workspace,
        );
      })
      .addCase(updateWorkspaceAndStore.rejected, (state, action) => {
        state.saving = false;
        state.error = action.payload ?? action.error.message ?? null;
      });
  },
});

export const { selectWorkspace, clearWorkspaceSelection } = workspaceSlice.actions;
export const workspaceReducer = workspaceSlice.reducer;

export const selectWorkspaces = (state: RootState) => state.workspace.items;
export const selectSelectedWorkspaceId = (state: RootState) => state.workspace.selectedId;
export const selectWorkspaceLoading = (state: RootState) => state.workspace.loading;
export const selectWorkspaceSaving = (state: RootState) => state.workspace.saving;
export const selectWorkspaceError = (state: RootState) => state.workspace.error;

export const selectSelectedWorkspace = createSelector(
  [selectWorkspaces, selectSelectedWorkspaceId],
  (items, selectedId) => items.find((workspace) => workspace.id === selectedId) ?? null,
);
