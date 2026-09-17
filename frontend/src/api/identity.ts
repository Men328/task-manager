/** identity service HTTP calls. */

import { asList, identityApi, request } from './client';
import type { CreateProfileInput, Profile } from '../types';

const PROFILES_PATH = '/v1/profiles';

/** GET /v1/profiles */
export async function listProfiles(): Promise<Profile[]> {
  const payload = await request<unknown>(identityApi, PROFILES_PATH, { method: 'GET' });
  return asList<Profile>(payload, 'profiles');
}

/** POST /v1/profiles */
export function createProfile(input: CreateProfileInput): Promise<Profile> {
  return request<unknown>(identityApi, PROFILES_PATH, {
    method: 'POST',
    body: JSON.stringify(input),
  }).then((payload) => {
    // gateway trả envelope { "profile": {...} }
    if (payload !== null && typeof payload === 'object' && 'profile' in payload) {
      return (payload as { profile: Profile }).profile;
    }
    return payload as Profile;
  });
}
