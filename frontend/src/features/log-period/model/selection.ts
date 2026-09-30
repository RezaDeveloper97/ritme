import { addDays, diffInDays, fromApiDate, toApiDate } from '@/shared/lib/date';

import type { LoggedPeriod, PeriodSegment } from '../api/mutations';

/**
 * The edit-period grid's state: `selected` days are bleeding days the user
 * logged or tapped (solid); `suggested` days were filled in from the usual
 * period length (dashed). Both are saved; the split only changes how they look
 * and how a tap treats them. ISO `YYYY-MM-DD` strings throughout.
 */
export interface PeriodSelection {
  selected: Set<string>;
  suggested: Set<string>;
}

const next = (iso: string, n = 1) => toApiDate(addDays(fromApiDate(iso), n));

/** Every ISO day in the inclusive [start, end] range. */
export function isoRange(start: string, end: string): string[] {
  const out: string[] = [];
  const first = fromApiDate(start);
  const count = diffInDays(fromApiDate(end), first) + 1;
  for (let i = 0; i < count; i += 1) out.push(toApiDate(addDays(first, i)));
  return out;
}

/**
 * Opening state from the logged history. A closed period is solid; an open one
 * (only its start is stored) is solid up to today and suggested for the rest of
 * the usual length, which is exactly the "dashed days" of the design.
 */
export function seedSelection(history: readonly LoggedPeriod[], periodDuration: number, todayIso: string): PeriodSelection {
  const selected = new Set<string>();
  const suggested = new Set<string>();
  for (const p of history) {
    if (p.period_end_date) {
      for (const d of isoRange(p.period_start_date, p.period_end_date)) selected.add(d);
      continue;
    }
    for (const d of isoRange(p.period_start_date, next(p.period_start_date, Math.max(0, periodDuration - 1)))) {
      if (d <= todayIso) selected.add(d);
      else suggested.add(d);
    }
  }
  return { selected, suggested };
}

/**
 * One tap on `iso`:
 * - a suggested day trims the run from there (the period ended earlier);
 * - a selected day is cleared;
 * - an empty day is selected, and when it starts a new run (no neighbour on
 *   either side) the rest of the usual length is suggested after it, stopping
 *   at any day already in another run.
 */
export function toggleDay(prev: PeriodSelection, iso: string, periodDuration: number): PeriodSelection {
  const selected = new Set(prev.selected);
  const suggested = new Set(prev.suggested);
  const has = (d: string) => selected.has(d) || suggested.has(d);

  if (suggested.has(iso)) {
    for (let d = iso; suggested.has(d); d = next(d)) suggested.delete(d);
    return { selected, suggested };
  }
  if (selected.has(iso)) {
    selected.delete(iso);
    return { selected, suggested };
  }
  const startsRun = !has(next(iso, -1)) && !has(next(iso));
  selected.add(iso);
  if (startsRun) {
    for (let i = 1; i < periodDuration; i += 1) {
      const d = next(iso, i);
      if (has(d)) break;
      suggested.add(d);
    }
  }
  return { selected, suggested };
}

/** Contiguous runs of the given days, chronological. */
export function toSegments(days: Iterable<string>): PeriodSegment[] {
  const segments: PeriodSegment[] = [];
  for (const iso of [...days].sort()) {
    const last = segments[segments.length - 1];
    if (last && next(last.end) === iso) last.end = iso;
    else segments.push({ start: iso, end: iso });
  }
  return segments;
}

/** First day of every run (for the check badge). */
export function runStarts(days: Iterable<string>): Set<string> {
  return new Set(toSegments(days).map((s) => s.start));
}

/**
 * The run the start / end tiles describe: the one containing today, else the
 * latest run that has started, else the earliest upcoming one. Null when empty.
 */
export function focusRun(days: Iterable<string>, todayIso: string): PeriodSegment | null {
  const segments = toSegments(days);
  const current = segments.find((s) => s.start <= todayIso && s.end >= todayIso);
  if (current) return current;
  const started = segments.filter((s) => s.start <= todayIso);
  return started[started.length - 1] ?? segments[0] ?? null;
}
