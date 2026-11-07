import { useCallback } from 'react';

import {
  acceptFriendRequest,
  cancelFriendRequest,
  declineFriendRequest,
  removeFriend,
  searchFriend,
  sendFriendRequest,
} from '../../api/social';
import { useAppDispatch, useAppSelector } from '../hooks';
import { fetchFriends } from './friendsSlice';

export function useFriends() {
  const dispatch = useAppDispatch();
  const token = useAppSelector((state) => state.session.token);
  const friends = useAppSelector((state) => state.friends.friends);
  const incoming = useAppSelector((state) => state.friends.incoming);
  const outgoing = useAppSelector((state) => state.friends.outgoing);
  const loading = useAppSelector((state) => state.friends.loading);
  const error = useAppSelector((state) => state.friends.error);

  const reload = useCallback(() => {
    void dispatch(fetchFriends());
  }, [dispatch]);

  /** Chạy thao tác rồi tải lại danh sách; lỗi ném ra để trang hiện thông báo. */
  const mutate = useCallback(
    async (action: (sessionToken: string) => Promise<unknown>) => {
      if (!token) {
        throw new Error('no session');
      }
      await action(token);
      void dispatch(fetchFriends());
    },
    [dispatch, token],
  );

  const search = useCallback(
    (email: string) => {
      if (!token) {
        throw new Error('no session');
      }
      return searchFriend(token, email);
    },
    [token],
  );

  const send = useCallback(
    (targetId: string) => mutate((value) => sendFriendRequest(value, targetId)),
    [mutate],
  );
  const accept = useCallback(
    (id: string) => mutate((value) => acceptFriendRequest(value, id)),
    [mutate],
  );
  const decline = useCallback(
    (id: string) => mutate((value) => declineFriendRequest(value, id)),
    [mutate],
  );
  const cancel = useCallback(
    (id: string) => mutate((value) => cancelFriendRequest(value, id)),
    [mutate],
  );
  const remove = useCallback(
    (friendId: string) => mutate((value) => removeFriend(value, friendId)),
    [mutate],
  );

  return { friends, incoming, outgoing, loading, error, reload, search, send, accept, decline, cancel, remove };
}
