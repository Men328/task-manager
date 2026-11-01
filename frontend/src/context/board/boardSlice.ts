import { createAction, createAsyncThunk, createSelector, createSlice } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  changeTaskStatus,
  createTask,
  deleteTask,
  updateTask,
  listStatuses,
  listTasks,
  listTransitions,
  seedDefaultStatuses,
} from '../../api/task';
import i18n from '../../i18n';
import { isDoneCategory } from '../../lib/tokens';
import type { CreateTaskInput, StatusTransition, Task, TaskStatus } from '../../types';
import type { RootState } from '../store';

export type NewTaskInput = Omit<CreateTaskInput, 'profileId'>;

interface BoardData {
  statuses: TaskStatus[];
  transitions: StatusTransition[];
  tasks: Task[];
}

const optimisticMove = createAction<{ taskId: string; statusId: string }>('board/optimisticMove');

interface FetchBoardArgs {
  profileId: string;
  /** null = chưa có project hợp lệ -> không tải task. */
  projectId: string | null;
}

/** Project đang chọn; dùng cho các thunk refetch sau khi ghi. */
const currentProjectId = (state: RootState) => state.project.currentId;

export const fetchBoard = createAsyncThunk<BoardData, FetchBoardArgs, { rejectValue: string }>(
  'board/fetchBoard',
  async ({ profileId, projectId }, { rejectWithValue }) => {
    try {
      const [statuses, transitions, tasks] = await Promise.all([
        listStatuses(profileId),
        listTransitions(profileId),
        projectId
          ? listTasks({ profileId, projectId, includeSubtasks: true })
          : Promise.resolve<Task[]>([]),
      ]);
      return { statuses, transitions, tasks };
    } catch (cause) {
      return rejectWithValue(getErrorMessage(cause));
    }
  },
);

export const createTaskAndRefresh = createAsyncThunk<
  void,
  { profileId: string; input: NewTaskInput },
  { state: RootState; rejectValue: string }
>('board/createTask', async ({ profileId, input }, { dispatch, getState, rejectWithValue }) => {
  try {
    await createTask({ ...input, profileId });
    await dispatch(fetchBoard({ profileId, projectId: currentProjectId(getState()) })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const moveTask = createAsyncThunk<
  void,
  { profileId: string; taskId: string; statusId: string; note?: string },
  { state: RootState; rejectValue: string }
>('board/moveTask', async (args, { dispatch, getState, rejectWithValue }) => {
  const previous = getState().board.tasks.find((task) => task.id === args.taskId);
  if (!previous || previous.statusId === args.statusId) {
    return;
  }

  dispatch(optimisticMove({ taskId: args.taskId, statusId: args.statusId }));

  try {
    await changeTaskStatus(args.taskId, args.statusId, args.note);
    await dispatch(
      fetchBoard({ profileId: args.profileId, projectId: currentProjectId(getState()) }),
    ).unwrap();
  } catch (cause) {
    dispatch(optimisticMove({ taskId: args.taskId, statusId: previous.statusId }));
    return rejectWithValue(getErrorMessage(cause));
  }
});

/** Chuyển task sang project khác rồi tải lại board của project đang chọn. */
export const moveTaskToProject = createAsyncThunk<
  void,
  { profileId: string; taskId: string; projectId: string },
  { state: RootState; rejectValue: string }
>('board/moveToProject', async ({ profileId, taskId, projectId }, { dispatch, getState, rejectWithValue }) => {
  try {
    await updateTask(taskId, { projectId });
    await dispatch(fetchBoard({ profileId, projectId: currentProjectId(getState()) })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const removeTask = createAsyncThunk<
  void,
  { profileId: string; taskId: string },
  { state: RootState; rejectValue: string }
>('board/removeTask', async ({ profileId, taskId }, { dispatch, getState, rejectWithValue }) => {
  try {
    await deleteTask(taskId);
    await dispatch(fetchBoard({ profileId, projectId: currentProjectId(getState()) })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const seedBoardStatuses = createAsyncThunk<
  void,
  { profileId: string },
  { state: RootState; rejectValue: string }
>('board/seedStatuses', async ({ profileId }, { dispatch, getState, rejectWithValue }) => {
  try {
    await seedDefaultStatuses(profileId);
    await dispatch(fetchBoard({ profileId, projectId: currentProjectId(getState()) })).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export interface BoardState {
  statuses: TaskStatus[];
  transitions: StatusTransition[];
  tasks: Task[];
  loading: boolean;
  error: string | null;
  /** requestId của fetchBoard mới nhất; response cũ hơn bị bỏ qua. */
  latestRequestId: string | null;
}

const initialState: BoardState = {
  statuses: [],
  transitions: [],
  tasks: [],
  loading: false,
  error: null,
  latestRequestId: null,
};

const boardSlice = createSlice({
  name: 'board',
  initialState,
  reducers: {},
  extraReducers: (builder) => {
    builder
      .addCase(fetchBoard.pending, (state, action) => {
        state.latestRequestId = action.meta.requestId;
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchBoard.fulfilled, (state, action) => {
        if (action.meta.requestId !== state.latestRequestId) {
          return;
        }
        state.loading = false;
        state.statuses = action.payload.statuses;
        state.transitions = action.payload.transitions;
        state.tasks = action.payload.tasks;
      })
      .addCase(fetchBoard.rejected, (state, action) => {
        if (action.meta.requestId !== state.latestRequestId) {
          return;
        }
        state.loading = false;
        state.error = action.payload ?? action.error.message ?? i18n.t('errors.loadBoard');
      })
      .addCase(optimisticMove, (state, action) => {
        const { taskId, statusId } = action.payload;
        state.tasks = state.tasks.map((task) =>
          task.id === taskId ? { ...task, statusId } : task,
        );
      });
  },
});

export const boardReducer = boardSlice.reducer;

export const selectStatuses = (state: RootState) => state.board.statuses;
export const selectTransitions = (state: RootState) => state.board.transitions;
export const selectTasks = (state: RootState) => state.board.tasks;
export const selectBoardLoading = (state: RootState) => state.board.loading;
export const selectBoardError = (state: RootState) => state.board.error;

export const selectStatusById = createSelector([selectStatuses], (statuses) => {
  const map = new Map<string, TaskStatus>();
  statuses.forEach((status) => map.set(status.id, status));
  return map;
});

export const selectTaskById = createSelector([selectTasks], (tasks) => {
  const map = new Map<string, Task>();
  tasks.forEach((task) => map.set(task.id, task));
  return map;
});

export const selectIsStatusDone = (statusId: string) => (state: RootState) =>
  isDoneCategory(state.board.statuses.find((status) => status.id === statusId)?.category);
