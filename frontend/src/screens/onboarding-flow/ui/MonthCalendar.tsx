'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';
import { clsx } from 'clsx';

import { type Locale, useDirection } from '@/shared/i18n';
import {
  diffInDays,
  formatMonthLabel,
  formatNumber,
  monthMatrix,
  shiftMonth,
  toParts,
  weekdayLabels,
} from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

interface MonthCalendarProps {
  value: Date | null;
  onChange: (date: Date) => void;
  /** Selectable range (inclusive). */
  min: Date;
  max: Date;
  /** Paint the selected day + the next `span − 1` days as one period (nbl_Onb_Cycle). */
  span?: number;
  /** Accessible name of the grid. */
  label: string;
  disabled?: boolean;
}

/**
 * The onboarding month grid: round day cells, the picked day solid and (for
 * a period) the following days tinted. Jalali for fa, Gregorian otherwise
 * (CLAUDE.md §7) — every date goes through `shared/lib/date`.
 */
export function MonthCalendar({ value, onChange, min, max, span = 1, label, disabled }: MonthCalendarProps) {
  const locale = useLocale() as Locale;
  const t = useTranslations('onboarding.flow.calendar');
  const rtl = useDirection() === 'rtl';
  const start = toParts(value ?? max, locale);
  const [view, setView] = useState({ year: start.year, month: start.month });
  const weeks = useMemo(() => monthMatrix(view.year, view.month, locale), [view, locale]);
  const minParts = toParts(min, locale);
  const maxParts = toParts(max, locale);
  const key = (y: number, m: number) => y * 12 + m;
  const canPrev = key(view.year, view.month) > key(minParts.year, minParts.month);
  const canNext = key(view.year, view.month) < key(maxParts.year, maxParts.month);

  return (
    <div className={clsx('nb-card onb2-cal', disabled && 'is-disabled')}>
      <div className="onb2-cal-nav">
        <button
          type="button"
          className="onb2-cal-btn"
          disabled={!canPrev || disabled}
          aria-label={t('prev')}
          onClick={() => setView(shiftMonth(view.year, view.month, -1))}
        >
          <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={20} />
        </button>
        <span className="onb2-cal-month" aria-live="polite">
          {formatMonthLabel(view.year, view.month, locale)}
        </span>
        <button
          type="button"
          className="onb2-cal-btn"
          disabled={!canNext || disabled}
          aria-label={t('next')}
          onClick={() => setView(shiftMonth(view.year, view.month, 1))}
        >
          <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={20} />
        </button>
      </div>
      <div className="onb2-cal-grid" role="grid" aria-label={label}>
        <div className="onb2-cal-row" role="row">
          {weekdayLabels(locale).map((w) => (
            <span key={w} className="onb2-cal-wd" role="columnheader">
              {w}
            </span>
          ))}
        </div>
        {weeks.map((week, wi) => (
          <div className="onb2-cal-row" role="row" key={wi}>
            {week.map((cell, ci) => {
              if (!cell) return <span key={ci} className="onb2-cal-pad" role="gridcell" />;
              const out = diffInDays(cell.date, min) < 0 || diffInDays(cell.date, max) > 0;
              const offset = value ? diffInDays(cell.date, value) : -1;
              const picked = offset === 0;
              const inSpan = offset > 0 && offset < span;
              return (
                <span key={ci} role="gridcell" className="onb2-cal-cell">
                  <button
                    type="button"
                    className={clsx('onb2-cal-day', picked && 'is-picked', inSpan && 'is-span')}
                    aria-pressed={picked}
                    disabled={out || disabled}
                    onClick={() => onChange(cell.date)}
                  >
                    {formatNumber(cell.day, locale)}
                  </button>
                </span>
              );
            })}
          </div>
        ))}
      </div>
    </div>
  );
}

