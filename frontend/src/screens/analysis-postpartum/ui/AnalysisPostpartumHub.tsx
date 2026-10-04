'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useChild, useChildren, type ChildHome } from '@/entities/child';
import { useLogDays, type LogDay } from '@/entities/health-log';
import { usePlusLocked } from '@/entities/plus';
import { usePostpartum } from '@/entities/postpartum';
import { feedingHref, hoursText, useBabyLogSummary, type BabyLogSummary } from '@/features/baby-log';
import { useRouter, type Locale } from '@/shared/i18n';
import {
  formatDayMonth,
  formatDecimal,
  formatNumber,
  fromApiDate,
  toApiDate,
  today as todayDate,
  weekdayKeys,
  weekdayLabels,
  WEEKDAY_KEYS,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, InfoNote, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';
import { ChartFigure, ColumnChart, TrendLine } from '@/widgets/charts';

import { useEpdsHistory, type EpdsHistory } from '../api/epds';
import { bleedingTrend, epdsTrend, hubChildId, hubRange, motherSleepAverage, weekAfterBirth, weightTrend } from '../model/hub';
import { EpdsChart } from './EpdsChart';
import { PostCard } from './PostCard';

type T = ReturnType<typeof useTranslations<'analysisPostpartum.hub'>>;

function useFmt() {
  const loc = useLocale() as Locale;
  const num = (n: number) => formatNumber(n, loc);
  const dec = (n: number) => formatDecimal(String(Math.round(n * 10) / 10), loc);
  /** «−۴٫۴» with U+2212 for a loss, as the artboard prints it. */
  const signed = (n: number) => `${n > 0 ? '+' : n < 0 ? '−' : ''}${dec(Math.abs(n))}`;
  return { loc, num, dec, signed };
}

/** Weekday letter of a `Y-m-d` in the locale's grid order. */
function useWeekday() {
  const loc = useLocale() as Locale;
  const keys = weekdayKeys(loc);
  const short = weekdayLabels(loc);
  return (date: string) => {
    // getUTCDay: 0 = Sunday; WEEKDAY_KEYS starts on Saturday.
    const [y, m, d] = date.split('-').map(Number);
    const key = WEEKDAY_KEYS[(new Date(Date.UTC(y, m - 1, d)).getUTCDay() + 1) % 7];
    return short[keys.indexOf(key)] ?? '';
  };
}

function CardSkeleton() {
  return <Skeleton shape="block" />;
}

// ── EPDS ───────────────────────────────────────────────────────
function EpdsCard({ history, pending, error, t }: { history: EpdsHistory | undefined; pending: boolean; error: boolean; t: T }) {
  const { loc, num } = useFmt();
  const trend = history ? epdsTrend(history) : null;
  const rule = t('epds.rule', { likely: num(history?.thresholds.fullLikely ?? 13) });
  let body;
  if (pending) body = <CardSkeleton />;
  else if (error) body = <p className="an-card-note">{t('cardError')}</p>;
  else if (!trend) body = <p className="an-card-note">{t('epds.empty')}</p>;
  else {
    const n = trend.points.length;
    const step = Math.max(1, Math.ceil(n / 5));
    const weekText = (w: number | null, date: string) => (w ? t('epds.week', { n: num(w) }) : formatDayMonth(fromApiDate(date), loc));
    const axis = trend.points.map((p, i) => (i % step === 0 || i === n - 1 ? weekText(p.week, p.takenOn) : ''));
    const latest = trend.latest;
    const lines: string[] = [];
    if (trend.status === 'down' && trend.peak) {
      lines.push(t('epds.down', { week: weekText(trend.peak.week, trend.peak.takenOn), score: num(trend.peak.total), th: num(trend.lower) }));
    } else if (trend.status === 'urgent' || trend.status === 'high') {
      lines.push(t('epds.high', { score: num(latest.total), th: num(trend.lower) }));
    } else {
      lines.push(t('epds.low', { score: num(latest.total), max: num(latest.max) }));
    }
    body = (
      <>
        {n > 1 ? (
          <EpdsChart
            trend={trend}
            label={t('epds.chart', {
              list: trend.points.map((p) => `${weekText(p.week, p.takenOn)} ${num(p.total)}`).join(t('listSeparator')),
            })}
            table={{
              caption: t('epds.title'),
              columns: [t('epds.colWeek'), t('epds.colScore')],
              rows: trend.points.map((p) => [weekText(p.week, p.takenOn), num(p.total)]),
            }}
            axis={axis}
            ticks={{ lower: num(trend.lower), upper: trend.upper !== null ? num(trend.upper) : null }}
          />
        ) : null}
        <p className={clsx('apg-line', trend.status === 'urgent' && 'is-alert')} role={trend.status === 'urgent' ? 'alert' : undefined}>
          {lines.join(' ')} {rule}
        </p>
      </>
    );
  }
  return (
    <PostCard title={t('epds.title')} sub={t('epds.sub')} href="/postpartum/mood">
      {body}
    </PostCard>
  );
}

// ── Bleeding ───────────────────────────────────────────────────
function BleedingCard({ days, today, birth, pending, error, t }: { days: readonly LogDay[]; today: string; birth: string | null; pending: boolean; error: boolean; t: T }) {
  const { loc, num } = useFmt();
  let body;
  if (pending) body = <CardSkeleton />;
  else if (error) body = <p className="an-card-note">{t('cardError')}</p>;
  else {
    const b = bleedingTrend(days, today, birth);
    if (!b.logged) body = <p className="an-card-note">{t('bleeding.empty')}</p>;
    else {
      const amount = (code: string | null) => t(`bleeding.amounts.${(code ?? 'unlogged') as 'none'}`);
      const parts = [
        b.trend ? t(`bleeding.trend.${b.trend}`, { n: num(b.weeks) }) : t('bleeding.few'),
        b.largeClots ? t('bleeding.clots') : t('bleeding.noClots'),
      ];
      body = (
        <>
          <ChartFigure
            label={t('bleeding.chart', {
              n: num(b.cells.length),
              list: b.cells.map((c) => `${formatDayMonth(fromApiDate(c.date), loc)} ${amount(c.code)}`).join(t('listSeparator')),
            })}
          >
            <div className="anp-strip" aria-hidden>
              {b.cells.map((c) => (
                <span key={c.date} className={clsx('anp-strip-cell', c.level === null ? 'is-empty' : `is-l${c.level}`)} />
              ))}
            </div>
          </ChartFigure>
          <p className={clsx('apg-line', (b.largeClots || b.trend === 'increasing') && 'is-alert')}>{parts.join(t('dot'))}</p>
        </>
      );
    }
  }
  return (
    <PostCard title={t('bleeding.title')} href="/postpartum/recovery">
      {body}
    </PostCard>
  );
}

// ── Feeds ──────────────────────────────────────────────────────
function FeedsCard({ childId, summary, pending, error, t }: { childId: number | null; summary: BabyLogSummary | undefined; pending: boolean; error: boolean; t: T }) {
  const { loc, num, dec } = useFmt();
  const router = useRouter();
  const weekday = useWeekday();
  if (!childId && !pending) {
    return (
      <PostCard title={t('feeds.title')}>
        <p className="an-card-note">{t('feeds.noChild')}</p>
        <SecondaryButton icon="plus" block={false} onClick={() => router.push('/children/new')}>
          {t('feeds.addChild')}
        </SecondaryButton>
      </PostCard>
    );
  }
  let body;
  if (pending) body = <CardSkeleton />;
  else if (error || !summary) body = <p className="an-card-note">{t('cardError')}</p>;
  else if (!summary.days.some((d) => d.feeds.count > 0)) body = <p className="an-card-note">{t('feeds.empty')}</p>;
  else {
    const a = summary.averages;
    const line = [t('feeds.avg', { n: dec(a.feedsPerDay) })];
    if (a.leftPercent !== null && a.rightPercent !== null) {
      line.push(t('feeds.split', { l: num(Math.round(a.leftPercent)), r: num(Math.round(a.rightPercent)) }));
    }
    body = (
      <>
        <ColumnChart
          label={t('feeds.chart', { list: summary.days.map((d) => `${weekday(d.date)} ${num(d.feeds.count)}`).join(t('listSeparator')) })}
          table={{
            caption: t('feeds.title'),
            columns: [t('feeds.colDay'), t('feeds.colCount')],
            rows: summary.days.map((d) => [formatDayMonth(fromApiDate(d.date), loc), num(d.feeds.count)]),
          }}
          columns={summary.days.map((d, i) => ({
            key: d.date,
            label: weekday(d.date),
            value: d.feeds.count,
            tone: 'data',
            solid: i === summary.days.length - 1,
          }))}
          floor={0.12}
          height={84}
        />
        <p className="apg-line">{line.join(t('dot'))}</p>
      </>
    );
  }
  return (
    <PostCard title={t('feeds.title')} href={childId ? feedingHref(childId) : undefined}>
      {body}
    </PostCard>
  );
}

// ── Sleep (Plus) ───────────────────────────────────────────────
function SleepCard({
  mother,
  babySeconds,
  babyName,
  locked,
  pending,
  t,
}: {
  mother: number | null;
  babySeconds: number | null;
  babyName: string | null;
  locked: boolean;
  pending: boolean;
  t: T;
}) {
  const { loc, dec } = useFmt();
  const tiles = (
    <dl className="anp-sleep">
      <div className="anp-sleep-tile is-mother">
        <dt>{t('sleep.you')}</dt>
        <dd className="anp-sleep-value">{locked ? ' ' : mother !== null ? dec(mother) : t('none')}</dd>
        <dd className="anp-sleep-unit">{t('sleep.hours')}</dd>
      </div>
      <div className="anp-sleep-tile is-baby">
        <dt>{babyName ?? t('sleep.baby')}</dt>
        <dd className="anp-sleep-value">{locked ? ' ' : babySeconds !== null ? hoursText(babySeconds, loc) : t('none')}</dd>
        <dd className="anp-sleep-unit">{t('sleep.hours')}</dd>
      </div>
    </dl>
  );
  return (
    <PostCard title={t('sleep.title')} plus locked={locked}>
      {pending && !locked ? <CardSkeleton /> : tiles}
      {!locked && !pending ? <p className="apg-line">{t('sleep.note')}</p> : null}
    </PostCard>
  );
}

// ── Growth ─────────────────────────────────────────────────────
function GrowthCard({ child, pending, t }: { child: ChildHome | undefined; pending: boolean; t: T }) {
  const { num } = useFmt();
  if (!child && !pending) return null;
  const latest = child?.latest;
  const chips = latest
    ? (
        [
          ['weight', latest.weight],
          ['length', latest.length],
          ['head', latest.head],
        ] as const
      ).flatMap(([key, v]) => (v?.percentile != null ? [{ key, text: t(`growth.${key}`, { n: num(Math.round(v.percentile)) }), inBand: v.inBand }] : []))
    : [];
  return (
    <PostCard title={child ? t('growth.title', { name: child.name }) : t('growth.titleNoName')} href={child ? `/children/${child.id}/growth` : undefined}>
      {pending ? (
        <CardSkeleton />
      ) : chips.length ? (
        <>
          <ul className="anp-chips">
            {chips.map((c) => (
              <li key={c.key} className={clsx('anp-chip', c.inBand === false && 'is-check')}>
                {c.text}
              </li>
            ))}
          </ul>
          <p className="apg-line">{t('growth.who')}</p>
        </>
      ) : (
        <p className="an-card-note">{t('growth.empty')}</p>
      )}
    </PostCard>
  );
}

// ── Recovery weight ────────────────────────────────────────────
function WeightCard({ days, birth, pending, error, t }: { days: readonly LogDay[]; birth: string | null; pending: boolean; error: boolean; t: T }) {
  const { loc, num, dec, signed } = useFmt();
  let body;
  if (pending) body = <CardSkeleton />;
  else if (error) body = <p className="an-card-note">{t('cardError')}</p>;
  else {
    const w = weightTrend(days, birth);
    if (!w) body = <p className="an-card-note">{t('weight.empty')}</p>;
    else if (w.points.length < 2) body = <p className="an-card-note">{t('weight.one', { kg: dec(w.first.kg) })}</p>;
    else
      body = (
        <>
          <TrendLine
            label={t('weight.chart', { list: w.points.map((p) => `${formatDayMonth(fromApiDate(p.date), loc)} ${dec(p.kg)}`).join(t('listSeparator')) })}
            table={{
              caption: t('weight.title'),
              columns: [t('weight.colDate'), t('weight.colKg')],
              rows: w.points.map((p) => [formatDayMonth(fromApiDate(p.date), loc), dec(p.kg)]),
            }}
            values={w.points.map((p) => p.kg)}
            height={80}
          />
          <p className="apg-line">
            <bdi dir="ltr">{signed(w.delta)}</bdi> {t('weight.change', { n: num(w.fromWeek) })}
          </p>
        </>
      );
  }
  return <PostCard title={t('weight.title')}>{body}</PostCard>;
}

/**
 * The postpartum analysis hub (An_Hub_Post, B-N5-07) rendered by `/analysis`
 * in postpartum mode: EPDS trend, lochia over three weeks, feeds with the L/R
 * split, mother's and baby's sleep (Plus), the baby's WHO percentiles and the
 * mother's weight since the birth. Computed client-side from existing reads;
 * descriptive only — every alert line points to the doctor.
 */
export function AnalysisPostpartumHub() {
  const t = useTranslations('analysisPostpartum.hub');
  const router = useRouter();
  const mounted = useMounted();
  const { num } = useFmt();
  const overview = usePostpartum();
  const today = toApiDate(todayDate());
  const birth = overview.data?.profile?.birthDate ?? null;
  const active = overview.data?.active ?? false;
  const range = hubRange(today, birth);
  const logDays = useLogDays(range.from, range.to, active);
  const epds = useEpdsHistory();
  const list = useChildren({ enabled: active });
  const childId = list.data ? hubChildId(list.data.children, birth) : null;
  const child = useChild(childId);
  const summary = useBabyLogSummary(childId, 7);
  const sleepLocked = usePlusLocked('plus.deep_analysis');

  const week = overview.data?.status?.week ?? (birth ? weekAfterBirth(today, birth) : null);
  const days = logDays.data?.days ?? [];
  const logsPending = logDays.isPending && logDays.fetchStatus !== 'idle';
  const childPending = list.isPending || (childId !== null && child.isPending);

  let body;
  if (!mounted || overview.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="an-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (overview.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <SecondaryButton icon="refresh" onClick={() => void overview.refetch()} loading={overview.isFetching}>
            {t('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (!active) {
    body = (
      <EmptyState
        icon="heart"
        title={t('inactive.title')}
        body={t('inactive.body')}
        action={
          <PrimaryButton block={false} onClick={() => router.push('/postpartum/setup')}>
            {t('inactive.cta')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = (
      <div className="an-content">
        <EpdsCard history={epds.data} pending={epds.isPending} error={epds.isError} t={t} />
        <BleedingCard days={days} today={today} birth={birth} pending={logsPending} error={logDays.isError} t={t} />
        <FeedsCard
          childId={childId}
          summary={summary.data}
          pending={list.isPending || (childId !== null && summary.isPending)}
          error={summary.isError}
          t={t}
        />
        <SleepCard
          mother={motherSleepAverage(days, today)}
          babySeconds={summary.data && summary.data.days.some((d) => d.sleep.count > 0) ? summary.data.averages.sleepSecondsPerDay : null}
          babyName={child.data?.name ?? null}
          locked={sleepLocked}
          pending={logsPending}
          t={t}
        />
        <GrowthCard child={child.data} pending={childPending} t={t} />
        <WeightCard days={days} birth={birth} pending={logsPending} error={logDays.isError} t={t} />
        <InfoNote>{t('footnote')}</InfoNote>
      </div>
    );
  }

  return (
    <div className="an-hub">
      <header className="apg-head">
        <h1 className="an-title">{t('title')}</h1>
        {active && week ? <span className="apg-week anp-week">{t('week', { n: num(week) })}</span> : null}
      </header>
      {body}
    </div>
  );
}
