'use client';

import type { Locale } from '@/shared/i18n';

import { formatLocaleInteger, parseLocaleInteger } from './locale-number';

/**
 * A labelled whole-number input that shows the locale's digits (۸ in fa) and
 * accepts any digit script. A text input with a numeric keyboard, because
 * `type="number"` always renders ASCII digits. `max` only limits how many
 * digits can be typed; clamping to `min`..`max` stays with the caller.
 */
export function LocaleNumberField({
  label,
  locale,
  value,
  onChange,
  max,
  placeholder,
}: {
  label: string;
  locale: Locale;
  value: number | undefined;
  onChange: (value: number | undefined) => void;
  /** The caller clamps; `max` limits the digit count. */
  max?: number;
  placeholder?: string;
}) {
  const maxDigits = max !== undefined ? String(Math.trunc(Math.abs(max))).length : 6;
  return (
    <label className="fld-label">
      <span className="fld-label-t">{label}</span>
      <input
        className="field fld-input"
        type="text"
        inputMode="numeric"
        autoComplete="off"
        value={formatLocaleInteger(value, locale)}
        placeholder={placeholder}
        maxLength={maxDigits}
        onChange={(e) => onChange(parseLocaleInteger(e.target.value, maxDigits))}
      />
    </label>
  );
}
