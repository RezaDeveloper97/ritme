'use client';

import { useTranslations } from 'next-intl';

import type { PmddChart } from '@/entities/condition';
import { useDirection } from '@/shared/i18n';
import { Card, Skeleton } from '@/shared/ui';

import { chartGeometry, pmddNotes } from '../model/chart';
import './cpm.css';

const W = 300;
const H = 100;
const PAD = 8;

/**
 * «این سیکل» (nbl_Cond_PMDD): the daily mean of the six items over the current cycle with the period and the
 * last-10-days bands, then the pattern sentence. Day 1 sits at the reading start (right in RTL).
 */
export function PmddChartCard({ chart, loading }: { chart: PmddChart | undefined; loading: boolean }) {
  const t = useTranslations('conditions.pmdd');
  const rtl = useDirection() === 'rtl';

  if (loading) return <Skeleton shape="card" className="cpm-chart-skel" />;
  const cycle = chart?.cycles[0];
  const notes = pmddNotes(chart?.alerts ?? []);

  const sentence = [
    notes.pattern
      ? notes.pattern.body
        ? t('patternJoin', { title: notes.pattern.title ?? '', body: notes.pattern.body })
        : notes.pattern.title
      : null,
    notes.needsTwo?.title ?? (chart && !chart.ready ? t('needsTwo') : null),
  ]
    .filter(Boolean)
    .join(' ');

  let plot;
  if (!cycle) {
    plot = <p className="cpm-chart-empty">{t('noCycle')}</p>;
  } else {
    const g = chartGeometry(cycle);
    const x = (v: number) => (rtl ? 1 - v : v) * W;
    const y = (v: number) => H - PAD - v * (H - 2 * PAD);
    const band = (b: { x0: number; x1: number }) => {
      const a = x(b.x0);
      const z = x(b.x1);
      return { x: Math.min(a, z), width: Math.abs(z - a) };
    };
    plot = (
      <>
        <svg
          className="cpm-svg"
          viewBox={`0 0 ${W} ${H}`}
          preserveAspectRatio="none"
          role="img"
          aria-label={t('chartLabel', { days: g.points.length })}
        >
          {g.period ? <rect className="cpm-band is-period" y={0} height={H} {...band(g.period)} /> : null}
          {g.lateLuteal ? <rect className="cpm-band is-luteal" y={0} height={H} {...band(g.lateLuteal)} /> : null}
          {g.points.length > 1 ? (
            <polyline
              className="cpm-line"
              points={g.points.map((p) => `${x(p.x)},${y(p.y)}`).join(' ')}
              vectorEffect="non-scaling-stroke"
            />
          ) : null}
        </svg>
        <div className="cpm-axis" aria-hidden="true">
          <span>{t('axisPeriod')}</span>
          <span>{t('axisMid')}</span>
          <span>{t('axisLate')}</span>
        </div>
        {!g.points.length ? <p className="cpm-chart-empty">{t('chartEmpty')}</p> : null}
        <ul className="cpm-legend">
          <li>
            <span className="cpm-swatch is-period" aria-hidden="true" />
            {t('legendPeriod')}
          </li>
          <li>
            <span className="cpm-swatch is-luteal" aria-hidden="true" />
            {t('legendLate')}
          </li>
        </ul>
      </>
    );
  }

  return (
    <Card className="cpm-chart">
      {plot}
      {sentence ? <p className="cpm-pattern">{sentence}</p> : null}
    </Card>
  );
}
