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

/** notification service: NOTIFICATION.notices — đối tượng notice trỏ tới. */
export type NoticeTargetType = 'task' | 'schedule' | 'event' | 'backlog';

/**
 * notification service: NOTIFICATION.notices
 *
 * HTTP list trả enum `targetType` dạng tên proto (`NOTICE_TARGET_TYPE_TASK`),
 * còn payload soketi trả dạng ngắn (`task`); dùng `normalizeTargetType` để quy về
 * một dạng trước khi so sánh.
 */
export interface Notice {
  id: string;
  profileId: string;
  type: string;
  title: string;
  body?: string | null;
  targetType: string;
  targetId: string;
  source?: string | null;
  isRead: boolean;
  createdAt?: string | null;
  readAt?: string | null;
}

const NOTICE_TARGET_TYPES: NoticeTargetType[] = ['task', 'schedule', 'event', 'backlog'];

export function normalizeTargetType(value: string | null | undefined): NoticeTargetType | null {
  if (typeof value !== 'string') {
    return null;
  }
  const short = value.trim().toLowerCase().replace(/^notice_target_type_/, '');
  return (NOTICE_TARGET_TYPES as string[]).includes(short) ? (short as NoticeTargetType) : null;
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

/* -------------------------------------------------------------------------- */
/* attachment service                                                         */
/* -------------------------------------------------------------------------- */

export type AttachmentOwnerType = 'task' | 'schedule' | 'event' | 'backlog';

export const ATTACHMENT_OWNER_TYPES: AttachmentOwnerType[] = [
  'task',
  'schedule',
  'event',
  'backlog',
];

/** attachment service: attachment.attachments */
export interface Attachment {
  id: string;
  profileId: string;
  ownerType: AttachmentOwnerType;
  ownerId: string;
  fileName: string;
  contentType: string;
  /** Kích thước byte. */
  size: number;
  createdAt?: string;
}

/* -------------------------------------------------------------------------- */
/* report service                                                             */
/* -------------------------------------------------------------------------- */

export type ReportInterval =
  | 'REPORT_INTERVAL_UNSPECIFIED'
  | 'REPORT_INTERVAL_DAY'
  | 'REPORT_INTERVAL_WEEK'
  | 'REPORT_INTERVAL_MONTH';

export const REPORT_INTERVALS: ReportInterval[] = [
  'REPORT_INTERVAL_DAY',
  'REPORT_INTERVAL_WEEK',
  'REPORT_INTERVAL_MONTH',
];

/** Category thô của status trong báo cáo (khớp DB enum `task.status_category`). */
export type ReportStatusCategory = 'todo' | 'in_progress' | 'done' | 'cancelled';

/** report service: ReportOverview (đã bỏ field timestamp dạng chuỗi ISO). */
export interface ReportOverview {
  profileId: string;
  from?: string;
  to?: string;
  totalTasks: number;
  rootTasks: number;
  subtaskCount: number;
  todoTasks: number;
  inProgressTasks: number;
  doneTasks: number;
  cancelledTasks: number;
  overdueTasks: number;
  dueSoonTasks: number;
  createdInRange: number;
  completedInRange: number;
  /** 0..100 */
  completionRate: number;
  /** 0..100 */
  overdueRate: number;
}

export interface ReportStatusBreakdownItem {
  statusId: string;
  statusName: string;
  statusSlug: string;
  color?: string | null;
  category: ReportStatusCategory;
  count: number;
  /** 0..100 */
  percentage: number;
}

export interface ReportStatusBreakdown {
  items: ReportStatusBreakdownItem[];
  total: number;
}

export interface ReportTimeSeriesPoint {
  bucketStart?: string;
  /** YYYY-MM-DD (UTC) */
  bucket: string;
  created: number;
  completed: number;
}

export interface ReportTimeSeries {
  interval: ReportInterval;
  points: ReportTimeSeriesPoint[];
  totalCreated: number;
  totalCompleted: number;
}

export interface ReportQueryParams {
  profileId: string;
  /** ISO-8601; bỏ trống = 30 ngày gần nhất. */
  from?: string;
  to?: string;
  includeArchived?: boolean;
}

export interface ReportTimeSeriesParams extends ReportQueryParams {
  interval?: ReportInterval;
}

export interface ReportBundle {
  overview: ReportOverview;
  breakdown: ReportStatusBreakdown;
  series: ReportTimeSeries;
}

export interface ReportCategoryCount {
  category: string;
  count: number;
}

export interface ReportEventStats {
  totalEvents: number;
  plannedEvents: number;
  confirmedEvents: number;
  cancelledEvents: number;
  upcomingEvents: number;
  allDayEvents: number;
}

export interface ReportScheduleStats {
  totalSchedules: number;
  allDaySchedules: number;
  upcomingSchedules: number;
  todaySchedules: number;
}

export interface ReportBacklogStats {
  totalBacklogs: number;
  newBacklogs: number;
  triagedBacklogs: number;
  archivedBacklogs: number;
  byCategory: ReportCategoryCount[];
}

/** report service: thống kê sự kiện + lịch + backlog trong khoảng thời gian. */
export interface ReportActivity {
  profileId: string;
  from?: string;
  to?: string;
  events: ReportEventStats;
  schedules: ReportScheduleStats;
  backlogs: ReportBacklogStats;
}


