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
 * Cell background per marker at each intensity step. The step comes from
 * {@link markerIntensityByDate}: days with a higher conception probability get
 * the deeper tint, so the calendar reads as a gradient within each color.
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

/**
 * Below this spread (in percentage points of conception probability) a set of
 * days reads as "all the same" and no gradient is drawn from it.
 */
const FLAT_RANGE_PP = 0.5;

const bucket = (t: number): MarkerIntensity =>
  t < 1 / 3 ? 'faint' : t < 2 / 3 ? 'medium' : 'strong';

/**
 * Marker intensity for each calendar day (keyed by `calculationDate`), from the
 * day's conception probability (`fertilityPercent`). Only fertile-window and
 * ovulation days are graded ({@link GRADED_MARKERS}); period/PMS days are left
 * out and render at their base tint. Graded days are normalized within their
 * own marker group — the fertile-window day closest to ovulation paints strong,
 * the opening day faint. A group with no internal spread (e.g. a lone ovulation
 * day) falls back to its position in the graded days' overall probability
 * range, and to `medium` when even that is flat. Purely visual and
 * informational (§11).
 */
export function markerIntensityByDate(
  calcs: Iterable<CycleCalculation>,
): Map<string, MarkerIntensity> {
  const groups = new Map<CycleDayMarker, CycleCalculation[]>();
  let globalMin = Infinity;
  let globalMax = -Infinity;
  for (const calc of calcs) {
    const marker = cycleDayMarker(calc);
    if (!marker || !GRADED_MARKERS.has(marker)) continue;
    const group = groups.get(marker);
    if (group) group.push(calc);
    else groups.set(marker, [calc]);
    globalMin = Math.min(globalMin, calc.fertilityPercent);
    globalMax = Math.max(globalMax, calc.fertilityPercent);
  }

  const out = new Map<string, MarkerIntensity>();
  for (const days of groups.values()) {
    let min = Infinity;
    let max = -Infinity;
    for (const d of days) {
      min = Math.min(min, d.fertilityPercent);
      max = Math.max(max, d.fertilityPercent);
    }
    for (const d of days) {
      let intensity: MarkerIntensity;
      if (max - min >= FLAT_RANGE_PP) {
        intensity = bucket((d.fertilityPercent - min) / (max - min));
      } else if (globalMax - globalMin >= FLAT_RANGE_PP) {
        intensity = bucket((d.fertilityPercent - globalMin) / (globalMax - globalMin));
      } else {
        intensity = 'medium';
      }
      out.set(d.calculationDate, intensity);
    }
  }
  return out;
}
