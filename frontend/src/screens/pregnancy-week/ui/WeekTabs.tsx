'use client';

import { useLocale, useTranslations } from 'next-intl';

import { V2_MAX_WEEK, V2_MIN_WEEK } from '@/entities/pregnancy';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { type SegmentedTab, SegmentedTabs } from '@/shared/ui';

/**
 * «هفته ۷ · هفته ۸ · هفته ۹» — the viewed week between its neighbours
 * (PregFull_Week). Route-driven: picking a tab replaces `/pregnancy/weeks/[n]`.
 * At the ends of the 1–42 range the window slides so there are always three.
 */
export function WeekTabs({ week }: { week: number }) {
  const t = useTranslations('pregnancyV2.week');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const first = Math.min(Math.max(V2_MIN_WEEK, week - 1), V2_MAX_WEEK - 2);
  const tabs: SegmentedTab<`${number}`>[] = [first, first + 1, first + 2].map((n) => ({
    value: `${n}`,
    label: t('tabWeek', { week: formatNumber(n, locale) }),
  }));
  return (
    <SegmentedTabs
      tabs={tabs}
      value={`${week}`}
      label={t('stripLabel')}
      onChange={(v) => router.replace(`/pregnancy/weeks/${v}`, { scroll: false })}
    />
  );
}
