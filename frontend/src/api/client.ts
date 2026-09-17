/**
 * Tiny JSON HTTP client on top of `fetch`.
 *
 * Every failure is normalised into an `ApiError` (never a bare TypeError), so
 * pages can catch one error type and render an Alert/Notification instead of
 * crashing when the Go/gRPC services are not running yet.
 */

export const IDENTITY_BASE_URL: string =
  (import.meta.env.VITE_IDENTITY_API_URL as string | undefined) ?? '/api/identity';

export const TASK_BASE_URL: string =
  (import.meta.env.VITE_TASK_API_URL as string | undefined) ?? '/api/task';

/** Base URL used by the identity service calls. */
export const identityApi = IDENTITY_BASE_URL;

/** Base URL used by the task service calls. */
export const taskApi = TASK_BASE_URL;

export class ApiError extends Error {
  /** HTTP status code, or 0 when the request never reached the server. */
  readonly status: number;
  /** Fully qualified request URL. */
  readonly url: string;
  /** Parsed response body (when any). */
  readonly payload: unknown;

  constructor(
    message: string,
    options: { status: number; url: string; payload?: unknown; cause?: unknown },
  ) {
    super(message, options.cause === undefined ? undefined : { cause: options.cause });
    this.name = 'ApiError';
    this.status = options.status;
    this.url = options.url;
    this.payload = options.payload;
  }
}

function joinUrl(baseUrl: string, path: string): string {
  const base = baseUrl.endsWith('/') ? baseUrl.slice(0, -1) : baseUrl;
  const suffix = path.startsWith('/') ? path : `/${path}`;
  return `${base}${suffix}`;
}

function parseBody(raw: string): unknown {
  if (!raw) {
    return undefined;
  }
  try {
    return JSON.parse(raw) as unknown;
  } catch {
    return raw;
  }
}

function describePayload(payload: unknown): string {
  if (payload === undefined || payload === null) {
    return '';
  }
  if (typeof payload === 'string') {
    return payload ? ` - ${payload.slice(0, 200)}` : '';
  }
  if (typeof payload === 'object') {
    const record = payload as Record<string, unknown>;
    const detail = record.message ?? record.error ?? record.detail ?? record.code;
    if (typeof detail === 'string' && detail) {
      return ` - ${detail}`;
    }
    try {
      return ` - ${JSON.stringify(payload).slice(0, 200)}`;
    } catch {
      return '';
    }
  }
  return ` - ${String(payload)}`;
}

/**
 * Perform a JSON request against `baseUrl + path`.
 *
 * @throws {ApiError} on network failure or any non-2xx response.
 */
export async function request<T>(
  baseUrl: string,
  path: string,
  init: RequestInit = {},
): Promise<T> {
  const url = joinUrl(baseUrl, path);
  const method = (init.method ?? 'GET').toUpperCase();

  const headers = new Headers(init.headers);
  if (!headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }
  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json');
  }

  let response: Response;
  try {
    response = await fetch(url, { ...init, headers });
  } catch (cause) {
    throw new ApiError(
      `Cannot reach the API (${method} ${url}). The backend may be offline - check that the service is running.`,
      { status: 0, url, cause },
    );
  }

  const raw = await response.text();
  const payload = parseBody(raw);

  if (!response.ok) {
    const statusText = response.statusText ? ` ${response.statusText}` : '';
    throw new ApiError(
      `HTTP ${response.status}${statusText} for ${method} ${url}${describePayload(payload)}`,
      { status: response.status, url, payload },
    );
  }

  return payload as T;
}

/** Best-effort, user friendly message for any thrown value. */
export function getErrorMessage(error: unknown): string {
  if (error instanceof ApiError) {
    return error.message;
  }
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === 'string') {
    return error;
  }
  try {
    return JSON.stringify(error);
  } catch {
    return 'Unknown error';
  }
}

/**
 * Tolerate both a bare array (`[...]`) and the gRPC-gateway envelope
 * (`{ "<key>": [...], "nextPageToken": "" }`).
 */
export function asList<T>(payload: unknown, key: string): T[] {
  if (Array.isArray(payload)) {
    return payload as T[];
  }
  if (payload !== null && typeof payload === 'object') {
    const value = (payload as Record<string, unknown>)[key];
    if (Array.isArray(value)) {
      return value as T[];
    }
  }
  return [];
}

/** Build a query string from defined values only. */
export function buildQuery(params: Record<string, string | number | boolean | undefined | null>): string {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      search.set(key, String(value));
    }
  }
  const query = search.toString();
  return query ? `?${query}` : '';
}
