'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useMemo, useRef } from 'react';

import type { JalaliParts } from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';
import { WheelPicker } from '@/shared/ui';

const FA = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];
const localizeNum = (value: string | number, loc: Locale) =>
  loc === 'fa' ? String(value).replace(/[0-9]/g, (d) => FA[Number(d)]) : String(value);

interface JalaliDateWheelsProps {
  /** Unique prefix for the underlying wheel ids (one instance per screen). */
  idPrefix: string;
  value: JalaliParts;
  onChange: (value: JalaliParts) => void;
  /** First selectable Jalali year (inclusive). */
  minYear: number;
  /** Last selectable Jalali year (inclusive). */
  maxYear: number;
}

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

/**
 * Day / month / year wheel triplet for picking a Jalali date — the same
 * interaction as the onboarding birthday step, packaged for the edit-profile
 * forms. Month names come from the `profileEdit` namespace and digits localize
 * per locale (CLAUDE.md §6); the wheels never show a Gregorian date (§7).
 */
export function JalaliDateWheels({ idPrefix, value, onChange, minYear, maxYear }: JalaliDateWheelsProps) {
  const t = useTranslations('profileEdit');
  const loc = useLocale() as Locale;

  // WheelPicker binds its scroll handler once on mount, so the callbacks below
  // are the first-render closures. Reading the latest parts from a ref (instead
  // of the captured `value`) keeps the three wheels from clobbering each other.
  const valueRef = useRef(value);
  useEffect(() => {
    valueRef.current = value;
  }, [value]);
  const emit = (patch: Partial<JalaliParts>) => {
    valueRef.current = { ...valueRef.current, ...patch };
    onChange(valueRef.current);
  };

  // Static keys so the typed-message check can verify every month exists (§6).
  const months = [
    t('months.m1'), t('months.m2'), t('months.m3'), t('months.m4'),
    t('months.m5'), t('months.m6'), t('months.m7'), t('months.m8'),
    t('months.m9'), t('months.m10'), t('months.m11'), t('months.m12'),
  ];

  const days = useMemo(
    () => Array.from({ length: 31 }, (_, i) => localizeNum(i + 1, loc)),
    [loc],
  );
  const years = useMemo(
    () => Array.from({ length: maxYear - minYear + 1 }, (_, i) => localizeNum(minYear + i, loc)),
    [loc, minYear, maxYear],
  );

  return (
    <div style={{ display: 'flex', gap: 8, justifyContent: 'center', position: 'relative' }}>
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
