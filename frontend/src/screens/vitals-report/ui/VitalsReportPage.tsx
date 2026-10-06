'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import {
  DEFAULT_RANGE,
  ReadingRow,
  classifyBp,
  fromMgDl,
  uiTone,
  unitSymbol,
  useDeleteReading,
  useVitalFormat,
  useVitalReadings,
  useVitalReport,
  useVitalThresholds,
  vitalAddPath,
  type BpReport,
  type DistributionEntry,
  type GlucoseFilter,
  type GlucoseReport,
  type GlucoseUnit,
  type HrReport,
  type ReportRange,
  type VitalReading,
  type VitalType,
} from '@/entities/vital';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, fromApiDate } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  InfoNote,
  PillChip,
  PrimaryButton,
  ProgressRing,
  ScreenHeader,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';

import { BpChart, VitalLineChart } from './VitalCharts';

type T = ReturnType<typeof useTranslations<'vitals'>>;

const RANGE_TABS: Record<VitalType, readonly ReportRange[]> = { bp: ['7d', '30d', '90d'], hr: ['7d', '30d', '90d'], glucose: ['14d', '30d', '90d'] };
const GLUCOSE_FILTERS: readonly GlucoseFilter[] = ['all', 'fasting', 'after_meal'];
const LIST_STEP = 20;

/** Short x label of a day: weekday initial for a week, «۲ مهر» beyond. */
function dayLabel(date: string, rangeDays: number, t: T, locale: Locale): string {
  const d = fromApiDate(date);
  return rangeDays <= 7 ? t(`weekdays.${(d.getDay() + 1) % 7}` as 'weekdays.0') : formatDayMonth(d, locale);
}

/**
 * `/vitals/bp`, `/vitals/glucose`, `/vitals/heart-rate` — the reports
 * (nbl_Vitals_BPReport / GlucoseReport; heart rate follows the BP board):
 * range tabs (glucose: context filter + range), average, daily chart with the
 * normal / target band, lowest / highest, class distribution, morning vs
 * night, time in range and every reading of the range (delete for own rows;
 * log-sheet values are read-only). Back header, no bottom nav.
 */
