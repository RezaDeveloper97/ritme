'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { type ChildHome, type ChildMeasurement, roundPercentile, useChild } from '@/entities/child';
import { getApiErrorStatus } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDecimal, formatLongDate, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  InfoNote,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { useGrowthSeries, useMeasurements } from '../api/queries';
import { GROWTH_INDICATORS, type GrowthIndicator, type GrowthSeries } from '../model/types';
import { GrowthChart } from './GrowthChart';
import { MeasurementSheet } from './MeasurementSheet';

type T = ReturnType<typeof useTranslations<'children'>>;

function Shell({ header, children }: { header: ReactNode; children: ReactNode }) {
  return (
    <div className="view chd-screen cgr-screen">
      <SkyLayer />
      <div className="scroll">
        {header}
        <div className="chd-body">{children}</div>
      </div>
    </div>
  );
}

/** The editor sheet target: a new measurement, or an existing one. */
type Editing = { kind: 'new' } | { kind: 'edit'; measurement: ChildMeasurement } | null;

/**
 * `/children/[id]/growth` (nbl_v16_Growth): weight / length / head tabs, the
 * WHO chart (P3–P97 band, median, the child's values), the measurement history
 * and add / edit / delete in a sheet. A spouse sees a shared child read-only
 * (no «+», no editing). Percentiles are estimates — the disclaimer stays visible.
 */
export function ChildGrowthPage({ id }: { id: number }) {
  const t = useTranslations('children');
  const router = useRouter();
  const mounted = useMounted();
  const valid = Number.isFinite(id) && id > 0;
  const child = useChild(valid ? id : null);
  const [indicator, setIndicator] = useState<GrowthIndicator>('weight');
  const [editing, setEditing] = useState<Editing>(null);
  const series = useGrowthSeries(valid ? id : 0, indicator);
  const list = useMeasurements(valid ? id : 0);

  const back = () => router.push(valid ? `/children/${id}` : '/children');
  const name = child.data?.name ?? '';
  const canEdit = child.data?.canEdit ?? false;
  const header = (
    <ScreenHeader
      title={t('section.growth', { name })}
      subtitle={t('growth.subtitle')}
      onBack={back}
      backLabel={t('common.back')}
      action={
        canEdit ? (
          <HeaderButton variant="soft" icon="plus" label={t('growth.add')} onClick={() => setEditing({ kind: 'new' })} />
        ) : undefined
      }
    />
  );

  if (!valid || getApiErrorStatus(child.error) === 404) {
    return (
      <Shell header={header}>
        <EmptyState
          icon="sprout"
          title={t('home.notFoundTitle')}
          body={t('home.notFoundBody')}
          action={<PrimaryButton onClick={() => router.push('/children')}>{t('home.toList')}</PrimaryButton>}
        />
      </Shell>
    );
  }

  if (!mounted || child.isPending || (series.isPending && !series.data) || list.isPending) {
    return (
      <Shell header={header}>
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton shape="block" />
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </SkeletonGroup>
      </Shell>
    );
  }

  if (child.isError || series.isError || list.isError) {
    const retry = () => {
      void child.refetch();
      void series.refetch();
      void list.refetch();
    };
    return (
      <Shell header={header}>
        <Card className="chd-state" role="alert">
          <IconCircle icon="warning" tone="danger" size="lg" />
          <p className="chd-state-text">{t('common.loadError')}</p>
          <SecondaryButton icon="refresh" block={false} loading={series.isFetching || list.isFetching} onClick={retry}>
            {t('common.retry')}
          </SecondaryButton>
        </Card>
      </Shell>
    );
  }

  const home = child.data;
  const measurements = list.data.measurements;
  return (
    <Shell header={header}>
      <SegmentedTabs<GrowthIndicator>
        label={t('growth.tabsLabel')}
        value={indicator}
        onChange={setIndicator}
        tabs={GROWTH_INDICATORS.map((v) => ({ value: v, label: t(`growth.${v}`) }))}
      />
      {home.role === 'shared' ? (
        <p className="cgr-readonly">
          <Icon name="eye" size={14} />
          {home.ownerName ? t('home.sharedBy', { name: home.ownerName }) : t('home.readOnly')}
        </p>
      ) : null}
      <ChartCard series={series.data} child={home} t={t} />
      {measurements.length > 0 ? (
        <History
          measurements={measurements}
          indicator={indicator}
          canEdit={canEdit}
          onEdit={(m) => setEditing({ kind: 'edit', measurement: m })}
          t={t}
        />
      ) : null}
      {series.data.disclaimer ? <p className="cgr-disclaimer">{series.data.disclaimer}</p> : null}
      {canEdit ? (
        <PrimaryButton icon="ruler" onClick={() => setEditing({ kind: 'new' })}>
          {t('growth.cta')}
        </PrimaryButton>
      ) : null}
      {canEdit ? (
        <MeasurementSheet
          childId={home.id}
          birthDate={home.birthDate}
          measurement={editing?.kind === 'edit' ? editing.measurement : null}
          open={editing !== null}
          onClose={() => setEditing(null)}
        />
      ) : null}
    </Shell>
  );
}

