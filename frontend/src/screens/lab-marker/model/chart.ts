/** Geometry of the marker trend chart (nbl_Lab_Marker «روند در آزمایش‌های تو»): oldest left → newest right. */
export interface TrendGeometry {
  width: number;
  height: number;
  points: { x: number; y: number; value: number }[];
  /** The reference band as y range (top < bottom), or null. */
  band: { top: number; bottom: number } | null;
  /** Gridline values with their y. */
  ticks: { value: number; y: number }[];
}

export const CHART_W = 320;
export const CHART_H = 150;
const PAD_X = 28;
const PAD_TOP = 22;
const PAD_BOTTOM = 14;

/** Scale hugging the values and the range (with 10 % headroom); a flat series still gets a span. */
export function trendGeometry(values: readonly number[], low: number | null, high: number | null): TrendGeometry {
  const all = [...values, ...(low !== null ? [low] : []), ...(high !== null ? [high] : [])].filter(Number.isFinite);
  let min = all.length ? Math.min(...all) : 0;
  let max = all.length ? Math.max(...all) : 1;
  if (max - min < 1e-9) {
    const d = Math.max(Math.abs(max) * 0.2, 1);
    min -= d;
    max += d;
  }
  const head = (max - min) * 0.1;
  // never dip below zero for values that can't be negative
  min = all.length && Math.min(...all) >= 0 ? Math.max(0, min - head) : min - head;
  max += head;
  const y = (v: number) => PAD_TOP + (CHART_H - PAD_TOP - PAD_BOTTOM) * (1 - (v - min) / (max - min));
  const n = values.length;
  const x = (i: number) => (n <= 1 ? CHART_W / 2 : PAD_X + ((CHART_W - 2 * PAD_X) * i) / (n - 1));
  const round = (v: number) => Math.round(v * 10) / 10;
  const tickValues = [...new Set([low, high].filter((v): v is number => v !== null))];
  return {
    width: CHART_W,
    height: CHART_H,
    points: values.map((v, i) => ({ x: round(x(i)), y: round(y(v)), value: v })),
    band:
      low !== null || high !== null
        ? { top: round(y(high ?? max)), bottom: round(y(low ?? min)) }
        : null,
    ticks: tickValues.map((v) => ({ value: v, y: round(y(v)) })),
  };
}
