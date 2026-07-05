/**
 * Domain-agnostic helpers for Iranian mobile numbers and Persian digits.
 * Lives in `shared/lib` so any layer can normalize/validate a phone number the
 * same way — the OTP flow, forms, and display all go through here (CLAUDE.md §7
 * keeps digit formatting centralized rather than sprinkled across components).
 */

const FA_DIGITS = '۰۱۲۳۴۵۶۷۸۹';
const AR_DIGITS = '٠١٢٣٤٥٦٧٨٩';

/** Converts Persian/Arabic-Indic digits in a string to ASCII `0-9`. */
export function toAsciiDigits(value: string): string {
  return value
    .replace(/[۰-۹]/g, (d) => String(FA_DIGITS.indexOf(d)))
    .replace(/[٠-٩]/g, (d) => String(AR_DIGITS.indexOf(d)));
}

/** Renders ASCII digits in a string as Persian digits for display. */
export function toPersianDigits(value: string | number): string {
  return String(value).replace(/[0-9]/g, (d) => FA_DIGITS[Number(d)]);
}

/**
 * Normalizes user input into a bare Iranian mobile number: Persian/Arabic
 * digits become ASCII, everything non-numeric is stripped, and common prefixes
 * (`+98`, `0098`, `98`) are collapsed to the local `0…` form.
 */
export function normalizeMobile(input: string): string {
  let digits = toAsciiDigits(input).replace(/\D/g, '');
  if (digits.startsWith('0098')) digits = digits.slice(4);
  else if (digits.startsWith('98') && digits.length === 12) digits = digits.slice(2);
  if (digits.length === 10 && digits.startsWith('9')) digits = `0${digits}`;
  return digits.slice(0, 11);
}

/** True when `value` is a valid 11-digit Iranian mobile number (`09XXXXXXXXX`). */
export function isValidMobile(value: string): boolean {
  return /^09\d{9}$/.test(normalizeMobile(value));
}
