import { toAsciiDigits } from '@/shared/lib/phone';

import type { ChildSex, VaccineVisit } from './types';

/** How far a vaccine visit is, for «۱۸ روز دیگر» / «۲ ماه دیگر» / «امروز» / overdue copy. */
export type DueIn =
  | { kind: 'overdue' }
  | { kind: 'today' }
  | { kind: 'days'; n: number }
  | { kind: 'months'; n: number };

/** Up to 59 days in days, beyond that in whole months (v15_Children «چکاپ دندان · ۲ ماه دیگر»). */
export function dueIn(daysLeft: number): DueIn {
  if (daysLeft < 0) return { kind: 'overdue' };
  if (daysLeft === 0) return { kind: 'today' };
  if (daysLeft < 60) return { kind: 'days', n: daysLeft };
  return { kind: 'months', n: Math.round(daysLeft / 30) };
}

/** True when the visit needs attention now (overdue or due within the reminder window). */
export function isVisitUrgent(visit: Pick<VaccineVisit, 'status' | 'daysLeft'>): boolean {
  return visit.status === 'overdue' || visit.status === 'due' || visit.daysLeft < 0;
}

/** Percentile as a whole number for a chip (`47.2` → 47); null stays null. */
export function roundPercentile(p: number | null | undefined): number | null {
  if (p == null || !Number.isFinite(p)) return null;
  return Math.min(100, Math.max(0, Math.round(p)));
}

/** Avatar tone per sex (artboard: pink girl, teal boy); unknown → brand. */
export function childTone(sex: ChildSex | null): 'bloom' | 'data' | 'brand' {
  if (sex === 'girl') return 'bloom';
  if (sex === 'boy') return 'data';
  return 'brand';
}

/**
 * Typed decimal text → its canonical form: any digit script, `٫` `,` `/` read as
 * the decimal point, at most one point and `maxDecimals` decimals. Returns the
 * cleaned text (what the field keeps) and the number (`undefined` when empty).
 */
export function parseDecimalInput(text: string, maxDecimals = 2): { text: string; value: number | undefined } {
  const ascii = toAsciiDigits(text).replace(/[٫,/]/g, '.');
  let out = '';
  let seenPoint = false;
  let decimals = 0;
  for (const ch of ascii) {
    if (ch >= '0' && ch <= '9') {
      if (seenPoint) {
        if (decimals >= maxDecimals) continue;
        decimals += 1;
      }
      if (!seenPoint && out.replace('.', '').length >= 3) continue;
      out += ch;
    } else if (ch === '.' && !seenPoint && maxDecimals > 0) {
      seenPoint = true;
      out += out === '' ? '0.' : '.';
    }
  }
  const value = out === '' || out === '0.' ? undefined : Number(out);
  return { text: out, value: value !== undefined && Number.isFinite(value) ? value : undefined };
}
