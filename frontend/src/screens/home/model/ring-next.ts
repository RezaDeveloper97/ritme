import type { CycleDayMarker } from '@/entities/cycle';

/** What the ring centre counts toward when the selected day is not a period day. */
export interface RingMilestone {
  kind: Exclude<CycleDayMarker, 'period'>;
  /** Days from the selected day; 0 means the selected day itself. */
  days: number;
}

/**
 * The next coloured stretch on the ring after the selected day, so a quiet
 * (grey) day reads «N روز تا پنجرهٔ باروری» instead of always counting to the
 * next period. Returns null when the next thing is the period itself (or
 * nothing on the ring is ahead): the caller keeps its period countdown then.
 *
 * - grey day → the next marker ahead
 * - fertile day → the next different marker ahead (ovulation, then PMS)
 * - ovulation day → the day itself (`days: 0`)
 * - PMS / period day → null (period countdown / period day stay as they are)
 */
export function ringNextMilestone(
  markers: ReadonlyArray<CycleDayMarker | null>,
  index: number | null,
): RingMilestone | null {
  if (index == null || index < 0 || index >= markers.length) return null;
  const current = markers[index];
  if (current === 'period' || current === 'pms') return null;
  if (current === 'ovulation') return { kind: 'ovulation', days: 0 };
  for (let j = index + 1; j < markers.length; j++) {
    const m = markers[j];
    if (!m || m === current) continue;
    // The fertile window wraps the ovulation day: its tail after ovulation is not "next".
    if (m === 'fertile' && current === null && markers.slice(index + 1, j).includes('ovulation')) continue;
    return m === 'period' ? null : { kind: m, days: j - index };
  }
  return null;
}
