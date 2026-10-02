'use client';

import { useLocale, useTranslations } from 'next-intl';

import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import { ChartFigure } from '@/widgets/charts';

import type { WeightGain } from '../api/schema';
import { GAIN_W, bandAt, gainChart } from '../model/chart';

const r1 = (n: number) => String(Math.round(n * 10) / 10);

/**
 * The weight-gain curve over the IOM band (An_Hub_Preg card, An_PregWeight):
 * turquoise band, rose line, a dot on the latest week. One accessible name
 * plus a hidden table of the weekly points.
 */
export function GainChartView({ data, height = 150 }: { data: WeightGain; height?: number }) {
  const t = useTranslations('analysisPregnancy.weight');
  const loc = useLocale() as Locale;
  const g = gainChart(data, height);
  const cur = data.current;
  const rec = cur?.recommended ?? (cur ? bandAt(data.band, cur.gaDays / 7) : null);
  const dec = (n: number) => formatDecimal(r1(n), loc);
  const label =
    cur && rec
      ? t('chart', { week: formatNumber(cur.week, loc), gain: dec(cur.gain), min: dec(rec.min), max: dec(rec.max) })
      : t('title');
  return (
    <ChartFigure
      label={label}
      table={{
        caption: t('title'),
        columns: [t('colWeek'), t('colGain')],
        rows: data.points.map((p) => [formatNumber(p.week, loc), dec(p.gain)]),
      }}
    >
      <svg viewBox={`0 0 ${GAIN_W} ${height}`} width="100%" className="axc-svg apg-gain" aria-hidden focusable="false">
        {g.band ? <path className="apg-gain-band" d={g.band} /> : null}
        {g.line ? <path className="axc-line nb-tone-bloom" d={g.line} /> : null}
        {g.end ? <circle className="axc-line-end nb-tone-bloom" cx={g.end.x} cy={g.end.y} r={5} /> : null}
        {g.ticks.map((tick) => (
          <text key={tick.week} className="axc-axis" x={tick.x} y={height - 4} textAnchor="middle">
            {formatNumber(tick.week, loc)}
          </text>
        ))}
      </svg>
    </ChartFigure>
  );
}
