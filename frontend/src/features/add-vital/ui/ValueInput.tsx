'use client';

import { clsx } from 'clsx';
import { useId, useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';
import { Icon, type Tone } from '@/shared/ui';

import { parseLocaleNumber } from '../model/validate';

function show(value: number | null, decimals: number, locale: Locale): string {
  return value === null ? '' : formatDecimal(value.toFixed(decimals), locale);
}

/**
 * The big number of an add form: + / − (44px) around a typed field that shows
 * the locale's digits (۱۱۸) and accepts any digit script. `layout="tile"` is
 * the AddBP column (label · + · number · − · unit); `row` is the AddGlucose /
 * AddHR line (− number +). The caller clamps nothing — validation reports.
 */
export function ValueInput({
  label,
  value,
  onChange,
  step,
  decimals = 0,
  min,
  max,
  unit,
  tone = 'brand',
  layout = 'tile',
  locale,
  increaseLabel,
  decreaseLabel,
  placeholder = '—',
  invalid,
  errorId,
  hideLabel,
  startAt,
}: {
  label: string;
  value: number | null;
  onChange: (value: number | null) => void;
  step: number;
  decimals?: number;
  min: number;
  max: number;
  unit?: string;
  tone?: Tone;
  layout?: 'tile' | 'row';
  locale: Locale;
  increaseLabel: string;
  decreaseLabel: string;
  placeholder?: string;
  invalid?: boolean;
  errorId?: string;
  hideLabel?: boolean;
  /** First value of an empty field on − / + (an optional pulse starts at 70). */
  startAt?: number;
}) {
  const id = useId();
  // The text being typed; re-derived from `value` whenever it no longer parses to it (−/+, unit switch).
  const [text, setText] = useState(() => show(value, decimals, locale));
  const parsed = parseLocaleNumber(text, decimals);
  const shown = parsed === value ? text : show(value, decimals, locale);
  const round = (n: number) => Math.round(n * 10 ** decimals) / 10 ** decimals;
  const bump = (dir: 1 | -1) => {
    const next = round(Math.min(max, Math.max(min, value === null ? (startAt ?? min) : value + dir * step)));
    setText(show(next, decimals, locale));
    onChange(next);
  };
  const plus = (
    <button type="button" className="vt-step" aria-label={increaseLabel} disabled={value !== null && value >= max} onClick={() => bump(1)}>
      <Icon name="plus" size={18} strokeWidth={2.4} />
    </button>
  );
  const minus = (
    <button type="button" className="vt-step" aria-label={decreaseLabel} disabled={value !== null && value <= min} onClick={() => bump(-1)}>
      <Icon name="minus" size={18} strokeWidth={2.4} />
    </button>
  );
  const field = (
    <input
      id={id}
      className="vt-num"
      type="text"
      inputMode={decimals ? 'decimal' : 'numeric'}
      autoComplete="off"
      dir="ltr"
      value={shown}
      placeholder={placeholder}
      aria-invalid={invalid || undefined}
      aria-describedby={errorId}
      maxLength={decimals ? 5 : 3}
      onChange={(e) => {
        setText(e.target.value);
        onChange(parseLocaleNumber(e.target.value, decimals));
      }}
      // Typed Latin digits («185») settle into the locale's digits («۱۸۵») once she leaves the field.
      onBlur={() => {
        if (value !== null) setText(show(value, decimals, locale));
      }}
    />
  );
  return (
    <div className={clsx('vt-value', `is-${layout}`, `nb-tone-${tone}`, invalid && 'is-invalid')}>
      <label htmlFor={id} className={clsx('vt-value-label', hideLabel && 'sr-only')}>
        {label}
      </label>
      {layout === 'tile' ? (
        <>
          {plus}
          {field}
          {minus}
        </>
      ) : (
        <div className="vt-value-line">
          {minus}
          {field}
          {plus}
        </div>
      )}
      {unit ? (
        <span className="vt-value-unit" dir="ltr">
          {unit}
        </span>
      ) : null}
    </div>
  );
}
