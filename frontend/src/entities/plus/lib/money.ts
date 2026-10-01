import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

/** Rials (what the API sends, `currency: IRR`) → toman (what people read): ÷10, rounded. */
export function rialsToToman(rials: number): number {
  return Math.round(rials / 10);
}

/**
 * A rial amount as a grouped toman number in the locale's digits, without the
 * unit: «۲۳۷٬۰۰۰» in fa (Persian digits + the Arabic thousands separator, the
 * one sanctioned Persian-specific branch, CLAUDE.md §6.1), "237,000" elsewhere.
 */
export function formatToman(rials: number, locale: Locale): string {
  const grouped = String(rialsToToman(Math.abs(rials))).replace(/\B(?=(\d{3})+(?!\d))/g, ',');
  const text = locale === 'fa' ? grouped.replace(/,/g, '٬') : grouped;
  return formatNumber(text, locale);
}

/**
 * A rial amount as thousands of toman with at most one decimal, in the locale's
 * digits and without the unit — the trial sheet's «۲۳۷» / «۱۱۸٫۵» (the message
 * adds «هزار تومان»). fa uses the Arabic decimal separator «٫» (CLAUDE.md §6.1).
 */
export function formatTomanThousands(rials: number, locale: Locale): string {
  const thousands = Math.round(rialsToToman(Math.abs(rials)) / 100) / 10;
  const [int, frac] = thousands.toFixed(1).split('.');
  const grouped = int.replace(/\B(?=(\d{3})+(?!\d))/g, locale === 'fa' ? '٬' : ',');
  const text = frac === '0' ? grouped : `${grouped}${locale === 'fa' ? '٫' : '.'}${frac}`;
  return formatNumber(text, locale);
}
