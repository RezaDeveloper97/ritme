import type { Tone } from '@/shared/ui';

import { groupedSlots, heightShare, maxOf } from '../lib/geometry';
import { ChartFigure, type DataTable } from './ChartFigure';

export interface ColumnGroup {
  key: string;
  /** Axis label under the group (localized, «روز ۷»). */
  label: string;
  /** One value per series, in `tones` order. */
  values: readonly number[];
}

interface GroupedColumnsProps {
  label: string;
  table?: DataTable;
  groups: readonly ColumnGroup[];
  /** Tone of each series (side by side inside a group). */
  tones: readonly Tone[];
  height?: number;
  className?: string;
}

const W = 320;
const AXIS = 22;

/**
 * Side-by-side columns per bucket (IVF follicle growth: 10–14 mm next to
 * ≥ 15 mm per scan day) — one shared scale, solid tones, oldest group on the
 * left in both locales. A zero value is a short baseline dash.
 */
export function GroupedColumns({ label, table, groups, tones, height = 150, className }: GroupedColumnsProps) {
  const plot = height - AXIS;
  const max = maxOf(groups.flatMap((g) => g.values));
  const slots = groupedSlots(groups.length, tones.length, W);
  return (
    <ChartFigure label={label} table={table} className={className}>
      <svg viewBox={`0 0 ${W} ${height}`} width="100%" className="axc-svg" aria-hidden focusable="false">
        {groups.map((g, gi) => {
          const group = slots[gi];
          if (!group) return null;
          return (
            <g key={g.key}>
              {tones.map((tone, si) => {
                const bar = group.bars[si];
                const h = heightShare(g.values[si] ?? 0, max, 0.06) * plot;
                if (!bar) return null;
                return (
                  <g key={si} className={`nb-tone-${tone}`}>
                    {h > 0 ? (
                      <rect className="axc-col is-solid" x={bar.x} y={plot - h} width={bar.w} height={h} rx={Math.min(5, bar.w / 3)} />
                    ) : (
                      <rect className="axc-col-zero" x={bar.x} y={plot - 3} width={bar.w} height={3} rx={1.5} />
                    )}
                  </g>
                );
              })}
              <text className="axc-axis" x={group.center} y={height - 5} textAnchor="middle">
                {g.label}
              </text>
            </g>
          );
        })}
      </svg>
    </ChartFigure>
  );
}
