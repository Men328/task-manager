import config from './config.json';

export interface LanguageOption {
  code: string;
  label: string;
  short: string;
}

/**
 * Toàn bộ cấu hình i18n nằm ở `config.json` (không hardcode trong TypeScript).
 * File này chỉ là lớp type hoá mỏng cho phần còn lại của app.
 */
export const LANGUAGES: LanguageOption[] = config.languages;

export type Language = string;

export const SUPPORTED_LANGUAGES: string[] = LANGUAGES.map((item) => item.code);

export const DEFAULT_LANGUAGE: string = config.defaultLanguage;

export const FALLBACK_LANGUAGE: string = config.fallbackLanguage;

export const LOAD_MODE = config.load as 'all' | 'currentOnly' | 'languageOnly';

export const NON_EXPLICIT_SUPPORTED_LNGS: boolean = config.nonExplicitSupportedLngs;

export const DETECTION = config.detection;

export const LANGUAGE_STORAGE_KEY: string = config.detection.storageKey;

export const LANGUAGE_QUERY_PARAM: string = config.detection.queryParam;
