import type { Locale } from '@/shared/i18n';
import type { Tone } from '@/shared/ui';
import type { NavMode } from '@/widgets/bottom-nav';

const FA_DIGITS = ['۰', '۱', '۲', '۳', '۴', '۵', '۶', '۷', '۸', '۹'];

/** Western digits → the locale's digits (Persian in fa). */
export function localizeDigits(value: string | number, locale: Locale): string {
  return locale === 'fa' ? String(value).replace(/[0-9]/g, (d) => FA_DIGITS[Number(d)]) : String(value);
}

/**
 * The hub shows the number masked (`nbl_Me_Hub`: «۰۹۱۲ ••• ••۴۵») — first four
 * and last two digits, so a glance over the shoulder reveals little.
 */
export function maskMobile(mobile: string, locale: Locale): string {
  const digits = mobile.replace(/\D/g, '');
  if (digits.length < 7) return localizeDigits(digits, locale);
  return localizeDigits(`${digits.slice(0, 4)} ••• ••${digits.slice(-2)}`, locale);
}

/** «0912 345 6789» grouping for the account screen (4-3-4, as drawn). */
export function groupMobile(mobile: string, locale: Locale): string {
  const digits = mobile.replace(/\D/g, '');
  if (digits.length !== 11) return localizeDigits(digits, locale);
  return localizeDigits(`${digits.slice(0, 4)} ${digits.slice(4, 7)} ${digits.slice(7)}`, locale);
}

/** Avatar initial: the first letter of the name (grapheme-safe enough for fa/en). */
export function firstLetter(name: string): string {
  return Array.from(name.trim())[0] ?? '';
}

/** Mode accent, as on the artboards (cycle = period rose). */
export const MODE_TONE: Record<NavMode, Tone> = {
  cycle: 'period',
  ttc: 'warm',
  pregnancy: 'bloom',
  postpartum: 'bloom',
  menopause: 'data',
  teen: 'brand',
  companion: 'brand',
};
