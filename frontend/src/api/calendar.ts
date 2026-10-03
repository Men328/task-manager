/** calendar service HTTP calls. */

import { asList, buildQuery, calendarApi, request } from './client';
import type {
  CreateScheduleInput,
  ListSchedulesParams,
  Schedule,
  UpdateScheduleInput,
} from '../types';

// Path khớp `option (google.api.http)` trong common/proto/calendar/v1/schedule.proto.
// Base URL là service root (/api/calendar), proxy của Vite/nginx bỏ prefix rồi forward.
const SCHEDULES_PATH = '/v1/schedules';

function toCreateBody(input: CreateScheduleInput): Record<string, unknown> {
  return {
    profile_id: input.profileId,
    title: input.title,
    description: input.description,
    location: input.location,
    start_at: input.startAt,
    end_at: input.endAt,
    all_day: input.allDay,
    color: input.color,
  };
}

/**
 * Chỉ gửi field nào thực sự được set: PATCH dùng presence của proto field
 * (bỏ field = không đổi).
 */
function toUpdateBody(input: UpdateScheduleInput): Record<string, unknown> {
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
  return body;
}

/** GET /v1/schedules?profile_id=...&from=...&to=... */
export async function listSchedules(params: ListSchedulesParams): Promise<Schedule[]> {
  const payload = await request<unknown>(
    calendarApi,
    `${SCHEDULES_PATH}${buildQuery({
      profile_id: params.profileId,
      from: params.from,
      to: params.to,
    })}`,
    { method: 'GET' },
  );
  return asList<Schedule>(payload, 'schedules');
}

/** GET /v1/schedules/{id} */
export function getSchedule(id: string): Promise<Schedule> {
  return request<unknown>(calendarApi, `${SCHEDULES_PATH}/${encodeURIComponent(id)}`, {
    method: 'GET',
  }).then((payload) => unwrap<Schedule>(payload, 'schedule'));
}

/** POST /v1/schedules */
export function createSchedule(input: CreateScheduleInput): Promise<Schedule> {
  return request<unknown>(calendarApi, SCHEDULES_PATH, {
    method: 'POST',
    body: JSON.stringify(toCreateBody(input)),
  }).then((payload) => unwrap<Schedule>(payload, 'schedule'));
}

/** PATCH /v1/schedules/{id} */
export function updateSchedule(id: string, input: UpdateScheduleInput): Promise<Schedule> {
  return request<unknown>(calendarApi, `${SCHEDULES_PATH}/${encodeURIComponent(id)}`, {
    method: 'PATCH',
    body: JSON.stringify(toUpdateBody(input)),
  }).then((payload) => unwrap<Schedule>(payload, 'schedule'));
}

/** DELETE /v1/schedules/{id} */
export function deleteSchedule(id: string): Promise<void> {
  return request<void>(calendarApi, `${SCHEDULES_PATH}/${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

/**
 * Response của gateway là envelope 1 field (`{ "schedule": {...} }`).
 * Hàm này bóc envelope, đồng thời chấp nhận cả payload trần.
 */
function unwrap<T>(payload: unknown, key: string): T {
  if (payload !== null && typeof payload === 'object' && key in (payload as Record<string, unknown>)) {
    return (payload as Record<string, T>)[key]!;
  }
  return payload as T;
}
