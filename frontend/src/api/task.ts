/** task service HTTP calls. */

import { asList, buildQuery, request, taskApi } from './client';
import type {
  CreateStatusInput,
  CreateTaskInput,
  CreateTransitionInput,
  ListTasksParams,
  StatusTransition,
  Task,
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

/** GET /v1/tasks?profile_id=... */
export async function listTasks(params: ListTasksParams): Promise<Task[]> {
  const payload = await request<unknown>(
    taskApi,
    `${TASKS_PATH}${buildQuery({
      profile_id: params.profileId,
      status_id: params.statusId,
      parent_task_id: params.parentTaskId,
      root_only: params.rootOnly,
      include_archived: params.includeArchived,
      include_subtasks: params.includeSubtasks,
    })}`,
    { method: 'GET' },
  );
  return asList<Task>(payload, 'tasks');
}

/** POST /v1/tasks */
export function createTask(input: CreateTaskInput): Promise<Task> {
  return request<Task>(taskApi, TASKS_PATH, {
    method: 'POST',
    body: JSON.stringify(toTaskBody(input)),
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
