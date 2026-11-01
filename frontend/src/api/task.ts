/** task service HTTP calls. */

import { asList, buildQuery, request, taskApi } from './client';
import type {
  CreateStatusInput,
  CreateTaskInput,
  CreateTransitionInput,
  UpdateTaskInput,
  ListTasksParams,
  StatusTransition,
  Task,
  TaskPage,
  TaskStatus,
} from '../types';

// Path khớp `option (google.api.http)` trong common/proto/task/v1/*.proto.
// Base URL là service root (/api/task), proxy của Vite bỏ prefix rồi forward.
const STATUSES_PATH = '/v1/statuses';
const TRANSITIONS_PATH = '/v1/transitions';
const TASKS_PATH = '/v1/tasks';

/**
 * Build the JSON body for a status, using the proto field names (snake_case)
 * that the gRPC-gateway accepts.
 */
function toStatusBody(input: CreateStatusInput): Record<string, unknown> {
  return {
    profile_id: input.profileId,
    name: input.name,
    slug: input.slug,
    description: input.description,
    color: input.color,
    category: input.category,
    is_default: input.isDefault,
    is_terminal: input.isTerminal,
    position: input.position,
  };
}

function toTaskBody(input: CreateTaskInput): Record<string, unknown> {
  return {
    profile_id: input.profileId,
    project_id: input.projectId,
    title: input.title,
    status_id: input.statusId,
    description: input.description,
    priority: input.priority,
    parent_task_id: input.parentTaskId,
    start_at: input.startAt,
    due_at: input.dueAt,
  };
}

/** GET /v1/statuses?profile_id=... */
export async function listStatuses(profileId: string): Promise<TaskStatus[]> {
  const payload = await request<unknown>(
    taskApi,
    `${STATUSES_PATH}${buildQuery({ profile_id: profileId })}`,
    { method: 'GET' },
  );
  return asList<TaskStatus>(payload, 'statuses');
}

/** POST /v1/statuses */
export function createStatus(input: CreateStatusInput): Promise<TaskStatus> {
  return request<TaskStatus>(taskApi, STATUSES_PATH, {
    method: 'POST',
    body: JSON.stringify(toStatusBody(input)),
  }).then((payload) => unwrap<TaskStatus>(payload, 'status'));
}

