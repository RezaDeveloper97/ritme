'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatDecimal } from '@/shared/lib/date';
import type { Locale } from '@/shared/i18n';
import { type TrendPoint, stateTone, trimNumber } from '@/entities/lab';
import { ChartFigure } from '@/widgets/charts';

import { trendGeometry } from '../model/chart';

interface TrendChartProps {
  points: readonly TrendPoint[];
  low: number | null;
  high: number | null;
  /** Localized date label per point («مهر ۰۵»). */
  dateLabel: (date: string) => string;
}

/**
 * The marker's values across the user's labs: the reference band, the
 * dashed line in the latest state's tone, every value labelled, dates below.
 * A `role=img` figure with a hidden data table (widgets/charts).
 */
export function TrendChart({ points, low, high, dateLabel }: TrendChartProps) {
  const t = useTranslations('labs.marker');
  const locale = useLocale() as Locale;
  const g = trendGeometry(points.map((p) => p.value), low, high);
  const last = points[points.length - 1];
  const tone = last ? stateTone(last.state) : 'data';
  const fmt = (v: number) => formatDecimal(trimNumber(v), locale);
  const d = g.points.map((p, i) => `${i ? 'L' : 'M'}${p.x} ${p.y}`).join('');
  return (
    <ChartFigure
      label={t('chartLabel', { values: points.map((p) => fmt(p.value)).join('، ') })}
      table={{
        caption: t('chartCaption'),
        columns: [t('chartDate'), t('chartValue')],
        rows: points.map((p) => [dateLabel(p.date), fmt(p.value)]),
      }}
    >
      <svg viewBox={`0 0 ${g.width} ${g.height + 22}`} width="100%" className={`axc-svg lab-trend nb-tone-${tone}`} aria-hidden focusable="false">
        {g.band ? <rect className="lab-trend-band" x={0} y={g.band.top} width={g.width} height={Math.max(0, g.band.bottom - g.band.top)} /> : null}
        {g.ticks.map((tk) => (
          <g key={tk.value}>
            <line className="lab-trend-grid" x1={0} x2={g.width} y1={tk.y} y2={tk.y} />
            <text className="lab-trend-tick" x={2} y={tk.y - 3}>
              {fmt(tk.value)}
            </text>
          </g>
        ))}
        {g.points.length > 1 ? <path className="lab-trend-line" d={d} /> : null}
        {g.points.map((p, i) => (
          <g key={i}>
            <circle className="lab-trend-dot" cx={p.x} cy={p.y} r={6} />
            <text className="lab-trend-value" x={p.x} y={p.y - 10} textAnchor="middle">
              {fmt(p.value)}
            </text>
            <text className="lab-trend-date" x={p.x} y={g.height + 16} textAnchor="middle">
              {dateLabel(points[i]!.date)}
            </text>
          </g>
        ))}
      </svg>
    </ChartFigure>
  );
}
