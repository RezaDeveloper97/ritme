'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { REPORT_DEFAULT_RANGE, REPORT_RANGES, usePeriodReport } from '@/entities/analysis';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { EmptyState, InfoNote, PrimaryButton, type Tone } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ChartCard, HBarList, HeatGrid, ReportFrame, StatTiles, type HeatRow, type StatTile } from '@/widgets/charts';

import { flowColumns, hasFlow, LEVELS, levelOf } from '../model/period';

type Range = (typeof REPORT_RANGES.period)[number];

const CO_TONES: readonly Tone[] = ['bloom', 'warm', 'brand', 'data', 'period'];

/**
 * `/analysis/period` (An_Period, B-N3-09): median period length (FIGO ≤ 8),
 * peak day, spotting, flow by day for the recent periods + the average, the
 * symptoms logged alongside, and the FIGO 2018 «talk to a doctor» footnote.
 */
export function AnalysisPeriodPage() {
  const t = useTranslations('analysis.reports');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const [range, setRange] = useState<Range>(REPORT_DEFAULT_RANGE.period);
  const query = usePeriodReport(range);
  const r = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const month = (iso: string) => monthName(toParts(fromApiDate(iso), loc).month, loc);
  const status = !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : query.isError || !r ? 'error' : 'ready';

  let content = null;
  if (r) {
    const median = r.periodLength.median;
    const tiles: StatTile[] = [
      {
        key: 'length',
        label: t('period.tiles.length'),
        value: median != null ? num(median) : t('none'),
        sub: r.periodLength.status ? t(`period.verdict.${r.periodLength.status}`, { max: num(r.periodMax) }) : undefined,
        tone: r.periodLength.status === 'prolonged' ? 'warm' : 'data',
      },
      {
        key: 'peak',
        label: t('period.tiles.peak'),
        value: r.peak ? num(r.peak.day) : t('none'),
        sub: r.peak ? t('period.peakSub') : undefined,
        tone: 'period',
      },
      {
        key: 'spotting',
        label: t('period.tiles.spotting'),
        value: num(r.spotting.days),
        sub: r.spotting.ofCycles > 0 ? t('period.spottingSub', { n: num(r.spotting.ofCycles) }) : undefined,
        tone: 'warm',
      },
    ];
    const cols = flowColumns(r);
    const levelName = (l: (typeof LEVELS)[number] | null) => (l ? t(`period.levels.${l}`) : t('none'));
    const rows: HeatRow[] = [
      { key: 'avg', label: t('period.flow.average'), cells: r.average.map((a) => levelOf(a.level)) },
      ...r.periods.map((p) => ({ key: p.start, label: month(p.start), cells: p.days.map((d) => levelOf(d.flow)) })),
    ];
    const dayCols = Array.from({ length: cols }, (_, i) => (i === 0 ? t('period.flow.day1', { n: num(1) }) : num(i + 1)));
    const co = r.coSymptoms;

    content =
      r.periods.length === 0 ? (
        <EmptyState
          icon="drop"
          title={t('empty.title')}
          body={t('empty.body')}
          action={
            <PrimaryButton icon="plus" block={false} onClick={() => openSheet('log')}>
              {t('empty.cta')}
            </PrimaryButton>
          }
        />
      ) : (
        <>
          <StatTiles label={t('period.tilesLabel')} tiles={tiles} />
          {!r.ready ? <p className="anr-note">{t('period.notReady')}</p> : null}

          <ChartCard title={t('period.flow.title')}>
            {hasFlow(r) && cols > 0 ? (
              <HeatGrid
                label={t('period.flow.chart', { n: num(r.periods.length) })}
                table={{
                  caption: t('period.flow.title'),
                  columns: [t('period.flow.colPeriod'), ...dayCols],
                  rows: [
                    [t('period.flow.average'), ...Array.from({ length: cols }, (_, i) => levelName(r.average[i]?.level ?? null))],
                    ...r.periods.map((p) => [
                      month(p.start),
                      ...Array.from({ length: cols }, (_, i) => levelName(p.days[i]?.flow ?? null)),
                    ]),
                  ],
                }}
                rows={rows}
                columns={dayCols}
                legend={[t('period.levels.light'), t('period.levels.medium'), t('period.levels.heavy'), t('period.levels.very_heavy')]}
              />
            ) : (
              <p className="anr-note">{t('period.flow.noFlow')}</p>
            )}
          </ChartCard>

          <ChartCard title={t('period.co.title')}>
            {!co.ready ? (
              <p className="anr-note">{t('period.co.notReady', { n: num(co.periodsNeeded) })}</p>
            ) : co.items.length ? (
              <HBarList
                label={t('period.co.label')}
                bars={co.items.map((s, i) => ({
                  key: s.key,
                  label: s.label,
                  pct: s.pct,
                  valueLabel: t('period.co.pct', { n: num(s.pct) }),
                  tone: CO_TONES[i % CO_TONES.length],
                }))}
              />
            ) : (
              <p className="anr-note">{t('period.co.none')}</p>
            )}
          </ChartCard>

          <InfoNote>{t('period.figo', { max: num(r.periodMax) })}</InfoNote>
        </>
      );
  }

  return (
    <ReportFrame
      title={t('period.title')}
      subtitle={r && r.basedOnPeriods > 0 ? t('period.sub', { n: num(r.basedOnPeriods) }) : t('recent', { range: t(`ranges.${range}`) })}
      backLabel={t('back')}
      onBack={() => router.push('/analysis')}
      tabs={{
        label: t('rangeLabel'),
        value: range,
        options: REPORT_RANGES.period.map((v) => ({ value: v, label: t(`ranges.${v}`) })),
        onChange: setRange,
      }}
      status={status}
      updating={query.isPlaceholderData && query.isFetching}
      loadingLabel={t('loading')}
      error={{
        title: t('error.title'),
        body: t('error.body'),
        retry: t('error.retry'),
        onRetry: () => void query.refetch(),
        retrying: query.isFetching,
      }}
      after={<BottomNav />}
    >
      {content}
    </ReportFrame>
  );
}
