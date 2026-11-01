import { useMemo } from 'react';

import { activeFilterCount, matchesBoardFilters, sortBoardTasks } from '../../lib/boardFilters';
import { hasText } from '../../lib/format';
import { PRIORITY_ORDER } from '../../lib/tokens';
import type { Task } from '../../types';
import { useAppSelector } from '../hooks';
import { selectStatuses, selectTasks } from './boardSlice';
import { useBoardFilters } from './useBoardFilters';

/**
 * Dữ liệu Board sau khi áp bộ lọc trên URL + ô tìm kiếm: cột Kanban (chỉ task gốc,
 * thứ tự kéo-thả) và hàng của bảng (mọi task, theo kiểu sắp xếp đã chọn).
 */
export function useBoardView() {
  const statuses = useAppSelector(selectStatuses);
  const tasks = useAppSelector(selectTasks);
  const query = useAppSelector((state) => state.session.query);
  const { filters } = useBoardFilters();

  return useMemo(() => {
    const closed = new Set(
      statuses
        .filter((status) => status.category === 'TASK_STATUS_CATEGORY_DONE' || status.category === 'TASK_STATUS_CATEGORY_CANCELLED')
        .map((status) => status.id),
    );
    const matching = tasks.filter((task) =>
      matchesBoardFilters(task, filters, { isClosed: (statusId) => closed.has(statusId), query }),
    );

    const cardsByStatus = new Map<string, Task[]>();
    statuses.forEach((status) => cardsByStatus.set(status.id, []));
    matching.forEach((task) => {
      if (!hasText(task.parentTaskId)) {
        cardsByStatus.get(task.statusId)?.push(task);
      }
    });
    cardsByStatus.forEach((bucket) =>
      bucket.sort(
        (a, b) =>
          (a.position ?? 0) - (b.position ?? 0) ||
          PRIORITY_ORDER[a.priority] - PRIORITY_ORDER[b.priority] ||
          (a.createdAt ?? '').localeCompare(b.createdAt ?? ''),
      ),
    );

    const kanbanCount = Array.from(cardsByStatus.values()).reduce((total, bucket) => total + bucket.length, 0);
    return {
      filters,
      cardsByStatus,
      kanbanCount,
      tableTasks: sortBoardTasks(matching, filters.sort),
      hasFilter: activeFilterCount(filters) > 0 || query.trim().length > 0,
    };
  }, [statuses, tasks, query, filters]);
}
