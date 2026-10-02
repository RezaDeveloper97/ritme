'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { useFetalMovement } from '@/entities/pregnancy';
import type { Locale } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';

/** The `pregnancy.kicks` quick tile key (nbl_Log_Sheet_Preg «حرکات جنین»). */
export const KICKS_TILE = 'pregnancy.kicks';

/**
 * Reads the day's fetal-movement log (`/pregnancy/fetal-movement`, the kick counter's store) and hands the
 * «حرکات جنین» tile its caption: «۸ امروز». Rendered only on the pregnancy sheet, so no one else fetches it.
 */
export function KickCountSub({ date, children }: { date: string; children: (subs: Record<string, string>) => ReactNode }) {
  const t = useTranslations('logSheet.presets.pregnancy');
  const locale = useLocale() as Locale;
  const movement = useFetalMovement(date);
  const count = movement.data?.movement_count;
  const subs: Record<string, string> = {};
  if (typeof count === 'number' && count > 0) {
    const n = formatNumber(count, locale);
    subs[KICKS_TILE] = date === toApiDate(today()) ? t('kicksToday', { count: n }) : t('kicksDay', { count: n });
  }
  return <>{children(subs)}</>;
}
