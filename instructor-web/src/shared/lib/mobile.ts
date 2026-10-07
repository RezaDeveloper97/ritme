/**
 * Iranian mobile numbers as the API wants them: `09xxxxxxxxx` (POST
 * /auth/send-otp validates `^09[0-9]{9}$`). Accepts Persian/Arabic digits,
 * spaces, dashes and the +98 / 0098 / 98 prefixes people paste.
 */
const PERSIAN = '۰۱۲۳۴۵۶۷۸۹';
const ARABIC = '٠١٢٣٤٥٦٧٨٩';

export function toLatinDigits(value: string): string {
  return value.replace(/[۰-۹٠-٩]/g, (ch) => {
    const fa = PERSIAN.indexOf(ch);
    return String(fa >= 0 ? fa : ARABIC.indexOf(ch));
  });
}

/** Digits only, Latin. */
export function digitsOnly(value: string): string {
  return toLatinDigits(value).replace(/\D/g, '');
}

/** `09123456789`, or null when the input cannot be an Iranian mobile. */
export function normalizeMobile(value: string): string | null {
  let d = digitsOnly(value);
  if (d.startsWith('0098')) d = d.slice(4);
  else if (d.startsWith('98') && d.length === 12) d = d.slice(2);
  if (d.length === 10 && d.startsWith('9')) d = `0${d}`;
  return /^09\d{9}$/.test(d) ? d : null;
}

/** `0912 345 6789` for display (Latin digits; the UI font renders them). */
export function formatMobile(mobile: string): string {
  const d = digitsOnly(mobile);
  if (d.length !== 11) return d;
  return `${d.slice(0, 4)} ${d.slice(4, 7)} ${d.slice(7)}`;
}

/** Persian digits for numbers shown in running text. */
export function toPersianDigits(value: string | number): string {
  return String(value).replace(/\d/g, (n) => PERSIAN[Number(n)] ?? n);
}
