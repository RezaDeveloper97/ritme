'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useId, useState, type ReactNode } from 'react';

import {
  PBAC_CLOTS,
  PBAC_LEVEL_POINTS,
  PBAC_LEVELS,
  PBAC_MAX_PADS,
  PROGRAMS_HUB_PATH,
  type PbacDay,
  type PbacEntry,
  type PbacLevel,
  useConditions,
  useEnrolCondition,
  usePbacDay,
  useSavePbacDay,
} from '@/entities/condition';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber, toApiDate, today } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  Card,
  ChipGroup,
  EmptyState,
  IconCircle,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
} from '@/shared/ui';

import { entryFrom, isDirty, liveScores, payloadFor } from '../model/form';

/**
 * «جدول خونریزی» (`/programs/bleeding`, CB-COND-05, nbl_Cond_Bleed): today's pads per soak level (1 / 5 / 20
 * points), the clot chip, flooding, today's and this period's PBAC score — live while she edits — and the «over 100»
 * note that suggests talking to a doctor about an anaemia test. A form with a back header: no bottom nav. Saving
 * returns to the programs hub (the board's flow). Not in the heavy-bleeding program yet → a join card first.
 */
export function BleedingChartPage() {
  const t = useTranslations('conditions.bleeding');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const date = toApiDate(today());
  const overview = useConditions();
  const query = usePbacDay(date);
  const enrolled = overview.data?.programs.find((p) => p.code === 'heavy_bleeding')?.enrolled;
  const day = query.data;

  let body: ReactNode;
  if (overview.isPending || query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="cnb-body">
        <Skeleton shape="card" className="cnb-skel-tall" />
        <Skeleton shape="card" className="cnb-skel" />
        <Skeleton shape="card" className="cnb-skel" />
      </SkeletonGroup>
    );
  } else if (overview.isError || query.isError || !day) {
    body = (
      <div className="cnb-body">
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <SecondaryButton
              icon="refresh"
              block={false}
              loading={overview.isFetching || query.isFetching}
              onClick={() => {
                void overview.refetch();
                void query.refetch();
              }}
            >
              {t('retry')}
            </SecondaryButton>
          }
        />
      </div>
    );
  } else if (!enrolled) {
    body = <JoinCard />;
  } else {
    body = <ChartForm key={day.date} day={day} onSaved={() => router.push(PROGRAMS_HUB_PATH)} />;
  }

  return (
    <div className="view cnb-page">
      <SkyLayer />
      <div className="scroll cnb-scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={day?.period ? t('periodDay', { day: formatNumber(day.period.day, locale) }) : t('today')}
          onBack={() => router.push(PROGRAMS_HUB_PATH)}
          backLabel={t('back')}
        />
        {body}
      </div>
    </div>
  );
}

function JoinCard() {
  const t = useTranslations('conditions.bleeding');
  const enrol = useEnrolCondition();
  return (
    <div className="cnb-body">
      <Card className="cnb-join">
        <IconCircle icon="drop" tone="period" size="lg" />
        <h2 className="cnb-join-title">{t('join.title')}</h2>
        <p className="cnb-join-body">{t('join.body')}</p>
        {enrol.isError ? (
          <p className="cnb-error" role="alert">
            {getApiSaveErrorMessage(enrol.error, t('join.error'))}
          </p>
        ) : null}
        <PrimaryButton loading={enrol.isPending} onClick={() => enrol.mutate('heavy_bleeding')} {...pilot('programs.bleeding-join')}>
          {t('join.action')}
        </PrimaryButton>
      </Card>
    </div>
  );
}

