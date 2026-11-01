/** Bộ lọc + sắp xếp của Board, lưu trên URL để reload/chia sẻ vẫn giữ nguyên. */

import { agendaRanges } from './dateRanges';
import { PRIORITY_ORDER } from './tokens';
import type { Task, TaskPriority } from '../types';

export type DueFilter = 'all' | 'overdue' | 'today' | 'week' | 'none';
export type SortField = 'position' | 'due' | 'priority' | 'created' | 'updated';

export interface BoardSort {
  field: SortField;
  desc: boolean;
}

export interface BoardFilters {
  due: DueFilter;
  priorities: TaskPriority[];
  hideDone: boolean;
  sort: BoardSort;
}

export const DUE_FILTERS: DueFilter[] = ['all', 'overdue', 'today', 'week', 'none'];
export const SORT_FIELDS: SortField[] = ['position', 'due', 'priority', 'created', 'updated'];
export const DEFAULT_SORT: BoardSort = { field: 'position', desc: false };

const PRIORITY_PARAM: Record<string, TaskPriority> = {
  low: 'TASK_PRIORITY_LOW',
  medium: 'TASK_PRIORITY_MEDIUM',
  high: 'TASK_PRIORITY_HIGH',
  urgent: 'TASK_PRIORITY_URGENT',
};

const PARAM = { due: 'due', priorities: 'prio', hideDone: 'hideDone', sort: 'sort' } as const;

/** Đọc bộ lọc từ query string; giá trị lạ bị bỏ qua thay vì làm hỏng trang. */
export function parseBoardFilters(params: URLSearchParams): BoardFilters {
  const due = params.get(PARAM.due) as DueFilter | null;
  const priorities = (params.get(PARAM.priorities) ?? '')
    .split(',')
    .map((value) => PRIORITY_PARAM[value.trim().toLowerCase()])
    .filter((value): value is TaskPriority => Boolean(value));
  const [field, direction] = (params.get(PARAM.sort) ?? '').split(':');

  return {
    due: due && DUE_FILTERS.includes(due) ? due : 'all',
    priorities: [...new Set(priorities)],
    hideDone: params.get(PARAM.hideDone) === '1',
    sort: SORT_FIELDS.includes(field as SortField)
      ? { field: field as SortField, desc: direction === 'desc' }
      : DEFAULT_SORT,
  };
}

/** Ghi bộ lọc vào query string; giá trị mặc định thì xoá param cho URL gọn. Giữ nguyên param khác. */
export function writeBoardFilters(params: URLSearchParams, filters: BoardFilters): URLSearchParams {
  const next = new URLSearchParams(params);
  const set = (key: string, value: string | null) => (value ? next.set(key, value) : next.delete(key));
  const priorityParam = Object.entries(PRIORITY_PARAM)
    .filter(([, priority]) => filters.priorities.includes(priority))
    .map(([key]) => key)
    .join(',');
  const isDefaultSort = filters.sort.field === DEFAULT_SORT.field && filters.sort.desc === DEFAULT_SORT.desc;

  set(PARAM.due, filters.due === 'all' ? null : filters.due);
  set(PARAM.priorities, priorityParam || null);
  set(PARAM.hideDone, filters.hideDone ? '1' : null);
  set(PARAM.sort, isDefaultSort ? null : `${filters.sort.field}:${filters.sort.desc ? 'desc' : 'asc'}`);
  return next;
}

/** Số bộ lọc đang bật (không tính sắp xếp). */
export function activeFilterCount(filters: BoardFilters): number {
  return (filters.due === 'all' ? 0 : 1) + (filters.priorities.length > 0 ? 1 : 0) + (filters.hideDone ? 1 : 0);
}

export interface MatchContext {
  /** true nếu status thuộc nhóm done/cancelled. */
  isClosed: (statusId: string) => boolean;
  now?: Date;
  /** Chuỗi tìm kiếm ở ô search (so không phân biệt hoa thường). */
  query?: string;
}

/**
 * Task có khớp bộ lọc không. Mốc ngày là 0h giờ địa phương (task chỉ có ngày lưu ở 0h),
 * giống màn agenda: "Quá hạn" = hạn trước hôm nay và task chưa đóng.
 */
export function matchesBoardFilters(task: Task, filters: BoardFilters, context: MatchContext): boolean {
  const closed = context.isClosed(task.statusId);
  if (filters.hideDone && closed) {
    return false;
  }
  if (filters.priorities.length > 0 && !filters.priorities.includes(task.priority)) {
    return false;
  }

  const due = task.dueAt ? new Date(task.dueAt) : null;
  const { today0, tomorrow0, week0 } = agendaRanges(context.now);
  switch (filters.due) {
    case 'overdue':
      if (!due || due >= today0 || closed) {
        return false;
      }
      break;
    case 'today':
      if (!due || due < today0 || due >= tomorrow0) {
        return false;
      }
      break;
    case 'week':
      if (!due || due < today0 || due >= week0) {
        return false;
      }
      break;
    case 'none':
      if (due) {
        return false;
      }
      break;
    default:
      break;
  }

  const needle = context.query?.trim().toLowerCase() ?? '';
  if (needle.length > 0) {
    const haystack = `${task.title} ${task.description ?? ''}`.toLowerCase();
    if (!haystack.includes(needle)) {
      return false;
    }
  }
  return true;
}

/** So theo thời điểm thật (không so chuỗi: timestamp có thể có hoặc không phần lẻ giây). */
function time(value: string | null | undefined): number {
  const ms = value ? Date.parse(value) : Number.NaN;
  return Number.isNaN(ms) ? 0 : ms;
}

function compareField(a: Task, b: Task, field: SortField): number {
  switch (field) {
    case 'due':
      return time(a.dueAt) - time(b.dueAt);
    case 'priority':
      // PRIORITY_ORDER: urgent = 0 … low = 3 -> đảo dấu để tăng dần là low -> urgent.
      return PRIORITY_ORDER[b.priority] - PRIORITY_ORDER[a.priority];
    case 'created':
      return time(a.createdAt) - time(b.createdAt);
    case 'updated':
      return time(a.updatedAt) - time(b.updatedAt);
    default:
      return (a.position ?? 0) - (b.position ?? 0) || time(a.createdAt) - time(b.createdAt);
  }
}

/** Sắp xếp (không đổi mảng gốc). Task không có hạn luôn ở cuối khi sắp theo hạn, giống API. */
export function sortBoardTasks(tasks: Task[], sort: BoardSort): Task[] {
  return [...tasks].sort((a, b) => {
    if (sort.field === 'due' && !a.dueAt !== !b.dueAt) {
      return a.dueAt ? -1 : 1;
    }
    const result = compareField(a, b, sort.field);
    if (result !== 0) {
      return sort.desc ? -result : result;
    }
    return a.id.localeCompare(b.id);
  });
}
