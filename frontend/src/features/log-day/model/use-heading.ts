'use client';

import { useQuery } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';

import { useCycleForDate } from '@/entities/cycle';
import { useLogTaxonomy } from '@/entities/health-log';
import { fetchPregnancyStatus, pregnancyKeys, trimesterOfWeek } from '@/entities/pregnancy';
import type { Locale } from '@/shared/i18n';
import { diffInDays, formatNumber, formatWeekdayDayMonth, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { isAuthenticated } from '@/shared/session';

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
  // Pregnancy / postpartum sheets (B-N3-06): «امروز چه خبر؟» over «هفته ۳۲ · سه‌ماهه سوم».
  const pregnancy = useQuery({
    queryKey: pregnancyKeys.status(),
    queryFn: fetchPregnancyStatus,
    enabled: mode === 'pregnancy' && isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
  if (mode === 'pregnancy' || mode === 'postpartum') {
    const title = isToday ? t('heading.today') : t('title.day');
    let subtitle = mode === 'postpartum' ? t('heading.postpartum', { date: dateText }) : dateText;
    const current = mode === 'pregnancy' ? (pregnancy.data?.currentWeek ?? null) : null;
    if (current !== null) {
      // The week of the logged day, counted back from today's week.
      const week = Math.max(1, current - Math.floor(diffInDays(today(), fromApiDate(day)) / 7));
      subtitle = isToday
        ? t('heading.pregnancy', { week: formatNumber(week, locale), trimester: t(`heading.trimester.${trimesterOfWeek(week)}`) })
        : t('heading.pregnancyDay', { date: dateText, week: formatNumber(week, locale) });
    }
    return { title, subtitle, panelSub: shortDate, isToday };
  }
  return {
    title: isToday ? t('title.today') : t('title.day'),
    subtitle: cycleDay ? t('subtitle', { date: dateText, day: formatNumber(cycleDay, locale) }) : dateText,
    panelSub: cycleDay ? t('panel.sub', { date: shortDate, day: formatNumber(cycleDay, locale) }) : shortDate,
    isToday,
  };
}
