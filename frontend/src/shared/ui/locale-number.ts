import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { toAsciiDigits } from '@/shared/lib/phone';

/*
 * Pure helpers of `LocaleNumberField`: a whole number shown in the locale's
 * digits (۸ in fa, 8 in en) and read back from whatever digits the keyboard
 * typed (Persian, Arabic-Indic or ASCII).
 */

/** The field's text for a value: '' when unset, else the locale's digits. */
export function formatLocaleInteger(value: number | undefined, locale: Locale): string {
  return value === undefined || !Number.isFinite(value) ? '' : formatNumber(Math.trunc(value), locale);
}

/**
 * Typed text → a whole number: any digit script, non-digits dropped, at most
 * `maxDigits` digits kept; `undefined` when nothing numeric is left.
 */
export function parseLocaleInteger(text: string, maxDigits = 6): number | undefined {
  const digits = toAsciiDigits(text).replace(/\D/g, '').slice(0, maxDigits);
  return digits === '' ? undefined : Number(digits);
}
