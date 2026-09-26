import type { BbtCycle } from "@/entities/fertility";

/** Pixel box the chart draws into (SVG user units). */
export interface ChartFrame {
  width: number;
  height: number;
  top: number;
  bottom: number;
  left: number;
  right: number;
}

export const DEFAULT_FRAME: ChartFrame = {
  width: 320,
  height: 190,
  top: 12,
  bottom: 26,
  left: 36,
  right: 10,
};

/** The artboard's y grid; widened (in 0.1 °C steps) when a reading falls outside it. */
export const BASE_Y_MIN = 36.2;
export const BASE_Y_MAX = 36.8;
/** The x axis always shows at least a typical cycle. */
export const MIN_DAYS = 28;

export interface YDomain {
  min: number;
  max: number;
  ticks: number[];
}

const round1 = (v: number): number => Math.round(v * 10) / 10;

/** 36.2–36.8 by default; auto-scaled outward to the nearest 0.1 °C to fit every value. */
export function yDomain(values: readonly number[]): YDomain {
  let min = BASE_Y_MIN;
  let max = BASE_Y_MAX;
  for (const v of values) {
    if (v < min) min = Math.floor(v * 10 + 1e-9) / 10;
    if (v > max) max = Math.ceil(v * 10 - 1e-9) / 10;
  }
  min = round1(min);
  max = round1(max);
  const span = round1(max - min);
  // ≤ 7 grid lines: 0.1 steps on the default span, coarser when auto-scaled wide.
  const step = span <= 0.6 + 1e-9 ? 0.1 : span <= 1.2 + 1e-9 ? 0.2 : 0.5;
  const ticks: number[] = [];
  for (let t = min; t <= max + 1e-9; t = round1(t + step))
    ticks.push(round1(t));
  return { min, max, ticks };
}

/** Last cycle day the x axis must reach. */
export function maxCycleDay(cycles: readonly BbtCycle[]): number {
  let max = MIN_DAYS;
  for (const c of cycles) {
    for (const p of c.points) max = Math.max(max, p.cycleDay);
    if (c.fertileWindow) max = Math.max(max, c.fertileWindow.toDay);
  }
  return max;
}

export interface BbtScales {
  frame: ChartFrame;
  domain: YDomain;
  maxDay: number;
  x: (day: number) => number;
  y: (value: number) => number;
  xTicks: number[];
}

export function buildScales(
  cycles: readonly BbtCycle[],
  frame: ChartFrame = DEFAULT_FRAME,
): BbtScales {
  const domain = yDomain(
    cycles.flatMap((c) => [
      ...c.points.map((p) => p.value),
      ...(c.coverline === null ? [] : [c.coverline]),
    ]),
  );
  const maxDay = maxCycleDay(cycles);
  const plotW = frame.width - frame.left - frame.right;
  const plotH = frame.height - frame.top - frame.bottom;
  const x = (day: number): number =>
    frame.left + ((day - 1) / Math.max(1, maxDay - 1)) * plotW;
  const y = (value: number): number =>
    frame.top + ((domain.max - value) / (domain.max - domain.min)) * plotH;
  const xTicks: number[] = [1];
  for (let d = 7; d < maxDay; d += 7) xTicks.push(d);
  if (xTicks[xTicks.length - 1] !== maxDay) xTicks.push(maxDay);
  return { frame, domain, maxDay, x, y, xTicks };
}

/** y of the dashed coverline; null when the cycle has none yet. */
export function coverlineY(
  scales: BbtScales,
  coverline: number | null,
): number | null {
  return coverline === null ? null : scales.y(coverline);
}

/** Horizontal extent of the fertile band: half a day either side of the inclusive range. */
export function fertileBandX(
  scales: BbtScales,
  window: { fromDay: number; toDay: number } | null,
): { x: number; width: number } | null {
  if (!window) return null;
  const { left, width, right } = scales.frame;
  const half = (scales.x(2) - scales.x(1)) / 2;
  const x0 = Math.max(left, scales.x(window.fromDay) - half);
  const x1 = Math.min(width - right, scales.x(window.toDay) + half);
  return { x: x0, width: Math.max(0, x1 - x0) };
}

/** SVG path through the points in day order. */
export function linePath(
  scales: BbtScales,
  points: readonly { cycleDay: number; value: number }[],
): string {
  return [...points]
    .sort((a, b) => a.cycleDay - b.cycleDay)
    .map(
      (p, i) =>
        `${i === 0 ? "M" : "L"}${scales.x(p.cycleDay).toFixed(1)} ${scales.y(p.value).toFixed(1)}`,
    )
    .join(" ");
}

/** The line closed down to the plot floor, for the area gradient. */
export function areaPath(
  scales: BbtScales,
  points: readonly { cycleDay: number; value: number }[],
): string {
  if (points.length === 0) return "";
  const sorted = [...points].sort((a, b) => a.cycleDay - b.cycleDay);
  const floor = scales.frame.height - scales.frame.bottom;
  const first = sorted[0]!;
  const last = sorted[sorted.length - 1]!;
  return `${linePath(scales, sorted)} L${scales.x(last.cycleDay).toFixed(1)} ${floor} L${scales.x(first.cycleDay).toFixed(1)} ${floor} Z`;
}
