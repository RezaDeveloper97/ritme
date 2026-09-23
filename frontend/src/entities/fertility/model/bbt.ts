import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

/*
 * Basal body temperature helpers (°C). Values are handled in hundredths
 * internally so ±0.01 steps never drift through float error (36.1 + 0.01 …).
 *
 * The API range is 35.00–38.50 (422 outside it, docs/fertility-ttc/README.md).
 */

export const BBT_MIN = 35;
export const BBT_MAX = 38.5;
/** One tap of − / + on the log screen. */
export const BBT_STEP = 0.01;
/** Where the input starts when nothing is logged yet (a typical follicular reading). */
export const BBT_DEFAULT = 36.5;

/** Persian decimal separator «٫» (U+066B). */
export const FA_DECIMAL_SEPARATOR = '٫';

/** `toFixed(6)` first so 36.005 × 100 (= 3600.4999…) still rounds half-up. */
const toHundredths = (value: number): number => Math.round(Number((value * 100).toFixed(6)));
const fromHundredths = (value: number): number => value / 100;

/** Round to 2 decimals (the column is decimal(4,2)). */
export function roundBbt(value: number): number {
  return fromHundredths(toHundredths(value));
}

export function isBbtInRange(value: number): boolean {
  const h = toHundredths(value);
  return Number.isFinite(value) && h >= toHundredths(BBT_MIN) && h <= toHundredths(BBT_MAX);
}

export function clampBbt(value: number): number {
  return fromHundredths(
    Math.min(toHundredths(BBT_MAX), Math.max(toHundredths(BBT_MIN), toHundredths(value))),
  );
}

/**
 * One − / + tap: moves by {@link BBT_STEP} × `steps` and clamps to the valid
 * range. From an empty field the first tap starts at {@link BBT_DEFAULT}.
 */
export function stepBbt(value: number | null, steps: number): number {
  if (value === null || !Number.isFinite(value)) return BBT_DEFAULT;
  return clampBbt(fromHundredths(toHundredths(value) + Math.round(steps)));
}

/**
 * Format a temperature for display: fixed decimals (2 by default, 1 for chart
 * axis ticks), Persian digits and «٫» in `fa`, `.` elsewhere.
 * `null` → empty string (the caller shows «ثبت نشده»).
 */
export function formatBbt(value: number | null, locale: Locale, fractionDigits = 2): string {
  if (value === null || !Number.isFinite(value)) return '';
  const fixed = value.toFixed(fractionDigits);
  if (locale !== 'fa') return fixed;
  return formatNumber(fixed, locale).replace('.', FA_DECIMAL_SEPARATOR);
}

/** Persian (U+06F0…) and Arabic-Indic (U+0660…) digits → ASCII. */
function toAsciiDigits(value: string): string {
  return value.replace(/[۰-۹٠-٩]/g, (d) => {
    const code = d.charCodeAt(0);
    return String(code >= 0x06f0 ? code - 0x06f0 : code - 0x0660);
  });
}

/**
 * Parse what the user typed into the BBT field. Accepts Persian / Arabic /
 * Latin digits and `.`, `,`, `/` or «٫» as the decimal separator;
 * rounds to 2 decimals. Returns `null` for an empty or non-numeric input.
 * The range is NOT enforced here — pair with {@link isBbtInRange} so the form
 * can show why a value is rejected.
 */
export function parseBbt(input: string): number | null {
  const normalized = toAsciiDigits(input.trim())
    .replace(/[٫,/]/g, '.')
    .replace(/\s|°|c$/gi, '');
  if (!/^\d{1,2}(\.\d*)?$|^\.\d+$/.test(normalized)) return null;
  const value = Number(normalized);
  return Number.isFinite(value) ? roundBbt(value) : null;
}
