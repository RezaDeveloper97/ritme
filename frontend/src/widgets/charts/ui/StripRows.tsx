import type { Tone } from '@/shared/ui';

import { axisTicks, cellOpacity } from '../lib/geometry';
import { ChartFigure, type DataTable } from './ChartFigure';

export interface StripRow {
  key: string;
  label: string;
  tone: Tone;
  /** Share (0–1) per day of the typical cycle, day 1 first. */
  values: readonly number[];
}

interface StripRowsProps {
  label: string;
  table?: DataTable;
  rows: readonly StripRow[];
  /** Axis label per day index (`dayLabel(1)` → «روز ۱»); only first / middle / last are drawn. */
  dayLabel: (day: number) => string;
}

const W = 300;
const H = 14;
const GAP = 2;

/**
 * Symptom pattern on the typical cycle (An_Symptoms «روی سیکل معمول»): one
 * row of day cells per symptom, darker = logged on that day in more cycles.
 */
export function StripRows({ label, table, rows, dayLabel }: StripRowsProps) {
  const days = Math.max(1, ...rows.map((r) => r.values.length));
  const cw = (W - GAP * (days - 1)) / days;
  return (
    <ChartFigure label={label} table={table}>
      <div className="axc-strips" aria-hidden>
        {rows.map((row) => (
          <div key={row.key} className={`axc-strip-row nb-tone-${row.tone}`}>
            <span className="axc-strip-label">{row.label}</span>
            <svg viewBox={`0 0 ${W} ${H}`} width="100%" className="axc-svg" focusable="false">
              {row.values.map((v, i) => (
                <rect key={i} className="axc-strip-cell" x={i * (cw + GAP)} y={0} width={cw} height={H} rx={3} opacity={cellOpacity(v)} />
              ))}
            </svg>
          </div>
        ))}
        <div className="axc-strip-row is-axis">
          <span className="axc-strip-label" />
          <svg viewBox={`0 0 ${W} 14`} width="100%" className="axc-svg" focusable="false">
            {axisTicks(days).map((i) => (
              <text
                key={i}
                className="axc-axis"
                x={i * (cw + GAP) + cw / 2}
                y={11}
                textAnchor={i === 0 ? 'start' : i === days - 1 ? 'end' : 'middle'}
              >
                {dayLabel(i + 1)}
              </text>
            ))}
          </svg>
        </div>
      </div>
    </ChartFigure>
  );
}
