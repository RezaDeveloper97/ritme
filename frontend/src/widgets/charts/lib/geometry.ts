/*
 * Pure layout math of the analysis charts (B-N3-09) — no React, unit-tested.
 * Every plot runs left → right (`direction: ltr`) in both locales: index 0 is
 * the oldest point / day 1, as the artboards draw them.
 */

/** A bar height as a share (0–1) of `max`, with a floor so a non-zero value stays visible. */
export function heightShare(value: number, max: number, floor = 0.08): number {
  if (!(max > 0) || !(value > 0)) return 0;
  return Math.max(floor, Math.min(1, value / max));
}

/** Evenly spaced column slots across `width`: x and width of slot `i` of `count` with `gap` between. */
export function slot(i: number, count: number, width: number, gap: number): { x: number; w: number } {
  if (count <= 0) return { x: 0, w: 0 };
  const w = (width - gap * (count - 1)) / count;
  return { x: i * (w + gap), w };
}

/** Segments of a stacked bar: start offset and width of each part, in `width` units. */
export function stackSegments(parts: readonly number[], width: number): Array<{ x: number; w: number }> {
  const total = parts.reduce((s, p) => s + Math.max(0, p), 0);
  let x = 0;
  return parts.map((p) => {
    const w = total > 0 ? (Math.max(0, p) / total) * width : 0;
    const seg = { x, w };
    x += w;
    return seg;
  });
}

/** Opacity of a pattern cell from its share (0–1): a faint floor so empty days still read as cells. */
export function cellOpacity(share: number): number {
  const v = Math.max(0, Math.min(1, share));
  return Math.round((0.12 + 0.88 * v) * 100) / 100;
}

/** Indices to label on a long axis: first, last and (for ≥ 3) the middle. */
export function axisTicks(count: number): number[] {
  if (count <= 0) return [];
  if (count === 1) return [0];
  if (count === 2) return [0, 1];
  return [0, Math.floor((count - 1) / 2), count - 1];
}

/** The maximum over every finite value (0 when none). */
export function maxOf(values: ReadonlyArray<number | null>): number {
  let m = 0;
  for (const v of values) if (v !== null && Number.isFinite(v) && v > m) m = v;
  return m;
}
