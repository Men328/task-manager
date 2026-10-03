import type {
  BacklogStatus,
  EventStatus,
  TaskPriority,
  TaskStatusCategory,
} from '../types';

/** Key i18n cho nhãn priority (màu nằm trong PriorityBadge.module.css). */
export const PRIORITY_LABEL_KEY: Record<TaskPriority, string> = {
  TASK_PRIORITY_UNSPECIFIED: 'priority.unspecified',
  TASK_PRIORITY_LOW: 'priority.low',
  TASK_PRIORITY_MEDIUM: 'priority.medium',
  TASK_PRIORITY_HIGH: 'priority.high',
  TASK_PRIORITY_URGENT: 'priority.urgent',
};

/** Key i18n cho nhãn status category. */
export const CATEGORY_LABEL_KEY: Record<TaskStatusCategory, string> = {
  TASK_STATUS_CATEGORY_UNSPECIFIED: 'category.unspecified',
  TASK_STATUS_CATEGORY_TODO: 'category.todo',
  TASK_STATUS_CATEGORY_IN_PROGRESS: 'category.inProgress',
  TASK_STATUS_CATEGORY_DONE: 'category.done',
  TASK_STATUS_CATEGORY_CANCELLED: 'category.cancelled',
};

/** Thứ tự sort card trong 1 cột: urgent lên trước. */
export const PRIORITY_ORDER: Record<TaskPriority, number> = {
  TASK_PRIORITY_URGENT: 0,
  TASK_PRIORITY_HIGH: 1,
  TASK_PRIORITY_MEDIUM: 2,
  TASK_PRIORITY_LOW: 3,
  TASK_PRIORITY_UNSPECIFIED: 4,
};

/** Màu dot của cột khi status chưa có color. */
export const FALLBACK_STATUS_COLORS = ['#9aa0ae', '#f59f00', '#37b24d', '#f06595', '#4c6ef5', '#12b886'];

export function statusColor(color: string | null | undefined, index: number): string {
  const value = (color ?? '').trim();
  return value.length > 0 ? value : FALLBACK_STATUS_COLORS[index % FALLBACK_STATUS_COLORS.length]!;
}

export function isDoneCategory(category: TaskStatusCategory | undefined): boolean {
  return category === 'TASK_STATUS_CATEGORY_DONE';
}

/** Key i18n cho nhãn trạng thái sự kiện. */
export const EVENT_STATUS_LABEL_KEY: Record<EventStatus, string> = {
  EVENT_STATUS_UNSPECIFIED: 'eventStatus.unspecified',
  EVENT_STATUS_PLANNED: 'eventStatus.planned',
  EVENT_STATUS_CONFIRMED: 'eventStatus.confirmed',
  EVENT_STATUS_CANCELLED: 'eventStatus.cancelled',
};

export const EVENT_STATUS_COLOR: Record<EventStatus, string> = {
  EVENT_STATUS_UNSPECIFIED: 'gray',
  EVENT_STATUS_PLANNED: 'blue',
  EVENT_STATUS_CONFIRMED: 'teal',
  EVENT_STATUS_CANCELLED: 'red',
};

/** Key i18n cho nhãn trạng thái backlog. */
export const BACKLOG_STATUS_LABEL_KEY: Record<BacklogStatus, string> = {
  BACKLOG_STATUS_UNSPECIFIED: 'backlogStatus.unspecified',
  BACKLOG_STATUS_NEW: 'backlogStatus.new',
  BACKLOG_STATUS_TRIAGED: 'backlogStatus.triaged',
  BACKLOG_STATUS_ARCHIVED: 'backlogStatus.archived',
};

export const BACKLOG_STATUS_COLOR: Record<BacklogStatus, string> = {
  BACKLOG_STATUS_UNSPECIFIED: 'gray',
  BACKLOG_STATUS_NEW: 'brand',
  BACKLOG_STATUS_TRIAGED: 'teal',
  BACKLOG_STATUS_ARCHIVED: 'gray',
};

/** Key i18n cho nhãn category thô của backlog (mail worker gán). */
export const BACKLOG_CATEGORY_LABEL_KEY: Record<string, string> = {
  task: 'backlogCategory.task',
  schedule: 'backlogCategory.schedule',
  event: 'backlogCategory.event',
  other: 'backlogCategory.other',
};
