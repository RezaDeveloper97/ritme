'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { DEFAULT_RANGE, HUB_RANGES, useAnalysisSummary, type AnalysisSummary, type HubRangeKey } from '@/entities/analysis';
import { Link, type Locale } from '@/shared/i18n';
import { calendarSystem, monthName, todayParts } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import {
  EmptyState,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  SecondaryButton,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
} from '@/shared/ui';

import { availableCategories, groupCards, visibleCards, type HubCard, type HubCategory, type HubVariant } from '../model/hub';
import {
  CycleCard,
  LabsCard,
  MoodPhaseCard,
  RecentCyclesCard,
  SleepMoodCard,
  SymptomsCard,
  VitalsCard,
  WeightCard,
} from './HubCards';

/** `/analysis/monthly/<ym>` for this month in the reader's calendar («گزارش مهر»). */
function useReportLink(): { href: string; month: string } {
  const loc = useLocale() as Locale;
  const { year, month } = todayParts(loc);
  const ym = `${year}-${String(month).padStart(2, '0')}`;
  return { href: `/analysis/monthly/${ym}?calendar=${calendarSystem(loc)}`, month: monthName(month, loc) };
}

function HubHead() {
  const t = useTranslations('analysis.hub');
  const report = useReportLink();
  return (
    <header className="an-head">
      <h1 className="an-title">{t('title')}</h1>
      <Link href={report.href} className="an-report">
        <Icon name="note" size={15} strokeWidth={1.8} />
        {t('report', { month: report.month })}
      </Link>
    </header>
  );
}

function HubSkeleton({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="an-skel">
      <Skeleton shape="block" className="an-skel-finding" />
      <Skeleton shape="card" />
      <Skeleton shape="card" />
      <Skeleton shape="card" />
    </SkeletonGroup>
  );
}

function renderCard(card: HubCard, summary: AnalysisSummary, variant: HubVariant) {
  const s = summary.sections;
  switch (card) {
    case 'cycle':
      return <CycleCard key={card} section={s.cycle} />;
    case 'recentCycles':
      return <RecentCyclesCard key={card} section={s.recentCycles} hideFertility={variant !== 'cycle'} />;
    case 'symptoms':
      return <SymptomsCard key={card} section={s.symptoms} />;
    case 'moodByPhase':
      return <MoodPhaseCard key={card} section={s.moodByPhase} />;
    case 'sleepMood':
      return <SleepMoodCard key={card} section={s.sleepMood} />;
    case 'weight':
      return <WeightCard key={card} section={s.weight} />;
    case 'vitals':
      return <VitalsCard key={card} section={s.vitals} />;
    case 'labs':
      return <LabsCard key={card} section={s.labs} />;
  }
}

/**
 * The analysis hub (An_Hub, B-N3-08) for cycle / teen / menopause: range tabs,
 * category chips, «مهم‌ترین یافته» and one card per section, each linking to
 * its detail screen (B-N3-09/10). Plus sections arrive locked (no data) for a
 * free user and render as a blurred teaser behind the paywall button.
 */
export function AnalysisHub({ variant }: { variant: HubVariant }) {
  const t = useTranslations('analysis.hub');
  const tCorr = useTranslations('analysis.correlations');
  const mounted = useMounted();
  const [range, setRange] = useState<HubRangeKey>(DEFAULT_RANGE);
  const [category, setCategory] = useState<HubCategory>('all');
  const query = useAnalysisSummary(range);
  const summary = query.data;

  const cards = useMemo(() => (summary ? visibleCards(summary, variant) : []), [summary, variant]);
  const categories = availableCategories(cards);
  const active = categories.includes(category) ? category : 'all';
  const groups = groupCards(cards, active);
  const noData = summary?.topFinding.kind === 'no_data';
  const updating = query.isPlaceholderData && query.isFetching;
  const corrShown =
    summary &&
    groups.some((g) => g.cards.some((c) => (c === 'moodByPhase' || c === 'sleepMood') && summary.sections[c].ready));

  let body;
  if (!mounted || (query.isPending && query.fetchStatus !== 'idle')) {
    body = <HubSkeleton label={t('loading')} />;
  } else if (query.isError || !summary) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <SecondaryButton icon="refresh" onClick={() => void query.refetch()} loading={query.isFetching}>
            {t('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else {
    body = (
      <div className={clsx('an-content', updating && 'is-updating')} aria-busy={updating || undefined}>
        <section className="an-finding-card" aria-labelledby="an-finding-title">
          <Icon name="sparkle" size={20} className="an-finding-icon" />
          <div>
            <h2 id="an-finding-title" className="an-finding-title">
              {t('topFinding', { range: t(`ranges.${range}`) })}
            </h2>
            <p className="an-finding-text">{summary.topFinding.text}</p>
          </div>
        </section>

        {noData ? (
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
        ) : groups.length ? (
          groups.map((g) => (
            <section key={g.group} className="an-group" aria-labelledby={`an-g-${g.group}`}>
              <h2 id={`an-g-${g.group}`} className="an-group-title">
                {t(`groups.${g.group}`)}
              </h2>
              {g.cards.map((card) => renderCard(card, summary, variant))}
            </section>
          ))
        ) : (
          <p className="an-card-note">{t('noFilterMatch')}</p>
        )}
        {corrShown ? <InfoNote>{tCorr('not_causal')}</InfoNote> : null}
      </div>
    );
  }

  return (
    <div className="an-hub">
      <HubHead />
      <SegmentedTabs
        label={t('rangeLabel')}
        value={range}
        tabs={HUB_RANGES.map((r) => ({ value: r, label: t(`ranges.${r}`) }))}
        onChange={setRange}
      />
      {summary && !noData && categories.length > 2 ? (
        <div className="an-chips" role="group" aria-label={t('filterLabel')}>
          {categories.map((c) => (
            <PillChip key={c} pressed={active === c} onPressedChange={() => setCategory(c)} className="an-chip">
              {t(`categories.${c}`)}
            </PillChip>
          ))}
        </div>
      ) : null}
      {body}
    </div>
  );
}
