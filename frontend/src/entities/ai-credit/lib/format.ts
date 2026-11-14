import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

/** A credit count in the locale's digits («۸۸»). */
export function formatCredits(value: number, locale: Locale): string {
  return formatNumber(Math.max(0, Math.round(value)), locale);
}