// ── Chart card ─────────────────────────────────────────────────
function ChartCard({ series, child, t }: { series: GrowthSeries; child: ChildHome; t: T }) {
  const locale = useLocale() as Locale;
  const router = useRouter();
  const latest = series.latest;
  const indicatorLabel = t(`growth.${series.indicator}`);
  const sexUnknown = !series.available && series.reason === 'sex_unknown';
  const when = latest
    ? latest.measuredOn === toApiDate(today())
      ? t('growth.today')
      : formatLongDate(fromApiDate(latest.measuredOn), locale)
    : '';
  const label = latest
    ? t('growth.chartLabel', {
        indicator: indicatorLabel,
        name: child.name,
        count: formatNumber(series.points.length, locale),
        value: formatDecimal(latest.value, locale),
        unit: series.unitLabel,
        age: formatDecimal(latest.ageMonths, locale),
      })
    : t('growth.chartLabelEmpty', { indicator: indicatorLabel, name: child.name });

  return (
    <Card as="section" className="cgr-card">
      {latest ? (
        <div className="cgr-latest">
          <div className="cgr-latest-text">
            <span className="cgr-latest-when">{t('growth.lastEntry', { when })}</span>
            <span className="cgr-latest-value">
              <b className="cgr-latest-num">{formatDecimal(latest.value, locale)}</b>
              <span className="cgr-latest-unit">{series.unitLabel}</span>
            </span>
          </div>
          {latest.inBand === true ? (
            <StatusPill tone="success" icon="check" className="cgr-pill">
              {t('growth.inBand')}
            </StatusPill>
          ) : latest.inBand === false ? (
            <StatusPill tone="warm" icon="info" className="cgr-pill">
              {t('growth.outOfBand')}
            </StatusPill>
          ) : null}
        </div>
      ) : null}

      {series.points.length > 0 || series.available ? (
        <>
          <GrowthChart
            series={series}
            label={label}
            table={{
              caption: t('growth.tableCaption', { indicator: indicatorLabel, name: child.name }),
              date: t('growth.colDate'),
              age: t('growth.colAge'),
              value: t('growth.colValue', { unit: series.unitLabel }),
              percentile: t('growth.colPercentile'),
            }}
          />
          <div className="cgr-axes" aria-hidden>
            <span>{t('growth.months')}</span>
            <span>{series.unitLabel}</span>
          </div>
          <ul className="cgr-legend" aria-hidden>
            <li>
              <span className="cgr-swatch is-child" />
              {t('growth.legendChild', { indicator: indicatorLabel, name: child.name })}
            </li>
            {series.available && series.medianLabel ? (
              <li>
                <span className="cgr-swatch is-median" />
                {series.medianLabel}
              </li>
            ) : null}
            {series.available ? (
              <li>
                <span className="cgr-swatch is-band" />
                {series.bandLabel}
              </li>
            ) : null}
          </ul>
        </>
      ) : null}

      {series.points.length === 0 ? (
        <div className="cgr-empty">
          <IconCircle icon="chart" tone="data" size="lg" />
          <b className="cgr-empty-title">{t('growth.emptyTitle')}</b>
          <p className="cgr-empty-body">{child.canEdit ? t('growth.emptyBody') : t('growth.emptyShared')}</p>
        </div>
      ) : null}

      {sexUnknown ? (
        <InfoNote icon="info" className="cgr-note">
          <b className="cgr-note-title">{t('growth.sexUnknownTitle')}</b>
          <span>{t('growth.sexUnknownBody')}</span>
          {child.canEdit ? (
            <button type="button" className="cgr-note-link" onClick={() => router.push(`/children/${child.id}/edit`)}>
              {t('growth.editProfile')}
            </button>
          ) : null}
        </InfoNote>
      ) : null}
    </Card>
  );
}

// ── History ────────────────────────────────────────────────────
function History({
  measurements,
  indicator,
  canEdit,
  onEdit,
  t,
}: {
  measurements: readonly ChildMeasurement[];
  indicator: GrowthIndicator;
  canEdit: boolean;
  onEdit: (m: ChildMeasurement) => void;
  t: T;
}) {
  const locale = useLocale() as Locale;
  return (
    <Card as="section" className="cgr-history" aria-labelledby="cgr-history-title">
      <h2 id="cgr-history-title" className="sr-only">
        {t('growth.historyTitle')}
      </h2>
      <ul className="cgr-rows">
        {measurements.map((m) => {
          const isBirth = m.source === 'birth' || m.id === null;
          const date = formatLongDate(fromApiDate(m.measuredOn), locale);
          const title = isBirth ? t('growth.birth') : t('growth.rowWhen', { date, age: m.ageLabel ?? '' });
          const values = [
            m.weight ? t('growth.kgValue', { v: formatDecimal(m.weight.value, locale) }) : null,
            m.length ? t('growth.cmValue', { v: formatDecimal(m.length.value, locale) }) : null,
            m.head ? t('growth.headValue', { v: formatDecimal(m.head.value, locale) }) : null,
          ].filter((v): v is string => v !== null);
          const current = m[indicator];
          const p = roundPercentile(current?.percentile);
          const body = (
            <>
              <span className="cgr-row-text">
                <b className="cgr-row-title">{title}</b>
                <span className="cgr-row-values">{values.join(' · ')}</span>
              </span>
              {current && p !== null ? (
                <span className={clsx('cgr-row-p', current.inBand === false && 'is-out')}>
                  {current.inBand === false ? t('growth.outOfBand') : t('growth.percentile', { p: formatNumber(p, locale) })}
                </span>
              ) : null}
            </>
          );
          const key = m.id ?? `birth-${m.measuredOn}`;
          return (
            <li key={key} className="cgr-row-item">
              {canEdit && !isBirth ? (
                <button
                  type="button"
                  className="cgr-row is-button"
                  aria-label={`${t('growth.rowActions', { date })} — ${values.join(' · ')}`}
                  onClick={() => onEdit(m)}
                >
                  {body}
                </button>
              ) : (
                <div className="cgr-row">{body}</div>
              )}
            </li>
          );
        })}
      </ul>
    </Card>
  );
}
