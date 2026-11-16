import { asList, buildQuery, messageApi, request } from './client';
import type { ChatMessage, ChatTaskDraft, Conversation } from '../types';

const CONVERSATIONS_PATH = '/v1/conversations';

function authHeader(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

function messagesPath(peerId: string): string {
  return `${CONVERSATIONS_PATH}/${encodeURIComponent(peerId)}/messages`;
}

export async function listConversations(token: string): Promise<Conversation[]> {
  const payload = await request<unknown>(messageApi, CONVERSATIONS_PATH, {
    method: 'GET',
    headers: authHeader(token),
  });
  return asList<Conversation>(payload, 'conversations');
}

export interface MessagePage {
  messages: ChatMessage[];
  hasMore: boolean;
}

/** `before` rỗng = trang mới nhất; kết quả xếp cũ -> mới. */
export async function listMessages(token: string, peerId: string, before?: string): Promise<MessagePage> {
  const payload = await request<{ messages?: ChatMessage[]; hasMore?: boolean }>(
    messageApi,
    `${messagesPath(peerId)}${buildQuery({ before, limit: 50 })}`,
    { method: 'GET', headers: authHeader(token) },
  );
  return { messages: payload.messages ?? [], hasMore: Boolean(payload.hasMore) };
}

export async function sendMessage(
  token: string,
  peerId: string,
  body: string,
  clientId: string,
): Promise<ChatMessage> {
  const payload = await request<{ message: ChatMessage }>(messageApi, messagesPath(peerId), {
    method: 'POST',
    headers: authHeader(token),
    body: JSON.stringify({ body, client_id: clientId }),
  });
  return payload.message;
}

/** Giao task cho bạn: task vào project mặc định của người nhận, tin nhắn kèm thẻ task. */
export async function sendTaskMessage(
  token: string,
  peerId: string,
  draft: ChatTaskDraft,
  clientId: string,
): Promise<ChatMessage> {
  const payload = await request<{ message: ChatMessage }>(
    messageApi,
    `${CONVERSATIONS_PATH}/${encodeURIComponent(peerId)}/tasks`,
    {
      method: 'POST',
      headers: authHeader(token),
      body: JSON.stringify({
        title: draft.title,
        description: draft.description,
        priority: draft.priority,
        due_at: draft.dueAt || undefined,
        client_id: clientId,
      }),
    },
  );
  return payload.message;
}

export async function getMessage(token: string, messageId: string): Promise<ChatMessage> {
  const payload = await request<{ message: ChatMessage }>(messageApi, `/v1/messages/${encodeURIComponent(messageId)}`, {
    method: 'GET',
    headers: authHeader(token),
  });
  return payload.message;
}
