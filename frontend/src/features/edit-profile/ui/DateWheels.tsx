'use client';

import { useLocale } from 'next-intl';
import { useEffect, useMemo, useRef } from 'react';

import { allMonthNames, type DateParts, formatNumber } from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';
import { WheelPicker } from '@/shared/ui';

interface DateWheelsProps {
  /** Unique prefix for the underlying wheel ids (one instance per screen). */
  idPrefix: string;
  value: DateParts;
  onChange: (value: DateParts) => void;
  /** First selectable year (inclusive), in the locale's calendar. */
  minYear: number;
  /** Last selectable year (inclusive), in the locale's calendar. */
  maxYear: number;
}

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

/**
 * Day / month / year wheel triplet for picking a date — the same interaction as
 * the onboarding birthday step, packaged for the edit-profile forms.
 *
 * The wheels speak the calendar the user's locale reads: Jalali months and
 * years for `fa`, Gregorian for `en`. Month names and digits both come from the
 * date layer (CLAUDE.md §6, §7) so the two calendars can never be mixed.
 */
export function DateWheels({ idPrefix, value, onChange, minYear, maxYear }: DateWheelsProps) {
  const loc = useLocale() as Locale;

  // WheelPicker binds its scroll handler once on mount, so the callbacks below
  // are the first-render closures. Reading the latest parts from a ref (instead
  // of the captured `value`) keeps the three wheels from clobbering each other.
  const valueRef = useRef(value);
  useEffect(() => {
    valueRef.current = value;
  }, [value]);
  const emit = (patch: Partial<DateParts>) => {
    valueRef.current = { ...valueRef.current, ...patch };
    onChange(valueRef.current);
  };

  const months = useMemo(() => [...allMonthNames(loc)], [loc]);
  const days = useMemo(
    () => Array.from({ length: 31 }, (_, i) => formatNumber(i + 1, loc)),
    [loc],
  );
  const years = useMemo(
    () => Array.from({ length: maxYear - minYear + 1 }, (_, i) => formatNumber(minYear + i, loc)),
    [loc, minYear, maxYear],
  );

  return (
    <div className="jdw-row">
      <div className="wheel-band" />
      <WheelPicker
        id={`${idPrefix}-day`}
        items={days}
        selectedIndex={clamp(value.day - 1, 0, 30)}
        width={56}
        onChange={(i) => emit({ day: i + 1 })}
      />
      <WheelPicker
        id={`${idPrefix}-month`}
        items={months}
        selectedIndex={clamp(value.month - 1, 0, 11)}
        width={112}
        onChange={(i) => emit({ month: i + 1 })}
      />
      <WheelPicker
        id={`${idPrefix}-year`}
        items={years}
        selectedIndex={clamp(value.year - minYear, 0, maxYear - minYear)}
        width={80}
        onChange={(i) => emit({ year: minYear + i })}
      />
    </div>
  );
}
