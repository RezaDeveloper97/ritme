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
