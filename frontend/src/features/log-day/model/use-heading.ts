'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useCycleForDate } from '@/entities/cycle';
import { useLogTaxonomy } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import { formatNumber, formatWeekdayDayMonth, fromApiDate, toApiDate, today } from '@/shared/lib/date';

/** Modes whose log heading shows the cycle day («روز ۲۵ سیکل»). */
const CYCLE_MODES = ['cycle', 'ttc', 'teen'];

export interface LogDayHeading {
  title: string;
  subtitle: string;
  /** Short form for detail panels: «امروز · روز ۲۵» / «۱۲ مهر · روز ۲۵». */
  panelSub: string;
  isToday: boolean;
}

/** «ثبت امروز» + «شنبه، ۱۲ مهر · روز ۲۵ سیکل» for the day being logged. */
export function useLogDayHeading(date: string | null, modeOverride?: string): LogDayHeading {
  const t = useTranslations('logSheet');
  const locale = useLocale() as Locale;
  const day = date ?? toApiDate(today());
  const isToday = day === toApiDate(today());
  const taxonomy = useLogTaxonomy(modeOverride);
  const mode = taxonomy.data?.mode ?? modeOverride ?? null;
  const withCycle = mode !== null && CYCLE_MODES.includes(mode);
  const cycle = useCycleForDate(day, withCycle);
  const cycleDay = withCycle ? (cycle.data?.cycleView?.cycleDay ?? null) : null;
  const dateText = formatWeekdayDayMonth(fromApiDate(day), locale);
  const shortDate = isToday ? t('panel.today') : formatWeekdayDayMonth(fromApiDate(day), locale);
  return {
    title: isToday ? t('title.today') : t('title.day'),
    subtitle: cycleDay ? t('subtitle', { date: dateText, day: formatNumber(cycleDay, locale) }) : dateText,
    panelSub: cycleDay ? t('panel.sub', { date: shortDate, day: formatNumber(cycleDay, locale) }) : shortDate,
    isToday,
  };
}
