import { createAction, createAsyncThunk, createSelector, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import {
  changeTaskStatus,
  createTask,
  deleteTask,
  listStatuses,
  listTasks,
  listTransitions,
} from '../../api/task';
import { hasText } from '../../lib/format';
import i18n from '../../i18n';
import { PRIORITY_ORDER, isDoneCategory } from '../../lib/tokens';
import type { CreateTaskInput, StatusTransition, Task, TaskPriority, TaskStatus } from '../../types';
import { seedDefaultStatuses } from '../seed';
import type { RootState } from '../store';

export type NewTaskInput = Omit<CreateTaskInput, 'profileId'>;

interface BoardData {
  statuses: TaskStatus[];
  transitions: StatusTransition[];
  tasks: Task[];
}

const optimisticMove = createAction<{ taskId: string; statusId: string }>('board/optimisticMove');

export const fetchBoard = createAsyncThunk<BoardData, string, { rejectValue: string }>(
  'board/fetchBoard',
  async (profileId, { rejectWithValue }) => {
    try {
      const [statuses, transitions, tasks] = await Promise.all([
        listStatuses(profileId),
        listTransitions(profileId),
        listTasks({ profileId, includeSubtasks: true }),
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
  { rejectValue: string }
>('board/createTask', async ({ profileId, input }, { dispatch, rejectWithValue }) => {
  try {
    await createTask({ ...input, profileId });
    await dispatch(fetchBoard(profileId)).unwrap();
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
    await dispatch(fetchBoard(args.profileId)).unwrap();
  } catch (cause) {
    dispatch(optimisticMove({ taskId: args.taskId, statusId: previous.statusId }));
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const removeTask = createAsyncThunk<
  void,
  { profileId: string; taskId: string },
  { rejectValue: string }
>('board/removeTask', async ({ profileId, taskId }, { dispatch, rejectWithValue }) => {
  try {
    await deleteTask(taskId);
    await dispatch(fetchBoard(profileId)).unwrap();
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const seedBoardStatuses = createAsyncThunk<void, string, { rejectValue: string }>(
  'board/seedStatuses',
  async (profileId, { dispatch, rejectWithValue }) => {
    try {
      await seedDefaultStatuses(profileId);
      await dispatch(fetchBoard(profileId)).unwrap();
    } catch (cause) {
      return rejectWithValue(getErrorMessage(cause));
    }
  },
);

export interface BoardState {
  statuses: TaskStatus[];
  transitions: StatusTransition[];
  tasks: Task[];
  priorityFilter: TaskPriority[];
  loading: boolean;
  error: string | null;
}

const initialState: BoardState = {
  statuses: [],
  transitions: [],
  tasks: [],
  priorityFilter: [],
  loading: false,
  error: null,
};

const boardSlice = createSlice({
  name: 'board',
  initialState,
  reducers: {
    togglePriority(state, action: PayloadAction<TaskPriority>) {
      const priority = action.payload;
      state.priorityFilter = state.priorityFilter.includes(priority)
        ? state.priorityFilter.filter((item) => item !== priority)
        : [...state.priorityFilter, priority];
    },
    clearFilters(state) {
      state.priorityFilter = [];
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchBoard.pending, (state) => {
        state.loading = true;
        state.error = null;
      })
      .addCase(fetchBoard.fulfilled, (state, action) => {
        state.loading = false;
        state.statuses = action.payload.statuses;
        state.transitions = action.payload.transitions;
        state.tasks = action.payload.tasks;
      })
      .addCase(fetchBoard.rejected, (state, action) => {
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

export const { togglePriority, clearFilters } = boardSlice.actions;
export const boardReducer = boardSlice.reducer;

export const selectStatuses = (state: RootState) => state.board.statuses;
export const selectTransitions = (state: RootState) => state.board.transitions;
export const selectTasks = (state: RootState) => state.board.tasks;
export const selectPriorityFilter = (state: RootState) => state.board.priorityFilter;
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

export const selectCardsByStatus = createSelector(
  [selectStatuses, selectTasks, selectPriorityFilter, (state: RootState) => state.session.query],
  (statuses, tasks, priorityFilter, query) => {
    const map = new Map<string, Task[]>();
    statuses.forEach((status) => map.set(status.id, []));

    const needle = query.trim().toLowerCase();

    tasks.forEach((task) => {
      if (hasText(task.parentTaskId)) {
        return;
      }
      const bucket = map.get(task.statusId);
      if (!bucket) {
        return;
      }
      if (priorityFilter.length > 0 && !priorityFilter.includes(task.priority)) {
        return;
      }
      if (needle.length > 0) {
        const haystack = `${task.title} ${task.description ?? ''}`.toLowerCase();
        if (!haystack.includes(needle)) {
          return;
        }
      }
      bucket.push(task);
    });

    map.forEach((bucket) => {
      bucket.sort((a, b) => {
        const position = (a.position ?? 0) - (b.position ?? 0);
        if (position !== 0) {
          return position;
        }
        const priority = PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority];
        if (priority !== 0) {
          return priority;
        }
        return (a.createdAt ?? '').localeCompare(b.createdAt ?? '');
      });
    });

    return map;
  },
);

export const selectVisibleTaskCount = createSelector([selectCardsByStatus], (cardsByStatus) =>
  Array.from(cardsByStatus.values()).reduce((total, bucket) => total + bucket.length, 0),
);

export const selectIsStatusDone = (statusId: string) => (state: RootState) =>
  isDoneCategory(state.board.statuses.find((status) => status.id === statusId)?.category);
