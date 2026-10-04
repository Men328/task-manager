import { asList, buildQuery, notificationApi, request } from './client';
import type { Notice } from '../types';

const NOTICES_PATH = '/v1/notices';

function authHeader(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

export async function listNotices(
  token: string,
  options: { limit?: number; unreadOnly?: boolean } = {},
): Promise<Notice[]> {
  const query = buildQuery({
    limit: options.limit,
    unread_only: options.unreadOnly === true ? true : undefined,
  });
  const payload = await request<unknown>(notificationApi, `${NOTICES_PATH}${query}`, {
    method: 'GET',
    headers: authHeader(token),
  });
  return asList<Notice>(payload, 'notices');
}

export async function countUnreadNotices(token: string): Promise<number> {
  const payload = await request<{ count?: number }>(notificationApi, `${NOTICES_PATH}/unread-count`, {
    method: 'GET',
    headers: authHeader(token),
  });
  return typeof payload?.count === 'number' ? payload.count : 0;
}

/** ids = ['*'] để đánh dấu đã đọc tất cả notice của user đang đăng nhập. */
export async function markNoticesRead(token: string, ids: string[]): Promise<number> {
  const payload = await request<{ marked?: number }>(notificationApi, `${NOTICES_PATH}/read`, {
    method: 'POST',
    headers: authHeader(token),
    body: JSON.stringify({ ids }),
  });
  return typeof payload?.marked === 'number' ? payload.marked : 0;
}

export function markAllNoticesRead(token: string): Promise<number> {
  return markNoticesRead(token, ['*']);
}
