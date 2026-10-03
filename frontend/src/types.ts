/**
 * Domain types shared by the UI.
 *
 * Field names follow the JSON representation of the gRPC/Go services
 * (lowerCamelCase). Enum values mirror the protobuf enum names defined in
 * `design/db_schema.dbml`.
 */

/* -------------------------------------------------------------------------- */
/* Enums                                                                      */
/* -------------------------------------------------------------------------- */

export type TaskPriority =
  | 'TASK_PRIORITY_UNSPECIFIED'
  | 'TASK_PRIORITY_LOW'
  | 'TASK_PRIORITY_MEDIUM'
  | 'TASK_PRIORITY_HIGH'
  | 'TASK_PRIORITY_URGENT';

export type TaskStatusCategory =
  | 'TASK_STATUS_CATEGORY_UNSPECIFIED'
  | 'TASK_STATUS_CATEGORY_TODO'
  | 'TASK_STATUS_CATEGORY_IN_PROGRESS'
  | 'TASK_STATUS_CATEGORY_DONE'
  | 'TASK_STATUS_CATEGORY_CANCELLED';

/** Các priority chọn được trong form (không gồm UNSPECIFIED). */
export const TASK_PRIORITIES: TaskPriority[] = [
  'TASK_PRIORITY_LOW',
  'TASK_PRIORITY_MEDIUM',
  'TASK_PRIORITY_HIGH',
  'TASK_PRIORITY_URGENT',
];

export const TASK_STATUS_CATEGORIES: TaskStatusCategory[] = [
  'TASK_STATUS_CATEGORY_TODO',
  'TASK_STATUS_CATEGORY_IN_PROGRESS',
  'TASK_STATUS_CATEGORY_DONE',
  'TASK_STATUS_CATEGORY_CANCELLED',
];

/* -------------------------------------------------------------------------- */
/* Entities                                                                   */
/* -------------------------------------------------------------------------- */

/** identity service: PROFILES */
export interface Profile {
  id: string;
  email: string;
  displayName: string;
  avatarUrl?: string | null;
  timezone?: string;
  locale?: string;
  isActive?: boolean;
  lastLoginAt?: string | null;
  createdAt?: string;
  updatedAt?: string;
}

/** task service: TASK_STATUSES */
export interface TaskStatus {
  id: string;
  profileId: string;
  name: string;
  slug: string;
  description?: string | null;
  /** `#RRGGBB` or `#RRGGBBAA` */
  color?: string | null;
  category: TaskStatusCategory;
  isDefault: boolean;
  isTerminal: boolean;
  position?: number;
  isArchived?: boolean;
  createdAt?: string;
  updatedAt?: string;
}

/** task service: STATUS_TRANSITIONS */
export interface StatusTransition {
  id: string;
  profileId: string;
  fromStatusId: string;
  toStatusId: string;
  isActive: boolean;
  requiresNote?: boolean;
  description?: string | null;
  createdAt?: string;
  updatedAt?: string;
}

/** task service: TASKS */
export interface Task {
  id: string;
  profileId: string;
  parentTaskId?: string | null;
  statusId: string;
  title: string;
  description?: string | null;
  priority: TaskPriority;
  position?: number;
  startAt?: string | null;
  dueAt?: string | null;
  completedAt?: string | null;
  isArchived?: boolean;
  createdAt?: string;
  updatedAt?: string;
  /** chỉ có khi request kèm include_subtasks=true */
  subtasks?: Task[];
}

/* -------------------------------------------------------------------------- */
/* Request payloads                                                           */
/* -------------------------------------------------------------------------- */

export interface CreateProfileInput {
  email: string;
  displayName: string;
  password?: string;
  avatarUrl?: string;
  timezone?: string;
  locale?: string;
}

export interface CreateStatusInput {
  profileId: string;
  name: string;
  slug: string;
  description?: string;
  color?: string;
  category?: TaskStatusCategory;
  isDefault?: boolean;
  isTerminal?: boolean;
  position?: number;
}

export interface ListTasksParams {
  profileId: string;
  statusId?: string;
  parentTaskId?: string;
  rootOnly?: boolean;
  includeArchived?: boolean;
  includeSubtasks?: boolean;
}

export interface CreateTransitionInput {
  profileId: string;
  fromStatusId: string;
  toStatusId: string;
  description?: string;
}

