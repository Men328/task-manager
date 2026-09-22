import { asList, buildQuery, request, workspaceApi } from './client';
import type { CreateWorkspaceInput, UpdateWorkspaceInput, Workspace } from '../types';

const WORKSPACES_PATH = '/v1/workspaces';

export async function listWorkspaces(profileId: string): Promise<Workspace[]> {
  const payload = await request<unknown>(
    workspaceApi,
    `${WORKSPACES_PATH}${buildQuery({ owner_profile_id: profileId })}`,
    { method: 'GET' },
  );
  return asList<Workspace>(payload, 'workspaces');
}

export function createWorkspace(input: CreateWorkspaceInput): Promise<Workspace> {
  return request<Workspace>(workspaceApi, WORKSPACES_PATH, {
    method: 'POST',
    body: JSON.stringify({
      owner_profile_id: input.ownerProfileId,
      name: input.name,
      slug: input.slug,
      description: input.description,
      color: input.color,
      icon: input.icon,
      is_default: input.isDefault,
      position: input.position,
    }),
  }).then((payload) => unwrap<Workspace>(payload, 'workspace'));
}

export function updateWorkspace(id: string, input: UpdateWorkspaceInput): Promise<Workspace> {
  const body: Record<string, unknown> = {};
  if (input.name !== undefined) body.name = input.name;
  if (input.description !== undefined) body.description = input.description;
  if (input.color !== undefined) body.color = input.color;
  if (input.icon !== undefined) body.icon = input.icon;
  if (input.isDefault !== undefined) body.is_default = input.isDefault;
  if (input.position !== undefined) body.position = input.position;
  if (input.isArchived !== undefined) body.is_archived = input.isArchived;

  return request<Workspace>(workspaceApi, `${WORKSPACES_PATH}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  }).then((payload) => unwrap<Workspace>(payload, 'workspace'));
}

function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}
