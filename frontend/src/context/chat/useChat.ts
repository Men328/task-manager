import { useCallback, useMemo } from 'react';

import { newClientId } from '../../lib/chatMessages';
import { useAppDispatch, useAppSelector } from '../hooks';
import {
  fetchConversations,
  fetchMessages,
  sendChatMessage,
  sendChatTask,
  setActivePeer,
  type ChatThread,
} from './chatSlice';
import type { ChatTaskDraft } from '../../types';

const EMPTY_THREAD: ChatThread = { messages: [], hasMore: false, loading: false, loaded: false, error: null };

export function useChat(peerId: string | null) {
  const dispatch = useAppDispatch();
  const profileId = useAppSelector((state) => state.session.profileId);
  const lastByPeer = useAppSelector((state) => state.chat.lastByPeer);
  const unreadByPeer = useAppSelector((state) => state.chat.unreadByPeer);
  const conversationsError = useAppSelector((state) => state.chat.conversationsError);
  const thread = useAppSelector((state) => (peerId ? state.chat.threads[peerId] : undefined)) ?? EMPTY_THREAD;

  const loadLatest = useCallback(() => {
    if (peerId) {
      void dispatch(fetchMessages({ peerId }));
    }
  }, [dispatch, peerId]);

  const loadOlder = useCallback(() => {
    const oldest = thread.messages.find((message) => !message.pending && !message.failed);
    if (peerId && oldest) {
      void dispatch(fetchMessages({ peerId, before: oldest.createdAt }));
    }
  }, [dispatch, peerId, thread.messages]);

  const send = useCallback(
    (body: string, clientId: string = newClientId()) => {
      if (peerId) {
        void dispatch(sendChatMessage({ peerId, body, clientId }));
      }
    },
    [dispatch, peerId],
  );

  const sendTask = useCallback(
    (draft: ChatTaskDraft) => {
      if (!peerId) {
        return Promise.reject(new Error('no peer'));
      }
      return dispatch(sendChatTask({ peerId, draft, clientId: newClientId() })).unwrap();
    },
    [dispatch, peerId],
  );

  const activate = useCallback(
    (value: string | null) => {
      dispatch(setActivePeer(value));
    },
    [dispatch],
  );

  const reloadConversations = useCallback(() => {
    void dispatch(fetchConversations());
  }, [dispatch]);

  return useMemo(
    () => ({
      profileId,
      thread,
      lastByPeer,
      unreadByPeer,
      conversationsError,
      loadLatest,
      loadOlder,
      send,
      sendTask,
      activate,
      reloadConversations,
    }),
    [
      profileId,
      thread,
      lastByPeer,
      unreadByPeer,
      conversationsError,
      loadLatest,
      loadOlder,
      send,
      sendTask,
      activate,
      reloadConversations,
    ],
  );
}

export function useUnreadMessageCount(): number {
  return useAppSelector((state) =>
    Object.values(state.chat.unreadByPeer).reduce((total, count) => total + count, 0),
  );
}
