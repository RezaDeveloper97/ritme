import { clsx } from 'clsx';

import { ChartFigure, type DataTable } from './ChartFigure';

export interface CycleDotRow {
  key: string;
  /** Row label at the start («شهریور»). */
  label: string;
  length: number;
  periodDays: number;
  /** 0 = no fertile window drawn (teen / menopause). */
  fertileStart: number;
  fertileEnd: number;
  ovulationDay: number;
  /** Number printed at the end of the row («۲۹»). */
  lengthLabel: string;
  /** Faint row: not counted in the median. */
  muted?: boolean;
}

const STEP = 11;
const R = 4;

function kindOf(row: CycleDotRow, day: number): 'period' | 'ovulation' | 'fertile' | 'other' {
  if (day <= row.periodDays) return 'period';
  if (row.ovulationDay > 0 && day === row.ovulationDay) return 'ovulation';
  if (row.fertileStart > 0 && day >= row.fertileStart && day <= row.fertileEnd) return 'fertile';
  return 'other';
}

/**
 * Cycle history as rows of day dots (An_Cycle «تاریخچه»): period red, fertile
 * amber, ovulation turquoise, day 1 at the left. All rows share one scale (the
 * longest cycle), so lengths compare at a glance.
 */
export function CycleDots({ label, table, rows }: { label: string; table?: DataTable; rows: readonly CycleDotRow[] }) {
  const longest = Math.max(1, ...rows.map((r) => r.length));
  const width = longest * STEP;
  return (
    <ChartFigure label={label} table={table}>
      <ul className="axc-dots" aria-hidden>
        {rows.map((row) => (
          <li key={row.key} className={clsx('axc-dots-row', row.muted && 'is-muted')}>
            <span className="axc-dots-label">{row.label}</span>
            <svg viewBox={`0 0 ${width} ${STEP}`} width="100%" className="axc-svg axc-dots-svg" focusable="false">
              {Array.from({ length: row.length }, (_, i) => (
                <circle key={i} className={`axc-dot is-${kindOf(row, i + 1)}`} cx={i * STEP + STEP / 2} cy={STEP / 2} r={R} />
              ))}
            </svg>
            <span className="axc-dots-len">{row.lengthLabel}</span>
          </li>
        ))}
      </ul>
    </ChartFigure>
  );
}
