'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useCallback } from 'react';

import { formatNumber } from '@/shared/lib';

type Timing = {
  interval_months: number;
  interval_months_max: number | null;
  age_min: number | null;
  age_max: number | null;
  cycle_day_from: number | null;
  cycle_day_to: number | null;
};

/** Labels for the plain value lists `/checkup-types/options` returns. */
export function useCheckupLabels() {
  const t = useTranslations('checkupTypes');
  const locale = useLocale();
  // Numbers go through the admin formatter so fa reads «هر ۱۲ ماه», not «هر 12 ماه» (known 5c).
  const n = useCallback((v: number) => formatNumber(v, locale), [locale]);
  const label = useCallback(
    (group: 'category' | 'performer' | 'tone' | 'audience', value: string) => {
      const key = `${group}Opt.${value}` as 'categoryOpt.monthly';
      return t.has(key) ? t(key) : value;
    },
    [t],
  );
  const interval = useCallback(
    (r: Pick<Timing, 'interval_months' | 'interval_months_max'>) =>
      r.interval_months_max && r.interval_months_max > r.interval_months
        ? t('intervalRange', { min: n(r.interval_months), max: n(r.interval_months_max) })
        : t('intervalEvery', { n: n(r.interval_months) }),
    [n, t],
  );
  const age = useCallback(
    (r: Pick<Timing, 'age_min' | 'age_max'>) =>
      r.age_min !== null && r.age_max !== null
        ? t('ageRange', { min: n(r.age_min), max: n(r.age_max) })
        : r.age_min !== null
          ? t('ageFrom', { min: n(r.age_min) })
          : r.age_max !== null
            ? t('ageTo', { max: n(r.age_max) })
            : t('ageAny'),
    [n, t],
  );
  const timing = useCallback(
    (r: Timing) => {
      const parts = [interval(r)];
      if (r.age_min !== null || r.age_max !== null) parts.push(age(r));
      if (r.cycle_day_from !== null && r.cycle_day_to !== null) {
        parts.push(t('cycleDays', { from: n(r.cycle_day_from), to: n(r.cycle_day_to) }));
      }
      return parts.join(' · ');
    },
    [age, interval, n, t],
  );
  return { label, interval, age, timing };
}