export interface CreateTaskInput {
  profileId: string;
  title: string;
  statusId?: string;
  description?: string;
  priority?: TaskPriority;
  parentTaskId?: string;
  dueAt?: string;
  startAt?: string;
}

/** calendar service: SCHEDULES */
export interface Schedule {
  id: string;
  profileId: string;
  title: string;
  description?: string | null;
  location?: string | null;
  /** ISO-8601 UTC */
  startAt: string;
  endAt?: string | null;
  allDay: boolean;
  /** `#RRGGBB` or `#RRGGBBAA` */
  color?: string | null;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateScheduleInput {
  profileId: string;
  title: string;
  description?: string;
  location?: string;
  startAt: string;
  endAt?: string;
  allDay?: boolean;
  color?: string;
}

export interface UpdateScheduleInput {
  title?: string;
  description?: string;
  location?: string;
  startAt?: string;
  endAt?: string;
  allDay?: boolean;
  color?: string;
}

export interface ListSchedulesParams {
  profileId: string;
  /** ISO-8601; chỉ lấy lịch giao với [from, to). */
  from?: string;
  to?: string;
}

/* -------------------------------------------------------------------------- */
/* event service                                                              */
/* -------------------------------------------------------------------------- */

export type EventStatus =
  | 'EVENT_STATUS_UNSPECIFIED'
  | 'EVENT_STATUS_PLANNED'
  | 'EVENT_STATUS_CONFIRMED'
  | 'EVENT_STATUS_CANCELLED';

export const EVENT_STATUSES: EventStatus[] = [
  'EVENT_STATUS_PLANNED',
  'EVENT_STATUS_CONFIRMED',
  'EVENT_STATUS_CANCELLED',
];

/** event service: EVENTS */
export interface Event {
  id: string;
  profileId: string;
  title: string;
  description?: string | null;
  location?: string | null;
  /** ISO-8601 UTC */
  startAt: string;
  endAt?: string | null;
  allDay: boolean;
  /** `#RRGGBB` or `#RRGGBBAA` */
  color?: string | null;
  status: EventStatus;
  /** Id email nguồn (rỗng = người dùng tạo trên UI). */
  source?: string | null;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateEventInput {
  profileId: string;
  title: string;
  description?: string;
  location?: string;
  startAt: string;
  endAt?: string;
  allDay?: boolean;
  color?: string;
  status?: EventStatus;
  source?: string;
}

export interface UpdateEventInput {
  title?: string;
  description?: string;
  location?: string;
  startAt?: string;
  endAt?: string;
  allDay?: boolean;
  color?: string;
  status?: EventStatus;
}

export interface ListEventsParams {
  profileId: string;
  /** ISO-8601; chỉ lấy sự kiện giao với [from, to). */
  from?: string;
  to?: string;
  status?: EventStatus;
}

/* -------------------------------------------------------------------------- */
/* backlog service                                                            */
/* -------------------------------------------------------------------------- */

export type BacklogStatus =
  | 'BACKLOG_STATUS_UNSPECIFIED'
  | 'BACKLOG_STATUS_NEW'
  | 'BACKLOG_STATUS_TRIAGED'
  | 'BACKLOG_STATUS_ARCHIVED';

export const BACKLOG_STATUSES: BacklogStatus[] = [
  'BACKLOG_STATUS_NEW',
  'BACKLOG_STATUS_TRIAGED',
  'BACKLOG_STATUS_ARCHIVED',
];

/** backlog service: BACKLOGS */
export interface Backlog {
  id: string;
  profileId: string;
  title: string;
  description?: string | null;
  /** Header From của email nguồn. */
  sender?: string | null;
  /** Gmail message id của email nguồn. */
  source?: string | null;
  /** Nhãn phân loại thô: task | schedule | event | other. */
  category: string;
  reason?: string | null;
  /** Object key của email gốc trên object storage (S3). */
  objectKey?: string | null;
  status: BacklogStatus;
  createdAt?: string;
  updatedAt?: string;
}

export interface CreateBacklogInput {
  profileId: string;
  title: string;
  description?: string;
  sender?: string;
  source?: string;
  category?: string;
  reason?: string;
  objectKey?: string;
  status?: BacklogStatus;
}

export interface UpdateBacklogInput {
  title?: string;
  description?: string;
  reason?: string;
  status?: BacklogStatus;
}

export interface ListBacklogsParams {
  profileId: string;
  status?: BacklogStatus;
  category?: string;
}
