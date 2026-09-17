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
