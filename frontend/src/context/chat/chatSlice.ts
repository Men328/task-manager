import { createAsyncThunk, createSlice, type PayloadAction } from '@reduxjs/toolkit';

import { getErrorMessage } from '../../api/client';
import { listConversations, listMessages, sendMessage } from '../../api/message';
import { compareMessages, mergeMessages, peerOf } from '../../lib/chatMessages';
import { signOut } from '../session/sessionSlice';
import type { ChatMessage, Conversation } from '../../types';

interface ChatRootState {
  session: { token: string | null; profileId: string | null };
}

export interface ChatThread {
  messages: ChatMessage[];
  hasMore: boolean;
  loading: boolean;
  loaded: boolean;
  error: string | null;
}

export interface ChatState {
  threads: Record<string, ChatThread>;
  lastByPeer: Record<string, ChatMessage>;
  unreadByPeer: Record<string, number>;
  activePeerId: string | null;
  conversationsLoading: boolean;
  conversationsError: string | null;
}

const initialState: ChatState = {
  threads: {},
  lastByPeer: {},
  unreadByPeer: {},
  activePeerId: null,
  conversationsLoading: false,
  conversationsError: null,
};

const emptyThread = (): ChatThread => ({ messages: [], hasMore: false, loading: false, loaded: false, error: null });

function threadOf(state: ChatState, peerId: string): ChatThread {
  state.threads[peerId] ??= emptyThread();
  return state.threads[peerId];
}

function rememberLast(state: ChatState, peerId: string, message: ChatMessage) {
  const current = state.lastByPeer[peerId];
  if (!current || compareMessages(current, message) <= 0) {
    state.lastByPeer[peerId] = message;
  }
}

function requireToken(state: ChatRootState): string {
  const token = state.session.token;
  if (!token) {
    throw new Error('no session');
  }
  return token;
}

export const fetchConversations = createAsyncThunk<
  Conversation[],
  void,
  { state: ChatRootState; rejectValue: string }
>('chat/fetchConversations', async (_arg, { getState, rejectWithValue }) => {
  try {
    return await listConversations(requireToken(getState()));
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const fetchMessages = createAsyncThunk<
  { messages: ChatMessage[]; hasMore: boolean },
  { peerId: string; before?: string },
  { state: ChatRootState; rejectValue: string }
>('chat/fetchMessages', async ({ peerId, before }, { getState, rejectWithValue }) => {
  try {
    return await listMessages(requireToken(getState()), peerId, before);
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

export const sendChatMessage = createAsyncThunk<
  ChatMessage,
  { peerId: string; body: string; clientId: string },
  { state: ChatRootState; rejectValue: string }
>('chat/send', async ({ peerId, body, clientId }, { getState, rejectWithValue }) => {
  try {
    return await sendMessage(requireToken(getState()), peerId, body, clientId);
  } catch (cause) {
    return rejectWithValue(getErrorMessage(cause));
  }
});

const chatSlice = createSlice({
  name: 'chat',
  initialState,
  reducers: {
    messageReceived(state, action: PayloadAction<{ message: ChatMessage; selfId: string }>) {
      const { message, selfId } = action.payload;
      const peerId = peerOf(message, selfId);
      const thread = threadOf(state, peerId);
      const known = thread.messages.some((item) => item.id === message.id);

      thread.messages = mergeMessages(thread.messages, [message]);
      rememberLast(state, peerId, message);
      if (!known && message.senderId !== selfId && state.activePeerId !== peerId) {
        state.unreadByPeer[peerId] = (state.unreadByPeer[peerId] ?? 0) + 1;
      }
    },
    setActivePeer(state, action: PayloadAction<string | null>) {
      state.activePeerId = action.payload;
      if (action.payload) {
        delete state.unreadByPeer[action.payload];
      }
    },
  },
  extraReducers: (builder) => {
    builder
      .addCase(signOut, () => initialState)
      .addCase(fetchConversations.pending, (state) => {
        state.conversationsLoading = true;
        state.conversationsError = null;
      })
      .addCase(fetchConversations.fulfilled, (state, action) => {
        state.conversationsLoading = false;
        for (const conversation of action.payload) {
          if (conversation.lastMessage) {
            rememberLast(state, conversation.peerId, conversation.lastMessage);
          }
        }
      })
      .addCase(fetchConversations.rejected, (state, action) => {
        state.conversationsLoading = false;
        state.conversationsError = action.payload ?? null;
      })
      .addCase(fetchMessages.pending, (state, action) => {
        const thread = threadOf(state, action.meta.arg.peerId);
        thread.loading = true;
        thread.error = null;
      })
      .addCase(fetchMessages.fulfilled, (state, action) => {
        const { peerId, before } = action.meta.arg;
        const thread = threadOf(state, peerId);
        thread.loading = false;
        thread.messages = mergeMessages(thread.messages, action.payload.messages);
        if (before || !thread.loaded) {
          thread.hasMore = action.payload.hasMore;
        }
        thread.loaded = true;
        const newest = action.payload.messages[action.payload.messages.length - 1];
        if (newest) {
          rememberLast(state, peerId, newest);
        }
      })
      .addCase(fetchMessages.rejected, (state, action) => {
        const thread = threadOf(state, action.meta.arg.peerId);
        thread.loading = false;
        thread.error = action.payload ?? null;
      })
      .addCase(sendChatMessage.pending, (state, action) => {
        const { peerId, body, clientId } = action.meta.arg;
        const optimistic: ChatMessage = {
          id: `local-${clientId}`,
          conversationId: '',
          senderId: '',
          recipientId: peerId,
          body,
          clientId,
          createdAt: new Date().toISOString(),
          pending: true,
        };
        const thread = threadOf(state, peerId);
        thread.messages = mergeMessages(thread.messages, [optimistic]);
      })
      .addCase(sendChatMessage.fulfilled, (state, action) => {
        const peerId = action.meta.arg.peerId;
        const thread = threadOf(state, peerId);
        thread.messages = mergeMessages(thread.messages, [action.payload]);
        rememberLast(state, peerId, action.payload);
      })
      .addCase(sendChatMessage.rejected, (state, action) => {
        const { peerId, clientId } = action.meta.arg;
        const thread = threadOf(state, peerId);
        thread.messages = thread.messages.map((item) =>
          item.clientId === clientId && item.pending ? { ...item, pending: false, failed: true } : item,
        );
      });
  },
});

export const { messageReceived, setActivePeer } = chatSlice.actions;
export const chatReducer = chatSlice.reducer;
