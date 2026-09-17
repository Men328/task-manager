import { useCallback } from 'react';

import { isDoneCategory } from '../../lib/tokens';
import i18n from '../../i18n';
import type { TaskPriority } from '../../types';
import { useAppDispatch, useAppSelector } from '../hooks';
import { setQuery } from '../session/sessionSlice';
import {
  clearFilters,
  createTaskAndRefresh,
  fetchBoard,
  moveTask as moveTaskThunk,
  removeTask as removeTaskThunk,
  seedBoardStatuses,
  selectBoardError,
  selectBoardLoading,
  selectCardsByStatus,
  selectPriorityFilter,
  selectStatusById,
  selectStatuses,
  selectTaskById,
  selectTasks,
  selectTransitions,
  selectVisibleTaskCount,
  togglePriority,
} from './boardSlice';
import type { NewTaskInput } from './boardSlice';

export function useBoard() {
  const dispatch = useAppDispatch();
  const statuses = useAppSelector(selectStatuses);
  const transitions = useAppSelector(selectTransitions);
  const tasks = useAppSelector(selectTasks);
  const statusById = useAppSelector(selectStatusById);
  const taskById = useAppSelector(selectTaskById);
  const cardsByStatus = useAppSelector(selectCardsByStatus);
  const visibleTaskCount = useAppSelector(selectVisibleTaskCount);
  const priorityFilter = useAppSelector(selectPriorityFilter);
  const loading = useAppSelector(selectBoardLoading);
  const error = useAppSelector(selectBoardError);
  const profileId = useAppSelector((state) => state.session.profileId);

  const isStatusDone = useCallback(
    (statusId: string) => isDoneCategory(statusById.get(statusId)?.category),
    [statusById],
  );

  const allowedTargets = useCallback(
    (statusId: string) => {
      const targets = new Set<string>();
      transitions.forEach((transition) => {
        if (transition.isActive && transition.fromStatusId === statusId) {
          targets.add(transition.toStatusId);
        }
      });
      return targets;
    },
    [transitions],
  );

  return {
    statuses,
    transitions,
    tasks,
    statusById,
    taskById,
    cardsByStatus,
    visibleTaskCount,
    priorityFilter,
    loading,
    error,
    isStatusDone,
    allowedTargets,
    togglePriority: (priority: TaskPriority) => dispatch(togglePriority(priority)),
    clearFilters: () => {
      dispatch(clearFilters());
      dispatch(setQuery(''));
    },
    refresh: () => {
      if (profileId) {
        void dispatch(fetchBoard(profileId));
      }
    },
    addTask: async (input: NewTaskInput) => {
      if (!profileId) {
        throw new Error(i18n.t('errors.noProfile'));
      }
      await dispatch(createTaskAndRefresh({ profileId, input })).unwrap();
    },
    moveTask: async (taskId: string, statusId: string, note?: string) => {
      if (!profileId) {
        return;
      }
      await dispatch(moveTaskThunk({ profileId, taskId, statusId, note })).unwrap();
    },
    removeTask: async (taskId: string) => {
      if (!profileId) {
        return;
      }
      await dispatch(removeTaskThunk({ profileId, taskId })).unwrap();
    },
    seedDefaultStatuses: async () => {
      if (!profileId) {
        throw new Error(i18n.t('errors.noProfile'));
      }
      await dispatch(seedBoardStatuses(profileId)).unwrap();
    },
  };
}