export function VitalsReportPage({ type }: { type: VitalType }) {
  const t = useTranslations('vitals');
  const router = useRouter();
  const f = useVitalFormat();
  const [range, setRange] = useState<ReportRange>(DEFAULT_RANGE[type]);
  const [filter, setFilter] = useState<GlucoseFilter>('all');
  const report = useVitalReport(type, range, type === 'glucose' ? filter : null);
  const r = report.data;

  const subtitle = r ? t('report.subtitle', { range: t(`report.ranges.${r.range.key}`), n: f.num(r.readings) }) : undefined;

  let body: ReactNode;
  if (report.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="vt-report">
        <Skeleton shape="card" className="vt-skel-chart" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (report.isError || !r) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('common.loadError')}
          action={
            <PrimaryButton icon="refresh" loading={report.isFetching} onClick={() => void report.refetch()}>
              {t('common.retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else if (r.readings === 0) {
    body = (
      <Card>
        <EmptyState
          icon="chart"
          title={t('report.empty.title')}
          body={t('report.empty.body')}
          action={
            <Link href={vitalAddPath(type)} className="nb-btn is-primary is-block">
              <Icon name="plus" size={18} />
              {t('report.addCta', { type: t(`types.${type}`) })}
            </Link>
          }
        />
      </Card>
    );
  } else {
    body = (
      <div className={clsx('vt-report', report.isPlaceholderData && 'is-stale')} aria-busy={report.isFetching || undefined}>
        {r.type === 'bp' ? <BpBody r={r} /> : r.type === 'glucose' ? <GlucoseBody r={r} /> : <HrBody r={r} />}
        <ReadingsList type={type} from={r.range.from} to={r.range.to} />
        {r.type === 'glucose' ? <p className="vt-foot-note">{t('report.generalNote')}</p> : null}
      </div>
    );
  }

  return (
    <div className="view vt-screen vt-report-screen">
      <SkyLayer />
      <div className="scroll vt-scroll">
        <ScreenHeader title={t(`report.titles.${type}`)} subtitle={subtitle} onBack={() => router.push('/vitals')} backLabel={t('common.back')} />
        <div className="vt-report-tabs">
          {type === 'glucose' ? (
            <>
              <SegmentedTabs
                label={t('report.filterTabs')}
                value={filter}
                onChange={setFilter}
                tabs={GLUCOSE_FILTERS.map((v) => ({ value: v, label: v === 'all' ? t('report.all') : t(`glucoseContextsShort.${v}`) }))}
              />
              <ChipGroup label={t('report.rangeTabs')} className="vt-range-chips">
                {RANGE_TABS.glucose.map((v) => (
                  <PillChip key={v} pressed={range === v} onPressedChange={() => setRange(v)}>
                    {t(`report.ranges.${v}`)}
                  </PillChip>
                ))}
              </ChipGroup>
            </>
          ) : (
            <SegmentedTabs
              label={t('report.rangeTabs')}
              value={range}
              onChange={setRange}
              tabs={RANGE_TABS[type].map((v) => ({ value: v, label: t(`report.ranges.${v}`) }))}
            />
          )}
        </div>
        {body}
      </div>
    </div>
  );
}

function Legend({ items }: { items: ReadonlyArray<{ key: string; label: string; kind: 'hollow' | 'dot' | 'band'; tone: string }> }) {
  return (
    <ul className="vtc-legend">
      {items.map((i) => (
        <li key={i.key} className={clsx('vtc-legend-item', `nb-tone-${i.tone}`)}>
          <span className={clsx('vtc-legend-mark', `is-${i.kind}`)} aria-hidden />
          {i.label}
        </li>
      ))}
    </ul>
  );
}

function Tiles({ tiles }: { tiles: ReadonlyArray<{ key: string; label: string; value: string; sub?: string }> }) {
  return (
    <dl className="vt-tiles-row">
      {tiles.map((x) => (
        <div key={x.key} className="nb-card vt-tile">
          <dt className="vt-tile-label">{x.label}</dt>
          <dd className="vt-tile-value">
            <bdi dir="ltr">{x.value}</bdi>
          </dd>
          {x.sub ? <dd className="vt-tile-sub">{x.sub}</dd> : null}
        </div>
      ))}
    </dl>
  );
}

function Distribution({ type, entries }: { type: VitalType; entries: readonly DistributionEntry[] }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const shown = entries.filter((e) => e.count > 0 || e.code !== 'crisis');
  return (
    <Card as="section" className="vt-card">
      <h2 className="vt-card-title">{t('report.distribution')}</h2>
      <div className="vt-dist-bar" aria-hidden>
        {entries
          .filter((e) => e.percent > 0)
          .map((e) => (
            <span key={e.code} className={clsx('vt-dist-seg', `is-${e.code}`, `nb-tone-${uiTone(e.tone)}`)} style={{ flexGrow: e.percent }} />
          ))}
      </div>
      <ul className="vt-dist-legend">
        {shown.map((e) => (
          <li key={e.code} className={clsx('vt-dist-item', `is-${e.code}`, `nb-tone-${uiTone(e.tone)}`)}>
            <span className="vt-dist-dot" aria-hidden />
            {t('report.distributionItem', { class: f.classLabel(type, e.code), pct: f.num(e.percent) })}
          </li>
        ))}
      </ul>
    </Card>
  );
}

function MorningNight({ morning, night, note }: { morning: string | null; night: string | null; note: string | null }) {
  const t = useTranslations('vitals');
  return (
    <>
      <Card as="section" className="vt-card">
        <h2 className="vt-card-title">{t('report.morningVsNight')}</h2>
        <div className="vt-mvn">
          <div className="vt-mvn-cell is-morning">
            <span className="vt-mvn-label">{t('report.morning')}</span>
            <bdi dir="ltr" className="vt-mvn-value">
              {morning ?? t('report.noReadings')}
            </bdi>
          </div>
          <div className="vt-mvn-cell is-night">
            <span className="vt-mvn-label">{t('report.night')}</span>
            <bdi dir="ltr" className="vt-mvn-value">
              {night ?? t('report.noReadings')}
            </bdi>
          </div>
        </div>
      </Card>
      {note ? (
        <InfoNote icon="warning" className="vt-caution">
          {note}
        </InfoNote>
      ) : null}
    </>
  );
}

function tableOf(t: T, rows: ReadonlyArray<readonly string[]>, caption: string) {
  return { caption, columns: [t('report.colDate'), t('report.colValue')], rows };
}

function BpBody({ r }: { r: BpReport }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const th = useVitalThresholds();
  const labels = r.series.map((p) => dayLabel(p.date, r.range.days, t, f.locale));
  const values = r.series.map((p) => f.bp(p.systolic, p.diastolic));
  const extreme = (x: VitalReading | null) => (x?.bloodPressure ? f.bp(x.bloodPressure.systolic, x.bloodPressure.diastolic) : '—');
  const extremeSub = (x: VitalReading | null) => (x ? (x.period === 'night' ? `${f.day(x.date)} ${t('report.night')}` : f.day(x.date)) : undefined);
  return (
    <>
      <Card as="section" className="vt-card vt-chart-card">
        <div className="vt-avg">
          <div className="vt-avg-main">
            <span className="vt-avg-label">{t('report.average')}</span>
            <span className="vt-avg-line">
              <bdi dir="ltr" className="vt-big">
                {r.average ? f.bp(r.average.systolic, r.average.diastolic) : '—'}
              </bdi>
              <span className="vt-unit-text" dir="ltr">
                {t('units.mmhg')}
              </span>
            </span>
          </div>
          {r.average?.classification ? (
            <StatusPill tone={uiTone(r.average.classification.tone)}>{f.classLabel('bp', r.average.classification.code)}</StatusPill>
          ) : null}
        </div>
        {r.series.length ? (
          <BpChart
            points={r.series.map((p) => ({ systolic: p.systolic, diastolic: p.diastolic, tone: classifyBp(p.systolic, p.diastolic, th).tone === 'ok' ? 'brand' : 'warm' }))}
            labels={labels}
            band={[r.target.diastolicMax, r.target.systolicMax]}
            fmt={f.num}
            label={t('report.chartBp', { n: f.num(r.series.length), list: values.join(t('common.listSeparator')) })}
            table={tableOf(t, r.series.map((p, i) => [formatDayMonth(fromApiDate(p.date), f.locale), values[i]]), t('report.titles.bp'))}
          />
        ) : null}
        <Legend
          items={[
            { key: 's', label: t('report.legend.systolic'), kind: 'hollow', tone: 'brand' },
            { key: 'd', label: t('report.legend.diastolic'), kind: 'dot', tone: 'brand' },
            { key: 'b', label: t('report.legend.normal'), kind: 'band', tone: 'data' },
          ]}
        />
      </Card>
      <Tiles
        tiles={[
          { key: 'min', label: t('report.min'), value: extreme(r.min), sub: extremeSub(r.min) },
          { key: 'max', label: t('report.max'), value: extreme(r.max), sub: extremeSub(r.max) },
          { key: 'pulse', label: t('report.pulse'), value: r.pulse ? f.num(r.pulse.bpm) : '—', sub: t('report.pulseAvg') },
        ]}
      />
      <Distribution type="bp" entries={r.distribution} />
      <MorningNight
        morning={r.morningVsNight.morning ? f.bp(r.morningVsNight.morning.systolic, r.morningVsNight.morning.diastolic) : null}
        night={r.morningVsNight.night ? f.bp(r.morningVsNight.night.systolic, r.morningVsNight.night.diastolic) : null}
        note={r.morningVsNight.nightOutOfRange > 0 ? t('report.nightNote.bp', { n: f.num(r.morningVsNight.nightOutOfRange) }) : null}
      />
    </>
  );
}

/** The glucose unit to show: the one of the newest reading in range (mg/dL by default). */
function displayUnit(r: GlucoseReport): GlucoseUnit {
  return r.max?.glucose?.unit ?? r.min?.glucose?.unit ?? 'mg_dl';
}

function GlucoseBody({ r }: { r: GlucoseReport }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const unit = displayUnit(r);
  const g = (mg: number) => f.glucose(fromMgDl(mg, unit), unit);
  const ctx = (c: 'fasting' | 'after_meal') => r.byContext.find((x) => x.context === c);
  const fasting = ctx('fasting');
  const after = ctx('after_meal');
  const labels = r.series.map((p) => dayLabel(p.date, r.range.days, t, f.locale));
  const values = r.series.map((p) => [p.fasting, p.afterMeal, p.other].filter((v): v is number => v !== null).map(g).join('، '));
  const band: [number, number] = r.filter === 'fasting' || r.filter === 'before_meal' ? [70, 100] : [70, 140];
  const scaled = (v: number | null) => (v === null ? null : fromMgDl(v, unit));
  const avgBlock = (label: string, mg: number | null | undefined, main: boolean) => (
    <div className={clsx('vt-avg-main', !main && 'is-secondary')}>
      <span className="vt-avg-label">{label}</span>
      <span className="vt-avg-line">
        <bdi dir="ltr" className={main ? 'vt-big' : 'vt-mid'}>
          {mg != null ? g(mg) : '—'}
        </bdi>
        {main ? (
          <span className="vt-unit-text" dir="ltr">
            {unitSymbol(unit)}
          </span>
        ) : null}
      </span>
    </div>
  );
  const tir = r.timeInRange;
  return (
    <>
      <Card as="section" className="vt-card vt-chart-card">
        <div className="vt-avg">
          {r.filter === 'all' ? (
            <>
              {avgBlock(t('report.averageOf', { context: t('glucoseContextsShort.fasting') }), fasting?.average?.mgDl, true)}
              {after ? avgBlock(t('report.averageOf', { context: t('glucoseContextsShort.after_meal') }), after.average?.mgDl, false) : null}
            </>
          ) : (
            avgBlock(t('report.averageOf', { context: t(`glucoseContextsShort.${r.filter}`) }), r.average?.mgDl, true)
          )}
        </div>
        {r.series.length ? (
          <VitalLineChart
            series={[
              { key: 'fasting', values: r.series.map((p) => scaled(p.fasting)), hollow: true, tone: 'warm' },
              { key: 'after', values: r.series.map((p) => scaled(p.afterMeal)), tone: 'warm' },
              { key: 'other', values: r.series.map((p) => scaled(p.other)), tone: 'warm' },
            ]}
            labels={labels}
            band={[fromMgDl(band[0], unit), fromMgDl(band[1], unit)]}
            fmt={(n) => f.glucose(n, unit)}
            label={t('report.chartGlucose', { n: f.num(r.series.length), list: values.join(t('common.listSeparator')) })}
            table={tableOf(t, r.series.map((p, i) => [formatDayMonth(fromApiDate(p.date), f.locale), values[i]]), t('report.titles.glucose'))}
          />
        ) : null}
        <Legend
          items={[
            { key: 'f', label: t('glucoseContextsShort.fasting'), kind: 'hollow', tone: 'warm' },
            { key: 'a', label: t('glucoseContextsShort.after_meal'), kind: 'dot', tone: 'warm' },
            { key: 'b', label: t('report.legend.target'), kind: 'band', tone: 'data' },
          ]}
        />
      </Card>
      <Card as="section" className="vt-card">
        <h2 className="vt-card-title">{t('report.tir')}</h2>
        <div className="vt-tir">
          <ProgressRing value={tir.percent / 100} size={96} thickness={12} tone="data" label={t('report.tir')} valueText={t('report.tirPct', { pct: f.num(tir.percent) })} className="vt-tir-ring">
            <span className="vt-tir-pct">{t('report.tirPct', { pct: f.num(tir.percent) })}</span>
          </ProgressRing>
          <p className="vt-tir-text">
            {t('report.tirText', { in: f.num(tir.inRange), n: f.num(tir.readings) })}
            {tir.above ? ` ${t('report.tirAbove', { n: f.num(tir.above) })}` : ''}
            {tir.below ? ` ${t('report.tirBelow', { n: f.num(tir.below) })}` : ''}
          </p>
        </div>
      </Card>
      <Tiles
        tiles={[
          { key: 'min', label: t('report.min'), value: r.min?.glucose ? f.glucose(r.min.glucose.value, r.min.glucose.unit) : '—', sub: r.min ? f.day(r.min.date) : undefined },
          { key: 'max', label: t('report.max'), value: r.max?.glucose ? f.glucose(r.max.glucose.value, r.max.glucose.unit) : '—', sub: r.max ? f.day(r.max.date) : undefined },
        ]}
      />
      {r.morningVsNight.morning && r.morningVsNight.night ? (
        <MorningNight
          morning={g(r.morningVsNight.morning.mgDl)}
          night={g(r.morningVsNight.night.mgDl)}
          note={r.morningVsNight.nightOutOfRange > 0 ? t('report.nightNote.glucose', { n: f.num(r.morningVsNight.nightOutOfRange) }) : null}
        />
      ) : null}
    </>
  );
}

function HrBody({ r }: { r: HrReport }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const labels = r.series.map((p) => dayLabel(p.date, r.range.days, t, f.locale));
  const values = r.series.map((p) => f.num(p.bpm));
  return (
    <>
      <Card as="section" className="vt-card vt-chart-card">
        <div className="vt-avg">
          <div className="vt-avg-main">
            <span className="vt-avg-label">{t('report.average')}</span>
            <span className="vt-avg-line">
              <bdi dir="ltr" className="vt-big">
                {r.average ? f.num(r.average.bpm) : '—'}
              </bdi>
              <span className="vt-unit-text" dir="ltr">
                {t('units.bpm')}
              </span>
            </span>
          </div>
          {r.restingAverage ? (
            <div className="vt-avg-main is-secondary">
              <span className="vt-avg-label">{t('report.restingAverage')}</span>
              <bdi dir="ltr" className="vt-mid">
                {f.num(r.restingAverage.bpm)}
              </bdi>
            </div>
          ) : null}
        </div>
        {r.series.length ? (
          <VitalLineChart
            series={[{ key: 'bpm', values: r.series.map((p) => p.bpm), tone: 'bloom' }]}
            labels={labels}
            band={[r.target.min, r.target.max]}
            fmt={f.num}
            label={t('report.chartHr', { n: f.num(r.series.length), list: values.join(t('common.listSeparator')) })}
            table={tableOf(t, r.series.map((p, i) => [formatDayMonth(fromApiDate(p.date), f.locale), values[i]]), t('report.titles.hr'))}
          />
        ) : null}
        <Legend
          items={[
            { key: 'p', label: t('report.legend.bpm'), kind: 'dot', tone: 'bloom' },
            { key: 'b', label: t('report.legend.normal'), kind: 'band', tone: 'data' },
          ]}
        />
      </Card>
      <Tiles
        tiles={[
          { key: 'min', label: t('report.min'), value: r.min?.heartRate ? f.num(r.min.heartRate.bpm) : '—', sub: r.min ? f.day(r.min.date) : undefined },
          { key: 'max', label: t('report.max'), value: r.max?.heartRate ? f.num(r.max.heartRate.bpm) : '—', sub: r.max ? f.day(r.max.date) : undefined },
        ]}
      />
      <Distribution type="hr" entries={r.distribution} />
      <MorningNight
        morning={r.morningVsNight.morning ? f.num(r.morningVsNight.morning.bpm) : null}
        night={r.morningVsNight.night ? f.num(r.morningVsNight.night.bpm) : null}
        note={r.morningVsNight.nightOutOfRange > 0 ? t('report.nightNote.hr', { n: f.num(r.morningVsNight.nightOutOfRange) }) : null}
      />
    </>
  );
}

function ReadingsList({ type, from, to }: { type: VitalType; from: string; to: string }) {
  const t = useTranslations('vitals');
  const f = useVitalFormat();
  const readings = useVitalReadings(type, from, to);
  const del = useDeleteReading();
  const [limit, setLimit] = useState(LIST_STEP);
  const list = readings.data ?? [];
  const remove = (r: VitalReading) => {
    if (r.id === null || !window.confirm(t('common.deleteConfirm'))) return;
    del.mutate(r.id);
  };
  return (
    <Card as="section" className="vt-card vt-list-card" aria-labelledby="vt-all-title">
      <div className="vt-card-head is-tight">
        <h2 id="vt-all-title" className="vt-card-title">
          {t('report.allReadings')}
        </h2>
      </div>
      {readings.isPending ? (
        <SkeletonGroup label={t('common.loading')}>
          <Skeleton />
          <Skeleton />
        </SkeletonGroup>
      ) : readings.isError ? (
        <p className="vt-error" role="alert">
          {t('common.loadError')}
        </p>
      ) : (
        <ul className="vt-rows">
          {list.slice(0, limit).map((r, i) => (
            <ReadingRow
              key={r.id ?? `log-${r.date}-${i}`}
              reading={r}
              withType={false}
              action={
                r.editable ? (
                  <button type="button" className="vt-icon-btn" aria-label={t('common.deleteAria', { value: f.title(r) })} disabled={del.isPending} onClick={() => remove(r)}>
                    <Icon name="trash" size={16} />
                  </button>
                ) : null
              }
            />
          ))}
        </ul>
      )}
      {del.isError ? (
        <p className="vt-error" role="alert">
          {t('common.deleteError')}
        </p>
      ) : null}
      {list.length > limit ? (
        <button type="button" className="vt-link-btn vt-more" onClick={() => setLimit((l) => l + LIST_STEP)}>
          {t('report.moreReadings')}
        </button>
      ) : null}
    </Card>
  );
}
