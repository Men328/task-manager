/** attachment service HTTP calls. */

import { asList, attachmentApi, buildQuery, request } from './client';
import type { Attachment, AttachmentOwnerType } from '../types';

// Path khớp `option (google.api.http)` trong common/proto/attachment/v1/attachment.proto.
const ATTACHMENTS_PATH = '/v1/attachments';

function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}

/**
 * protojson serialize int64 thành chuỗi (ví dụ `"46"`); chuẩn hoá về number
 * để `Attachment.size` luôn đúng kiểu.
 */
function normalize(item: Attachment): Attachment {
  return { ...item, size: Number(item.size) };
}

/** GET /v1/attachments?owner_type=...&owner_id=... */
export async function listAttachments(
  ownerType: AttachmentOwnerType,
  ownerId: string,
): Promise<Attachment[]> {
  const payload = await request<unknown>(
    attachmentApi,
    `${ATTACHMENTS_PATH}${buildQuery({ owner_type: ownerType, owner_id: ownerId })}`,
    { method: 'GET' },
  );
  return asList<Attachment>(payload, 'attachments').map(normalize);
}

/** GET /v1/attachments/{id} (metadata) */
export function getAttachment(id: string): Promise<Attachment> {
  return request<unknown>(attachmentApi, `${ATTACHMENTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'GET',
  })
    .then((payload) => unwrap<Attachment>(payload, 'attachment'))
    .then(normalize);
}

/** POST multipart /v1/attachments/upload */
export function uploadAttachment(input: {
  profileId: string;
  ownerType: AttachmentOwnerType;
  ownerId: string;
  file: File;
}): Promise<Attachment> {
  const form = new FormData();
  form.set('profile_id', input.profileId);
  form.set('owner_type', input.ownerType);
  form.set('owner_id', input.ownerId);
  form.set('file', input.file, input.file.name);

  return request<unknown>(attachmentApi, `${ATTACHMENTS_PATH}/upload`, {
    method: 'POST',
    body: form,
  })
    .then((payload) => unwrap<Attachment>(payload, 'attachment'))
    .then(normalize);
}

/** DELETE /v1/attachments/{id} */
export function deleteAttachment(id: string): Promise<void> {
  return request<void>(attachmentApi, `${ATTACHMENTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

/** URL tải tệp nhị phân (route custom, không qua JSON). */
export function attachmentDownloadUrl(id: string): string {
  const base = attachmentApi.endsWith('/') ? attachmentApi.slice(0, -1) : attachmentApi;
  return `${base}${ATTACHMENTS_PATH}/${encodeURIComponent(id)}/download`;
}
