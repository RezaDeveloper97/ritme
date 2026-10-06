/*
 * Geometry of the vitals report charts (pure, no React, no locale). Plots run
 * left → right, oldest first, in every direction (B-N3-09 LTR axes). The y
 * scale hugs the data but always shows the target band, padded to round ticks.
 */

export const CHART_W = 320;
export const PAD_X = 28; // room for the y labels on the left
export const PAD_Y = 10;

export interface Scale {
  min: number;
  max: number;
  ticks: number[];
  y: (v: number) => number;
  height: number;
}

/** Nice tick step for a span (10, 20, 25, 50 …). */
export function tickStep(span: number, target = 4): number {
  const raw = Math.max(span, 1) / target;
  const pow = 10 ** Math.floor(Math.log10(raw));
  for (const m of [1, 2, 2.5, 5, 10]) if (raw <= m * pow) return m * pow;
  return 10 * pow;
}

/** A y scale over `values` and the band, rounded out to whole ticks. */
export function makeScale(values: readonly number[], band: readonly [number, number], height: number): Scale {
  const all = [...values.filter(Number.isFinite), band[0], band[1]];
  const lo = Math.min(...all);
  const hi = Math.max(...all);
  const step = tickStep(hi - lo);
  const min = Math.floor((lo - step / 2) / step) * step;
  const max = Math.ceil((hi + step / 2) / step) * step;
  const ticks: number[] = [];
  for (let v = min; v <= max + 1e-9; v += step) ticks.push(Math.round(v * 10) / 10);
  const y = (v: number) => Math.round((PAD_Y + (height - 2 * PAD_Y) * (1 - (v - min) / (max - min || 1))) * 10) / 10;
  return { min, max, ticks, y, height };
}

/** x of the i-th of n slots (centre), inside the plot area. */
export function slotX(i: number, n: number): number {
  const left = PAD_X + 8;
  const right = CHART_W - 8;
  if (n <= 1) return (left + right) / 2;
  return Math.round((left + ((right - left) * i) / (n - 1)) * 10) / 10;
}

/** SVG path through the non-null points (a null breaks the line). */
export function linePath(points: ReadonlyArray<{ x: number; y: number } | null>): string {
  let d = '';
  let pen = false;
  for (const p of points) {
    if (!p) {
      pen = false;
      continue;
    }
    d += `${pen ? 'L' : 'M'}${p.x} ${p.y}`;
    pen = true;
  }
  return d;
}

/** Up to `max` evenly spaced indexes of n (always the first and the last), for x labels. */
export function labelIndexes(n: number, max: number): number[] {
  if (n <= 0) return [];
  if (n <= max) return Array.from({ length: n }, (_, i) => i);
  const out = new Set<number>([0, n - 1]);
  const step = (n - 1) / (max - 1);
  for (let k = 1; k < max - 1; k++) out.add(Math.round(k * step));
  return [...out].sort((a, b) => a - b);
}
