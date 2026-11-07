import { asList, buildQuery, request, socialApi } from './client';
import type { Friend, FriendRelation, FriendRequest, FriendSearchResult } from '../types';

const FRIENDS_PATH = '/v1/friends';
const REQUESTS_PATH = '/v1/friend-requests';

function authHeader(token: string): Record<string, string> {
  return { Authorization: `Bearer ${token}` };
}

export function searchFriend(token: string, email: string): Promise<FriendSearchResult> {
  return request<FriendSearchResult>(socialApi, `${FRIENDS_PATH}/search${buildQuery({ email })}`, {
    method: 'GET',
    headers: authHeader(token),
  });
}

export async function listFriends(token: string): Promise<Friend[]> {
  const payload = await request<unknown>(socialApi, FRIENDS_PATH, {
    method: 'GET',
    headers: authHeader(token),
  });
  return asList<Friend>(payload, 'friends');
}

export async function removeFriend(token: string, friendId: string): Promise<void> {
  await request<unknown>(socialApi, `${FRIENDS_PATH}/${encodeURIComponent(friendId)}`, {
    method: 'DELETE',
    headers: authHeader(token),
  });
}

export async function listFriendRequests(
  token: string,
): Promise<{ incoming: FriendRequest[]; outgoing: FriendRequest[] }> {
  const payload = await request<unknown>(socialApi, REQUESTS_PATH, {
    method: 'GET',
    headers: authHeader(token),
  });
  return {
    incoming: asList<FriendRequest>(payload, 'incoming'),
    outgoing: asList<FriendRequest>(payload, 'outgoing'),
  };
}

export async function sendFriendRequest(token: string, targetId: string): Promise<FriendRelation> {
  const payload = await request<{ relation: FriendRelation }>(socialApi, REQUESTS_PATH, {
    method: 'POST',
    headers: authHeader(token),
    body: JSON.stringify({ profile_id: targetId }),
  });
  return payload.relation;
}

export async function acceptFriendRequest(token: string, id: string): Promise<void> {
  await request<unknown>(socialApi, `${REQUESTS_PATH}/${encodeURIComponent(id)}/accept`, {
    method: 'POST',
    headers: authHeader(token),
  });
}

export async function declineFriendRequest(token: string, id: string): Promise<void> {
  await request<unknown>(socialApi, `${REQUESTS_PATH}/${encodeURIComponent(id)}/decline`, {
    method: 'POST',
    headers: authHeader(token),
  });
}

export async function cancelFriendRequest(token: string, id: string): Promise<void> {
  await request<unknown>(socialApi, `${REQUESTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
    headers: authHeader(token),
  });
}
