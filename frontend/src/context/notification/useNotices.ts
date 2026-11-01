import { useCallback, useEffect, useRef } from 'react';

import { NOTICE_EVENT_NAME, createSoketiClient, noticeChannelName } from '../../lib/soketi';
import { useAppDispatch, useAppSelector } from '../hooks';
import { useSession } from '../session/useSession';
import {
  clearNotices,
  fetchNotices,
  fetchUnreadCount,
  markAllNoticesReadThunk,
  markNoticesRead,
} from './noticeSlice';

import type { Notice } from '../../types';

/** Payload soketi có thể đến dạng object hoặc chuỗi JSON. */
function parseIncoming(data: unknown): Notice | null {
  try {
    const value = typeof data === 'string' ? JSON.parse(data) : data;
    return value && typeof value === 'object' && 'type' in value ? (value as Notice) : null;
  } catch {
    return null;
  }
}

/** `onIncoming` được gọi với mỗi notice realtime (vd. để hiện toast nhắc việc). */
export function useNotices(onIncoming?: (notice: Notice) => void) {
  const dispatch = useAppDispatch();
  const { token, profileId } = useSession();
  const items = useAppSelector((state) => state.notices.items);
  const unreadCount = useAppSelector((state) => state.notices.unreadCount);
  const loading = useAppSelector((state) => state.notices.loading);
  const error = useAppSelector((state) => state.notices.error);
  const incomingRef = useRef(onIncoming);
  incomingRef.current = onIncoming;

  useEffect(() => {
    if (!token || !profileId) {
      dispatch(clearNotices());
      return;
    }

    void dispatch(fetchUnreadCount());

    const client = createSoketiClient(token);
    const channelName = noticeChannelName(profileId);
    const channel = client.subscribe(channelName);
    const onNotice = (data: unknown) => {
      void dispatch(fetchUnreadCount());
      void dispatch(fetchNotices());
      const notice = parseIncoming(data);
      if (notice) {
        incomingRef.current?.(notice);
      }
    };
    channel.bind(NOTICE_EVENT_NAME, onNotice);

    return () => {
      channel.unbind(NOTICE_EVENT_NAME, onNotice);
      client.unsubscribe(channelName);
      client.disconnect();
    };
  }, [dispatch, token, profileId]);

  const refresh = useCallback(() => {
    void dispatch(fetchUnreadCount());
    void dispatch(fetchNotices());
  }, [dispatch]);

  const markRead = useCallback(
    async (ids: string[]) => {
      await dispatch(markNoticesRead(ids))
        .unwrap()
        .catch(() => undefined);
      void dispatch(fetchUnreadCount());
    },
    [dispatch],
  );

  const markAllRead = useCallback(async () => {
    await dispatch(markAllNoticesReadThunk())
      .unwrap()
      .catch(() => undefined);
    void dispatch(fetchUnreadCount());
  }, [dispatch]);

  return { items, unreadCount, loading, error, refresh, markRead, markAllRead };
}
