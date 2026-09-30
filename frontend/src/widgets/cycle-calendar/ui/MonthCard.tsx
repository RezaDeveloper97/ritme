'use client';

import clsx from 'clsx';
import type { TouchEventHandler } from 'react';

import { useDirection, type Locale } from '@/shared/i18n';
import { diffInDays, formatMonthLabel, monthMatrix, today } from '@/shared/lib/date';
import { HeaderButton } from '@/shared/ui';

import { bandEdges, type DayTone } from '../model/tones';

/** What a day cell needs to know about its date (computed by the screen). */
export interface CalendarDayInfo {
  tone: DayTone | null;
  /** Estimated ovulation — the TTC variant marks it with a turquoise dot. */
  ovulation: boolean;
  /** Spoken label (date + state); the visible digit is aria-hidden. */
  label: string;
}

export interface MonthNavProps {
  onPrev: () => void;
  onNext: () => void;
  prevLabel: string;
  nextLabel: string;
}

interface MonthCardProps {
  year: number;
  month: number;
  locale: Locale;
  /** Weekday initials in the locale's column order. */
  weekdays: readonly string[];
  dayInfo: (date: Date) => CalendarDayInfo;
  formatDay: (day: number) => string;
  selected: Date;
  onSelect: (date: Date) => void;
  /** Previous/next arrows around the title — only the card that drives the view. */
  nav?: MonthNavProps;
  /** Show the ovulation dot (TTC variant). */
  showOvulation?: boolean;
  onTouchStart?: TouchEventHandler;
  onTouchMove?: TouchEventHandler;
  onTouchEnd?: TouchEventHandler;
}

const sameDay = (a: Date, b: Date) => diffInDays(a, b) === 0;

/**
 * One month of the Night & Bloom calendar: Lalezar title (+ arrows), weekday row
 * and 44px day cells whose tones join into rounded bands within a week.
 */
export function MonthCard({
  year,
  month,
  locale,
  weekdays,
  dayInfo,
  formatDay,
  selected,
  onSelect,
  nav,
  showOvulation = false,
  onTouchStart,
  onTouchMove,
  onTouchEnd,
}: MonthCardProps) {
  const rtl = useDirection() === 'rtl';
  const weeks = monthMatrix(year, month, locale);
  const titleId = `cal-m-${year}-${month}`;
  const now = today();

  return (
    <section className="nb-card cc-month" aria-labelledby={titleId}>
      <div className={clsx('cc-month-head', !nav && 'is-plain')}>
        {nav ? (
          <HeaderButton variant="soft" icon={rtl ? 'chevronRight' : 'chevronLeft'} label={nav.prevLabel} onClick={nav.onPrev} />
        ) : null}
        <h2 id={titleId} className="cc-month-title">
          {formatMonthLabel(year, month, locale)}
        </h2>
        {nav ? (
          <HeaderButton variant="soft" icon={rtl ? 'chevronLeft' : 'chevronRight'} label={nav.nextLabel} onClick={nav.onNext} />
        ) : null}
      </div>

      <div className="cc-grid" aria-hidden>
        {weekdays.map((w, i) => (
          <span key={i} className="cc-weekday">
            {w}
          </span>
        ))}
      </div>

      <div
        className="cc-weeks"
        role="group"
        aria-labelledby={titleId}
        onTouchStart={onTouchStart}
        onTouchMove={onTouchMove}
        onTouchEnd={onTouchEnd}
      >
        {weeks.map((week, wi) => {
          const infos = week.map((cell) => (cell ? dayInfo(cell.date) : null));
          const edges = bandEdges(infos.map((info) => info?.tone ?? null));
          return (
            <div key={wi} className="cc-grid cc-week">
              {week.map((cell, ci) => {
                const info = infos[ci];
                if (!cell || !info) return <span key={ci} className="cc-pad" />;
                const edge = edges[ci];
                const isSelected = sameDay(cell.date, selected);
                const isToday = sameDay(cell.date, now);
                return (
                  <button
                    key={ci}
                    type="button"
                    className={clsx(
                      'cc-day',
                      info.tone && `is-${info.tone}`,
                      edge?.start && 'is-band-start',
                      edge?.end && 'is-band-end',
                      isSelected && 'is-selected',
                      isToday && 'is-today',
                    )}
                    onClick={() => onSelect(cell.date)}
                    aria-label={info.label}
                    aria-pressed={isSelected}
                    aria-current={isToday ? 'date' : undefined}
                  >
                    <span aria-hidden className="cc-day-num">
                      {formatDay(cell.day)}
                    </span>
                    {showOvulation && info.ovulation ? <span aria-hidden className="cc-ov-dot" /> : null}
                  </button>
                );
              })}
            </div>
          );
        })}
      </div>
    </section>
  );
}
