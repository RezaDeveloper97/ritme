/**
 * Geometry of the dotted rings drawn on the splash, the intro slides and the
 * welcome card (Night & Bloom `Splash` / `Intro_1` / `Intro_2` / `Onb_Welcome`).
 * Pure so the layout can be unit-tested without a DOM.
 *
 * Both rings sit on a 300×300 viewBox, radius 128, first dot at 12 o'clock and
 * going clockwise — the artboards draw them that way in both directions, so the
 * geometry is direction-agnostic and needs no RTL flip.
 */
export const RING_VIEW = 300;
const CENTER = RING_VIEW / 2;
const RADIUS = 128;

export interface RingPoint {
  x: number;
  y: number;
}

/** Position of dot `index` of `count`, evenly spaced clockwise from the top. */
export function ringPoint(index: number, count: number): RingPoint {
  const a = (index * 2 * Math.PI) / count;
  return {
    x: round(CENTER + RADIUS * Math.sin(a)),
    y: round(CENTER - RADIUS * Math.cos(a)),
  };
}

function round(n: number): number {
  return Math.round(n * 10) / 10;
}

/** The illustrative 29-day cycle the artboards draw (one dot per day). */
export type CycleDotKind = 'period' | 'neutral' | 'fertile' | 'ovulation' | 'pms';

export const CYCLE_DAYS = 29;

/** Day kind by 0-based index: 5 period, 5 quiet, fertile window 10–15 with ovulation on 14, 6 PMS days at the end. */
export function cycleDotKind(index: number): CycleDotKind {
  if (index < 5) return 'period';
  if (index < 10) return 'neutral';
  if (index === 14) return 'ovulation';
  if (index < 16) return 'fertile';
  if (index < 23) return 'neutral';
  return 'pms';
}

/** Dot radius per kind (artboard values). */
export const CYCLE_DOT_R: Record<CycleDotKind, number> = {
  period: 7,
  neutral: 5,
  fertile: 7,
  ovulation: 8,
  pms: 6,
};

/**
 * A day up to and including `today` is drawn filled, a later one as an
 * outline. Ovulation is always filled: it is the one data marker.
 */
export function isCycleDotFilled(index: number, today: number): boolean {
  return index <= today || cycleDotKind(index) === 'ovulation';
}

/** 40 pregnancy weeks, one dot per week. */
export const PREGNANCY_WEEKS = 40;

export type Trimester = 1 | 2 | 3;

/** Trimester of week `index` (0-based): weeks 1–13, 14–27, 28–40. */
export function trimesterOf(index: number): Trimester {
  if (index < 13) return 1;
  if (index < 27) return 2;
  return 3;
}
