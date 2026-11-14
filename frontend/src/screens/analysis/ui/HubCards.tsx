'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import {
  barShares,
  lowestIndex,
  shareOfMax,
  stripCells,
  type AnalysisSection,
  type Correlation,
  type HubCycle,
  type HubRecentCycle,
  type HubSymptoms,
  type HubVitals,
  type HubWeight,
  type MoodByPhase,
} from '@/entities/analysis';
import type { Locale } from '@/shared/i18n';
import { formatDecimal, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import { BarChart, LineChart, StatusPill, type Tone } from '@/shared/ui';

import { signedDecimal } from '../model/hub';
import { HubCard, NotReady } from './HubCard';

type T = ReturnType<typeof useTranslations<'analysis.hub'>>;

function useHub(): { t: T; loc: Locale; num: (n: number) => string; dec: (n: number) => string } {
  const t = useTranslations('analysis.hub');
  const loc = useLocale() as Locale;
  return {
    t,
    loc,
    num: (n: number) => formatNumber(n, loc),
    dec: (n: number) => formatDecimal(String(Math.round(n * 10) / 10), loc),
  };
}

const startMonth = (iso: string, loc: Locale): string => monthName(toParts(fromApiDate(iso), loc).month, loc);

/** «طول سیکل و پریود»: median cycle (Lalezar), period median, regularity pill, last 6 cycle bars. */
export function CycleCard({ section }: { section: AnalysisSection<HubCycle> }) {
  const { t, loc, num } = useHub();
  const c = section.data;
  const bars = c?.bars ?? [];
  const shares = barShares(bars.map((b) => b.length));
  const abbr = Number(t('cycle.monthAbbrLength')) || 3;
  const pill =
    c?.regularity === 'regular'
      ? { tone: 'data' as Tone, icon: 'check' as const, label: t('cycle.regular') }
      : c?.regularity === 'irregular'
        ? { tone: 'warm' as Tone, icon: undefined, label: t('cycle.irregular') }
        : null;
  return (
    <HubCard title={t('cycle.title')} href="/analysis/cycle">
      {section.ready && c?.medianCycle != null ? (
        <span className="an-stat">
          <span className="an-num">{num(c.medianCycle)}</span>
          <span className="an-stat-unit">
            {c.medianPeriod != null ? t('cycle.medianPeriod', { days: num(c.medianPeriod) }) : t('cycle.median')}
          </span>
          {pill ? (
            <StatusPill tone={pill.tone} icon={pill.icon} className="an-stat-pill">
              {pill.label}
            </StatusPill>
          ) : null}
        </span>
      ) : (
        <NotReady>{t('cycle.notReady')}</NotReady>
      )}
      {bars.length ? (
        <span
          className="an-bars"
          role="img"
          aria-label={t('cycle.chart', { count: num(bars.length), list: bars.map((b) => num(b.length)).join('، ') })}
        >
          <span className="an-bars-row" aria-hidden>
            {bars.map((b, i) => (
              <span
                key={b.start}
                className={clsx('an-bar', i === bars.length - 1 && 'is-last', !b.inFigoRange && 'is-out')}
                style={{ blockSize: `${Math.round(shares[i] * 100)}%` }}
              />
            ))}
          </span>
          <span className="an-bars-labels" aria-hidden>
            {bars.map((b) => (
              <span key={b.start}>{startMonth(b.start, loc).slice(0, abbr)}</span>
            ))}
          </span>
        </span>
      ) : null}
    </HubCard>
  );
}

/** «پریود و تاریخچه سیکل‌ها»: the last 3 cycles as day dots (period / fertile / ovulation). */
export function RecentCyclesCard({
  section,
  hideFertility,
}: {
  section: AnalysisSection<HubRecentCycle[]>;
  /** Teen and menopause: period days only (no fertility content, B-N2-03). */
  hideFertility: boolean;
}) {
  const { t, loc, num } = useHub();
  const cycles = section.data ?? [];
  return (
    <HubCard title={t('period.title')} href="/analysis/period">
      <ul className="an-strips">
        {cycles.map((c) => {
          const month = startMonth(c.start, loc);
          const text = c.isCurrent
            ? t('period.rowCurrent', {
                month,
                day: num(c.daysSoFar ?? 1),
                length: num(c.length),
                period: num(c.periodDays),
              })
            : t('period.row', { month, length: num(c.length), period: num(c.periodDays) });
          return (
            <li key={c.start} className="an-strip">
              <span className="an-strip-month" aria-hidden>
                {month}
              </span>
              <span className="an-strip-dots" aria-hidden style={{ ['--an-days' as string]: c.length }}>
                {stripCells(c).map((cell) => {
                  const kind = hideFertility && cell.kind !== 'period' ? 'other' : cell.kind;
                  return <span key={cell.day} className={clsx('an-dot', `is-${kind}`, cell.future && 'is-future')} />;
                })}
              </span>
              <span className="sr-only">{text}</span>
            </li>
          );
        })}
      </ul>
    </HubCard>
  );
}

const SYMPTOM_TONES = ['bloom', 'brand', 'warm'] as const;

/** «الگوی علائم»: the strongest pattern sentence + the top 3 symptoms as bars. */
export function SymptomsCard({ section, cycleNudge = true }: { section: AnalysisSection<HubSymptoms>; cycleNudge?: boolean }) {
  const { t, num } = useHub();
  const s = section.data;
  const top = s?.top ?? [];
  const shares = shareOfMax(top.map((x) => x.days));
  const h = s?.highlight ?? null;
  const missing = s ? Math.max(0, s.cyclesNeeded - s.cyclesCounted) : 0;
  const highlightKey = h
    ? h.relation === 'before_period' || h.relation === 'early'
      ? h.relation
      : h.startDay === h.endDay
        ? 'mid_day'
        : 'mid'
    : null;
  return (
    <HubCard
      title={t('symptoms.title')}
      sub={s && s.cyclesCounted > 0 ? t('symptoms.sub', { n: num(s.cyclesCounted) }) : undefined}
      href="/analysis/symptoms"
    >
      {h && highlightKey ? (
        <p className="an-highlight">
          {t.rich(`symptoms.highlight.${highlightKey}`, {
            symptom: h.label,
            days: num(h.days ?? 0),
            start: num(h.startDay),
            end: num(h.endDay),
            b: (chunks) => <b>{chunks}</b>,
          })}
        </p>
      ) : missing > 0 ? (
        <NotReady>{cycleNudge ? t('symptoms.notReady', { n: num(missing) }) : t('symptoms.notReadyNoCycle')}</NotReady>
      ) : null}
      {top.length ? (
        <ul className="an-hbars">
          {top.map((x, i) => (
            <li key={x.key} className={clsx('an-hbar', `nb-tone-${SYMPTOM_TONES[i % SYMPTOM_TONES.length]}`)}>
              <span className="an-hbar-label">{x.label}</span>
              <span className="an-hbar-track" aria-hidden>
                <span className="an-hbar-fill" style={{ inlineSize: `${shares[i]}%` }} />
              </span>
              <span className="an-hbar-value">
                <span aria-hidden>{num(x.days)}</span>
                <span className="sr-only">{t('symptoms.count', { n: num(x.days) })}</span>
              </span>
            </li>
          ))}
        </ul>
      ) : !h && missing === 0 ? (
        <NotReady>{t('symptoms.none')}</NotReady>
      ) : null}
    </HubCard>
  );
}

/** Decorative teaser for a locked card (blurred, hidden from assistive tech; no user data). */
function TeaserBars() {
  return (
    <BarChart
      label=""
      height={110}
      max={100}
      bars={[60, 85, 70, 45].map((value, i) => ({ label: '', value, tone: i === 3 ? 'bloom' : 'data' }))}
    />
  );
}

function TeaserLine({ tone }: { tone: Tone }) {
  return <LineChart label="" height={100} series={[{ values: [62, 70, 55, 44, 74, 66, 58], tone }]} highlightIndex={6} />;
}

/**
 * «حال در فازهای سیکل» (Plus, §4.1): share of good-mood days per phase — only
 * phases with 10 mood-logged days carry a value (the rest read «—»); the
 * lowest phase is marked and the most frequent mood of each is listed.
 */
export function MoodPhaseCard({ section }: { section: AnalysisSection<MoodByPhase> }) {
  const { t, num } = useHub();
  const m = section.data;
  const phases = m?.phases ?? [];
  const tops = phases.filter((p) => p.ready && p.moods.length > 0);
  const low = lowestIndex(phases.map((p) => p.goodPct));
  const label = (p: (typeof phases)[number]) => t(`mood.phases.${p.phase}`);
  return (
    <HubCard title={t('mood.title')} plus locked={section.locked} href="/analysis/correlations">
      {section.locked ? (
        <TeaserBars />
      ) : section.ready && phases.length ? (
        <>
          <BarChart
            label={t('mood.chart', {
              list: phases.map((p) => `${label(p)} ${p.goodPct == null ? '—' : t('mood.pct', { n: num(p.goodPct) })}`).join('، '),
            })}
            height={130}
            max={100}
            bars={phases.map((p, i) => ({
              label: label(p),
              value: p.goodPct ?? 0,
              tone: i === low ? 'bloom' : 'data',
              highlight: true,
              valueLabel: p.goodPct == null ? undefined : t('mood.pct', { n: num(p.goodPct) }),
            }))}
          />
          <span className="an-caption">{t('mood.caption')}</span>
          {tops.length ? (
            tops.map((p) => (
              <p key={p.phase} className="an-card-note">
                {t('mood.topMood', { phase: label(p), mood: p.moods[0].label, pct: num(p.moods[0].pct) })}
              </p>
            ))
          ) : null}
          {m?.finding ? <p className="an-finding">{m.finding.text}</p> : null}
        </>
      ) : (
        <NotReady>{m && m.daysNeeded > 0 ? t('mood.notReadyDays', { n: num(m.daysNeeded) }) : t('mood.notReady')}</NotReady>
      )}
    </HubCard>
  );
}

/** «خواب و حال» (Plus): irritable-mood share after short vs normal nights. */
export function SleepMoodCard({ section }: { section: AnalysisSection<Correlation> }) {
  const { t, num } = useHub();
  const c = section.data;
  return (
    <HubCard title={t('sleep.title')} plus locked={section.locked} href="/analysis/correlations">
      {section.locked ? (
        <TeaserLine tone="brand" />
      ) : section.ready && c ? (
        <>
          <ul className="an-hbars">
            {c.groups.map((g) => (
              <li key={g.key} className={clsx('an-hbar is-wide', g.key === 'sleep_under_6' ? 'nb-tone-brand' : 'nb-tone-data')}>
                <span className="an-hbar-label">{t(`sleep.groups.${g.key === 'sleep_under_6' ? 'sleep_under_6' : 'sleep_6_plus'}`)}</span>
                <span className="an-hbar-track" aria-hidden>
                  <span className="an-hbar-fill" style={{ inlineSize: `${Math.min(100, Math.max(0, g.pct))}%` }} />
                </span>
                <span className="an-hbar-value is-wide">{t('sleep.pct', { n: num(g.pct) })}</span>
              </li>
            ))}
          </ul>
          {c.finding ? <p className="an-finding">{c.finding.text}</p> : null}
        </>
      ) : (
        <NotReady>{t('sleep.notReady', { n: num(c?.minDays ?? 20) })}</NotReady>
      )}
    </HubCard>
  );
}

/**
 * «وزن»: current weight (Lalezar), 30-day change and the 7-day moving average
 * line — from 3 weigh-ins in the last 14 days (§4.1); below that the card
 * says how many more.
 */
export function WeightCard({ section }: { section: AnalysisSection<HubWeight> }) {
  const { t, loc, dec, num } = useHub();
  const w = section.data;
  const points = w?.points ?? [];
  const delta = w?.delta30d != null ? signedDecimal(w.delta30d) : null;
  return (
    <HubCard title={t('weight.title')} href="/analysis/body">
      {section.ready && w?.current != null ? (
        <>
          <span className="an-stat">
            <span className="an-num">{dec(w.current)}</span>
            {delta ? (
              <span className="an-delta">
                {t.rich('weight.delta', {
                  delta: `${delta.sign}${formatDecimal(delta.abs, loc)}`,
                  n: (chunks) => <bdi dir="ltr">{chunks}</bdi>,
                })}
              </span>
            ) : null}
          </span>
          {points.length > 1 ? (
            <LineChart
              label={t('weight.chart')}
              height={90}
              series={[{ values: points.map((p) => p.avg7 ?? p.value), tone: 'data' }]}
              highlightIndex={points.length - 1}
            />
          ) : null}
        </>
      ) : (
        <NotReady>
          {w && w.daysNeeded > 0 ? t('weight.notReadyLogs', { n: num(w.daysNeeded) }) : t('weight.notReady')}
        </NotReady>
      )}
    </HubCard>
  );
}

/**
 * «علائم حیاتی» (§4.1): BP, fasting sugar and heart rate — mean and min–max per type once it has 2 readings on
 * 2 different days; a type below that reads «—». Opens the vitals hub (B-N6-02).
 */
export function VitalsCard({ section }: { section: AnalysisSection<HubVitals> }) {
  const { t, num } = useHub();
  const v = section.data;
  const bp = v?.bloodPressure?.ready ? v.bloodPressure : null;
  const sugar = v?.bloodSugar?.ready ? v.bloodSugar : null;
  const hr = v?.heartRate?.ready ? v.heartRate : null;
  // LTR-isolated «۱۲۰/۸۰» inside the RTL sentence (U+2066 … U+2069).
  const pair = (p: { systolic: number; diastolic: number }) => `\u2066${num(p.systolic)}/${num(p.diastolic)}\u2069`;
  return (
    <HubCard title={t('vitals.title')} href="/vitals">
      {section.ready && v ? (
        <dl className="an-vitals">
          <div>
            <dt>{t('vitals.bp')}</dt>
            <dd className="an-num is-md">
              {bp && bp.systolic != null && bp.diastolic != null ? (
                <bdi dir="ltr">{`${num(bp.systolic)}/${num(bp.diastolic)}`}</bdi>
              ) : (
                t('vitals.none')
              )}
            </dd>
            {bp?.min && bp.max ? (
              <dd className="an-caption">{t('vitals.range', { min: pair(bp.min), max: pair(bp.max) })}</dd>
            ) : null}
          </div>
          <div>
            <dt>{t('vitals.sugar')}</dt>
            <dd className="an-num is-md">{sugar?.avg != null ? num(sugar.avg) : t('vitals.none')}</dd>
            {sugar?.min != null && sugar.max != null ? (
              <dd className="an-caption">{t('vitals.range', { min: num(sugar.min), max: num(sugar.max) })}</dd>
            ) : null}
          </div>
          <div>
            <dt>{t('vitals.hr')}</dt>
            <dd className="an-num is-md">{hr?.avg != null ? num(hr.avg) : t('vitals.none')}</dd>
            {hr?.min != null && hr.max != null ? (
              <dd className="an-caption">{t('vitals.range', { min: num(hr.min), max: num(hr.max) })}</dd>
            ) : null}
          </div>
        </dl>
      ) : (
        <NotReady>
          {v ? t('vitals.notReadyMin', { readings: num(v.minReadings), days: num(v.minDays) }) : t('vitals.notReady')}
        </NotReady>
      )}
    </HubCard>
  );
}

/** «روند آزمایش‌ها» (Plus): empty until lab results exist (B-N6-06). */
export function LabsCard({ section }: { section: AnalysisSection<unknown> }) {
  const { t } = useHub();
  return (
    <HubCard title={t('labs.title')} plus locked={section.locked} href="/analysis/labs">
      {section.locked ? <TeaserLine tone="warm" /> : <NotReady>{t('labs.empty')}</NotReady>}
    </HubCard>
  );
}
