/*
 * Pure geometry of the Night & Bloom SVG charts — no React, unit-tested.
 * Plots are always laid out left → right (the chart container is
 * `direction: ltr`), so index 0 is the oldest point in both RTL and LTR.
 */

export interface PlotBox {
  width: number;
  height: number;
  /** Inner padding: top, right, bottom, left (room for the axis labels). */
  pad: readonly [number, number, number, number];
}

export interface Domain {
  min: number;
  max: number;
}

/** Min/max over every finite value, widened so a flat series is still drawable. */
export function domainOf(series: ReadonlyArray<ReadonlyArray<number | null>>, fixed?: Partial<Domain>): Domain {
  const values = series.flat().filter((v): v is number => v !== null && Number.isFinite(v));
  let min = fixed?.min ?? (values.length ? Math.min(...values) : 0);
  let max = fixed?.max ?? (values.length ? Math.max(...values) : 1);
  if (max === min) {
    min -= 1;
    max += 1;
  }
  return { min, max };
}

/** x of point `index` of `count`, spread across the plot width. */
export function xAt(index: number, count: number, box: PlotBox): number {
  const [, right, , left] = box.pad;
  const inner = box.width - left - right;
  return count <= 1 ? left + inner / 2 : left + (inner * index) / (count - 1);
}

/** y of `value` in `domain` (SVG y grows downwards). */
export function yAt(value: number, domain: Domain, box: PlotBox): number {
  const [top, , bottom] = box.pad;
  const inner = box.height - top - bottom;
  return top + inner * (1 - (value - domain.min) / (domain.max - domain.min));
}

const r = (n: number) => Math.round(n * 100) / 100;

/**
 * SVG path for a series; `null` values break the line (a missed day is a gap,
 * not a guess). Returns '' when there is nothing to draw.
 */
export function linePath(values: ReadonlyArray<number | null>, domain: Domain, box: PlotBox): string {
  let d = '';
  let pen = false;
  values.forEach((v, i) => {
    if (v === null || !Number.isFinite(v)) {
      pen = false;
      return;
    }
    d += `${pen ? 'L' : 'M'}${r(xAt(i, values.length, box))} ${r(yAt(v, domain, box))}`;
    pen = true;
  });
  return d;
}

/** Closed band between two series (growth P3–P97, normal range). */
export function bandPath(
  lower: ReadonlyArray<number>,
  upper: ReadonlyArray<number>,
  domain: Domain,
  box: PlotBox,
): string {
  const n = Math.min(lower.length, upper.length);
  if (n === 0) return '';
  const top = upper.slice(0, n).map((v, i) => `${i ? 'L' : 'M'}${r(xAt(i, n, box))} ${r(yAt(v, domain, box))}`);
  const bottom = lower
    .slice(0, n)
    .map((v, i) => ({ v, i }))
    .reverse()
    .map(({ v, i }) => `L${r(xAt(i, n, box))} ${r(yAt(v, domain, box))}`);
  return `${top.join('')}${bottom.join('')}Z`;
}

export interface BarRect {
  x: number;
  y: number;
  width: number;
  height: number;
}

/** Bars from a zero baseline, evenly spaced; `gap` is the fraction of each slot left empty. */
export function barRects(values: ReadonlyArray<number>, max: number, box: PlotBox, gap = 0.35): BarRect[] {
  const [top, right, bottom, left] = box.pad;
  const innerW = box.width - left - right;
  const innerH = box.height - top - bottom;
  const slot = values.length ? innerW / values.length : 0;
  const width = slot * (1 - gap);
  const top0 = Math.max(max, 1e-9);
  return values.map((v, i) => {
    const h = innerH * Math.max(0, Math.min(1, v / top0));
    return {
      x: r(left + slot * i + (slot - width) / 2),
      y: r(top + innerH - h),
      width: r(width),
      height: r(h),
    };
  });
}

/** Stroke dash for a ring arc covering `fraction` (0–1) of the circumference. */
export function ringDash(fraction: number, radius: number): { dash: string; circumference: number } {
  const circumference = 2 * Math.PI * radius;
  const f = Math.max(0, Math.min(1, fraction));
  return { dash: `${r(circumference * f)} ${r(circumference)}`, circumference: r(circumference) };
}
