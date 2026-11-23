'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { feedingHref, hoursText } from '@/features/baby-log';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate, weekdayKeys, weekdayLabels, WEEKDAY_KEYS } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { EmptyState, InfoNote, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';
import { ChartFigure, ColumnChart, TrendLine } from '@/widgets/charts';

import {
  useHubReport,
  type BleedingCardData,
  type EpdsCardData,
  type FeedsCardData,
  type GrowthCardData,
  type HubCard,
  type SleepCardData,
  type WeightCardData,
} from '../api/report';
import { epdsTrend } from '../model/hub';
import { EpdsChart } from './EpdsChart';
import { PostCard } from './PostCard';
import '@/shared/ui/styles/an.css';
import '@/shared/ui/styles/apg.css';
import './anp.css';

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

/** «Log N more days» under a card that is not ready yet. */
function NeedMore({ n, t }: { n: number | null; t: T }) {
  const { num } = useFmt();
  if (!n) return null;
  return <p className="an-card-note">{t('daysNeeded', { count: n, n: num(n) })}</p>;
}

// ── EPDS (never Plus) ──────────────────────────────────────────
function EpdsCard({ data, t }: { data: EpdsCardData | null; t: T }) {
  const { loc, num } = useFmt();
  const trend = data ? epdsTrend(data.history) : null;
  const rule = t('epds.rule', { likely: num(data?.history.thresholds.fullLikely ?? 13) });
  let body;
  if (!data || !trend) body = <p className="an-card-note">{t('epds.empty')}</p>;
  else {
    const n = trend.points.length;
    const step = Math.max(1, Math.ceil(n / 5));
    const weekText = (w: number | null, date: string) => (w ? t('epds.week', { n: num(w) }) : formatDayMonth(fromApiDate(date), loc));
    const axis = trend.points.map((p, i) => (i % step === 0 || i === n - 1 ? weekText(p.week, p.takenOn) : ''));
    const latest = trend.latest;
    let line: string;
    if (trend.status === 'down' && trend.peak) {
      line = t('epds.down', { week: weekText(trend.peak.week, trend.peak.takenOn), score: num(trend.peak.total), th: num(trend.lower) });
    } else if (trend.status === 'urgent' || trend.status === 'high') {
      line = t('epds.high', { score: num(latest.total), th: num(trend.lower) });
    } else {
      line = t('epds.low', { score: num(latest.total), max: num(latest.max) });
    }
    body = (
      <>
        {data.trendReady ? (
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
          {line} {rule}
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
function BleedingCard({ card, t }: { card: HubCard<BleedingCardData>; t: T }) {
  const { loc, num } = useFmt();
  const b = card.data;
  let body;
  if (!b || !b.logged) body = <p className="an-card-note">{t('bleeding.empty')}</p>;
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
        {!card.ready ? <NeedMore n={card.daysNeeded} t={t} /> : null}
      </>
    );
  }
  return (
    // A red flag today goes to the postpartum home, where the urgent alerts and their call buttons live.
    <PostCard title={t('bleeding.title')} href={b?.alertToday ? '/postpartum' : '/postpartum/recovery'}>
      {b?.alertToday ? (
        <p className="apg-line is-alert" role="alert">
          {t('bleeding.alertToday')}
        </p>
      ) : null}
      {body}
    </PostCard>
  );
}

// ── Feeds ──────────────────────────────────────────────────────
function FeedsCard({ card, childrenAvailable, t }: { card: HubCard<FeedsCardData>; childrenAvailable: boolean; t: T }) {
  const { loc, num, dec } = useFmt();
  const router = useRouter();
  const weekday = useWeekday();
  const f = card.data;
  if (f && f.childId === null && f.logged === 0 && childrenAvailable) {
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
  if (!f || !f.logged) body = <p className="an-card-note">{t('feeds.empty')}</p>;
  else {
    const line = f.feedsPerDay !== null ? [t('feeds.avg', { n: dec(f.feedsPerDay) })] : [];
    if (f.leftPercent !== null && f.rightPercent !== null) {
      line.push(t('feeds.split', { l: num(Math.round(f.leftPercent)), r: num(Math.round(f.rightPercent)) }));
    }
    body = (
      <>
        <ColumnChart
          label={t('feeds.chart', { list: f.days.map((d) => `${weekday(d.date)} ${num(d.count ?? 0)}`).join(t('listSeparator')) })}
          table={{
            caption: t('feeds.title'),
            columns: [t('feeds.colDay'), t('feeds.colCount')],
            rows: f.days.map((d) => [formatDayMonth(fromApiDate(d.date), loc), num(d.count ?? 0)]),
          }}
          columns={f.days.map((d, i) => ({
            key: d.date,
            label: weekday(d.date),
            value: d.count ?? 0,
            tone: 'data',
            solid: i === f.days.length - 1,
          }))}
          floor={0.12}
          height={84}
        />
        <p className="apg-line">{line.join(t('dot'))}</p>
        {!card.ready ? <NeedMore n={card.daysNeeded} t={t} /> : null}
      </>
    );
  }
  return (
    <PostCard title={t('feeds.title')} href={f?.childId ? feedingHref(f.childId) : undefined}>
      {body}
    </PostCard>
  );
}

// ── Sleep (Plus) ───────────────────────────────────────────────
function SleepCard({ card, t }: { card: HubCard<SleepCardData>; t: T }) {
  const { loc, dec } = useFmt();
  const s = card.data;
  return (
    <PostCard title={t('sleep.title')} plus locked={card.locked}>
      <dl className="anp-sleep">
        <div className="anp-sleep-tile is-mother">
          <dt>{t('sleep.you')}</dt>
          <dd className="anp-sleep-value">{card.locked ? ' ' : s?.motherHours != null ? dec(s.motherHours) : t('none')}</dd>
          <dd className="anp-sleep-unit">{t('sleep.hours')}</dd>
        </div>
        <div className="anp-sleep-tile is-baby">
          <dt>{s?.babyName ?? t('sleep.baby')}</dt>
          <dd className="anp-sleep-value">{card.locked ? ' ' : s?.babySeconds != null ? hoursText(s.babySeconds, loc) : t('none')}</dd>
          <dd className="anp-sleep-unit">{t('sleep.hours')}</dd>
        </div>
      </dl>
      {!card.locked ? <p className="apg-line">{t('sleep.note')}</p> : null}
      {!card.locked && !card.ready ? <NeedMore n={card.daysNeeded} t={t} /> : null}
    </PostCard>
  );
}

// ── Growth ─────────────────────────────────────────────────────
function GrowthCard({ card, t }: { card: HubCard<GrowthCardData>; t: T }) {
  const { num } = useFmt();
  const g = card.data;
  if (!g || !g.available || !g.child) return null;
  const latest = g.latest;
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
    <PostCard title={t('growth.title', { name: g.child.name })} href={`/children/${g.child.id}/growth`}>
      {chips.length ? (
        <>
          <ul className="anp-chips">
            {chips.map((c) => (
              <li key={c.key} className={clsx('anp-chip', c.inBand === false && 'is-check')}>
                {c.text}
              </li>
            ))}
          </ul>
          {g.crossings.map((label) => (
            <p key={label} className="apg-line is-alert">
              {label}
            </p>
          ))}
          <p className="apg-line">{t('growth.who')}</p>
        </>
      ) : (
        <p className="an-card-note">{t('growth.empty')}</p>
      )}
    </PostCard>
  );
}

// ── Recovery weight (Plus) ─────────────────────────────────────
function WeightCard({ card, t }: { card: HubCard<WeightCardData>; t: T }) {
  const { loc, num, dec, signed } = useFmt();
  const w = card.data;
  let body;
  // A locked card computes nothing: a neutral block sits behind the Plus gate.
  if (card.locked) body = <Skeleton shape="block" />;
  else if (!w || !w.points.length) body = <p className="an-card-note">{t('weight.empty')}</p>;
  else if (w.points.length < 2) body = <p className="an-card-note">{t('weight.one', { kg: dec(w.points[0].kg) })}</p>;
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
        {w.deltaSinceFirst !== null ? (
          <p className="apg-line">
            <bdi dir="ltr">{signed(w.deltaSinceFirst)}</bdi> {t('weight.change', { n: num(w.fromWeek ?? 1) })}
          </p>
        ) : null}
        {w.deltaVsPre !== null ? (
          <p className="apg-line">
            <bdi dir="ltr">{signed(w.deltaVsPre)}</bdi> {t('weight.vsPre')}
          </p>
        ) : null}
      </>
    );
  return (
    <PostCard title={t('weight.title')} plus locked={card.locked}>
      {body}
    </PostCard>
  );
}

/**
 * The postpartum analysis hub (An_Hub_Post) rendered by `/analysis` in
 * postpartum mode. Every card comes from GET /analysis/postpartum (T-M10-27):
 * EPDS trend (never Plus), lochia over three weeks, feeds, mother's and baby's
 * sleep (Plus), the baby's WHO percentiles with the crossing hint, and the
 * mother's weight (Plus). On Laravel (404) only the EPDS card shows, from
 * GET /postpartum/epds. Descriptive only — every alert line points to the
 * doctor or to the existing urgent path.
 */
export function AnalysisPostpartumHub() {
  const t = useTranslations('analysisPostpartum.hub');
  const router = useRouter();
  const mounted = useMounted();
  const { num } = useFmt();
  const report = useHubReport();
  const r = report.data;

  let body;
  let week: number | null = null;
  if (!mounted || report.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="an-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (report.isError || !r) {
    body = (
      <EmptyState
        icon="warning"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <SecondaryButton icon="refresh" onClick={() => void report.refetch()} loading={report.isFetching}>
            {t('error.retry')}
          </SecondaryButton>
        }
      />
    );
  } else if (r.kind === 'inactive') {
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
  } else if (r.kind === 'legacy') {
    body = (
      <div className="an-content">
        <EpdsCard data={{ history: r.epds, trendReady: r.epds.checks.length > 1 }} t={t} />
        <InfoNote>{t('footnote')}</InfoNote>
      </div>
    );
  } else {
    week = r.week;
    const c = r.cards;
    body = (
      <div className="an-content">
        <EpdsCard data={c.epds.data} t={t} />
        <BleedingCard card={c.bleeding} t={t} />
        <FeedsCard card={c.breastfeeding} childrenAvailable={c.growth.data?.available ?? false} t={t} />
        <SleepCard card={c.sleep} t={t} />
        <GrowthCard card={c.growth} t={t} />
        <WeightCard card={c.weight} t={t} />
        <InfoNote>{t('footnote')}</InfoNote>
      </div>
    );
  }

  return (
    <div className="an-hub">
      <header className="apg-head">
        <h1 className="an-title">{t('title')}</h1>
        {week ? <span className="apg-week anp-week">{t('week', { n: num(week) })}</span> : null}
      </header>
      {body}
    </div>
  );
}
