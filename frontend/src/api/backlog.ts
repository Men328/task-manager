/** backlog service HTTP calls. */

import { asList, backlogApi, buildQuery, request } from './client';
import type {
  Backlog,
  CreateBacklogInput,
  ListBacklogsParams,
  UpdateBacklogInput,
} from '../types';

// Path khớp `option (google.api.http)` trong common/proto/backlog/v1/backlog.proto.
const BACKLOGS_PATH = '/v1/backlogs';

function toCreateBody(input: CreateBacklogInput): Record<string, unknown> {
  return {
    profile_id: input.profileId,
    title: input.title,
    description: input.description,
    sender: input.sender,
    source: input.source,
    category: input.category,
    reason: input.reason,
    object_key: input.objectKey,
    status: input.status,
  };
}

function toUpdateBody(input: UpdateBacklogInput): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.title !== undefined) {
    body.title = input.title;
  }
  if (input.description !== undefined) {
    body.description = input.description;
  }
  if (input.reason !== undefined) {
    body.reason = input.reason;
  }
  if (input.status !== undefined) {
    body.status = input.status;
  }
  return body;
}

/** GET /v1/backlogs?profile_id=...&status=...&category=... */
export async function listBacklogs(params: ListBacklogsParams): Promise<Backlog[]> {
  const payload = await request<unknown>(
    backlogApi,
    `${BACKLOGS_PATH}${buildQuery({
      profile_id: params.profileId,
      status: params.status,
      category: params.category,
    })}`,
    { method: 'GET' },
  );
  return asList<Backlog>(payload, 'backlogs');
}

/** GET /v1/backlogs/{id} */
export function getBacklog(id: string): Promise<Backlog> {
  return request<unknown>(backlogApi, `${BACKLOGS_PATH}/${encodeURIComponent(id)}`, {
    method: 'GET',
  }).then((payload) => unwrap<Backlog>(payload, 'backlog'));
}

/** POST /v1/backlogs */
export function createBacklog(input: CreateBacklogInput): Promise<Backlog> {
  return request<unknown>(backlogApi, BACKLOGS_PATH, {
    method: 'POST',
    body: JSON.stringify(toCreateBody(input)),
  }).then((payload) => unwrap<Backlog>(payload, 'backlog'));
}

/** PATCH /v1/backlogs/{id} */
export function updateBacklog(id: string, input: UpdateBacklogInput): Promise<Backlog> {
  return request<unknown>(backlogApi, `${BACKLOGS_PATH}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(toUpdateBody(input)),
  }).then((payload) => unwrap<Backlog>(payload, 'backlog'));
}

/** DELETE /v1/backlogs/{id} */
export function deleteBacklog(id: string): Promise<void> {
  return request<void>(backlogApi, `${BACKLOGS_PATH}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}
