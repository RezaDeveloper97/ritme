'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo } from 'react';

import type { Locale } from '@/shared/i18n';
import { addDays, formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';

import { clockTime, formatDuration, isoDay } from '../model/timing';

/** `m:ss` / `HH:mm` stay left-to-right inside RTL text (LRI … PDI isolate). */
const isolate = (text: string) => `⁦${text}⁩`;

/** Locale-aware formatting of the tools' numbers, durations and timestamps. */
export function useToolFormat() {
  const locale = useLocale() as Locale;
  const t = useTranslations('pregnancyTools.common');
  return useMemo(() => {
    const num = (n: number | string) => formatNumber(n, locale);
    const dur = (ms: number) => isolate(num(formatDuration(ms)));
    const time = (iso: string | null) => isolate(num(clockTime(iso)));
    /** «امروز · ۲۱:۱۵», «دیروز · …» or «۱۲ مهر · …». */
    const dayTime = (iso: string) => {
      const day = isoDay(iso);
      const now = today();
      const date =
        day === toApiDate(now)
          ? t('today')
          : day === toApiDate(addDays(now, -1))
            ? t('yesterday')
            : formatDayMonth(fromApiDate(day), locale);
      return t('dateTime', { date, time: time(iso) });
    };
    /** «۲ ساعت» / «۹۰ دقیقه» of a whole-minute window. */
    const span = (minutes: number) =>
      minutes % 60 === 0 ? t('hours', { n: minutes / 60 }) : t('minutes', { n: minutes });
    return { locale, num, dur, time, dayTime, span };
  }, [locale, t]);
}
