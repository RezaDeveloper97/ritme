import { clsx } from 'clsx';

import { ChartFigure, type DataTable } from '@/widgets/charts';

import { dayX, linePath, tempScale, tempY, type ChartBox } from '../model/ttc';
import type { FertilityCycle } from '../model/types';

interface BbtChartProps {
  data: FertilityCycle;
  label: string;
  table: DataTable;
  /** Localized labels drawn in the picture (already translated / digit-formatted). */
  text: { ovulation: string; coverline: string; period: string; lh: string; sex: string };
  formatTemp: (v: number) => string;
}

const W = 320;
const BOX: ChartBox = { width: W, height: 230, left: 40, right: 310, top: 26, bottom: 156 };
const ROW_Y = { period: 176, lh: 194, sex: 212 };
const CELL = 7;

/**
 * The An_Fertility chart: one cycle's basal temperatures by cycle day (time runs
 * left → right in both directions), the fertile window band, the ovulation line,
 * the dashed coverline and the three high readings of a confirmed shift in the
 * data tone; under it the period, LH and intercourse rows. Pure SVG on tokens.
 */
export function BbtChart({ data, label, table, text, formatTemp }: BbtChartProps) {
  const span = Math.max(data.cycle.span, data.cycle.days, ...data.chart.points.map((p) => p.day));
  const values = data.chart.points.map((p) => p.value);
  if (data.chart.coverline != null) values.push(data.chart.coverline);
  const scale = tempScale(values);
  const step = (BOX.right - BOX.left) / Math.max(1, span - 1);
  const x = (day: number) => dayX(day, span, BOX);
  const high = new Set(data.chart.highDays);
  const fw = data.fertileWindow;
  const ov = data.ovulation.day;
  const cover = data.chart.coverline;

  return (
    <ChartFigure label={label} table={table}>
      <svg viewBox={`0 0 ${W} ${BOX.height}`} width="100%" className="axc-svg ttc-fx-svg" aria-hidden focusable="false">
        {scale.ticks.map((tick) => {
          const y = tempY(tick, scale, BOX);
          return (
            <g key={tick}>
              <line className="ttc-fx-grid" x1={BOX.left} x2={BOX.right} y1={y} y2={y} />
              <text className="ttc-fx-tick" x={BOX.left - 6} y={y + 3} textAnchor="end">
                {formatTemp(tick)}
              </text>
            </g>
          );
        })}
        {fw ? (
          <rect
            className="ttc-fx-band"
            x={x(fw.from) - step / 2}
            y={BOX.top - 8}
            width={(fw.to - fw.from + 1) * step}
            height={BOX.bottom - BOX.top + 16}
          />
        ) : null}
        {ov != null ? (
          <g>
            <line className="ttc-fx-ov" x1={x(ov)} x2={x(ov)} y1={BOX.top - 8} y2={BOX.bottom + 8} />
            <text className="ttc-fx-ov-label" x={x(ov)} y={BOX.top - 13} textAnchor="middle">
              {text.ovulation}
            </text>
          </g>
        ) : null}
        {cover != null ? (
          <g>
            <line
              className="ttc-fx-cover"
              x1={BOX.left}
              x2={BOX.right}
              y1={tempY(cover, scale, BOX)}
              y2={tempY(cover, scale, BOX)}
            />
            <text className="ttc-fx-cover-label" x={BOX.right} y={tempY(cover, scale, BOX) - 5} textAnchor="end">
              {text.coverline}
            </text>
          </g>
        ) : null}
        {data.chart.points.length > 1 ? <path className="ttc-fx-line" d={linePath(data.chart.points, span, scale, BOX)} /> : null}
        {data.chart.points.map((p) => (
          <circle
            key={p.day}
            className={clsx('ttc-fx-pt', high.has(p.day) && 'is-high')}
            cx={x(p.day)}
            cy={tempY(p.value, scale, BOX)}
            r={high.has(p.day) ? 4.5 : 2.6}
          />
        ))}

        <text className="ttc-fx-row" x={BOX.left - 6} y={ROW_Y.period + 4} textAnchor="end">
          {text.period}
        </text>
        {Array.from({ length: Math.min(data.periodDays, span) }, (_, i) => (
          <rect
            key={i}
            className="ttc-fx-cell is-period"
            x={x(i + 1) - CELL / 2}
            y={ROW_Y.period - CELL / 2}
            width={CELL}
            height={CELL}
            rx={2}
          />
        ))}
        <text className="ttc-fx-row" x={BOX.left - 6} y={ROW_Y.lh + 4} textAnchor="end">
          {text.lh}
        </text>
        {data.lh.tests.map((t) => (
          <rect
            key={t.day}
            className={clsx('ttc-fx-cell', `is-${t.value}`)}
            x={x(t.day) - CELL / 2}
            y={ROW_Y.lh - CELL / 2}
            width={CELL}
            height={CELL}
            rx={2}
          />
        ))}
        <text className="ttc-fx-row" x={BOX.left - 6} y={ROW_Y.sex + 4} textAnchor="end">
          {text.sex}
        </text>
        {data.intercourseDays.map((d) => (
          <circle key={d} className="ttc-fx-sex" cx={x(d)} cy={ROW_Y.sex} r={3.6} />
        ))}
      </svg>
    </ChartFigure>
  );
}
