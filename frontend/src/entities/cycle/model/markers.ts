import type { CycleDayMarker, FertileLevel } from './types';

/**
 * The one palette for cycle day markers, shared by every calendar surface (the
 * full calendar screen, the home ring and week strip) so a day never reads as
 * one state in one place and another elsewhere. Follicular/luteal days carry no
 * marker and read as neutral. `color` is the solid fill of a marked day; `bg` is
 * the soft tint used behind a marker icon (day detail, legends).
 */
export const cycleMarkerStyle: Record<CycleDayMarker, { bg: string; color: string }> = {
  // Period stays red — bleeding days read as red to users, and that convention
  // outranks the brand palette. It is the one sanctioned red (§10.2).
  period: { bg: 'var(--period-soft)', color: 'var(--period)' },
  // The fertile window and ovulation are one turquoise-green ramp, graded by chance
  // ({@link fertileLevel}); ovulation is its darkest step.
  fertile: { bg: 'var(--fertile-soft)', color: 'var(--fertile-2)' },
  ovulation: { bg: 'var(--fertile-soft)', color: 'var(--fertile-4)' },
  pms: { bg: 'var(--violet-soft)', color: 'var(--violet)' },
};

/** How far {@link fertileLevelAt} looks for the ovulation day around a fertile day. */
const OVULATION_SEARCH_DAYS = 6;

/**
 * Chance step of a fertile-window day from its distance to ovulation
 * (`daysToOvulation` = ovulation − day): ovulation itself is the peak (4), the
 * two days before it nearly as high (3), the two before those lower (2), and
 * the window's edges — five days before, or after ovulation — the lowest (1).
 * Unknown distance reads as the middle step. Purely visual (§11).
 */
export function fertileLevel(daysToOvulation: number | null): FertileLevel {
  if (daysToOvulation == null) return 2;
  if (daysToOvulation === 0) return 4;
  if (daysToOvulation === 1 || daysToOvulation === 2) return 3;
  if (daysToOvulation === 3 || daysToOvulation === 4) return 2;
  return 1;
}

/**
 * Chance step of a day from the markers around it: `markerAt(offset)` returns
 * the marker `offset` days from the day (0 = the day itself). Null for a day
 * outside the fertile window.
 */
export function fertileLevelAt(markerAt: (offset: number) => CycleDayMarker | null): FertileLevel | null {
  const own = markerAt(0);
  if (own === 'ovulation') return 4;
  if (own !== 'fertile') return null;
  for (let k = 1; k <= OVULATION_SEARCH_DAYS; k += 1) {
    if (markerAt(k) === 'ovulation') return fertileLevel(k);
    if (markerAt(-k) === 'ovulation') return fertileLevel(-k);
  }
  return fertileLevel(null);
}
