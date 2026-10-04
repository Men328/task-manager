import Pusher from 'pusher-js';

import { identityApi } from '../api/client';

/** App key là công khai (client dùng để kết nối), app secret chỉ nằm ở backend. */
export const SOKETI_APP_KEY: string =
  (import.meta.env.VITE_SOKETI_APP_KEY as string | undefined) ?? 'task-manager';

export const SOKETI_CLUSTER: string =
  (import.meta.env.VITE_SOKETI_CLUSTER as string | undefined) ?? 'mt1';

/** Cùng origin với SPA; nginx/Vite proxy sang soketi (xem nginx.conf / vite.config.ts). */
export const SOKETI_WS_PATH: string =
  (import.meta.env.VITE_SOKETI_WS_PATH as string | undefined) ?? '/api/soketi';

export const NOTICE_EVENT_NAME = 'notice.created';

/** Mỗi user 1 kênh private: private-<prefix><profile_id>. */
export function noticeChannelName(profileId: string): string {
  return `private-noti-internal-${profileId}`;
}

export function createSoketiClient(token: string): Pusher {
  const secure = window.location.protocol === 'https:';
  const port = window.location.port ? Number(window.location.port) : secure ? 443 : 80;

  return new Pusher(SOKETI_APP_KEY, {
    cluster: SOKETI_CLUSTER,
    forceTLS: secure,
    wsHost: window.location.hostname,
    wsPort: port,
    wssPort: port,
    wsPath: SOKETI_WS_PATH,
    enabledTransports: ['ws', 'wss'],
    channelAuthorization: {
      transport: 'ajax',
      endpoint: `${identityApi}/v1/auth/soketi`,
      headers: { Authorization: `Bearer ${token}` },
      params: { token },
    },
  });
}
