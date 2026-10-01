'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { REPORT_DEFAULT_RANGE, REPORT_RANGES, useCorrelationsReport, type CorrelationsReport } from '@/entities/analysis';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, IconCircle, InfoNote, StatusPill } from '@/shared/ui';
import { PlusGate } from '@/shared/ui/plus-gate';
import { BottomNav, useNavMode } from '@/widgets/bottom-nav';
import { ChartCard, ColumnChart, ReportFrame } from '@/widgets/charts';

import { isPairGroup, isPhaseGroup, PAIR_STYLE, strengthTone, TEASER } from '../model/correlations';

type Range = (typeof REPORT_RANGES.correlations)[number];
type Item = NonNullable<CorrelationsReport['data']>['items'][number];

/**
 * `/analysis/correlations` (An_Correlations, B-N3-09) — Plus
 * (`plus.deep_analysis`). A free user gets `locked: true` and no data from the
 * API, so the screen shows a blurred teaser behind the paywall button. Every
 * card states the co-occurrence in the user's own data and the page closes
 * with the «not causal» note.
 */
export function AnalysisCorrelationsPage() {
  const t = useTranslations('analysis.reports');
  const tA = useTranslations('analysis');
  const tPlus = useTranslations('plus.gate');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const { mode } = useNavMode();
  const [range, setRange] = useState<Range>(REPORT_DEFAULT_RANGE.correlations);
  const query = useCorrelationsReport(range);
  const r = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const status = !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : query.isError || !r ? 'error' : 'ready';
  const rangeLabel = t(`ranges.${range}`);

  const groupLabel = (key: string): string =>
    isPhaseGroup(key) ? tA(`phases.${key}`) : isPairGroup(key) ? t(`correlations.groups.${key}`) : key;

  const card = (c: Item) => {
    const style = PAIR_STYLE[c.key];
    const title = t(`correlations.items.${c.key}`);
    const strength = c.strength === 'strong' || c.strength === 'medium' || c.strength === 'weak' ? c.strength : null;
    const showBars = c.status === 'ready' || (c.status === 'no_association' && c.n > 0);
    return (
      <ChartCard
        key={c.key}
        title={title}
        icon={<IconCircle icon={style.icon} tone={style.tone} size="md" />}
        aside={
          strength && c.status === 'ready' ? (
            <StatusPill tone={strengthTone(strength)} className="anr-strength">
              {t(`correlations.strength.${strength}`)}
            </StatusPill>
          ) : null
        }
      >
        {c.status === 'ready' && c.finding ? <p className="anr-finding">{c.finding.text}</p> : null}
        {c.status === 'no_association' ? <p className="anr-note">{t('correlations.noAssociation')}</p> : null}
        {c.status === 'not_enough_data' ? (
          <p className="anr-note">{t('correlations.notEnough', { n: num(c.minDays), have: num(c.n) })}</p>
        ) : null}
        {showBars ? (
          <ColumnChart
            label={t('correlations.chart', {
              title,
              list: c.groups
                .map((g) => t('correlations.groupValue', { group: groupLabel(g.key), pct: num(g.pct) }))
                .join(tA('format.list_separator')),
            })}
            table={{
              caption: title,
              columns: [t('correlations.colGroup'), t('correlations.colPct'), t('correlations.colDays')],
              rows: c.groups.map((g) => [
                groupLabel(g.key),
                t('period.co.pct', { n: num(g.pct) }),
                t('correlations.hitsOf', { hits: num(g.hits), days: num(g.days) }),
              ]),
            }}
            columns={c.groups.map((g, i) => ({
              key: g.key,
              label: groupLabel(g.key),
              value: g.pct,
              tone: style.tone,
              solid: i === c.groups.length - 1,
            }))}
            max={100}
            floor={0.12}
            gap={12}
            height={130}
          />
        ) : null}
      </ChartCard>
    );
  };

  let content = null;
  if (r) {
    if (r.locked) {
      content =
        mode === 'teen' ? (
          // Teens get no Plus upsell (B-N2-03).
          <EmptyState icon="lock" title={t('correlations.locked.title')} body={t('correlations.locked.body')} />
        ) : (
          <>
            <p className="anr-lead">{t('correlations.locked.body')}</p>
            <section className="nb-card axc-card anr-locked" aria-label={t('correlations.locked.title')}>
              <PlusGate locked label={tPlus('label')} lockedText={tPlus('lockedText')} onUnlock={() => router.push('/plus')}>
                <ColumnChart
                  label=""
                  columns={TEASER.map((b, i) => ({ key: b.key, label: '', value: b.value, tone: 'brand', solid: i === 1 }))}
                  max={100}
                  height={130}
                />
              </PlusGate>
            </section>
          </>
        );
    } else {
      const items = r.data?.items ?? [];
      content = items.length ? items.map(card) : <p className="anr-note">{t('empty.body')}</p>;
    }
  }

  const days = r?.data?.daysLogged;
  return (
    <ReportFrame
      title={t('correlations.title')}
      subtitle={days != null ? t('correlations.sub', { range: rangeLabel, n: num(days) }) : t('recent', { range: rangeLabel })}
      backLabel={t('back')}
      onBack={() => router.push('/analysis')}
      tabs={{
        label: t('rangeLabel'),
        value: range,
        options: REPORT_RANGES.correlations.map((v) => ({ value: v, label: t(`ranges.${v}`) })),
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
      {r && (r.notCausal || r.locked) ? <InfoNote>{tA('correlations.not_causal')}</InfoNote> : null}
    </ReportFrame>
  );
}
