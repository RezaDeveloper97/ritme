import { cycleDayMarker } from './predictions';
import type { CycleCalculation, CycleDayMarker, MarkerIntensity } from './types';

/**
 * The one palette for cycle day markers, shared by every calendar surface (the
 * full calendar screen and the home mini calendar) so a period day is never
 * pink in one place and something else in another. Follicular/luteal days carry
 * no marker and read as neutral. `bg` is the medium intensity step — day cells
 * pick their exact step from {@link cycleMarkerBg}.
 */
export const cycleMarkerStyle: Record<CycleDayMarker, { bg: string; color: string }> = {
  // Period stays red — bleeding days read as red to users, and that convention
  // outranks the brand palette. It is the one sanctioned red (§10.2).
  period: { bg: 'var(--period-soft)', color: 'var(--period)' },
  fertile: { bg: 'var(--amber-soft)', color: 'var(--amber)' },
  // Ovulation is algorithmic data — it carries the turquoise data accent (§10.2).
  ovulation: { bg: 'var(--data-soft)', color: 'var(--data-deep)' },
  pms: { bg: 'var(--violet-soft)', color: 'var(--violet)' },
};

/**
 * Cell background per marker at each intensity step.
 */
export const cycleMarkerBg: Record<CycleDayMarker, Record<MarkerIntensity, string>> = {
  period: {
    faint: 'var(--period-soft-faint)',
    medium: 'var(--period-soft)',
    strong: 'var(--period-soft-strong)',
  },
  fertile: {
    faint: 'var(--amber-soft-faint)',
    medium: 'var(--amber-soft)',
    strong: 'var(--amber-soft-strong)',
  },
  ovulation: {
    faint: 'var(--data-soft-faint)',
    medium: 'var(--data-soft)',
    strong: 'var(--data-soft-strong)',
  },
  pms: {
    faint: 'var(--violet-soft-faint)',
    medium: 'var(--violet-soft)',
    strong: 'var(--violet-soft-strong)',
  },
};

/**
 * Only the fertility-related markers are graded by conception probability;
 * period and PMS days keep their single base tint (product decision — the
 * ramp is about visualizing pregnancy chance, which those days don't carry).
 */
const GRADED_MARKERS: ReadonlySet<CycleDayMarker> = new Set(['fertile', 'ovulation']);

/** Below this spread (percentage points) a set of days reads as "all the same". */
const FLAT_RANGE_PP = 0.5;

const bucket = (t: number): MarkerIntensity => (t < 1 / 3 ? 'faint' : t < 2 / 3 ? 'medium' : 'strong');

const percentOf = (calc: CycleCalculation): number => calc.fertilityPercent ?? 0;

/**
 * Marker intensity per calendar day (keyed by `calculationDate`), from the
 * day's conception probability. Only fertile-window and ovulation days are
 * graded, normalized within their own marker group — the fertile day closest
 * to ovulation paints strong, the opening day faint. A group with no internal
 * spread falls back to the graded days' overall range, then to `medium`.
 * Purely visual and informational (§11).
 */
export function markerIntensityByDate(calcs: Iterable<CycleCalculation>): Map<string, MarkerIntensity> {
  const groups = new Map<CycleDayMarker, CycleCalculation[]>();
  let globalMin = Infinity;
  let globalMax = -Infinity;
  for (const calc of calcs) {
    const marker = cycleDayMarker(calc);
    if (!marker || !GRADED_MARKERS.has(marker)) continue;
    const group = groups.get(marker);
    if (group) group.push(calc);
    else groups.set(marker, [calc]);
    globalMin = Math.min(globalMin, percentOf(calc));
    globalMax = Math.max(globalMax, percentOf(calc));
  }

  const out = new Map<string, MarkerIntensity>();
  for (const days of groups.values()) {
    let min = Infinity;
    let max = -Infinity;
    for (const d of days) {
      min = Math.min(min, percentOf(d));
      max = Math.max(max, percentOf(d));
    }
    for (const d of days) {
      let intensity: MarkerIntensity;
      if (max - min >= FLAT_RANGE_PP) intensity = bucket((percentOf(d) - min) / (max - min));
      else if (globalMax - globalMin >= FLAT_RANGE_PP) intensity = bucket((percentOf(d) - globalMin) / (globalMax - globalMin));
      else intensity = 'medium';
      out.set(d.calculationDate, intensity);
    }
  }
  return out;
}
