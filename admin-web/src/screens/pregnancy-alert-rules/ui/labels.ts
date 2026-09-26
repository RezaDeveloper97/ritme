'use client';

import { useTranslations } from 'next-intl';

/** Rule / level / action / param key → its editorial name (unknown keys as-is). */
export function useAlertLabels() {
  const t = useTranslations('pregnancyAlertRules');
  const pick = (ns: string, key: string) => (t.has(`${ns}.${key}` as 'title') ? t(`${ns}.${key}` as 'title') : key);
  return {
    rule: (key: string) => pick('rules', key),
    level: (key: string) => pick('levels', key),
    action: (key: string) => pick('actions', key),
    param: (key: string) => pick('params', key),
  };
}
