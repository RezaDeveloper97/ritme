import { fromApiDate } from '@/shared/lib/date';

import type { WeightGain } from '../api/schema';

/** Plot box of the weight-gain chart (viewBox units; the SVG scales to the card width). */
export const GAIN_W = 320;
export const GAIN_PAD_X = 10;
export const GAIN_PAD_Y = 10;
export const GAIN_AXIS = 20;

/** The axis ticks the artboard prints (weeks 0 · 13 · 28 · 40). */
export const GAIN_TICKS = [0, 13, 28, 40] as const;

export interface GainChart {
  height: number;
  /** Closed path of the recommended band (empty without one). */
  band: string;
  /** The user's gain line (empty with fewer than one point). */
  line: string;
  /** The latest point. */
  end: { x: number; y: number } | null;
  ticks: Array<{ week: number; x: number }>;
  axisY: number;
}

const r1 = (n: number) => Math.round(n * 10) / 10;

/** Recommended band value at gestational week `w` — the piecewise-linear band the server sends. */
export function bandAt(band: WeightGain['band'], w: number): { min: number; max: number } | null {
  if (!band.length) return null;
  if (w <= band[0].gaWeeks) return { min: band[0].min, max: band[0].max };
  for (let i = 1; i < band.length; i++) {
    const a = band[i - 1];
    const b = band[i];
    if (w <= b.gaWeeks) {
      const f = (w - a.gaWeeks) / (b.gaWeeks - a.gaWeeks || 1);
      return { min: a.min + (b.min - a.min) * f, max: a.max + (b.max - a.max) * f };
    }
  }
  const last = band[band.length - 1];
  return { min: last.min, max: last.max };
}

/**
 * Geometry of An_PregWeight's chart: x = gestational week 0–40 (further if
 * overdue), y = kg gained from 0 (or the lowest gain) to the band's top at
 * term (or the highest gain). Plots left → right in both directions.
 */
export function gainChart(w: Pick<WeightGain, 'band' | 'points'>, height = 150): GainChart {
  const lastWeek = w.points.length ? w.points[w.points.length - 1].gaDays / 7 : 0;
  const xMax = Math.max(40, Math.ceil(lastWeek));
  const gains = w.points.map((p) => p.gain);
  const bandTop = w.band.length ? Math.max(...w.band.map((b) => b.max)) : 0;
  const yMax = Math.max(bandTop, ...gains, 1);
  const yMin = Math.min(0, ...gains);
  const axisY = height - GAIN_AXIS;
  const x = (week: number) => r1(GAIN_PAD_X + ((GAIN_W - 2 * GAIN_PAD_X) * week) / xMax);
  const y = (kg: number) => r1(GAIN_PAD_Y + (axisY - 2 * GAIN_PAD_Y) * (1 - (kg - yMin) / (yMax - yMin || 1)));

  let band = '';
  if (w.band.length) {
    const xs = [...w.band.map((b) => b.gaWeeks), xMax].filter((v, i, a) => a.indexOf(v) === i).sort((a, b) => a - b);
    const top = xs.map((wk) => `${x(wk)} ${y(bandAt(w.band, wk)?.max ?? 0)}`);
    const bottom = [...xs].reverse().map((wk) => `${x(wk)} ${y(bandAt(w.band, wk)?.min ?? 0)}`);
    band = `M${top.join('L')}L${bottom.join('L')}Z`;
  }
  let line = '';
  let end: GainChart['end'] = null;
  w.points.forEach((p, i) => {
    const px = x(p.gaDays / 7);
    const py = y(p.gain);
    line += `${i ? 'L' : 'M'}${px} ${py}`;
    end = { x: px, y: py };
  });
  return {
    height,
    band,
    line,
    end,
    ticks: GAIN_TICKS.filter((t) => t <= xMax).map((t) => ({ week: t, x: x(t) })),
    axisY,
  };
}

/** Which IOM row the user's pre-pregnancy BMI falls in (by category key). */
export function isUserRow(row: { category: string }, category: string | null): boolean {
  return category != null && row.category === category;
}

const JS_WEEKDAY = ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat'] as const;

/** Weekday key of a `YYYY-MM-DD` date (calendar-independent). */
export function weekdayKeyOf(apiDate: string): (typeof JS_WEEKDAY)[number] {
  return JS_WEEKDAY[fromApiDate(apiDate).getDay()];
}

/** The two items a trimester tile shows (the payload sends three). */
export function tileItems<T>(items: readonly T[]): T[] {
  return items.slice(0, 2);
}

export interface BpChart {
  line: string;
  end: { x: number; y: number } | null;
  /** y of the systolic threshold line (140). */
  thresholdY: number;
  /** Readings at or above the threshold, as dots. */
  highs: Array<{ x: number; y: number }>;
}

/**
 * The hub's blood-pressure line (systolic, oldest first) on a fixed clinical
 * domain — at least 90–150 mmHg, wider when a reading falls outside — so a
 * 5 mmHg wobble stays a wobble and the threshold line sits where it means
 * something.
 */
export function bpChart(
  readings: ReadonlyArray<{ systolic: number; high: boolean }>,
  threshold: number,
  height = 90,
  width = GAIN_W,
): BpChart {
  const vals = readings.map((r) => r.systolic);
  const lo = Math.min(90, ...vals.map((v) => v - 5));
  const hi = Math.max(threshold + 10, ...vals.map((v) => v + 5));
  const pad = 8;
  const n = readings.length;
  const x = (i: number) => r1(n <= 1 ? width / 2 : pad + ((width - 2 * pad) * i) / (n - 1));
  const y = (v: number) => r1(pad + (height - 2 * pad) * (1 - (v - lo) / (hi - lo || 1)));
  let line = '';
  let end: BpChart['end'] = null;
  const highs: BpChart['highs'] = [];
  readings.forEach((r, i) => {
    const p = { x: x(i), y: y(r.systolic) };
    line += `${i ? 'L' : 'M'}${p.x} ${p.y}`;
    end = p;
    if (r.high) highs.push(p);
  });
  return { line, end, thresholdY: y(threshold), highs };
}
