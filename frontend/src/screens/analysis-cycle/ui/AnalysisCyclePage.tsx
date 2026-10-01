'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { REPORT_DEFAULT_RANGE, REPORT_RANGES, useCycleReport } from '@/entities/analysis';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { EmptyState, InfoNote, PrimaryButton, type Tone } from '@/shared/ui';
import { BottomNav, useNavMode } from '@/widgets/bottom-nav';
import { ChartCard, ColumnChart, CycleDots, PhaseBar, ReportFrame, StatTiles, type StatTile } from '@/widgets/charts';

import { chronological, cycleVerdict, dayLayout, excludedCycles, MIN_CYCLES, periodVerdict, variationVerdict, type Verdict } from '../model/cycle';

type Range = (typeof REPORT_RANGES.cycle)[number];

const TONE: Record<Verdict, Tone> = { good: 'data', warn: 'warm', none: 'neutral' };

/**
 * `/analysis/cycle` (An_Cycle, B-N3-09): median cycle / period / variation
 * tiles with their FIGO verdicts, the length of each cycle, the typical phase
 * split, the cycle history as day dots, and the FIGO footnote.
 */
export function AnalysisCyclePage() {
  const t = useTranslations('analysis.reports');
  const tA = useTranslations('analysis');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const { mode } = useNavMode();
  const [range, setRange] = useState<Range>(REPORT_DEFAULT_RANGE.cycle);
  const query = useCycleReport(range);
  const r = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const month = (iso: string) => monthName(toParts(fromApiDate(iso), loc).month, loc);
  const abbr = Number(tA('hub.cycle.monthAbbrLength')) || 3;
  // Teens and menopause see period days only (no fertility content, B-N2-03).
  const hideFertility = mode === 'teen' || mode === 'menopause';

  const status = !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : query.isError || !r ? 'error' : 'ready';

  let content = null;
  if (r) {
    const figo = r.figo;
    const cv = cycleVerdict(r);
    const pv = periodVerdict(r);
    const vv = variationVerdict(r);
    const tiles: StatTile[] = [
      {
        key: 'cycle',
        label: t('cycle.tiles.cycle'),
        value: r.cycleLength.median != null ? num(r.cycleLength.median) : t('none'),
        sub:
          cv === 'none'
            ? r.cycleLength.median != null
              ? t('cycle.verdict.unrated')
              : undefined
            : t(`cycle.verdict.${r.cycleLength.status ?? 'normal'}`, { min: num(figo.cycleMin), max: num(figo.cycleMax) }),
        tone: TONE[cv],
      },
      {
        key: 'period',
        label: t('cycle.tiles.period'),
        value: r.periodLength.median != null ? num(r.periodLength.median) : t('none'),
        sub:
          pv === 'none'
            ? undefined
            : r.periodLength.status === 'prolonged'
              ? t('cycle.verdict.prolonged', { max: num(figo.periodMax) })
              : t('cycle.verdict.periodNormal', { max: num(figo.periodMax) }),
        tone: TONE[pv],
      },
      {
        key: 'variation',
        label: t('cycle.tiles.variation'),
        value: r.variation.days != null ? num(r.variation.days) : t('none'),
        sub:
          vv === 'none'
            ? t('cycle.verdict.needMore', { n: num(Math.max(1, r.variation.cyclesNeeded - r.basedOnCycles)) })
            : t(`cycle.verdict.${r.variation.status === 'regular' ? 'regular' : 'irregular'}`, { max: num(r.variation.max) }),
        tone: TONE[vv],
      },
    ];
    const bars = chronological(r.cycles);
    const excluded = excludedCycles(r.cycles);
    const typ = r.typical;
    const phaseParts = [
      { key: 'period', label: t('cycle.phases.legend', { phase: tA('phases.period'), days: num(typ.days.period) }), days: typ.days.period, tone: 'period' as const },
      { key: 'follicular', label: t('cycle.phases.legend', { phase: tA('phases.follicular'), days: num(typ.days.follicular) }), days: typ.days.follicular, tone: 'track' as const },
      { key: 'fertile', label: t('cycle.phases.legend', { phase: tA('phases.fertile'), days: num(typ.days.fertile) }), days: typ.days.fertile, tone: 'warm' as const },
      { key: 'luteal', label: t('cycle.phases.legend', { phase: tA('phases.luteal'), days: num(typ.days.luteal) }), days: typ.days.luteal, tone: 'brand' as const },
    ];

    content =
      r.cycles.length === 0 && !r.current ? (
        <EmptyState
          icon="chart"
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
          <StatTiles label={t('cycle.tilesLabel')} tiles={tiles} />
          {!r.ready ? <p className="anr-note">{t('cycle.notReady', { n: num(MIN_CYCLES) })}</p> : null}

          {bars.length ? (
            <ChartCard title={t('cycle.lengths.title')}>
              <ColumnChart
                label={t('cycle.lengths.chart', { n: num(bars.length), list: bars.map((c) => num(c.length)).join('، ') })}
                table={{
                  caption: t('cycle.lengths.title'),
                  columns: [t('cycle.lengths.colCycle'), t('cycle.lengths.colLength')],
                  rows: bars.map((c) => [
                    month(c.start),
                    c.counted ? num(c.length) : `${num(c.length)} (${t('cycle.lengths.excludedTag')})`,
                  ]),
                }}
                columns={bars.map((c, i) => ({
                  key: c.start,
                  label: month(c.start).slice(0, abbr),
                  value: c.length,
                  tone: 'brand',
                  solid: i === bars.length - 1,
                  muted: !c.counted,
                }))}
                baseline={14}
                floor={0.2}
                height={170}
              />
              {excluded.map((c) => (
                <p key={c.start} className="anr-note">
                  {t(`cycle.lengths.${c.excluded === 'implausible' ? 'implausible' : 'outlier'}`, {
                    month: month(c.start),
                    days: num(c.length),
                  })}
                </p>
              ))}
            </ChartCard>
          ) : null}

          {!hideFertility && r.ready ? (
            <ChartCard title={t('cycle.phases.title')}>
              <PhaseBar
                label={t('cycle.phases.chart', {
                  length: num(typ.cycleLength),
                  period: num(typ.days.period),
                  follicular: num(typ.days.follicular),
                  fertile: num(typ.days.fertile),
                  luteal: num(typ.days.luteal),
                })}
                parts={phaseParts}
              />
            </ChartCard>
          ) : null}

          {r.cycles.length ? (
            <ChartCard title={t('cycle.history.title')}>
              <CycleDots
                label={t('cycle.history.chart', { n: num(r.cycles.length) })}
                table={{
                  caption: t('cycle.history.title'),
                  columns: [t('cycle.history.colMonth'), t('cycle.history.colLength'), t('cycle.history.colPeriod')],
                  rows: r.cycles.map((c) => [month(c.start), num(c.length), num(c.periodDays)]),
                }}
                rows={r.cycles.map((c) => {
                  const l = dayLayout(c.length, c.periodDays);
                  return {
                    key: c.start,
                    label: month(c.start),
                    length: c.length,
                    periodDays: l.periodDays,
                    fertileStart: hideFertility ? 0 : l.fertileStart,
                    fertileEnd: hideFertility ? 0 : l.fertileEnd,
                    ovulationDay: hideFertility ? 0 : l.ovulationDay,
                    lengthLabel: num(c.length),
                    muted: !c.counted,
                  };
                })}
              />
            </ChartCard>
          ) : null}

          <InfoNote>{figo.applies ? t('cycle.figo') : t('cycle.figoNotApplied')}</InfoNote>
        </>
      );
  }

  return (
    <ReportFrame
      title={t('cycle.title')}
      subtitle={r && r.basedOnCycles > 0 ? t('cycle.sub', { n: num(r.basedOnCycles) }) : t('recent', { range: t(`ranges.${range}`) })}
      backLabel={t('back')}
      onBack={() => router.push('/analysis')}
      tabs={{
        label: t('rangeLabel'),
        value: range,
        options: REPORT_RANGES.cycle.map((v) => ({ value: v, label: t(`ranges.${v}`) })),
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
