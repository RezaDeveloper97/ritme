'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useDirection, useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { EmptyState, Icon, InfoNote, PrimaryButton, type Tone } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ChartCard, ReportFrame, StatTiles, type StatTile } from '@/widgets/charts';

import { useFertilityCycle } from '../api/ttc';
import { BbtChart } from './BbtChart';

const SOURCE_TONE: Record<string, Tone> = { bbt: 'data', lh: 'warm', estimate: 'neutral' };

/**
 * `/analysis/fertility` (An_Fertility, B-N3-11): one cycle's basal temperature
 * chart with the three-over-six confirmation, LH and intercourse rows, the
 * ovulation / LH / luteal tiles, «چطور تأیید شد؟» and the measuring tip. The
 * stepper walks the cycles the TTC hub lists (newest first on open).
 */
export function AnalysisFertilityPage() {
  const t = useTranslations('analysis.ttc.ui.detail');
  const tR = useTranslations('analysis.reports');
  const tPlus = useTranslations('plus.gate');
  const loc = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const router = useRouter();
  const mounted = useMounted();
  const [cycle, setCycle] = useState<string | null>(null);
  const query = useFertilityCycle(cycle);
  const f = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const temp = (v: number) => formatDecimal(v.toFixed(1), loc);
  const date = (iso: string) => formatDayMonth(fromApiDate(iso), loc);

  const status = !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : query.isError ? 'error' : 'ready';

  let content = null;
  let subtitle: string | undefined;
  if (status === 'ready' && f === null) {
    content = (
      <EmptyState
        icon="chart"
        title={t('empty.title')}
        body={t('empty.body')}
        action={
          <PrimaryButton icon="plus" block={false} onClick={() => openSheet('log')}>
            {tR('empty.cta')}
          </PrimaryButton>
        }
      />
    );
  } else if (f) {
    const c = f.cycle;
    subtitle = t('sub', { index: num(c.index), from: date(c.start), to: c.current ? t('today') : date(c.end) });
    const ov = f.ovulation;
    const lutealLocked = f.luteal.locked;
    const tiles: StatTile[] = [
      {
        key: 'ovulation',
        label: t('tiles.ovulation'),
        value: ov.day != null ? t('tiles.day', { day: num(ov.day) }) : t('tiles.none'),
        sub: ov.source ? t(`tiles.source.${ov.source}`) : undefined,
        tone: ov.source ? SOURCE_TONE[ov.source] : 'neutral',
      },
      {
        key: 'lh',
        label: t('tiles.lh'),
        value: f.lh.positiveDay != null ? t('tiles.day', { day: num(f.lh.positiveDay) }) : t('tiles.none'),
        sub: f.lh.daysBeforeOvulation != null ? t('tiles.lhBefore', { n: f.lh.daysBeforeOvulation }) : undefined,
        tone: 'warm',
      },
      {
        key: 'luteal',
        label: t('tiles.luteal'),
        value: !lutealLocked && f.luteal.data?.days != null ? num(f.luteal.data.days) : t('tiles.none'),
        sub: lutealLocked ? tPlus('label') : f.luteal.data?.days != null ? t('tiles.days') : undefined,
        tone: lutealLocked ? 'warm' : 'brand',
      },
    ];
    const chartTable = {
      caption: t('chart', { index: num(c.index), n: num(f.chart.points.length) }),
      columns: [t('colDay'), t('colTemp')],
      rows: f.chart.points.map((p) => [num(p.day), formatDecimal(p.value.toFixed(2), loc)]),
    };
    content = (
      <>
        <nav className="ttc-fx-nav" aria-label={t('nav', { index: num(c.index), total: num(c.total) })}>
          <button
            type="button"
            className="ttc-fx-step"
            aria-label={t('prev')}
            disabled={!f.prevStart}
            onClick={() => setCycle(f.prevStart)}
          >
            <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={18} strokeWidth={2} />
          </button>
          <b className="ttc-fx-nav-title" aria-live="polite">
            {t('nav', { index: num(c.index), total: num(c.total) })}
          </b>
          <button
            type="button"
            className="ttc-fx-step"
            aria-label={t('next')}
            disabled={!f.nextStart}
            onClick={() => setCycle(f.nextStart)}
          >
            <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} strokeWidth={2} />
          </button>
        </nav>
        <section className="nb-card axc-card ttc-fx-card">
          {f.chart.points.length ? (
            <BbtChart
              data={f}
              label={chartTable.caption}
              table={chartTable}
              text={{
                ovulation: t('ovulation'),
                coverline: t('coverline'),
                period: t('rows.period'),
                lh: t('rows.lh'),
                sex: t('rows.sex'),
              }}
              formatTemp={temp}
            />
          ) : (
            <p className="an-card-note">{t('noReadings')}</p>
          )}
        </section>
        <StatTiles label={t('tiles.label')} tiles={tiles} />
        <ChartCard title={t('how')}>
          <p className="ttc-fx-how">{f.explanation.text}</p>
        </ChartCard>
        <InfoNote>{f.tip.text}</InfoNote>
      </>
    );
  }

  return (
    <ReportFrame
      title={t('title')}
      subtitle={subtitle}
      backLabel={tR('back')}
      onBack={() => router.push('/analysis')}
      status={status}
      updating={query.isPlaceholderData && query.isFetching}
      loadingLabel={tR('loading')}
      error={{
        title: tR('error.title'),
        body: tR('error.body'),
        retry: tR('error.retry'),
        onRetry: () => void query.refetch(),
        retrying: query.isFetching,
      }}
      after={<BottomNav />}
    >
      {content}
    </ReportFrame>
  );
}
