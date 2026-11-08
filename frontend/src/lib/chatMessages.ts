import type { ChatMessage } from '../types';

function timeOf(message: ChatMessage): number {
  const value = Date.parse(message.createdAt);
  return Number.isNaN(value) ? 0 : value;
}

/** Thứ tự hiển thị: cũ -> mới; trùng thời gian thì theo id (UUIDv7 tăng dần theo thời gian). */
export function compareMessages(a: ChatMessage, b: ChatMessage): number {
  return timeOf(a) - timeOf(b) || (a.id < b.id ? -1 : a.id > b.id ? 1 : 0);
}

/**
 * Gộp tin mới vào danh sách: bỏ trùng theo id, và tin từ server thay thế bản gửi
 * lạc quan có cùng clientId (tin đó có thể về qua HTTP response hoặc soketi trước).
 */
export function mergeMessages(existing: ChatMessage[], incoming: ChatMessage[]): ChatMessage[] {
  const byKey = new Map<string, ChatMessage>();
  const keyOf = (message: ChatMessage) => (message.clientId ? `c:${message.clientId}` : `i:${message.id}`);

  for (const message of existing) {
    byKey.set(keyOf(message), message);
  }
  for (const message of incoming) {
    const key = keyOf(message);
    const current = byKey.get(key);
    if (current && !current.pending && !current.failed && message.pending) {
      continue;
    }
    byKey.set(key, message);
  }

  const seenIds = new Set<string>();
  const merged: ChatMessage[] = [];
  for (const message of byKey.values()) {
    if (seenIds.has(message.id)) {
      continue;
    }
    seenIds.add(message.id);
    merged.push(message);
  }
  return merged.sort(compareMessages);
}

/** Bản cập nhật thẻ task chỉ thay khi mới hơn (sự kiện có thể đến trễ hoặc lặp). */
export function isNewerTaskState(current: ChatMessage, incoming: ChatMessage): boolean {
  if (!incoming.task) {
    return false;
  }
  const before = Date.parse(current.task?.statusChangedAt ?? '');
  const after = Date.parse(incoming.task.statusChangedAt ?? '');
  if (Number.isNaN(after)) {
    return !current.task;
  }
  return Number.isNaN(before) || after >= before;
}

/** Người bên kia của tin nhắn, nhìn từ phía `selfId`. */
export function peerOf(message: ChatMessage, selfId: string): string {
  return message.senderId === selfId ? message.recipientId : message.senderId;
}

export function newClientId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID();
  }
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`;
}

/** Payload soketi có thể đến dạng object hoặc chuỗi JSON. */
export function parseChatMessage(data: unknown): ChatMessage | null {
  try {
    const value = typeof data === 'string' ? (JSON.parse(data) as unknown) : data;
    if (!value || typeof value !== 'object') {
      return null;
    }
    const record = value as Record<string, unknown>;
    const fields = ['id', 'conversationId', 'senderId', 'recipientId', 'body', 'createdAt'];
    if (!fields.every((field) => typeof record[field] === 'string')) {
      return null;
    }
    return { ...(record as unknown as ChatMessage), clientId: typeof record.clientId === 'string' ? record.clientId : '' };
  } catch {
    return null;
  }
}
