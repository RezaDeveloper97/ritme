import { clsx } from 'clsx';

import type { Tone } from '@/shared/ui';

import { heightShare, slot } from '../lib/geometry';
import { ChartFigure, type DataTable } from './ChartFigure';

export interface Column {
  key: string;
  /** Axis label under the bar (localized). */
  label: string;
  value: number | null;
  tone?: Tone;
  /** Solid fill — the current / focus bar; the rest are the soft tint. */
  solid?: boolean;
  /** Drawn dashed and faint: left out of the median (an outlier cycle). */
  muted?: boolean;
}

interface ColumnChartProps {
  label: string;
  table?: DataTable;
  columns: readonly Column[];
  /** Top of the scale; defaults to the largest value. */
  max?: number;
  /** Share of the plot a tiny value still gets (cycle lengths use a high floor so 28 vs 31 stay comparable). */
  floor?: number;
  /** Bars start from this value instead of 0 (cycle lengths read from ~15 days). */
  baseline?: number;
  height?: number;
  gap?: number;
  className?: string;
}

const W = 320;
const AXIS = 22;

/**
 * Vertical bars (cycle lengths, correlation groups, sleep by weekday, active
 * days per week) — pure SVG on tokens, left → right in both locales. A null
 * value draws a short baseline dash (no data that slot).
 */
export function ColumnChart({
  label,
  table,
  columns,
  max,
  floor = 0.08,
  baseline = 0,
  height = 150,
  gap = 8,
  className,
}: ColumnChartProps) {
  const plot = height - AXIS;
  const top = (max ?? Math.max(0, ...columns.map((c) => c.value ?? 0))) - baseline;
  return (
    <ChartFigure label={label} table={table} className={className}>
      <svg viewBox={`0 0 ${W} ${height}`} width="100%" className="axc-svg" aria-hidden focusable="false">
        {columns.map((c, i) => {
          const { x, w } = slot(i, columns.length, W, gap);
          const share = c.value === null ? 0 : heightShare(c.value - baseline, top, floor);
          const h = Math.max(0, share * plot);
          return (
            <g key={c.key} className={`nb-tone-${c.tone ?? 'brand'}`}>
              {h > 0 ? (
                <rect
                  className={clsx('axc-col', c.solid && 'is-solid', c.muted && 'is-muted')}
                  x={x}
                  y={plot - h}
                  width={w}
                  height={h}
                  rx={Math.min(10, w / 3)}
                />
              ) : (
                <rect className="axc-col-zero" x={x + w * 0.2} y={plot - 3} width={w * 0.6} height={3} rx={1.5} />
              )}
              {c.label ? (
                <text className="axc-axis" x={x + w / 2} y={height - 5} textAnchor="middle">
                  {c.label}
                </text>
              ) : null}
            </g>
          );
        })}
      </svg>
    </ChartFigure>
  );
}
