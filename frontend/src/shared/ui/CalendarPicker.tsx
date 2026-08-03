'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import {
  daysInCalendarMonth,
  type DateParts,
  formatMonthLabel,
  formatNumber,
  monthMatrix,
  todayParts,
  weekdayLabels,
} from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';

import { Icon } from './Icon';

interface CalendarPickerProps {
  /** The currently-selected full date, or `null` if nothing is picked. */
  value: DateParts | null;
  /** Reports the full picked date (year/month/day), not just the day number. */
  onSelect: (value: DateParts) => void;
}

/**
 * Month calendar for selecting a date, rendered in the calendar the user's
 * locale reads — Jalali for `fa`, Gregorian for `en` (CLAUDE.md §7). The parts
 * it reports are therefore in that calendar too.
 */
export function CalendarPicker({ value, onSelect }: CalendarPickerProps) {
  const locale = useLocale() as Locale;
  const t = useTranslations('common');
  const isRtl = locale === 'fa';

  const now = todayParts(locale);
  const [year, setYear] = useState(now.year);
  const [month, setMonth] = useState(now.month);

  // Length and weekday alignment both come from the date layer (§7) — Esfand is
  // 29 or 30 days depending on the leap year, and the first weekday can't be
  // derived from the month number.
  const daysInMonth = daysInCalendarMonth(year, month, locale);
  const offset = useMemo(
    () => monthMatrix(year, month, locale)[0]!.findIndex((cell) => cell !== null),
    [year, month, locale],
  );

  const prevMonth = () => {
    if (month === 1) { setYear(y => y - 1); setMonth(12); }
    else setMonth(m => m - 1);
  };
  const nextMonth = () => {
    if (month === 12) { setYear(y => y + 1); setMonth(1); }
    else setMonth(m => m + 1);
  };

  /*
   * In RTL layout (fa):
   *   – First flex child → appears on the RIGHT side
   *   – Last  flex child → appears on the LEFT  side
   *
   * So for a Jalali calendar in RTL:
   *   RIGHT = prevMonth (→  chevronRight, going to older month)
   *   LEFT  = nextMonth (←  chevronLeft,  going to newer month)
   *
   * In LTR layout (en) the positions swap automatically with flex.
   */
  return (
    <div className="card jcal">
      <div className="jcal-nav">
        {/* First child → RIGHT in RTL = "prev month" */}
        <button
          className="iconbtn"
          onClick={isRtl ? prevMonth : nextMonth}
          aria-label={isRtl ? t('calendar.prevMonth') : t('calendar.nextMonth')}
        >
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>

        <span className="jcal-month">{formatMonthLabel(year, month, locale)}</span>

        {/* Last child → LEFT in RTL = "next month" */}
        <button
          className="iconbtn"
          onClick={isRtl ? nextMonth : prevMonth}
          aria-label={isRtl ? t('calendar.nextMonth') : t('calendar.prevMonth')}
        >
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      {/* Day-of-week headers */}
      <div className="cal-grid jcal-weekdays">
        {weekdayLabels(locale).map(w => (
          <span key={w} className="jcal-weekday">{w}</span>
        ))}
      </div>

      {/* Days grid */}
      <div className="cal-grid">
        {Array.from({ length: offset }, (_, i) => <span key={`pad-${i}`} />)}
        {Array.from({ length: daysInMonth }, (_, i) => {
          const d = i + 1;
          // Highlight only when the selection falls inside the month on screen,
          // so navigating months doesn't leave a stale highlight on a same-numbered day.
          const isSelected = value?.year === year && value?.month === month && value?.day === d;
          return (
            <button
              key={d}
              className={`cday${isSelected ? ' on' : ''}`}
              onClick={() => onSelect({ year, month, day: d })}
            >
              {formatNumber(d, locale)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
