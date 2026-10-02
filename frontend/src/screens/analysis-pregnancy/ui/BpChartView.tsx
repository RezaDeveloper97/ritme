'use client';

import { ChartFigure, type DataTable } from '@/widgets/charts';

import { GAIN_W, bpChart } from '../model/chart';

/** Systolic line with the 140 mmHg threshold dashed across (An_Hub_Preg «فشار خون»). */
export function BpChartView({
  readings,
  threshold,
  label,
  table,
  height = 90,
}: {
  readings: ReadonlyArray<{ systolic: number; high: boolean }>;
  threshold: number;
  label: string;
  table: DataTable;
  height?: number;
}) {
  const g = bpChart(readings, threshold, height);
  return (
    <ChartFigure label={label} table={table}>
      <svg viewBox={`0 0 ${GAIN_W} ${height}`} width="100%" className="axc-svg" aria-hidden focusable="false">
        <line className="apg-bp-threshold" x1={0} x2={GAIN_W} y1={g.thresholdY} y2={g.thresholdY} />
        <path className="axc-line nb-tone-brand" d={g.line} />
        {g.highs.map((p) => (
          <circle key={`${p.x}-${p.y}`} className="apg-bp-high" cx={p.x} cy={p.y} r={4} />
        ))}
        {g.end ? <circle className="axc-line-end nb-tone-brand" cx={g.end.x} cy={g.end.y} r={5} /> : null}
      </svg>
    </ChartFigure>
  );
}
