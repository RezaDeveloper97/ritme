'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate, weekdayKeys, weekdayLabels } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, InfoNote, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup, StatusPill } from '@/shared/ui';
import { ColumnChart } from '@/widgets/charts';

import { isPregnancyInactive, usePregnancyAnalysis } from '../api/queries';
import type { PregnancySections } from '../api/schema';
import { gainUntilWeek, tileItems, weekdayKeyOf } from '../model/chart';
import { BpChartView } from './BpChartView';
import { GainChartView } from './GainChartView';
import { PregCard } from './PregCard';

type T = ReturnType<typeof useTranslations<'analysisPregnancy.hub'>>;

function useFmt() {
  const loc = useLocale() as Locale;
  const num = (n: number) => formatNumber(n, loc);
  const dec = (n: number) => formatDecimal(String(Math.round(n * 10) / 10), loc);
  /** «+۱۰٫۲» with U+2212 for a loss, as the artboards print it. */
  const signed = (n: number) => {
    const v = Math.round(n * 10) / 10;
    return `${v > 0 ? '+' : v < 0 ? '−' : ''}${dec(Math.abs(v))}`;
  };
  return { loc, num, dec, signed };
}

/** Decorative teaser behind a Plus lock (no user data, hidden from assistive tech). */
function TeaserTiles() {
  return (
    <div className="apg-tiles" aria-hidden>
      {[0, 1, 2].map((i) => (
        <div key={i} className="apg-tile apg-teaser" />
      ))}
    </div>
  );
}

function WeightCard({ s, t, week }: { s: PregnancySections['weightGain']; t: T; week: number }) {
  const { loc, num, dec, signed } = useFmt();
  const w = s.data;
  // B-N3-14b (N3 stage smoke B-6): the header chip is today's week; a weigh-in from an earlier week is
  // named by its date so the card never reads «هفته ۲۳» next to «تا هفته ۲۲».
  const gainUntil = (cur: { date: string; week: number }, today: number) =>
    gainUntilWeek(cur.week, today)
      ? t('weight.until', { n: num(cur.week || today) })
      : t('weight.untilDate', { date: formatDayMonth(fromApiDate(cur.date), loc) });
  const statusTone = w?.status === 'within' ? 'data' : 'warm';
  return (
    <PregCard
      title={t('weight.title')}
      href="/analysis/pregnancy-weight"
      sub={
        w?.bmi != null && w.target
          ? t('weight.sub', { bmi: dec(w.bmi), min: dec(w.target.min), max: dec(w.target.max) })
          : undefined
      }
    >
      {s.ready && w?.current ? (
        <>
          <div className="apg-gain-head">
            <span className="apg-gain-num">
              <bdi dir="ltr">{signed(w.current.gain)}</bdi>
            </span>
            <span className="an-stat-unit">{gainUntil(w.current, week)}</span>
            {w.status ? (
              <StatusPill tone={statusTone} className="an-stat-pill">
                {t(`weight.status.${w.status}`)}
              </StatusPill>
            ) : null}
          </div>
          <GainChartView data={w} height={130} />
        </>
      ) : (
        <p className="an-card-note">{t(`weight.missing.${w?.missing ?? 'weights'}`)}</p>
      )}
    </PregCard>
  );
}

/** «فشار خون» (weekly pregnancy BP); opens the Vitals BP report (B-N6-02). */
function BloodPressureCard({ s, t }: { s: PregnancySections['bloodPressure']; t: T }) {
  const { loc, num } = useFmt();
  const b = s.data;
  if (!s.ready || !b) {
    return (
      <PregCard title={t('bp.title')} href="/vitals/bp">
        <p className="an-card-note">{t('bp.empty')}</p>
      </PregCard>
    );
  }
  const th = b.status === 'severe' ? b.severeThreshold : b.threshold;
  const bp = (r: { systolic: number; diastolic: number }) => `${num(r.systolic)}/${num(r.diastolic)}`;
  const text =
    b.status === 'severe'
      ? t('bp.severe', { sys: num(th.systolic), dia: num(th.diastolic) })
      : b.status === 'high'
        ? t('bp.high', { n: num(b.highCount), sys: num(th.systolic), dia: num(th.diastolic) })
        : t('bp.below', { sys: num(th.systolic), dia: num(th.diastolic) });
  return (
    <PregCard title={t('bp.title')} href="/vitals/bp">
      {b.readings.length > 1 ? (
        <BpChartView
          label={t('bp.chart', {
            n: num(b.readings.length),
            list: b.readings.map((r) => bp(r)).join(t('listSeparator')),
          })}
          table={{
            caption: t('bp.title'),
            columns: [t('bp.colDate'), t('bp.colValue')],
            rows: b.readings.map((r) => [formatDayMonth(fromApiDate(r.date), loc), bp(r)]),
          }}
          readings={b.readings}
          threshold={b.threshold.systolic}
        />
      ) : null}
      <p className={clsx('apg-line', b.status !== 'below_threshold' && 'is-alert')} role={b.status === 'severe' ? 'alert' : undefined}>
        {text}
      </p>
    </PregCard>
  );
}

