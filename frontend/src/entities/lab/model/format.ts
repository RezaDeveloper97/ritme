import type { Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';

import type { LabMarker, MarkerReference } from './types';

/** «۱۱٫۲» / "11.2" — the number in the locale's digits, else the printed text value («منفی»), else «—». */
export function formatLabValue(marker: Pick<LabMarker, 'value' | 'valueText'>, locale: Locale): string {
  if (marker.value !== null) return formatDecimal(trimNumber(marker.value), locale);
  if (marker.valueText) return marker.valueText;
  return '—';
}

/** Up to 3 decimals, no trailing zeros (11.2, 0.4, 310). */
export function trimNumber(n: number): string {
  return String(Math.round(n * 1000) / 1000);
}

/** The range as printed («۱۲–۱۶»), or rebuilt from its bounds; null when there is none. */
export function formatReference(ref: MarkerReference, locale: Locale): string | null {
  if (ref.text) return formatDecimal(ref.text, locale);
  if (ref.low !== null && ref.high !== null) return formatDecimal(`${trimNumber(ref.low)}–${trimNumber(ref.high)}`, locale);
  if (ref.high !== null) return formatDecimal(`< ${trimNumber(ref.high)}`, locale);
  if (ref.low !== null) return formatDecimal(`> ${trimNumber(ref.low)}`, locale);
  return null;
}
