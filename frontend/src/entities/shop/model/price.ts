import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';

type Unit = 'plain' | 'thousand' | 'million';

/** Integer toman → compact display parts: 485000 → thousand «۴۸۵», 9800000 → million «۹٫۸» (board style). */
export function formatToman(price: number, locale: Locale): { unit: Unit; n: string } {
  const dec = locale === 'fa' ? '٫' : '.';
  const trim = (v: number) => formatNumber(String(Math.round(v * 10) / 10), locale).replace('.', dec);
  if (price >= 1_000_000) return { unit: 'million', n: trim(price / 1_000_000) };
  if (price >= 1_000) return { unit: 'thousand', n: trim(price / 1_000) };
  return { unit: 'plain', n: formatNumber(price, locale) };
}
