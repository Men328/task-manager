import { useTranslation } from 'react-i18next';

import { DEFAULT_LANGUAGE, LANGUAGES } from './types';
import type { Language } from './types';

export function useLanguage() {
  const { i18n } = useTranslation();
  const resolved = i18n.resolvedLanguage ?? i18n.language ?? DEFAULT_LANGUAGE;
  const language = (LANGUAGES.find((item) => item.code === resolved)?.code ??
    DEFAULT_LANGUAGE) as Language;

  return {
    language,
    languages: LANGUAGES,
    changeLanguage: (next: Language) => {
      void i18n.changeLanguage(next);
    },
  };
}
