'use client';

import clsx from 'clsx';

import { useDirection, type Locale } from '@/shared/i18n';
import { diffInDays, formatYear, monthMatrix, monthName, today } from '@/shared/lib/date';
import { HeaderButton } from '@/shared/ui';

import type { DayTone } from '../model/tones';

interface YearViewProps {
  year: number;
  locale: Locale;
  weekdays: readonly string[];
  toneOf: (date: Date) => DayTone | null;
  formatDay: (day: number) => string;
  onPickMonth: (month: number) => void;
  onPrevYear: () => void;
  onNextYear: () => void;
  prevYearLabel: string;
  nextYearLabel: string;
  /** «باز کردن مهر» — spoken name of a month tile. */
  openMonthLabel: (monthName: string) => string;
}

const MONTHS = Array.from({ length: 12 }, (_, i) => i + 1);

/**
 * «سال» view: twelve mini months in a 2-column grid with the same tones as the
 * month view (no bands at this size). Tapping a month opens it in the month view.
 */
export function YearView({
  year,
  locale,
  weekdays,
  toneOf,
  formatDay,
  onPickMonth,
  onPrevYear,
  onNextYear,
  prevYearLabel,
  nextYearLabel,
  openMonthLabel,
}: YearViewProps) {
  const rtl = useDirection() === 'rtl';
  const now = today();
  return (
    <section className="nb-card cc-year" aria-labelledby="cc-year-title">
      <div className="cc-month-head">
        <HeaderButton variant="soft" icon={rtl ? 'chevronRight' : 'chevronLeft'} label={prevYearLabel} onClick={onPrevYear} />
        <h2 id="cc-year-title" className="cc-month-title">
          {formatYear(year, locale)}
        </h2>
        <HeaderButton variant="soft" icon={rtl ? 'chevronLeft' : 'chevronRight'} label={nextYearLabel} onClick={onNextYear} />
      </div>
      <div className="cc-year-grid">
        {MONTHS.map((m) => {
          const name = monthName(m, locale);
          return (
            <button
              key={m}
              type="button"
              className="cc-mini"
              onClick={() => onPickMonth(m)}
              aria-label={openMonthLabel(name)}
            >
              <span className="cc-mini-title" aria-hidden>
                {name}
              </span>
              <span className="cc-mini-grid" aria-hidden>
                {weekdays.map((w, i) => (
                  <span key={`w${i}`} className="cc-mini-wd">
                    {w}
                  </span>
                ))}
                {monthMatrix(year, m, locale)
                  .flat()
                  .map((cell, i) => {
                    if (!cell) return <span key={i} />;
                    const tone = toneOf(cell.date);
                    return (
                      <span
                        key={i}
                        className={clsx(
                          'cc-mini-day',
                          tone && `is-${tone}`,
                          diffInDays(cell.date, now) === 0 && 'is-today',
                        )}
                      >
                        {formatDay(cell.day)}
                      </span>
                    );
                  })}
              </span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
