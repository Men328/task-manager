import { describe, expect, it } from 'vitest';

import { mergeMessages, parseChatMessage, peerOf } from './chatMessages';
import type { ChatMessage } from '../types';

function message(id: string, createdAt: string, extra: Partial<ChatMessage> = {}): ChatMessage {
  return {
    id,
    conversationId: 'c1',
    senderId: 'a',
    recipientId: 'b',
    body: id,
    clientId: '',
    createdAt,
    ...extra,
  };
}

describe('mergeMessages', () => {
  it('xếp cũ -> mới và bỏ trùng theo id', () => {
    const merged = mergeMessages(
      [message('m2', '2026-11-08T10:00:01Z')],
      [message('m1', '2026-11-08T10:00:00Z'), message('m2', '2026-11-08T10:00:01Z')],
    );

    expect(merged.map((m) => m.id)).toEqual(['m1', 'm2']);
  });

  it('so thời gian theo giá trị, không theo chuỗi', () => {
    const merged = mergeMessages(
      [message('late', '2026-11-08T10:00:00.5Z')],
      [message('early', '2026-11-08T10:00:00.123456Z')],
    );

    expect(merged.map((m) => m.id)).toEqual(['early', 'late']);
  });

  it('tin từ server thay bản gửi lạc quan cùng clientId', () => {
    const pending = message('local-x', '2026-11-08T10:00:05Z', { clientId: 'x', pending: true });
    const server = message('m9', '2026-11-08T10:00:04Z', { clientId: 'x' });

    const merged = mergeMessages([pending], [server]);

    expect(merged).toEqual([server]);
  });

  it('không để bản lạc quan đè lên tin server đã về trước', () => {
    const server = message('m9', '2026-11-08T10:00:04Z', { clientId: 'x' });
    const pending = message('local-x', '2026-11-08T10:00:05Z', { clientId: 'x', pending: true });

    expect(mergeMessages([server], [pending])).toEqual([server]);
  });

  it('tin server về 2 lần (HTTP + soketi) chỉ giữ 1', () => {
    const server = message('m9', '2026-11-08T10:00:04Z', { clientId: 'x' });

    expect(mergeMessages(mergeMessages([], [server]), [server])).toHaveLength(1);
  });
});

describe('parseChatMessage', () => {
  it('đọc được cả chuỗi JSON và object', () => {
    const raw = message('m1', '2026-11-08T10:00:00Z');

    expect(parseChatMessage(JSON.stringify(raw))?.id).toBe('m1');
    expect(parseChatMessage(raw)?.id).toBe('m1');
  });

  it('bỏ qua payload thiếu field', () => {
    expect(parseChatMessage({ id: 'm1' })).toBeNull();
    expect(parseChatMessage('not-json')).toBeNull();
  });
});

describe('peerOf', () => {
  it('trả người bên kia', () => {
    const m = message('m1', '2026-11-08T10:00:00Z');

    expect(peerOf(m, 'a')).toBe('b');
    expect(peerOf(m, 'b')).toBe('a');
  });
});
