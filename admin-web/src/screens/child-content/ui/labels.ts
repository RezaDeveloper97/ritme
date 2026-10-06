'use client';

import { useTranslations } from 'next-intl';

import { useNumber } from '@/shared/lib';

import { VISIT_MONTHS } from '../lib/meta';

/** Age in months → «بدو تولد» / «۴ ماهگی» / «۶ سالگی» (whole years from 24 months). */
export function useAgeLabel() {
  const t = useTranslations('childContent.age');
  const n = useNumber();
  return (months: number) => {
    if (months === 0) return t('birth');
    if (months >= 24 && months % 12 === 0) return t('years', { n: n(months / 12) });
    return t('months', { n: n(months) });
  };
}

/** A visit code → its age label for the seeded codes, else the code itself. */
export function useVisitLabel() {
  const age = useAgeLabel();
  return (visit: string) => {
    const m = VISIT_MONTHS[visit];
    return m === undefined ? visit : age(m);
  };
}

/** domain / topic code → its name (unknown codes as-is). */
export function useCodeLabels() {
  const t = useTranslations('childContent');
  const pick = (ns: 'domains' | 'topics', key: string) => (t.has(`${ns}.${key}` as 'title') ? t(`${ns}.${key}` as 'title') : key);
  return { domain: (k: string) => pick('domains', k), topic: (k: string) => pick('topics', k) };
}