function GlucoseCard({ s, t }: { s: PregnancySections['glucose']; t: T }) {
  const { num } = useFmt();
  const g = s.data;
  const above = g?.slots.some((x) => x.withinTarget === false);
  return (
    <PregCard title={t('glucose.title')} sub={t('glucose.sub')} plus locked={s.locked}>
      {s.locked ? (
        <TeaserTiles />
      ) : s.ready && g ? (
        <>
          <dl className="apg-tiles">
            {g.slots.map((slot) => (
              <div
                key={slot.slot}
                className={clsx('apg-tile', slot.slot === 'two_hour' ? 'is-warm' : 'is-data')}
                aria-label={
                  slot.avg != null
                    ? t('glucose.label', { slot: t(`glucose.slots.${slot.slot}`), avg: num(slot.avg), target: num(slot.targetMax) })
                    : undefined
                }
              >
                <dt className="apg-tile-label">{t(`glucose.slots.${slot.slot}`)}</dt>
                <dd className="apg-tile-value">{slot.avg != null ? num(slot.avg) : t('none')}</dd>
                <dd className="apg-tile-sub">
                  <bdi>{t('glucose.target', { n: num(slot.targetMax) })}</bdi>
                </dd>
              </div>
            ))}
          </dl>
          {above ? <p className="apg-line is-alert">{t('glucose.above')}</p> : null}
        </>
      ) : (
        <p className="an-card-note">{t('glucose.empty')}</p>
      )}
    </PregCard>
  );
}

function KicksCard({ s, t }: { s: PregnancySections['kicks']; t: T }) {
  const { loc, num } = useFmt();
  const k = s.data;
  if (!s.ready || !k) {
    return (
      <PregCard title={t('kicks.title')}>
        <p className="an-card-note">{t('kicks.empty', { week: num(k?.fromWeek ?? 28) })}</p>
      </PregCard>
    );
  }
  const keys = weekdayKeys(loc);
  const short = weekdayLabels(loc);
  const dayLabel = (date: string) => short[keys.indexOf(weekdayKeyOf(date))] ?? '';
  const timed = k.timedSessions > 0;
  const value = (d: (typeof k.days)[number]) => (timed ? d.minutes : d.count);
  const hours = num(k.windowMinutes / 60);
  const caption = timed ? t('kicks.caption', { n: num(k.target) }) : t('kicks.count', { n: num(k.target) });
  const valueText = (d: (typeof k.days)[number]) => {
    const v = value(d);
    if (v != null) return num(v);
    return d.count != null ? t('kicks.noTime') : t('none');
  };
  return (
    <PregCard title={t('kicks.title')}>
      <ColumnChart
        label={t('kicks.chart', {
          n: num(k.target),
          list: k.days.map((d) => `${dayLabel(d.date)} ${valueText(d)}`).join(t('listSeparator')),
        })}
        table={{
          caption: t('kicks.title'),
          columns: [t('kicks.colDay'), timed ? t('kicks.colMinutes') : t('kicks.title')],
          rows: k.days.map((d) => [formatDayMonth(fromApiDate(d.date), loc), valueText(d)]),
        }}
        columns={k.days.map((d, i) => ({
          key: d.date,
          label: dayLabel(d.date),
          value: value(d),
          tone: 'bloom',
          solid: i === k.days.length - 1,
        }))}
        floor={0.12}
        height={120}
      />
      <p className={clsx('apg-line', (k.allWithinWindow === false || k.lowCountDays > 0) && 'is-alert')}>
        {k.allWithinWindow === false || k.lowCountDays > 0
          ? t('kicks.notAll', { h: hours, n: num(k.target) })
          : timed
            ? `${caption}${t('dot')}${t('kicks.allWithin', { h: hours })}`
            : caption}
      </p>
    </PregCard>
  );
}

