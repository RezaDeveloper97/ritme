'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  PROGRAMS_HUB_PATH,
  type PmddAlert,
  type PmddDay,
  type PmddItem,
  useConditions,
  useEnrolCondition,
  usePmddChart,
  usePmddDay,
  usePmddItems,
  useSavePmddDay,
} from '@/entities/condition';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { pilot } from '@/shared/pilot';
import {
  Card,
  EmptyState,
  InfoNote,
  NumericScale,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

import { cycleDayOn, pmddNotes } from '../model/chart';
import { PmddChartCard } from './PmddChartCard';
import './cpm.css';

/**
 * «پرسشنامه خلق روزانه» (`/programs/pmdd`, CB-COND-04, nbl_Cond_PMDD): today's six `pmdd_items` rated 1–6
 * (prefilled when she already rated today), the current cycle's chart with the period and last-10-days bands and the
 * pattern sentence, and the crisis hotlines. Writes need the `pmdd` enrolment — without it the form becomes a join
 * card. A form: no bottom nav.
 */
export function PmddPage() {
  const t = useTranslations('conditions.pmdd');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const date = toApiDate(today());
  const overview = useConditions();
  const items = usePmddItems(locale);
  const day = usePmddDay(date);
  const chart = usePmddChart(locale);

  const cycleDay = cycleDayOn(chart.data?.cycles[0], date);
  const enrolled = overview.data?.programs.find((p) => p.code === 'pmdd')?.enrolled ?? false;
  const notes = pmddNotes(chart.data?.alerts ?? []);

  let form;
  if (items.isPending || day.isPending || overview.isPending) {
    form = (
      <SkeletonGroup label={t('loading')} className="cpm-skel">
        <Skeleton shape="card" className="cpm-form-skel" />
      </SkeletonGroup>
    );
  } else if (items.isError || day.isError || overview.isError || !items.data.length) {
    form = (
      <EmptyState
        icon="warning"
        title={t('loadError')}
        action={
          <SecondaryButton
            icon="refresh"
            block={false}
            loading={items.isFetching || day.isFetching || overview.isFetching}
            onClick={() => {
              void items.refetch();
              void day.refetch();
              void overview.refetch();
            }}
          >
            {t('retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (!enrolled) {
    form = <JoinCard />;
  } else {
    form = <PmddForm key={date} date={date} items={items.data} day={day.data} />;
  }

  return (
    <div className="view cpm-page">
      <SkyLayer />
      <div className="scroll cpm-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={cycleDay ? t('subtitle', { day: formatNumber(cycleDay, locale) }) : t('subtitleNoCycle')}
          onBack={() => router.back(PROGRAMS_HUB_PATH)}
          backLabel={t('back')}
        />
        <div className="cpm-body">
          {form}
          <section className="cpm-section" aria-labelledby="cpm-chart-title">
            <h2 id="cpm-chart-title" className="cpm-section-title">
              {t('chartTitle')}
            </h2>
            {chart.isError ? (
              <EmptyState
                icon="warning"
                title={t('chartError')}
                action={
                  <SecondaryButton icon="refresh" block={false} loading={chart.isFetching} onClick={() => void chart.refetch()}>
                    {t('retry')}
                  </SecondaryButton>
                }
              />
            ) : (
              <PmddChartCard chart={chart.data} loading={chart.isPending} />
            )}
          </section>
          <CrisisNote alert={notes.crisis} locale={locale} />
        </div>
      </div>
    </div>
  );
}

function PmddForm({ date, items, day }: { date: string; items: PmddItem[]; day: PmddDay }) {
  const t = useTranslations('conditions.pmdd');
  const locale = useLocale() as Locale;
  const save = useSavePmddDay();
  const [scores, setScores] = useState<Record<string, number>>(() => day.scores);
  const [saved, setSaved] = useState(false);

  const done = items.filter((i) => scores[i.code] !== undefined).length;
  const dirty = items.some((i) => scores[i.code] !== day.scores[i.code]);

  const submit = () => {
    if (!done || save.isPending) return;
    const body = Object.fromEntries(items.flatMap((i) => (scores[i.code] ? [[i.code, scores[i.code]]] : [])));
    save.mutate({ date, scores: body }, { onSuccess: () => setSaved(true) });
  };

  return (
    <>
      <Card className="cpm-card">
        <p className="cpm-hint">{t('scaleHint')}</p>
        {items.map((item) => (
          <div key={item.code} className="cpm-q">
            <NumericScale
              min={1}
              max={6}
              value={scores[item.code] ?? null}
              onChange={(v) => {
                setSaved(false);
                setScores((s) => ({ ...s, [item.code]: v }));
              }}
              locale={locale}
              label={<b className="cpm-q-title">{item.title}</b>}
              pilot={pilot(`programs.pmdd-q:${item.code}`, 'choice')}
            />
          </div>
        ))}
      </Card>
      <div className="cpm-footer">
        {save.isError ? (
          <p className="cpm-error" role="alert">
            {getApiSaveErrorMessage(save.error, t('saveError'))}
          </p>
        ) : null}
        <p className="cpm-progress" aria-live="polite">
          {saved && !dirty
            ? t('saved')
            : t('progress', { done: formatNumber(done, locale), total: formatNumber(items.length, locale) })}
        </p>
        <PrimaryButton loading={save.isPending} disabled={!done || (!dirty && saved)} onClick={submit} {...pilot('programs.pmdd-save')}>
          {t('save')}
        </PrimaryButton>
      </div>
    </>
  );
}

function JoinCard() {
  const t = useTranslations('conditions.pmdd');
  const enrol = useEnrolCondition();
  return (
    <Card className="cpm-join">
      <p className="cpm-join-title">{t('joinTitle')}</p>
      <p className="cpm-join-body">{t('joinBody')}</p>
      {enrol.isError ? (
        <p className="cpm-error" role="alert">
          {getApiSaveErrorMessage(enrol.error, t('saveError'))}
        </p>
      ) : null}
      <PrimaryButton loading={enrol.isPending} onClick={() => enrol.mutate('pmdd')} {...pilot('programs.pmdd-join')}>
        {t('join')}
      </PrimaryButton>
    </Card>
  );
}

const FALLBACK_HOTLINES = ['1480', '123'] as const;

/** The standing crisis line (catalog `pmdd_crisis`, with the 1480 / 123 fallback when the catalog is unreachable). */
function CrisisNote({ alert, locale }: { alert: PmddAlert | null; locale: Locale }) {
  const t = useTranslations('conditions.pmdd');
  const lines = alert?.hotlines.length
    ? alert.hotlines.map((h) => ({ number: h.number, label: h.label ?? '' }))
    : FALLBACK_HOTLINES.map((number) => ({ number, label: t(`hotline.${number}`) }));
  return (
    <>
      <UrgentCard
        variant="note"
        title={alert?.title ?? t('crisisTitle')}
        hotlines={lines.map((l) => ({ ...l, display: formatNumber(Number(l.number), locale) }))}
        hotlinesLabel={t('hotlinesLabel')}
      >
        {alert?.body ?? t('crisisBody')}
      </UrgentCard>
      <InfoNote className="cpm-note">{t('note')}</InfoNote>
    </>
  );
}