/** DELETE /v1/statuses/{id} */
export function deleteStatus(id: string): Promise<void> {
  return request<void>(taskApi, `${STATUSES_PATH}/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

/**
 * POST /v1/statuses/seed — tạo bộ status + rule lifecycle mặc định cho profile.
 * Backend idempotent: profile đã có status thì chỉ bổ sung phần còn thiếu.
 */
export function seedDefaultStatuses(profileId: string): Promise<void> {
  return request<unknown>(taskApi, `${STATUSES_PATH}/seed`, {
    method: 'POST',
    body: JSON.stringify({ profile_id: profileId }),
  }).then(() => undefined);
}

/** GET /v1/transitions?profile_id=... */
export async function listTransitions(profileId: string): Promise<StatusTransition[]> {
  const payload = await request<unknown>(
    taskApi,
    `${TRANSITIONS_PATH}${buildQuery({ profile_id: profileId })}`,
    { method: 'GET' },
  );
  return asList<StatusTransition>(payload, 'transitions');
}

/** POST /v1/transitions */
export function createTransition(input: CreateTransitionInput): Promise<StatusTransition> {
  return request<StatusTransition>(taskApi, TRANSITIONS_PATH, {
    method: 'POST',
    body: JSON.stringify({
      profile_id: input.profileId,
      from_status_id: input.fromStatusId,
      to_status_id: input.toStatusId,
      description: input.description,
    }),
  }).then((payload) => unwrap<StatusTransition>(payload, 'transition'));
}

function listTasksQuery(params: ListTasksParams): string {
  return buildQuery({
    profile_id: params.profileId,
    project_id: params.projectId,
    status_id: params.statusId,
    parent_task_id: params.parentTaskId,
    root_only: params.rootOnly,
    include_archived: params.includeArchived,
    include_subtasks: params.includeSubtasks,
    due_from: params.dueFrom,
    due_to: params.dueTo,
    has_due_at: params.hasDueAt,
    open_only: params.openOnly,
    priorities: params.priorities,
    sort_field: params.sortField,
    sort_desc: params.sortDesc,
    page_size: params.pageSize,
    page_token: params.pageToken,
    query: params.query,
  });
}

/** GET /v1/tasks?profile_id=... */
export async function listTasks(params: ListTasksParams): Promise<Task[]> {
  const payload = await request<unknown>(taskApi, `${TASKS_PATH}${listTasksQuery(params)}`, { method: 'GET' });
  return asList<Task>(payload, 'tasks');
}

/** GET /v1/tasks có phân trang: trả một trang và token trang sau. */
export async function listTasksPage(params: ListTasksParams): Promise<TaskPage> {
  const payload = await request<unknown>(taskApi, `${TASKS_PATH}${listTasksQuery(params)}`, { method: 'GET' });
  const nextPageToken =
    payload !== null && typeof payload === 'object'
      ? (payload as { nextPageToken?: unknown }).nextPageToken
      : undefined;
  return {
    tasks: asList<Task>(payload, 'tasks'),
    nextPageToken: typeof nextPageToken === 'string' ? nextPageToken : '',
  };
}

const LIST_ALL_PAGE_SIZE = 200;

/** Đi hết các trang của ListTasks (dùng cho màn gộp như agenda). */
export async function listAllTasks(params: Omit<ListTasksParams, 'pageSize' | 'pageToken'>): Promise<Task[]> {
  const all: Task[] = [];
  let pageToken = '';
  do {
    const page = await listTasksPage({ ...params, pageSize: LIST_ALL_PAGE_SIZE, pageToken: pageToken || undefined });
    all.push(...page.tasks);
    pageToken = page.nextPageToken;
  } while (pageToken);
  return all;
}

/** GET /v1/tasks/{id}?include_subtasks=true */
export function getTask(id: string, includeSubtasks = true): Promise<Task> {
  return request<Task>(
    taskApi,
    `${TASKS_PATH}/${encodeURIComponent(id)}${buildQuery({ include_subtasks: includeSubtasks })}`,
    { method: 'GET' },
  ).then((payload) => unwrap<Task>(payload, 'task'));
}

/** POST /v1/tasks */
export function createTask(input: CreateTaskInput): Promise<Task> {
  return request<Task>(taskApi, TASKS_PATH, {
    method: 'POST',
    body: JSON.stringify(toTaskBody(input)),
  }).then((payload) => unwrap<Task>(payload, 'task'));
}

/** PATCH /v1/tasks/{id} — hiện chỉ dùng để chuyển project. */
export function updateTask(id: string, input: UpdateTaskInput): Promise<Task> {
  const body: Record<string, unknown> = {};
  if (input.projectId !== undefined) {
    body.project_id = input.projectId;
  }
  return request<Task>(taskApi, `${TASKS_PATH}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  }).then((payload) => unwrap<Task>(payload, 'task'));
}

/** DELETE /v1/tasks/{id} */
export function deleteTask(id: string): Promise<void> {
  return request<void>(taskApi, `${TASKS_PATH}/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

/** POST /v1/tasks/{taskId}/status - move a task through the lifecycle. */
export function changeTaskStatus(
  taskId: string,
  statusId: string,
  note?: string,
): Promise<Task> {
  return request<Task>(taskApi, `${TASKS_PATH}/${encodeURIComponent(taskId)}/status`, {
    method: 'POST',
    body: JSON.stringify({ status_id: statusId, note }),
  }).then((payload) => unwrap<Task>(payload, 'task'));
}

/**
 * Response của gateway là envelope 1 field (`{ "task": {...} }`).
 * Hàm này bóc envelope, đồng thời chấp nhận cả payload trần.
 */
function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}
