/** event service HTTP calls. */

import { asList, buildQuery, eventApi, request } from './client';
import type {
  CreateEventInput,
  Event,
  ListEventsParams,
  UpdateEventInput,
} from '../types';

// Path khớp `option (google.api.http)` trong common/proto/event/v1/event.proto.
const EVENTS_PATH = '/v1/events';

function toCreateBody(input: CreateEventInput): Record<string, unknown> {
  return {
    profile_id: input.profileId,
    title: input.title,
    description: input.description,
    location: input.location,
    start_at: input.startAt,
    end_at: input.endAt,
    all_day: input.allDay,
    color: input.color,
    status: input.status,
    source: input.source,
  };
}

function toUpdateBody(input: UpdateEventInput): Record<string, unknown> {
  const body: Record<string, unknown> = {};
  if (input.title !== undefined) {
    body.title = input.title;
  }
  if (input.description !== undefined) {
    body.description = input.description;
  }
  if (input.location !== undefined) {
    body.location = input.location;
  }
  if (input.startAt !== undefined) {
    body.start_at = input.startAt;
  }
  if (input.endAt !== undefined) {
    body.end_at = input.endAt;
  }
  if (input.allDay !== undefined) {
    body.all_day = input.allDay;
  }
  if (input.color !== undefined) {
    body.color = input.color;
  }
  if (input.status !== undefined) {
    body.status = input.status;
  }
  return body;
}

/** GET /v1/events?profile_id=...&from=...&to=...&status=... */
export async function listEvents(params: ListEventsParams): Promise<Event[]> {
  const payload = await request<unknown>(
    eventApi,
    `${EVENTS_PATH}${buildQuery({
      profile_id: params.profileId,
      from: params.from,
      to: params.to,
      status: params.status,
    })}`,
    { method: 'GET' },
  );
  return asList<Event>(payload, 'events');
}

/** GET /v1/events/{id} */
export function getEvent(id: string): Promise<Event> {
  return request<unknown>(eventApi, `${EVENTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'GET',
  }).then((payload) => unwrap<Event>(payload, 'event'));
}

/** POST /v1/events */
export function createEvent(input: CreateEventInput): Promise<Event> {
  return request<unknown>(eventApi, EVENTS_PATH, {
    method: 'POST',
    body: JSON.stringify(toCreateBody(input)),
  }).then((payload) => unwrap<Event>(payload, 'event'));
}

/** PATCH /v1/events/{id} */
export function updateEvent(id: string, input: UpdateEventInput): Promise<Event> {
  return request<unknown>(eventApi, `${EVENTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(toUpdateBody(input)),
  }).then((payload) => unwrap<Event>(payload, 'event'));
}

/** DELETE /v1/events/{id} */
export function deleteEvent(id: string): Promise<void> {
  return request<void>(eventApi, `${EVENTS_PATH}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}
