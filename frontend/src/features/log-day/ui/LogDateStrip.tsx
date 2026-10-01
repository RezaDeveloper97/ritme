'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo } from 'react';

import { useLogDays } from '@/entities/health-log';
import type { Locale } from '@/shared/i18n';
import {
  addDays,
  formatNumber,
  formatWeekdayDayMonth,
  fromApiDate,
  toApiDate,
  toParts,
  today,
  weekdayLabels,
  weekOf,
} from '@/shared/lib/date';

/** Days shown before / after the selected one (nbl_Log_Sheet_Cycle: ۷ … ۱۲ … ۱۳). */
const BEFORE = 5;
const AFTER = 1;

function shortWeekday(date: Date, locale: Locale): string {
  const key = toApiDate(date);
  const index = weekOf(date, locale).findIndex((d) => toApiDate(d) === key);
  return weekdayLabels(locale)[index] ?? '';
}

/**
 * A rolling week around the logged day (5 before, 1 after) on the Night & Bloom DateStrip cells — the
 * locale-week `DateStrip` would put today first on a Saturday and leave the rest unpickable. Future days
 * are disabled; a day with entries gets a dot (red when it has a period flow).
 */
export function LogDateStrip({ date, onSelect }: { date: string; onSelect: (date: string) => void }) {
  const t = useTranslations('logSheet');
  const locale = useLocale() as Locale;
  const todayKey = toApiDate(today());
  const days = useMemo(() => {
    const base = fromApiDate(date);
    return Array.from({ length: BEFORE + AFTER + 1 }, (_, i) => addDays(base, i - BEFORE));
  }, [date]);
  const from = toApiDate(days[0]);
  const to = toApiDate(days[days.length - 1]) > todayKey ? todayKey : toApiDate(days[days.length - 1]);
  const logged = useLogDays(from, to, from <= to);
  const marks = useMemo(() => {
    const out = new Map<string, 'period' | 'brand'>();
    for (const d of logged.data?.days ?? []) {
      out.set(d.date, d.categories.bleeding?.flow ? 'period' : 'brand');
    }
    return out;
  }, [logged.data]);

  return (
    <div role="group" aria-label={t('dates.label')} className="nb-dates lday-dates">
      {days.map((day) => {
        const key = toApiDate(day);
        const selected = key === date;
        const tone = marks.get(key);
        const name = formatWeekdayDayMonth(day, locale);
        return (
          <button
            key={key}
            type="button"
            className={clsx('nb-date', selected && 'is-selected', key === todayKey && 'is-today')}
            aria-pressed={selected}
            aria-current={key === todayKey ? 'date' : undefined}
            aria-label={tone ? `${name} · ${t('dates.logged')}` : name}
            disabled={key > todayKey}
            onClick={() => onSelect(key)}
          >
            <span className="nb-date-wd" aria-hidden>
              {shortWeekday(day, locale)}
            </span>
            <span className="nb-date-num" aria-hidden>
              {formatNumber(toParts(day, locale).day, locale)}
            </span>
            {tone ? <span className={clsx('nb-date-dot', `nb-tone-${tone}`)} aria-hidden /> : null}
          </button>
        );
      })}
    </div>
  );
}
