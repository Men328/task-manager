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

export const MESSAGE_EVENT_NAME = 'message.created';

/** Worker gửi khi thẻ task trong tin nhắn đổi trạng thái. */
export const MESSAGE_UPDATED_EVENT_NAME = 'message.updated';

/** Mỗi user 1 kênh private: private-<prefix><profile_id>. */
export function noticeChannelName(profileId: string): string {
  return `private-noti-internal-${profileId}`;
}

/** Kênh private chat của mỗi user; identity chỉ ký cho chính chủ. */
export function chatChannelName(profileId: string): string {
  return `private-chat-${profileId}`;
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

let shared: { token: string; client: Pusher; refs: number } | null = null;

/**
 * Dùng chung 1 kết nối WebSocket cho mọi kênh (thông báo, chat) của cùng phiên.
 * Mỗi lần acquire phải đi kèm 1 lần release.
 */
export function acquireSoketiClient(token: string): Pusher {
  if (shared && shared.token !== token) {
    shared.client.disconnect();
    shared = null;
  }
  if (!shared) {
    shared = { token, client: createSoketiClient(token), refs: 0 };
  }
  shared.refs += 1;
  return shared.client;
}

export function releaseSoketiClient(client: Pusher): void {
  if (!shared || shared.client !== client) {
    client.disconnect();
    return;
  }
  shared.refs -= 1;
  if (shared.refs <= 0) {
    shared.client.disconnect();
    shared = null;
  }
}
