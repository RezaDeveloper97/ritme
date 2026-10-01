import { ChartFigure, type DataTable } from './ChartFigure';

/** Intensity step of a cell (1 = lightest … 4 = strongest), null = nothing logged. */
export type HeatLevel = 1 | 2 | 3 | 4 | null;

export interface HeatRow {
  key: string;
  label: string;
  cells: readonly HeatLevel[];
}

interface HeatGridProps {
  label: string;
  table?: DataTable;
  rows: readonly HeatRow[];
  /** Column labels under the grid («روز ۱», «۲», …), one per column. */
  columns: readonly string[];
  /** Legend, one label per level 1–4. */
  legend: readonly [string, string, string, string];
}

const CELL_H = 24;
const GAP = 4;
const W = 300;

/**
 * Flow by period day (An_Period «شدت هر روز»): one row per period plus the
 * average, columns = day 1… at the left, four red steps (`--period` ramp).
 */
export function HeatGrid({ label, table, rows, columns, legend }: HeatGridProps) {
  const cols = Math.max(1, columns.length);
  const cw = (W - GAP * (cols - 1)) / cols;
  return (
    <div className="axc-heat">
      <ChartFigure label={label} table={table}>
        <div className="axc-heat-grid" aria-hidden>
          {rows.map((row) => (
            <div key={row.key} className="axc-heat-row">
              <span className="axc-heat-label">{row.label}</span>
              <svg viewBox={`0 0 ${W} ${CELL_H}`} width="100%" className="axc-svg" focusable="false">
                {Array.from({ length: cols }, (_, i) => {
                  const level = row.cells[i] ?? null;
                  return level === null ? null : (
                    <rect
                      key={i}
                      className={`axc-heat-cell is-l${level}`}
                      x={i * (cw + GAP)}
                      y={0}
                      width={cw}
                      height={CELL_H}
                      rx={6}
                    />
                  );
                })}
              </svg>
            </div>
          ))}
          <div className="axc-heat-row is-axis">
            <span className="axc-heat-label" />
            <svg viewBox={`0 0 ${W} 14`} width="100%" className="axc-svg" focusable="false">
              {columns.map((c, i) => (
                <text key={i} className="axc-axis" x={i * (cw + GAP) + cw / 2} y={11} textAnchor="middle">
                  {c}
                </text>
              ))}
            </svg>
          </div>
        </div>
      </ChartFigure>
      <ul className="axc-legend is-heat" aria-hidden>
        {legend.map((l, i) => (
          <li key={l}>
            <span className={`axc-legend-dot axc-heat-cell is-l${i + 1}`} />
            {l}
          </li>
        ))}
      </ul>
    </div>
  );
}
