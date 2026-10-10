import { type PbacDay, type PbacEntry, pbacScore } from '@/entities/condition';

/** The editable part of a saved day. */
export function entryFrom(day: PbacDay): PbacEntry {
  return { light: day.light, medium: day.medium, heavy: day.heavy, clots: day.clots, flooding: day.flooding };
}

export function isDirty(saved: PbacEntry, entry: PbacEntry): boolean {
  return (
    saved.light !== entry.light ||
    saved.medium !== entry.medium ||
    saved.heavy !== entry.heavy ||
    saved.clots !== entry.clots ||
    saved.flooding !== entry.flooding
  );
}

/** Today's and the period's score while she edits: the saved period total with today's saved score swapped out. */
export function liveScores(day: PbacDay, entry: PbacEntry): { today: number; period: number } {
  const today = pbacScore(entry);
  return { today, period: Math.max(today, (day.period?.score ?? 0) - day.score + today) };
}

/**
 * The PUT body. `clots` goes only when it changed: the API takes the clot chip only for a life mode that logs clots,
 * and a `null` would clear one from the day log.
 */
export function payloadFor(saved: PbacEntry, entry: PbacEntry): Partial<PbacEntry> {
  const { clots, ...rest } = entry;
  return clots === saved.clots ? rest : { ...rest, clots };
}
