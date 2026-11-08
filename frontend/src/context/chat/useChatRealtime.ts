import { useEffect, useRef } from 'react';

import {
  MESSAGE_EVENT_NAME,
  acquireSoketiClient,
  chatChannelName,
  releaseSoketiClient,
} from '../../lib/soketi';
import { parseChatMessage } from '../../lib/chatMessages';
import { useAppDispatch, useAppSelector } from '../hooks';
import { fetchConversations, messageReceived } from './chatSlice';
import type { ChatMessage } from '../../types';

/**
 * Subscribe kênh private `private-chat-<profile_id>` một lần cho cả app.
 * `onIncoming` nhận tin của người khác gửi tới (vd. để hiện toast).
 */
export function useChatRealtime(onIncoming?: (message: ChatMessage) => void) {
  const dispatch = useAppDispatch();
  const token = useAppSelector((state) => state.session.token);
  const profileId = useAppSelector((state) => state.session.profileId);
  const incomingRef = useRef(onIncoming);
  incomingRef.current = onIncoming;

  useEffect(() => {
    if (!token || !profileId) {
      return;
    }

    void dispatch(fetchConversations());

    const client = acquireSoketiClient(token);
    const channelName = chatChannelName(profileId);
    const channel = client.subscribe(channelName);
    const onMessage = (data: unknown) => {
      const message = parseChatMessage(data);
      if (!message) {
        return;
      }
      dispatch(messageReceived({ message, selfId: profileId }));
      if (message.senderId !== profileId) {
        incomingRef.current?.(message);
      }
    };
    channel.bind(MESSAGE_EVENT_NAME, onMessage);

    return () => {
      channel.unbind(MESSAGE_EVENT_NAME, onMessage);
      client.unsubscribe(channelName);
      releaseSoketiClient(client);
    };
  }, [dispatch, token, profileId]);
}
