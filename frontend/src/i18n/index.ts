import i18n from 'i18next';
import type { Resource } from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

import {
  DEFAULT_LANGUAGE,
  DETECTION,
  FALLBACK_LANGUAGE,
  LANGUAGE_QUERY_PARAM,
  LANGUAGE_STORAGE_KEY,
  LOAD_MODE,
  NON_EXPLICIT_SUPPORTED_LNGS,
  SUPPORTED_LANGUAGES,
} from './types';

interface LocaleModule {
  default: Record<string, unknown>;
}

/**
 * Nạp mọi file `locales/*.json`. Thêm ngôn ngữ mới chỉ cần:
 *   1. thêm `locales/<code>.json`
 *   2. khai báo `<code>` trong `config.json` → `languages`
 * Không phải sửa code TypeScript.
 */
const localeModules = import.meta.glob<LocaleModule>('./locales/*.json', { eager: true });

const resources = Object.fromEntries(
  Object.entries(localeModules).map(([path, module]) => {
    const code = path.slice(path.lastIndexOf('/') + 1).replace(/\.json$/, '');
    return [code, { translation: module.default }];
  }),
) as Resource;

function applyDocumentLanguage(language: string): void {
  document.documentElement.lang = language;
  document.title = i18n.t('app.title');

  const description = document.querySelector('meta[name="description"]');
  if (description) {
    description.setAttribute('content', i18n.t('app.description'));
  }
}

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    supportedLngs: SUPPORTED_LANGUAGES,
    fallbackLng: FALLBACK_LANGUAGE,
    nonExplicitSupportedLngs: NON_EXPLICIT_SUPPORTED_LNGS,
    load: LOAD_MODE,
    detection: {
      order: DETECTION.order,
      caches: DETECTION.caches,
      lookupLocalStorage: LANGUAGE_STORAGE_KEY,
      lookupQuerystring: LANGUAGE_QUERY_PARAM,
    },
    interpolation: {
      escapeValue: false,
    },
  })
  .then(() => {
    applyDocumentLanguage(i18n.resolvedLanguage ?? DEFAULT_LANGUAGE);
  });

i18n.on('languageChanged', (language) => {
  applyDocumentLanguage(language);
});

export default i18n;
