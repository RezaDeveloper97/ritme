import { apiClient } from '@/shared/api';

import type { Language } from '../model/types';

interface LanguagesPayload {
  data: {
    default: string;
    languages: {
      code: string;
      name: string;
      english_name: string;
      direction: string;
      is_default: boolean;
    }[];
  };
}

/** The compiled-in locales, shown until the request resolves and if it fails. */
export const FALLBACK_LANGUAGES: Language[] = [
  { code: 'fa', name: 'فارسی', englishName: 'Persian', direction: 'rtl', isDefault: true },
  { code: 'en', name: 'English', englishName: 'English', direction: 'ltr', isDefault: false },
];

export async function fetchLanguages(): Promise<Language[]> {
  const { data } = await apiClient.get<LanguagesPayload>('/languages');

  const languages = data.data.languages.map<Language>((row) => ({
    code: row.code,
    name: row.name,
    englishName: row.english_name,
    direction: row.direction === 'rtl' ? 'rtl' : 'ltr',
    isDefault: row.is_default || row.code === data.data.default,
  }));

  return languages.length > 0 ? languages : FALLBACK_LANGUAGES;
}
