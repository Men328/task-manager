import { useAppSelector } from './hooks';

export function useSidebarCounts() {
  const boardTasks = useAppSelector((state) => state.board.tasks.length);
  const boardStatuses = useAppSelector((state) => state.board.statuses.length);
  const pageTasks = useAppSelector((state) => state.statusesPage.tasks.length);
  const pageStatuses = useAppSelector((state) => state.statusesPage.statuses.length);

  return {
    tasks: boardTasks || pageTasks,
    statuses: boardStatuses || pageStatuses,
  };
}
