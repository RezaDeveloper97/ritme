'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatNumber } from '@/shared/lib';

import { dayRange } from '../lib/day-range';

/** "Every day" / "Days 3–7" / "From day 3" / "Up to day 7" / "Day 3". */
export function useDayLabel() {
  const t = useTranslations('challenges.days');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  return (from: number | null, to: number | null): string => {
    const r = dayRange(from, to);
    switch (r.kind) {
      case 'all':
        return t('all');
      case 'single':
        return t('single', { day: n(r.day) });
      case 'range':
        return t('range', { from: n(r.from), to: n(r.to) });
      case 'from':
        return t('from', { from: n(r.from) });
      case 'to':
        return t('to', { to: n(r.to) });
    }
  };
}
