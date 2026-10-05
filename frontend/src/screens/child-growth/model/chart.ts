import type { GrowthPoint, GrowthReferenceRow } from './types';

/*
 * Geometry of the v16_Growth chart: a 330×190 SVG, months on x (left → right in
 * both directions — time axes stay LTR), the indicator's unit on y. Pure, so the
 * maths is unit-tested apart from React.
 */

export const CHART_W = 330;
export const CHART_H = 190;
/** Plot box inside the SVG (room for y labels on the left and month labels below). */
export const PLOT = { left: 30, right: 320, top: 8, bottom: 170 } as const;

export interface ChartScale {
  minMonth: number;
  maxMonth: number;
  minValue: number;
  maxValue: number;
  x: (month: number) => number;
  y: (value: number) => number;
  yTicks: number[];
  xTicks: number[];
}

const round1 = (n: number) => Math.round(n * 10) / 10;

/** A "nice" tick step (1, 2, 2.5, 5 × 10ⁿ) giving about `target` intervals over `span`. */
export function niceStep(span: number, target = 4): number {
  if (!(span > 0)) return 1;
  const raw = span / target;
  const pow = 10 ** Math.floor(Math.log10(raw));
  const unit = raw / pow;
  const nice = unit <= 1 ? 1 : unit <= 2 ? 2 : unit <= 2.5 ? 2.5 : unit <= 5 ? 5 : 10;
  return nice * pow;
}

/** Month labels: every month up to a year, then every 3 / 6 / 12 months. */
export function monthStep(span: number): number {
  if (span <= 12) return 1;
  if (span <= 24) return 3;
  if (span <= 36) return 6;
  return 12;
}

/**
 * The scales for a range of months and the values that must fit (reference band
 * + the child's points). The y range snaps outward to whole ticks.
 */
export function buildScale(
  fromMonth: number,
  toMonth: number,
  reference: readonly GrowthReferenceRow[],
  points: readonly GrowthPoint[],
): ChartScale {
  const minMonth = Math.max(0, Math.min(fromMonth, ...points.map((p) => p.ageMonths)));
  const maxMonth = Math.max(toMonth, minMonth + 1, ...points.map((p) => Math.ceil(p.ageMonths)));
  const values = [
    ...reference.filter((r) => r.month >= minMonth && r.month <= maxMonth).flatMap((r) => [r.p3, r.p97]),
    ...points.map((p) => p.value),
  ].filter((v) => Number.isFinite(v));
  let lo = values.length ? Math.min(...values) : 0;
  let hi = values.length ? Math.max(...values) : 1;
  if (hi - lo < 1) {
    lo -= 0.5;
    hi += 0.5;
  }
  const step = niceStep(hi - lo);
  const minValue = Math.floor(lo / step) * step;
  const maxValue = Math.ceil(hi / step) * step;
  const yTicks: number[] = [];
  for (let v = minValue; v <= maxValue + step / 1000; v += step) yTicks.push(Math.round(v * 1000) / 1000);
  const mStep = monthStep(maxMonth - minMonth);
  const xTicks: number[] = [];
  for (let m = Math.ceil(minMonth / mStep) * mStep; m <= maxMonth; m += mStep) xTicks.push(m);

  const x = (month: number) =>
    round1(PLOT.left + ((PLOT.right - PLOT.left) * (month - minMonth)) / (maxMonth - minMonth));
  const y = (value: number) =>
    round1(PLOT.bottom - ((PLOT.bottom - PLOT.top) * (value - minValue)) / (maxValue - minValue || 1));
  return { minMonth, maxMonth, minValue, maxValue, x, y, yTicks, xTicks };
}

/** Closed area between P3 (bottom edge) and P97 (top edge). '' with fewer than two rows. */
export function bandPath(reference: readonly GrowthReferenceRow[], s: ChartScale): string {
  const rows = reference.filter((r) => r.month >= s.minMonth && r.month <= s.maxMonth);
  if (rows.length < 2) return '';
  const low = rows.map((r) => `${s.x(r.month)},${s.y(r.p3)}`);
  const high = [...rows].reverse().map((r) => `${s.x(r.month)},${s.y(r.p97)}`);
  return `M${low.join(' L')} L${high.join(' L')} Z`;
}

/** An open polyline through `pts` ('' with fewer than two). */
export function linePath(pts: ReadonlyArray<{ x: number; y: number }>): string {
  if (pts.length < 2) return '';
  return `M${pts.map((p) => `${p.x},${p.y}`).join(' L')}`;
}

/** The median (P50) curve over the visible months. */
export function medianPath(reference: readonly GrowthReferenceRow[], s: ChartScale): string {
  return linePath(
    reference.filter((r) => r.month >= s.minMonth && r.month <= s.maxMonth).map((r) => ({ x: s.x(r.month), y: s.y(r.p50) })),
  );
}

/** The child's points in screen space, oldest first. */
export function pointCoords(points: readonly GrowthPoint[], s: ChartScale): Array<{ x: number; y: number; point: GrowthPoint }> {
  return [...points]
    .sort((a, b) => a.ageMonths - b.ageMonths)
    .map((point) => ({ x: s.x(point.ageMonths), y: s.y(point.value), point }));
}

/**
 * The last month drawn: the child's latest age plus 3 months, at least 6 (the
 * artboard's 0–6 for a 3-month-old), never past the server's range.
 */
export function visibleToMonth(series: { fromMonth: number; toMonth: number; points: readonly GrowthPoint[] }): number {
  const oldest = Math.max(0, ...series.points.map((p) => p.ageMonths));
  return Math.min(series.toMonth, Math.max(series.fromMonth + 6, Math.ceil(oldest) + 3));
}
