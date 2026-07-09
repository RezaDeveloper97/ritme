'use client';

import dayjs from 'dayjs';
import jalaliday from 'jalaliday';
import { useLocale } from 'next-intl';
import { useState } from 'react';

import type { JalaliParts } from '@/shared/lib/date';

import { Icon } from './Icon';

dayjs.extend(jalaliday as Parameters<typeof dayjs.extend>[0]);

const FA_MONTHS = [
  'فروردین','اردیبهشت','خرداد','تیر','مرداد','شهریور',
  'مهر','آبان','آذر','دی','بهمن','اسفند',
] as const;

const DAY_NAMES = ['ش','ی','د','س','چ','پ','ج'] as const;

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

interface JalaliCalendarProps {
  /** The currently-selected full Jalali date, or `null` if nothing is picked. */
  value: JalaliParts | null;
  /** Reports the full picked date (year/month/day), not just the day number. */
  onSelect: (value: JalaliParts) => void;
}

function getDaysInJalaliMonth(year: number, month: number): number {
  if (month <= 6) return 31;
  if (month <= 11) return 30;
  return 29;
}

function getJalaliMonthOffset(year: number, month: number): number {
  const d = (dayjs() as unknown as { calendar: (c: string) => typeof dayjs['prototype'] })
    .calendar('jalali')
    .year(year).month(month - 1).date(1).day();
  return (d + 1) % 7;
}

/** Jalali month calendar for selecting the last period date. */
export function JalaliCalendar({ value, onSelect }: JalaliCalendarProps) {
  const locale = useLocale();
  const isRtl = locale === 'fa';

  const now = dayjs() as unknown as { calendar: (c: string) => { year: () => number; month: () => number } };
  const j = now.calendar('jalali');
  const [year, setYear] = useState(j.year());
  const [month, setMonth] = useState(j.month() + 1);

  const daysInMonth = getDaysInJalaliMonth(year, month);
  const offset = getJalaliMonthOffset(year, month);

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
    <div className="card" style={{ padding: '16px 14px', marginTop: 2 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14 }}>
        {/* First child → RIGHT in RTL = "prev month" */}
        <button
          className="iconbtn"
          onClick={isRtl ? prevMonth : nextMonth}
          aria-label={isRtl ? 'ماه قبل' : 'Next month'}
        >
          <Icon name={isRtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>

        <span style={{ fontWeight: 800, fontSize: 15, color: 'var(--ink)' }}>
          {FA_MONTHS[month - 1]} {faNum(year)}
        </span>

        {/* Last child → LEFT in RTL = "next month" */}
        <button
          className="iconbtn"
          onClick={isRtl ? nextMonth : prevMonth}
          aria-label={isRtl ? 'ماه بعد' : 'Previous month'}
        >
          <Icon name={isRtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>

      {/* Day-of-week headers */}
      <div className="cal-grid" style={{ marginBottom: 8 }}>
        {DAY_NAMES.map(w => (
          <span key={w} style={{ fontSize: 11, color: 'var(--muted)', fontWeight: 700, textAlign: 'center' }}>{w}</span>
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
              {faNum(d)}
            </button>
          );
        })}
      </div>
    </div>
  );
}
