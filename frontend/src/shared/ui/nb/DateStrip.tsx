'use client';

import { clsx } from 'clsx';

import type { Locale } from '@/shared/i18n';
import {
  formatNumber,
  formatWeekdayDayMonth,
  toApiDate,
  toParts,
  weekdayLabels,
  weekOf,
} from '@/shared/lib/date';

import { toneClass, type Tone } from './tone';

interface DateStripProps {
  locale: Locale;
  selected: Date;
  onSelect: (date: Date) => void;
  /** Which week to show; defaults to the selected day's week. */
  anchor?: Date;
  /** Highlighted as «today» (outline) when it is not the selected day. */
  today?: Date;
  /** Dot under the day in a tone (`period` for bleeding, `brand` for a log…); `null` = none. */
  marker?: (date: Date) => Tone | null;
  /** Appended to a marked day's accessible name, e.g. «ثبت شده». */
  markedLabel?: string;
  /** Days after this one cannot be picked (no logging the future). */
  maxDate?: Date;
  /** Accessible name of the strip, e.g. «روزهای این هفته». */
  label: string;
  className?: string;
}

/**
 * One locale week (Saturday → Friday in Jalali) as 7 day cells, 42×54 radius 16:
 * weekday 10.5/700 over the day number. Selected = `--brand` outline + tint
 * (nbl_Log_Sheet_Cycle); today, when not selected, gets a brand weekday.
 * Calendar maths comes from `shared/lib/date` only (CLAUDE.md §7).
 */
export function DateStrip({
  locale,
  selected,
  onSelect,
  anchor,
  today,
  marker,
  markedLabel,
  maxDate,
  label,
  className,
}: DateStripProps) {
  const days = weekOf(anchor ?? selected, locale);
  const labels = weekdayLabels(locale);
  const selectedKey = toApiDate(selected);
  const todayKey = today ? toApiDate(today) : null;
  const maxKey = maxDate ? toApiDate(maxDate) : null;
  return (
    <div role="group" aria-label={label} className={clsx('nb-dates', className)}>
      {days.map((day, index) => {
        const key = toApiDate(day);
        const isSelected = key === selectedKey;
        const future = maxKey !== null && key > maxKey;
        const tone = marker?.(day) ?? null;
        const marked = tone !== null;
        const name = formatWeekdayDayMonth(day, locale);
        return (
          <button
            key={key}
            type="button"
            className={clsx('nb-date', isSelected && 'is-selected', key === todayKey && 'is-today')}
            aria-pressed={isSelected}
            aria-current={key === todayKey ? 'date' : undefined}
            aria-label={marked && markedLabel ? `${name} · ${markedLabel}` : name}
            disabled={future}
            onClick={() => onSelect(day)}
          >
            <span className="nb-date-wd" aria-hidden>
              {labels[index]}
            </span>
            <span className="nb-date-num" aria-hidden>
              {formatNumber(toParts(day, locale).day, locale)}
            </span>
            {tone ? <span className={clsx('nb-date-dot', toneClass(tone))} aria-hidden /> : null}
          </button>
        );
      })}
    </div>
  );
}
