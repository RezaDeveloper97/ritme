import type { BbtPoint, LhTest, LhValue } from './types';

/** The LH strip of the hub card: 7 cycle days. */
export const LH_STRIP_DAYS = 7;

export interface LhCell {
  day: number;
  value: LhValue | null;
}

/**
 * The 7 cycle days the LH strip shows (An_Hub_TTC «تست LH»): four days before
 * the positive test to two after it, else the 7 days ending on the last test.
 * Days without a test are `null`. Empty without any test.
 */
export function lhCells(tests: readonly LhTest[], positiveDay: number | null): LhCell[] {
  if (!tests.length) return [];
  const last = tests[tests.length - 1].day;
  const first = positiveDay != null ? Math.max(1, positiveDay - 4) : Math.max(1, last - LH_STRIP_DAYS + 1);
  const byDay = new Map(tests.map((t) => [t.day, t.value]));
  return Array.from({ length: LH_STRIP_DAYS }, (_, i) => ({ day: first + i, value: byDay.get(first + i) ?? null }));
}

/** Share of the referral threshold reached (the trying ring), 0–1. */
export function tryingShare(months: number, thresholdMonths: number): number {
  if (thresholdMonths <= 0) return 0;
  return Math.min(1, Math.max(0, months / thresholdMonths));
}

/** Bar heights (0–1) of the regularity card, relative to the longest. */
export function barShares(lengths: readonly number[]): number[] {
  const top = Math.max(1, ...lengths);
  return lengths.map((l) => Math.max(0.08, l / top));
}

export interface ChartBox {
  width: number;
  height: number;
  /** Inner plot edges. */
  left: number;
  right: number;
  top: number;
  bottom: number;
}

export interface TempScale {
  min: number;
  max: number;
  ticks: number[];
}

const STEP = 0.2;

/**
 * The y domain of the BBT chart: the readings and the coverline, padded and
 * snapped to 0.2 °C, at least 0.6 °C tall (a 0.1 wobble must not look like a
 * shift). Ticks every 0.2 °C.
 */
export function tempScale(values: readonly number[]): TempScale {
  const finite = values.filter((v) => Number.isFinite(v));
  const lo = finite.length ? Math.min(...finite) : 36.2;
  const hi = finite.length ? Math.max(...finite) : 36.8;
  let min = Math.floor((lo - 0.05) / STEP) * STEP;
  let max = Math.ceil((hi + 0.05) / STEP) * STEP;
  while (max - min < 0.6 - 1e-9) {
    min -= STEP / 2;
    max += STEP / 2;
  }
  min = Math.round(min * 100) / 100;
  max = Math.round(max * 100) / 100;
  const ticks: number[] = [];
  for (let t = Math.ceil(min / STEP - 1e-9) * STEP; t <= max + 1e-9; t += STEP) ticks.push(Math.round(t * 100) / 100);
  return { min, max, ticks };
}

/** x of cycle day `day` (1-based) in a cycle of `span` days. */
export function dayX(day: number, span: number, box: ChartBox): number {
  const n = Math.max(2, span);
  return box.left + ((box.right - box.left) * (day - 1)) / (n - 1);
}

/** y of a temperature. */
export function tempY(v: number, scale: TempScale, box: ChartBox): number {
  const span = scale.max - scale.min || 1;
  return box.bottom - ((box.bottom - box.top) * (v - scale.min)) / span;
}

/** The polyline of the readings (consecutive readings joined, gaps bridged — as the artboard draws it). */
export function linePath(points: readonly BbtPoint[], span: number, scale: TempScale, box: ChartBox): string {
  return points
    .map((p, i) => {
      const x = Math.round(dayX(p.day, span, box) * 10) / 10;
      const y = Math.round(tempY(p.value, scale, box) * 10) / 10;
      return `${i === 0 ? 'M' : 'L'}${x} ${y}`;
    })
    .join('');
}

/** Bidi-isolates an interpolated value (FSI … PDI, the string form of `<bdi>`), B-N3-14b. */
export function isolate(value: string): string {
  return `⁨${value}⁩`;
}
