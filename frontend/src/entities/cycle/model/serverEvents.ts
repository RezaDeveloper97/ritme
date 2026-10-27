import type { CycleFutureCycle, CycleMonthDay } from './types';
import { diffInDays, fromApiDate } from '@/shared/lib/date';

/** An inclusive run of Gregorian `YYYY-MM-DD` dates the server marked. */
export interface ServerRange {
  start: string;
  end: string;
}

export interface ServerFertileEvents {
  /** The next ovulation day that is not behind today, as the server reported it. */
  ovulation: string | null;
  /** The fertile window running today or the next one, as the server reported it. */
  fertile: ServerRange | null;
}

/**
 * The upcoming ovulation and fertile window on a v6 payload (T-M9-42c). The server already
 * decided every day (`main_phase = fertile`, `estimated_ovulation_day`) and every future cycle
 * (`future_cycles`: ovulation, fertile start/end) — this only picks the first one that is not
 * over yet. No cycle length, luteal offset or roll-forward happens here. ISO dates compare as
 * strings. Fertile events are empty under clinical suppression.
 */
export function serverFertileEvents(
  anchorOvulation: string | null | undefined,
  days: Iterable<CycleMonthDay>,
  future: readonly CycleFutureCycle[],
  todayIso: string,
  suppressed: boolean,
): ServerFertileEvents {
  if (suppressed) return { ovulation: null, fertile: null };

  const ovulations: string[] = [];
  if (anchorOvulation) ovulations.push(anchorOvulation);

  const fertileDays: string[] = [];
  for (const d of days) {
    if (d.mainPhase !== 'fertile') continue;
    fertileDays.push(d.date);
    if (d.cycleDay != null && d.cycleDay === d.estimatedOvulationDay) ovulations.push(d.date);
  }
  fertileDays.sort();

  // Group consecutive fertile days into the windows the server painted.
  const windows: ServerRange[] = [];
  for (const iso of fertileDays) {
    const last = windows[windows.length - 1];
    if (last && diffInDays(fromApiDate(iso), fromApiDate(last.end)) === 1) last.end = iso;
    else if (!last || last.end !== iso) windows.push({ start: iso, end: iso });
  }

  for (const c of future) {
    if (c.ovulation) ovulations.push(c.ovulation);
    if (c.fertileStart && c.fertileEnd) windows.push({ start: c.fertileStart, end: c.fertileEnd });
  }

  const ovulation = ovulations.filter((iso) => iso >= todayIso).sort()[0] ?? null;
  const fertile = windows.filter((w) => w.end >= todayIso).sort((a, b) => a.start.localeCompare(b.start))[0] ?? null;
  return { ovulation, fertile };
}
