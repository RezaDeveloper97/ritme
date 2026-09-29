import type { Locale } from '@/shared/i18n';
import { formatDecimal } from '@/shared/lib/date';

/**
 * A week stat as the admin stores it (ASCII text: `"1.6"`, `"<1"`, `"150-170"`,
 * README «Stored shapes») → its parts in the locale's digits. The screen
 * wraps them in copy: a single value is an average («~۱٫۶»), a range reads
 * «۱۵۰ تا ۱۷۰», and `<n` reads «کمتر از ۱».
 */
export type WeekStat =
  | { kind: 'approx'; value: string }
  | { kind: 'range'; from: string; to: string }
  | { kind: 'less'; value: string }
  | { kind: 'text'; value: string };

export function parseWeekStat(raw: string, locale: Locale): WeekStat {
  const v = raw.trim();
  const range = /^(\d+(?:\.\d+)?)\s*[-–]\s*(\d+(?:\.\d+)?)$/.exec(v);
  if (range) return { kind: 'range', from: formatDecimal(range[1], locale), to: formatDecimal(range[2], locale) };
  const less = /^<\s*(\d+(?:\.\d+)?)$/.exec(v);
  if (less) return { kind: 'less', value: formatDecimal(less[1], locale) };
  if (/^~?\d+(?:\.\d+)?$/.test(v)) return { kind: 'approx', value: formatDecimal(v.replace(/^~/, ''), locale) };
  return { kind: 'text', value: formatDecimal(v, locale) };
}
