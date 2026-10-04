'use client';

import { ChartFigure, type DataTable } from '@/widgets/charts';

import { EPDS_W, epdsGeometry, type EpdsTrend } from '../model/hub';

/**
 * EPDS scores over the weeks after the birth (An_Hub_Post «حال روحی»): the
 * likely band (≥ 13) on a period tint, the possible band (10–12) on a warm
 * tint, the cut-off values at the start, the line in brand with the checks
 * at or above the lower cut-off in warm.
 */
export function EpdsChart({
  trend,
  label,
  table,
  axis,
  ticks,
  height = 96,
}: {
  trend: EpdsTrend;
  label: string;
  table: DataTable;
  /** Localized x labels, one per point («هفته ۵»). */
  axis: readonly string[];
  /** Localized cut-off values (lower, upper). */
  ticks: { lower: string; upper: string | null };
  height?: number;
}) {
  const g = epdsGeometry(trend, height);
  const right = EPDS_W - 8;
  return (
    <ChartFigure label={label} table={table}>
      <svg viewBox={`0 0 ${EPDS_W} ${height + 20}`} width="100%" className="axc-svg" aria-hidden focusable="false">
        {g.upperY !== null ? (
          <>
            <rect className="anp-band-likely" x={g.padX} y={g.topY} width={right - g.padX} height={Math.max(0, g.upperY - g.topY)} />
            <rect className="anp-band-possible" x={g.padX} y={g.upperY} width={right - g.padX} height={Math.max(0, g.lowerY - g.upperY)} />
          </>
        ) : (
          <rect className="anp-band-possible" x={g.padX} y={g.topY} width={right - g.padX} height={Math.max(0, g.lowerY - g.topY)} />
        )}
        <line className="anp-cut" x1={g.padX} x2={right} y1={g.lowerY} y2={g.lowerY} />
        <text className="axc-axis" x={g.padX - 6} y={g.lowerY + 3} textAnchor="end">
          {ticks.lower}
        </text>
        {g.upperY !== null && ticks.upper ? (
          <text className="axc-axis" x={g.padX - 6} y={g.upperY + 3} textAnchor="end">
            {ticks.upper}
          </text>
        ) : null}
        <path className="axc-line nb-tone-brand" d={g.line} />
        {g.dots.map((d, i) => (
          <circle key={i} className={d.high ? 'anp-dot is-high' : 'anp-dot'} cx={d.x} cy={d.y} r={4.5} />
        ))}
        {g.dots.map((d, i) => (
          <text key={`l${i}`} className="axc-axis anp-week" x={d.x} y={height + 14} textAnchor="middle">
            {axis[i]}
          </text>
        ))}
      </svg>
    </ChartFigure>
  );
}
