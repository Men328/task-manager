import { describe, expect, it, vi } from 'vitest';

// Test chạy trong môi trường node: i18n thật cần `document` khi khởi tạo.
vi.mock('../../i18n', () => ({ default: { t: (key: string) => key } }));

import { signOut } from '../session/sessionSlice';
import { fetchFriends, friendsReducer } from './friendsSlice';
import type { Friend, FriendRequest } from '../../types';

const profile = { id: 'p1', email: 'a@x.io', displayName: 'A', avatarUrl: '' };
const friend: Friend = { profile, friendsSince: '2026-11-01T00:00:00Z' };
const request: FriendRequest = { id: 'r1', profile, createdAt: '2026-11-01T00:00:00Z' };
const payload = { friends: [friend], incoming: [request], outgoing: [request] };

describe('friendsSlice', () => {
  it('ghi đủ 3 danh sách khi tải xong', () => {
    const state = friendsReducer(undefined, fetchFriends.fulfilled(payload, 'req', undefined));

    expect(state.friends).toEqual([friend]);
    expect(state.incoming).toEqual([request]);
    expect(state.outgoing).toEqual([request]);
    expect(state.loading).toBe(false);
  });

  it('giữ dữ liệu cũ và ghi lỗi khi tải hỏng', () => {
    const loaded = friendsReducer(undefined, fetchFriends.fulfilled(payload, 'req', undefined));
    const failed = fetchFriends.rejected(null, 'req', undefined, 'mất mạng');

    const state = friendsReducer(loaded, failed);

    expect(state.error).toBe('mất mạng');
    expect(state.friends).toEqual([friend]);
  });

  it('xoá hết khi đăng xuất', () => {
    const loaded = friendsReducer(undefined, fetchFriends.fulfilled(payload, 'req', undefined));

    const state = friendsReducer(loaded, signOut());

    expect(state.friends).toEqual([]);
    expect(state.incoming).toEqual([]);
    expect(state.outgoing).toEqual([]);
  });
});
