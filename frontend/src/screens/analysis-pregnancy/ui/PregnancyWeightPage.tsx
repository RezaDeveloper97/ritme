'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useRouter, type Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, InfoNote, PrimaryButton } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ChartCard, ReportFrame, StatTiles } from '@/widgets/charts';

import { isPregnancyInactive, usePregnancyAnalysis } from '../api/queries';
import { isUserRow } from '../model/chart';
import { GainChartView } from './GainChartView';

/**
 * `/analysis/pregnancy-weight` (An_PregWeight, B-N3-12): gain so far, the
 * total IOM target and the last 4 weeks; the weekly curve over the
 * recommended band; the IOM 2009 table with the user's BMI row highlighted;
 * the starting-weight source and the singleton footnote.
 */
export function PregnancyWeightPage() {
  const t = useTranslations('analysisPregnancy.weight');
  const tHub = useTranslations('analysisPregnancy.hub');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const query = usePregnancyAnalysis();
  const r = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const dec = (n: number) => formatDecimal(String(Math.round(n * 10) / 10), loc);
  const signed = (n: number) => {
    const v = Math.round(n * 10) / 10;
    return `${v > 0 ? '+' : v < 0 ? '−' : ''}${dec(Math.abs(v))}`;
  };
  const inactive = query.isError && isPregnancyInactive(query.error);
  const status =
    !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : (query.isError && !inactive) || (!r && !inactive) ? 'error' : 'ready';

  let content = null;
  if (inactive) {
    content = (
      <EmptyState
        icon="heart"
        title={tHub('inactive.title')}
        body={tHub('inactive.body')}
        action={
          <PrimaryButton block={false} onClick={() => router.push('/pregnancy/setup')}>
            {tHub('inactive.cta')}
          </PrimaryButton>
        }
      />
    );
  } else if (r) {
    const s = r.sections.weightGain;
    const w = s.data;
    content = (
      <>
        <StatTiles
          label={t('tilesLabel')}
          tiles={[
            {
              key: 'toDate',
              label: t('toDate'),
              value: w?.current ? signed(w.current.gain) : tHub('none'),
              sub: t('unit'),
              tone: 'bloom',
            },
            {
              key: 'target',
              label: t('target'),
              value: w?.target ? t('range', { min: dec(w.target.min), max: dec(w.target.max) }) : tHub('none'),
              sub: t('unitIom'),
              tone: 'data',
            },
            {
              key: 'recent',
              label: t('recent'),
              value: w?.recent4w != null ? signed(w.recent4w) : tHub('none'),
              sub: t('unit'),
              tone: 'brand',
            },
          ]}
        />

        <ChartCard className="apg-chart-card">
          {s.ready && w ? (
            <>
              <GainChartView data={w} height={170} />
              <ul className="axc-legend apg-legend">
                <li className="nb-tone-bloom">
                  <span className="axc-legend-dot" aria-hidden />
                  {t('legendYou')}
                </li>
                <li className="nb-tone-data">
                  <span className="apg-legend-bar" aria-hidden />
                  {t('legendBand')}
                </li>
              </ul>
            </>
          ) : (
            <p className="anr-note">{tHub(`weight.missing.${w?.missing ?? 'weights'}`)}</p>
          )}
        </ChartCard>

        {w?.iomTable.length ? (
          <section className="nb-card apg-table-card" aria-label={t('tableGain')}>
            <table className="apg-table">
              <thead>
                <tr>
                  <th scope="col">{t('tableBmi')}</th>
                  <th scope="col">{t('tableGain')}</th>
                </tr>
              </thead>
              <tbody>
                {w.iomTable.map((row) => {
                  const mine = isUserRow(row, w.bmiCategory);
                  const bmi =
                    row.bmiMin == null && row.bmiMax != null
                      ? t('bmiUnder', { n: dec(row.bmiMax + 0.1) })
                      : row.bmiMax == null && row.bmiMin != null
                        ? t('bmiOver', { n: dec(row.bmiMin) })
                        : t('bmiRange', { min: dec(row.bmiMin ?? 0), max: dec(row.bmiMax ?? 0) });
                  return (
                    <tr key={row.category} className={clsx(mine && 'is-mine')} aria-current={mine ? 'true' : undefined}>
                      <th scope="row">
                        {bmi}
                        {mine ? <span className="sr-only"> ({t('yours')})</span> : null}
                      </th>
                      <td>{t('gainRange', { min: dec(row.gainMin), max: dec(row.gainMax) })}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </section>
        ) : null}

        {w?.baseline?.source ? (
          <p className="anr-note apg-baseline">{t(`baseline.${w.baseline.source}`, { w: dec(w.baseline.weight) })}</p>
        ) : null}
        <InfoNote>{t('footnote')}</InfoNote>
      </>
    );
  }

  return (
    <ReportFrame
      title={t('title')}
      subtitle={r ? t('sub', { n: num(r.pregnancy.week) }) : undefined}
      backLabel={t('back')}
      onBack={() => router.push('/analysis')}
      status={status}
      loadingLabel={t('loading')}
      error={{
        title: tHub('error.title'),
        body: tHub('error.body'),
        retry: tHub('error.retry'),
        onRetry: () => void query.refetch(),
        retrying: query.isFetching,
      }}
      after={<BottomNav />}
    >
      {content}
    </ReportFrame>
  );
}
