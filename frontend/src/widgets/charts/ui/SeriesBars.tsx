import type { Tone } from '@/shared/ui';

import { heightShare, maxOf, slot } from '../lib/geometry';
import { ChartFigure, type DataTable } from './ChartFigure';

export interface BarSeries {
  key: string;
  tone: Tone;
  /** One value per bucket, oldest first. */
  values: readonly number[];
}

interface SeriesBarsProps {
  label: string;
  table?: DataTable;
  series: readonly BarSeries[];
  /** Labels under the first and the last bucket («۲۹ شهریور», «۱۲ مهر»). */
  from: string;
  to: string;
  className?: string;
}

const W = 320;
const LANE = 64;
const LANE_GAP = 14;

/**
 * Symptom trend (An_Symptoms): one lane of bars per selected symptom, buckets
 * left → right; an empty bucket is a short baseline dash. Each lane scales to
 * its own maximum — the lanes compare timing, not counts (the table has those).
 */
export function SeriesBars({ label, table, series, from, to, className }: SeriesBarsProps) {
  const buckets = Math.max(1, ...series.map((s) => s.values.length));
  const height = series.length * LANE + Math.max(0, series.length - 1) * LANE_GAP;
  const gap = buckets > 40 ? 1 : buckets > 20 ? 2 : 4;
  return (
    <ChartFigure label={label} table={table} className={className}>
      <svg viewBox={`0 0 ${W} ${Math.max(height, 1)}`} width="100%" className="axc-svg" aria-hidden focusable="false">
        {series.map((s, si) => {
          const top = si * (LANE + LANE_GAP);
          const max = maxOf(s.values);
          return (
            <g key={s.key} className={`nb-tone-${s.tone}`}>
              {s.values.map((v, i) => {
                const { x, w } = slot(i, buckets, W, gap);
                const h = heightShare(v, max, 0.15) * LANE;
                return h > 0 ? (
                  <rect key={i} className="axc-col is-solid" x={x} y={top + LANE - h} width={w} height={h} rx={Math.min(4, w / 3)} />
                ) : (
                  <rect key={i} className="axc-col-zero" x={x} y={top + LANE - 3} width={w} height={3} rx={1.5} />
                );
              })}
            </g>
          );
        })}
      </svg>
      <span className="axc-range-axis" aria-hidden>
        <span>{from}</span>
        <span>{to}</span>
      </span>
    </ChartFigure>
  );
}