function SymptomsCard({ s, t }: { s: PregnancySections['symptoms']; t: T }) {
  const sy = s.data;
  return (
    <PregCard title={t('symptoms.title')} plus locked={s.locked}>
      {s.locked ? (
        <TeaserTiles />
      ) : s.ready && sy ? (
        <dl className="apg-tiles">
          {sy.trimesters.map((tri) => (
            <div key={tri.trimester} className={clsx('apg-tile is-plain', tri.isCurrent && 'is-current')}>
              <dt className="apg-tile-label">{t(`symptoms.trimesters.${String(tri.trimester) as '1' | '2' | '3'}`)}</dt>
              {tri.isFuture ? (
                <dd className="apg-tile-muted">{t('symptoms.future')}</dd>
              ) : tri.items.length ? (
                tileItems(tri.items).map((it) => (
                  <dd key={it.key} className="apg-tile-item">
                    {it.label}
                  </dd>
                ))
              ) : (
                <dd className="apg-tile-muted">{t('symptoms.nothing')}</dd>
              )}
            </div>
          ))}
        </dl>
      ) : (
        <p className="an-card-note">{t('symptoms.empty')}</p>
      )}
    </PregCard>
  );
}

function VisitsCard({ s, t }: { s: PregnancySections['visits']; t: T }) {
  const { num } = useFmt();
  const v = s.data;
  const parts: string[] = [];
  if (v && s.ready) {
    parts.push(t('visits.done', { n: num(v.done) }));
    if (v.next) parts.push(t('visits.next', { title: v.next.title, week: num(v.next.week) }));
    if (v.latestResult) parts.push(t('visits.result', { title: v.latestResult.title, note: v.latestResult.note }));
  }
  return (
    <PregCard title={t('visits.title')} href="/pregnancy/calendar">
      <p className="apg-line">{parts.length ? parts.join(t('dot')) : t('visits.empty')}</p>
    </PregCard>
  );
}

/**
 * The pregnancy analysis hub (An_Hub_Preg, B-N3-12) rendered by `/analysis`
 * in pregnancy mode: weight gain against the IOM band, blood pressure against
 * 140/90, GDM glucose targets and symptoms by trimester (Plus), kick counts
 * and visits. Descriptive only — every alert line says «talk to your doctor».
 */
export function AnalysisPregnancyHub() {
  const t = useTranslations('analysisPregnancy.hub');
  const router = useRouter();
  const mounted = useMounted();
  const { num } = useFmt();
  const query = usePregnancyAnalysis();
  const r = query.data;

  let body;
  if (!mounted || (query.isPending && query.fetchStatus !== 'idle')) {
    body = (
      <SkeletonGroup label={t('loading')} className="an-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError && isPregnancyInactive(query.error)) {
    body = (
      <EmptyState
        icon="heart"
        title={t('inactive.title')}
        body={t('inactive.body')}
        action={
          <PrimaryButton block={false} onClick={() => router.push('/pregnancy/setup')}>
            {t('inactive.cta')}
          </PrimaryButton>
        }
      />
    );
  } else if (query.isError || !r) {
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
    const s = r.sections;
    body = (
      <div className="an-content">
        <WeightCard s={s.weightGain} t={t} week={r.pregnancy.week} />
        <BloodPressureCard s={s.bloodPressure} t={t} />
        <GlucoseCard s={s.glucose} t={t} />
        <KicksCard s={s.kicks} t={t} />
        <SymptomsCard s={s.symptoms} t={t} />
        <VisitsCard s={s.visits} t={t} />
        <InfoNote>{t('footnote')}</InfoNote>
      </div>
    );
  }

  return (
    <div className="an-hub">
      <header className="apg-head">
        <h1 className="an-title">{t('title')}</h1>
        {r ? <span className="apg-week">{t('week', { n: num(r.pregnancy.week) })}</span> : null}
      </header>
      {body}
    </div>
  );
}
