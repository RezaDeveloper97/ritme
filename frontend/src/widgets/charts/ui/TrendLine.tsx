import type { Tone } from '@/shared/ui';

import { ChartFigure, type DataTable } from './ChartFigure';

interface TrendLineProps {
  label: string;
  table?: DataTable;
  /** Oldest first; null breaks the line. */
  values: ReadonlyArray<number | null>;
  tone?: Tone;
  height?: number;
}

const W = 320;
const PAD = 8;

/**
 * A single smoothed trend (An_Body weight, 7-day moving average): the line in
 * the data tone and a dot on the latest value. The y scale hugs the data with
 * a minimum span, so a 0.2 kg wobble doesn't look like a cliff.
 */
export function TrendLine({ label, table, values, tone = 'data', height = 110 }: TrendLineProps) {
  const finite = values.filter((v): v is number => v !== null && Number.isFinite(v));
  const lo = finite.length ? Math.min(...finite) : 0;
  const hi = finite.length ? Math.max(...finite) : 1;
  const span = Math.max(hi - lo, 1);
  const mid = (hi + lo) / 2;
  const min = mid - span / 2;
  const n = values.length;
  const x = (i: number) => (n <= 1 ? W / 2 : PAD + ((W - 2 * PAD) * i) / (n - 1));
  const y = (v: number) => PAD + (height - 2 * PAD) * (1 - (v - min) / span);
  let d = '';
  let pen = false;
  let last: { x: number; y: number } | null = null;
  values.forEach((v, i) => {
    if (v === null || !Number.isFinite(v)) {
      pen = false;
      return;
    }
    const px = Math.round(x(i) * 10) / 10;
    const py = Math.round(y(v) * 10) / 10;
    d += `${pen ? 'L' : 'M'}${px} ${py}`;
    pen = true;
    last = { x: px, y: py };
  });
  const end = last as { x: number; y: number } | null;
  return (
    <ChartFigure label={label} table={table}>
      <svg viewBox={`0 0 ${W} ${height}`} width="100%" className={`axc-svg nb-tone-${tone}`} aria-hidden focusable="false">
        <path className="axc-line" d={d} />
        {end ? <circle className="axc-line-end" cx={end.x} cy={end.y} r={5} /> : null}
      </svg>
    </ChartFigure>
  );
}
