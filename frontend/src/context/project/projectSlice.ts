import { createAsyncThunk, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  archiveProject,
  createProject,
  listProjects,
  unarchiveProject,
  updateProject,
} from '../../api/project';
import i18n from '../../i18n';
import type { CreateProjectInput, Project, UpdateProjectInput } from '../../types';
import type { RootState } from '../store';

const STORAGE_KEY = 'tm.currentProjectId';

function readStoredId(): string | null {
  try {
    return window.localStorage.getItem(STORAGE_KEY);
  } catch {
    return null;
  }
}

function writeStoredId(id: string | null): void {
  try {
    if (id) {
      window.localStorage.setItem(STORAGE_KEY, id);
    } else {
      window.localStorage.removeItem(STORAGE_KEY);
    }
  } catch {
    // Bỏ qua: storage bị chặn thì chỉ mất việc nhớ lựa chọn.
  }
}

/** Luôn gồm cả project đã lưu trữ để trang quản lý hiển thị được tab "Đã lưu trữ". */
export const fetchProjects = createAsyncThunk<Project[], string, { rejectValue: string }>(
  'project/fetchProjects',
  async (profileId, { rejectWithValue }) => {
    try {
      return await listProjects(profileId, true);
    } catch (cause) {
      return rejectWithValue(getErrorMessage(cause));
    }
  },
);

export const saveProject = createAsyncThunk<
  Project,
  { profileId: string; id?: string; input: CreateProjectInput | UpdateProjectInput },
  { rejectValue: string }
>('project/saveProject', async ({ profileId, id, input }, { dispatch, rejectWithValue }) => {
  try {
    const saved = id
      ? await updateProject(id, input as UpdateProjectInput)
      : await createProject({ ...(input as CreateProjectInput), profileId });
    await dispatch(fetchProjects(profileId)).unwrap();
    return saved;
  } catch (cause) {
    return rejectWithValue(typeof cause === 'string' ? cause : getErrorMessage(cause));
  }
});

export const setProjectArchived = createAsyncThunk<
  void,
  { id: string; archived: boolean },
  { state: RootState; rejectValue: string }
>('project/setArchived', async ({ id, archived }, { dispatch, getState, rejectWithValue }) => {
  try {
    const updated = archived ? await archiveProject(id) : await unarchiveProject(id);
    await dispatch(fetchProjects(updated.profileId || (getState().session.profileId ?? ''))).unwrap();
  } catch (cause) {
    return rejectWithValue(typeof cause === 'string' ? cause : getErrorMessage(cause));
  }
});

export interface ProjectState {
  items: Project[];
  currentId: string | null;
  loading: boolean;
  error: string | null;
}

const initialState: ProjectState = {
  items: [],
  currentId: readStoredId(),
  loading: false,
  error: null,
};

const projectSlice = createSlice({
  name: 'project',
  initialState,
  reducers: {
    setCurrentProject(state, action: PayloadAction<string>) {
      state.currentId = action.payload;
      writeStoredId(action.payload);
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchProjects.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchProjects.fulfilled, (state, action) => {
        state.loading = false;
        state.items = action.payload;
        const active = action.payload.filter((item) => !item.isArchived);
        if (!active.some((item) => item.id === state.currentId)) {
          const fallback = active.find((item) => item.isDefault) ?? active[0] ?? null;
          state.currentId = fallback?.id ?? null;
          writeStoredId(state.currentId);
        }
      })
      .addCase(fetchProjects.rejected, (state, action) => {
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadProjects');
      });
  },
});

export const { setCurrentProject } = projectSlice.actions;
export const projectReducer = projectSlice.reducer;

export const selectProjects = (state: RootState) => state.project.items;
export const selectProjectLoading = (state: RootState) => state.project.loading;
export const selectProjectError = (state: RootState) => state.project.error;
export const selectCurrentProjectId = (state: RootState) => state.project.currentId;
export const selectCurrentProject = (state: RootState): Project | null =>
  state.project.items.find((item) => item.id === state.project.currentId) ?? null;

let cachedItems: Project[] | null = null;
let cachedActive: Project[] = [];
/** Memo thủ công để selector trả cùng tham chiếu khi `items` không đổi. */
export const selectActiveProjects = (state: RootState): Project[] => {
  if (cachedItems !== state.project.items) {
    cachedItems = state.project.items;
    cachedActive = cachedItems.filter((item) => !item.isArchived);
  }
  return cachedActive;
};
