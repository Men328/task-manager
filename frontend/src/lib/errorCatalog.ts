import catalog from '../config/error_codes.json';

export interface LocalizedErrorMessage {
  vi: string;
  en: string;
}

export interface ErrorCodeEntry {
  owner?: string;
  grpcCode?: string;
  httpStatus?: number;
  message?: Partial<LocalizedErrorMessage>;
}

interface ErrorCatalog {
  version?: string;
  domain?: string;
  defaultCode?: string;
  codes: Record<string, ErrorCodeEntry>;
}

const file = catalog as unknown as ErrorCatalog;

export const ERROR_CODES: Record<string, ErrorCodeEntry> = file.codes;

export const DEFAULT_ERROR_CODE: string = file.defaultCode ?? 'COMMON_INTERNAL';

export const ERROR_CATALOG_VERSION: string = file.version ?? '0.0.0';

export type SupportedLanguage = 'vi' | 'en';

function normalizeLanguage(language: string | undefined): SupportedLanguage {
  return language?.toLowerCase().startsWith('en') ? 'en' : 'vi';
}

export function messageForCode(code: string | null | undefined, language?: string): string | null {
  if (!code) {
    return null;
  }
  const message = ERROR_CODES[code]?.message;
  if (!message) {
    return null;
  }
  const lang = normalizeLanguage(language);
  const preferred = message[lang];
  const fallback = lang === 'vi' ? message.en : message.vi;
  const text = (preferred ?? fallback ?? '').trim();
  return text.length > 0 ? text : null;
}

export function fallbackCodeForStatus(status: number): string {
  if (status === 0) {
    return 'CLIENT_NETWORK_ERROR';
  }
  if (status === 400 || status === 422) {
    return 'CLIENT_BAD_REQUEST';
  }
  if (status === 401) {
    return 'CLIENT_UNAUTHORIZED';
  }
  if (status === 403) {
    return 'CLIENT_FORBIDDEN';
  }
  if (status === 404) {
    return 'CLIENT_NOT_FOUND';
  }
  if (status === 409) {
    return 'CLIENT_CONFLICT';
  }
  if (status === 429) {
    return 'CLIENT_TOO_MANY_REQUESTS';
  }
  if (status === 503) {
    return 'CLIENT_UNAVAILABLE';
  }
  if (status >= 500) {
    return 'CLIENT_SERVER_ERROR';
  }
  if (status >= 400) {
    return 'CLIENT_BAD_REQUEST';
  }
  return 'CLIENT_UNKNOWN_ERROR';
}
