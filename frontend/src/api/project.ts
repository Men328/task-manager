/** project service HTTP calls. */

import { asList, buildQuery, projectApi, request } from './client';
import type { CreateProjectInput, Project, UpdateProjectInput } from '../types';

// Path khớp `option (google.api.http)` trong common/proto/project/v1/project.proto.
const PROJECTS_PATH = '/v1/projects';

function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}

/** GET /v1/projects?profile_id=...&include_archived=... */
export async function listProjects(profileId: string, includeArchived = false): Promise<Project[]> {
  const payload = await request<unknown>(
    projectApi,
    `${PROJECTS_PATH}${buildQuery({
      profile_id: profileId,
      include_archived: includeArchived ? 'true' : undefined,
    })}`,
    { method: 'GET' },
  );
  return asList<Project>(payload, 'projects');
}

/** POST /v1/projects */
export function createProject(input: CreateProjectInput): Promise<Project> {
  return request<unknown>(projectApi, PROJECTS_PATH, {
    method: 'POST',
    body: JSON.stringify({
      profile_id: input.profileId,
      name: input.name,
      description: input.description,
      color: input.color,
    }),
  }).then((payload) => unwrap<Project>(payload, 'project'));
}

/** PATCH /v1/projects/{id} — chỉ gửi field được set. */
export function updateProject(id: string, input: UpdateProjectInput): Promise<Project> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) {
    body.name = input.name;
  }
  if (input.description !== undefined) {
    body.description = input.description;
  }
  if (input.color !== undefined) {
    body.color = input.color;
  }
  return request<unknown>(projectApi, `${PROJECTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  }).then((payload) => unwrap<Project>(payload, 'project'));
}

/** POST /v1/projects/{id}/archive */
export function archiveProject(id: string): Promise<Project> {
  return request<unknown>(projectApi, `${PROJECTS_PATH}/${encodeURIComponent(id)}/archive`, {
    method: 'POST',
    body: '{}',
  }).then((payload) => unwrap<Project>(payload, 'project'));
}

/** POST /v1/projects/{id}/unarchive */
export function unarchiveProject(id: string): Promise<Project> {
  return request<unknown>(projectApi, `${PROJECTS_PATH}/${encodeURIComponent(id)}/unarchive`, {
    method: 'POST',
    body: '{}',
  }).then((payload) => unwrap<Project>(payload, 'project'));
}
