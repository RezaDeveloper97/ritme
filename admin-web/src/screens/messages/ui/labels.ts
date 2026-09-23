'use client';

import { useTranslations } from 'next-intl';

import { useContentLanguages } from '@/shared/i18n';

/** Group key → its editorial name; locale code → the content language's name. Unknown values as-is. */
export function useMessageLabels() {
  const t = useTranslations('smartMessages.groups');
  const languages = useContentLanguages().data?.languages;
  return {
    group: (group: string) => (t.has(group as 'pattern') ? t(group as 'pattern') : group),
    locale: (code: string) => languages?.find((l) => l.code === code)?.name ?? code,
    direction: (code: string): 'rtl' | 'ltr' | undefined => languages?.find((l) => l.code === code)?.direction,
  };
}
