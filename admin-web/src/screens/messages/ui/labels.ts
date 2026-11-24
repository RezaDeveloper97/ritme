'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useContentLanguages } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib';

import { areaOf, describeItemKey } from '../lib/where';

/**
 * Group key → its editorial name and where the app shows it; item key → a readable name; locale code → the content
 * language's name. Unknown values as-is.
 */
export function useMessageLabels() {
  const t = useTranslations('smartMessages');
  const uiLocale = useLocale();
  const languages = useContentLanguages().data?.languages;
  const known = (key: string) => t.has(`itemKeys.${key}` as 'itemKeys.base');
  const word = (key: string) => t(`itemKeys.${key}` as 'itemKeys.base');
  const n = (v: number) => formatNumber(v, uiLocale);
  return {
    group: (group: string) => (t.has(`groups.${group}` as 'groups.pattern') ? t(`groups.${group}` as 'groups.pattern') : group),
    /** Where the group appears in the user app (`null` for a group this panel does not know yet). */
    where: (group: string) => (t.has(`where.${group}` as 'where.pattern') ? t(`where.${group}` as 'where.pattern') : null),
    /** False for groups the app does not render anywhere right now. */
    shown: (group: string) => areaOf(group) !== 'notShown',
    area: (group: string) => t(`areas.${areaOf(group) ?? 'other'}` as 'areas.other'),
    /** A readable name of the item key, or `null` when there is none (show the raw key alone). */
    item: (group: string, key: string): string | null => {
      const d = describeItemKey(group, key, known);
      switch (d.kind) {
        case 'week':
          return t('itemWeek', { n: n(d.n) });
        case 'trimester':
          return t('itemTrimester', { n: n(d.n) });
        case 'phaseTip':
          return t('itemPhaseTip', { phase: word(d.phase), n: n(d.n) });
        case 'phaseNote':
          return t('itemPhaseNote', { phase: word(d.phase) });
        case 'key':
          return word(d.key);
        default:
          return null;
      }
    },
    locale: (code: string) => languages?.find((l) => l.code === code)?.name ?? code,
    direction: (code: string): 'rtl' | 'ltr' | undefined => languages?.find((l) => l.code === code)?.direction,
  };
}
