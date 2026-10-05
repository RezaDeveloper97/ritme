'use client';

import { useLocale } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { ChartFigure, type DataTable } from '@/widgets/charts';

import {
  bandPath,
  buildScale,
  CHART_H,
  CHART_W,
  linePath,
  medianPath,
  PLOT,
  pointCoords,
  visibleToMonth,
} from '../model/chart';
import type { GrowthSeries } from '../model/types';

interface Props {
  series: GrowthSeries;
  /** Accessible name of the picture. */
  label: string;
  /** Column titles of the visually hidden table (caption, date, age, value, percentile). */
  table: { caption: string; date: string; age: string; value: string; percentile: string };
}

/**
 * v16_Growth chart: the WHO P3–P97 band (data-soft), the median as a dashed
 * line, the child's values as a line with dots — months on x, always left →
 * right. One `role="img"` with a name + a visually hidden table of the values.
 * Without reference curves (sex unknown) only the child's points are drawn.
 */
export function GrowthChart({ series, label, table }: Props) {
  const locale = useLocale() as Locale;
  const reference = series.available ? series.reference : [];
  const s = buildScale(series.fromMonth, visibleToMonth(series), reference, series.points);
  const band = bandPath(reference, s);
  const median = medianPath(reference, s);
  const pts = pointCoords(series.points, s);
  const line = linePath(pts);

  const data: DataTable = {
    caption: table.caption,
    columns: [table.date, table.age, table.value, table.percentile],
    rows: pts.map(({ point }) => [
      formatLongDate(fromApiDate(point.measuredOn), locale),
      formatDecimal(point.ageMonths, locale),
      formatDecimal(point.value, locale),
      point.percentile === null ? '—' : formatNumber(Math.round(point.percentile), locale),
    ]),
  };

  return (
    <ChartFigure label={label} table={data} className="cgr-chart">
      <svg viewBox={`0 0 ${CHART_W} ${CHART_H}`} width="100%" className="axc-svg cgr-svg" aria-hidden focusable="false">
        {band ? <path className="cgr-band" d={band} /> : null}
        {s.yTicks.map((v) => (
          <g key={`y${v}`}>
            <line className="cgr-grid" x1={PLOT.left} x2={CHART_W} y1={s.y(v)} y2={s.y(v)} />
            <text className="cgr-tick" x={PLOT.left - 6} y={s.y(v) + 3} textAnchor="end">
              {formatDecimal(v, locale)}
            </text>
          </g>
        ))}
        {median ? <path className="cgr-median" d={median} /> : null}
        {line ? <path className="cgr-line" d={line} /> : null}
        {pts.map(({ x, y, point }) => (
          <circle key={`${point.measuredOn}-${point.id ?? 'b'}`} className="cgr-dot" cx={x} cy={y} r={4} />
        ))}
        {s.xTicks.map((m) => (
          <text key={`x${m}`} className="cgr-tick" x={s.x(m)} y={CHART_H - 4} textAnchor="middle">
            {formatNumber(m, locale)}
          </text>
        ))}
      </svg>
    </ChartFigure>
  );
}
