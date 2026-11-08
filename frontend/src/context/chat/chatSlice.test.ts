import { describe, expect, it, vi } from 'vitest';

vi.mock('../../i18n', () => ({ default: { t: (key: string) => key } }));

import { signOut } from '../session/sessionSlice';
import { chatReducer, fetchMessages, messageReceived, sendChatMessage, setActivePeer } from './chatSlice';
import type { ChatMessage } from '../../types';

const me = 'me';
const bob = 'bob';

function message(id: string, from: string, to: string, createdAt: string, clientId = ''): ChatMessage {
  return { id, conversationId: 'c', senderId: from, recipientId: to, body: id, clientId, createdAt };
}

describe('chatSlice', () => {
  it('tin đến từ người khác tăng unread khi không mở cuộc trò chuyện đó', () => {
    let state = chatReducer(undefined, messageReceived({ message: message('m1', bob, me, '2026-11-08T10:00:00Z'), selfId: me }));
    state = chatReducer(state, messageReceived({ message: message('m1', bob, me, '2026-11-08T10:00:00Z'), selfId: me }));

    expect(state.unreadByPeer[bob]).toBe(1);
    expect(state.threads[bob].messages).toHaveLength(1);
    expect(state.lastByPeer[bob].id).toBe('m1');
  });

  it('không tăng unread khi đang mở đúng cuộc trò chuyện, và mở thì xoá unread', () => {
    let state = chatReducer(undefined, messageReceived({ message: message('m1', bob, me, '2026-11-08T10:00:00Z'), selfId: me }));
    state = chatReducer(state, setActivePeer(bob));
    expect(state.unreadByPeer[bob]).toBeUndefined();

    state = chatReducer(state, messageReceived({ message: message('m2', bob, me, '2026-11-08T10:00:01Z'), selfId: me }));
    expect(state.unreadByPeer[bob]).toBeUndefined();
  });

  it('tin mình gửi từ tab khác vào đúng thread của người nhận, không tính unread', () => {
    const state = chatReducer(undefined, messageReceived({ message: message('m1', me, bob, '2026-11-08T10:00:00Z'), selfId: me }));

    expect(state.threads[bob].messages).toHaveLength(1);
    expect(state.unreadByPeer[bob]).toBeUndefined();
  });

  it('gửi lạc quan rồi thay bằng tin server; hỏng thì đánh dấu failed', () => {
    const arg = { peerId: bob, body: 'hi', clientId: 'x' };
    let state = chatReducer(undefined, sendChatMessage.pending('r1', arg));
    expect(state.threads[bob].messages[0].pending).toBe(true);

    const server = message('m9', me, bob, '2026-11-08T10:00:00Z', 'x');
    state = chatReducer(state, messageReceived({ message: server, selfId: me }));
    state = chatReducer(state, sendChatMessage.fulfilled(server, 'r1', arg));
    expect(state.threads[bob].messages).toEqual([server]);

    const failArg = { peerId: bob, body: 'oops', clientId: 'y' };
    state = chatReducer(state, sendChatMessage.pending('r2', failArg));
    state = chatReducer(state, sendChatMessage.rejected(null, 'r2', failArg, 'lỗi'));
    const failed = state.threads[bob].messages.find((m) => m.clientId === 'y');
    expect(failed?.failed).toBe(true);
    expect(failed?.pending).toBe(false);
  });

  it('tải lịch sử gộp với tin realtime đã có', () => {
    let state = chatReducer(undefined, messageReceived({ message: message('m3', bob, me, '2026-11-08T10:00:03Z'), selfId: me }));
    const page = { messages: [message('m1', me, bob, '2026-11-08T10:00:01Z'), message('m2', bob, me, '2026-11-08T10:00:02Z')], hasMore: true };
    state = chatReducer(state, fetchMessages.fulfilled(page, 'r', { peerId: bob }));

    expect(state.threads[bob].messages.map((m) => m.id)).toEqual(['m1', 'm2', 'm3']);
    expect(state.threads[bob].hasMore).toBe(true);
    expect(state.lastByPeer[bob].id).toBe('m3');
  });

  it('xoá sạch khi đăng xuất', () => {
    const state = chatReducer(
      chatReducer(undefined, messageReceived({ message: message('m1', bob, me, '2026-11-08T10:00:00Z'), selfId: me })),
      signOut(),
    );

    expect(state.threads).toEqual({});
    expect(state.unreadByPeer).toEqual({});
  });
});
