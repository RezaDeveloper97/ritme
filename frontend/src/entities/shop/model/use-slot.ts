'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatNumber, formatWeekdayDayMonth, fromApiDate } from '@/shared/lib/date';

interface SlotLike {
  date: string;
  starts_at: string;
  ends_at: string;
}

/** «شنبه، ۱۹ مهر · ۹ تا ۱۳» — a delivery window in the locale's calendar and digits. */
export function useSlotText() {
  const t = useTranslations('shop');
  const locale = useLocale() as Locale;
  const hour = (v: string) => formatNumber(v.slice(0, 5), locale);
  return (slot: SlotLike) =>
    t('order.slot', { day: formatWeekdayDayMonth(fromApiDate(slot.date), locale), from: hour(slot.starts_at), to: hour(slot.ends_at) });
}
