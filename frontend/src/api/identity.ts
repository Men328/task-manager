import { asList, identityApi, request } from './client';
import type { CreateProfileInput, Profile } from '../types';

const PROFILES_PATH = '/v1/profiles';
const AUTH_PATH = '/v1/auth';
const GOOGLE_LOGIN_PATH = '/v1/auth/google/login';

export async function listProfiles(): Promise<Profile[]> {
  const payload = await request<unknown>(identityApi, PROFILES_PATH, { method: 'GET' });
  return asList<Profile>(payload, 'profiles');
}

export function createProfile(input: CreateProfileInput): Promise<Profile> {
  return request<unknown>(identityApi, PROFILES_PATH, {
    method: 'POST',
    body: JSON.stringify(input),
  }).then(unwrapProfile);
}

export function googleLoginUrl(): string {
  return `${identityApi}${GOOGLE_LOGIN_PATH}`;
}

export function getCurrentProfile(token: string): Promise<Profile> {
  return request<unknown>(identityApi, `${AUTH_PATH}/me`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` },
  }).then(unwrapProfile);
}

function unwrapProfile(payload: unknown): Profile {
  if (payload !== null && typeof payload === 'object' && 'profile' in payload) {
    return (payload as { profile: Profile }).profile;
  }
  return payload as Profile;
}