function ChartForm({ day, onSaved }: { day: PbacDay; onSaved: () => void }) {
  const t = useTranslations('conditions.bleeding');
  const locale = useLocale() as Locale;
  const saved = entryFrom(day);
  const [entry, setEntry] = useState<PbacEntry>(saved);
  const save = useSavePbacDay();
  const padsId = useId();
  const clotsId = useId();
  const floodId = useId();

  const scores = liveScores(day, entry);
  const over = scores.period >= day.alertThreshold;
  const dirty = isDirty(saved, entry);
  const edit = (next: Partial<PbacEntry>) => {
    save.reset();
    setEntry({ ...entry, ...next });
  };
  const submit = () => {
    if (!dirty || save.isPending) return;
    save.mutate({ date: day.date, entry: payloadFor(saved, entry) }, { onSuccess: onSaved });
  };
  // The admin copy of `pbac_over_100` once the saved period crosses it; our own copy while she is still editing.
  const alertTitle = day.alert?.title ?? t('alert.title', { threshold: formatNumber(day.alertThreshold, locale) });
  const alertBody = day.alert?.body ?? t('alert.body');

  return (
    <>
      <div className="cnb-body">
        <section className="cnb-section" aria-labelledby={padsId}>
          <div className="cnb-head">
            <h2 id={padsId} className="cnb-title">
              {t('pads.title')}
            </h2>
            <p className="cnb-hint">{t('pads.hint')}</p>
          </div>
          <Card className="cnb-pads">
            {PBAC_LEVELS.map((level) => (
              <PadRow key={level} level={level} value={entry[level]} onChange={(n) => edit({ [level]: n })} />
            ))}
          </Card>
        </section>

        <Card className="cnb-extras">
          <div className="cnb-clots">
            <h2 id={clotsId} className="cnb-label">
              {t('clots.title')}
            </h2>
            <ChipGroup label={t('clots.title')} pilot={pilot('programs.pbac-clots', 'choice')}>
              {PBAC_CLOTS.map((clot) => (
                <PillChip
                  key={clot}
                  mode="multi"
                  tone="period"
                  pressed={entry.clots === clot}
                  {...pilotOption(clot)}
                  onPressedChange={(on) => edit({ clots: on ? clot : null })}
                >
                  {t(`clots.${clot}`)}
                </PillChip>
              ))}
            </ChipGroup>
          </div>
          <div className="cnb-flood">
            <span id={floodId} className="cnb-label">
              {t('flooding')}
            </span>
            <Switch checked={entry.flooding} onCheckedChange={(flooding) => edit({ flooding })} labelledBy={floodId} pilot={pilot('programs.pbac-flooding')} />
          </div>
        </Card>

        <Card className="cnb-scores" aria-label={t('scores.label')}>
          <ScoreTile label={t('scores.today')} value={formatNumber(scores.today, locale)} />
          <ScoreTile label={t('scores.period')} value={formatNumber(scores.period, locale)} />
        </Card>

        {over ? (
          <section className="cnb-alert" role="status" aria-live="polite">
            <IconCircle icon="warning" tone="period" size="md" />
            <div className="cnb-alert-text">
              <h2 className="cnb-alert-title">{alertTitle}</h2>
              <p className="cnb-alert-body">{alertBody}</p>
            </div>
          </section>
        ) : null}
      </div>

      <div className="cnb-footer">
        {save.isError ? (
          <p className="cnb-error" role="alert">
            {getApiSaveErrorMessage(save.error, t('saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} disabled={!dirty} onClick={submit} {...pilot('programs.pbac-save')}>
          {t('save')}
        </PrimaryButton>
      </div>
    </>
  );
}

function PadRow({ level, value, onChange }: { level: PbacLevel; value: number; onChange: (n: number) => void }) {
  const t = useTranslations('conditions.bleeding');
  const locale = useLocale() as Locale;
  const label = t(`levels.${level}`);
  return (
    <div className="cnb-pad-row">
      <span className={clsx('cnb-pad', `is-${level}`)} aria-hidden>
        <span className="cnb-pad-fill" />
      </span>
      <NumberStepper
        className="cnb-stepper"
        value={value}
        onChange={onChange}
        min={0}
        max={PBAC_MAX_PADS}
        label={label}
        description={t('points', { count: PBAC_LEVEL_POINTS[level], points: formatNumber(PBAC_LEVEL_POINTS[level], locale) })}
        decrementLabel={t('decrement', { label })}
        incrementLabel={t('increment', { label })}
        locale={locale}
        pilot={pilot(`programs.pbac-pads:${level}`, 'stepper')}
      />
    </div>
  );
}

function ScoreTile({ label, value }: { label: string; value: string }) {
  return (
    <div className="cnb-tile">
      <span className="cnb-tile-num">{value}</span>
      <span className="cnb-tile-label">{label}</span>
    </div>
  );
}

